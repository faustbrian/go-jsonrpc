package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"runtime/debug"
	"strings"
	"sync"
	"unicode/utf8"
)

// Registration errors identify reserved method names, duplicate methods, and
// nil handlers.
var (
	// ErrInvalidMethodName identifies rpc. application registration or non-rpc. system registration.
	ErrInvalidMethodName = errors.New("jsonrpc: invalid method name")
	// ErrMethodAlreadyRegistered identifies an attempt to replace an existing registry name.
	ErrMethodAlreadyRegistered = errors.New("jsonrpc: method already registered")
	// ErrNilHandler identifies registration without a callable handler.
	ErrNilHandler               = errors.New("jsonrpc: nil handler")
	errDispatchResponseTooLarge = errors.New("jsonrpc: dispatch response too large")
	parameterNames              sync.Map
)

const (
	defaultMaxDispatchBytes         int64 = 4 << 20
	defaultMaxDispatchResponseBytes int64 = 4 << 20
	defaultMaxBatchItems                  = 1024
	defaultMaxNestingDepth                = 10_000
)

// Handler implements one JSON-RPC method synchronously. Params is borrowed for
// the invocation; copy bytes before retaining or mutating them. Handlers own
// cooperative cancellation, authorization, replay safety and bounded returned
// values/custom marshaling. Concurrent dispatch may invoke a handler concurrently.
type Handler func(context.Context, json.RawMessage) (any, error)

// Middleware wraps a Handler with cross-cutting behavior. Its constructor runs
// for each method execution, outside registry locks, in reverse wrapping order.
// Both it and its returned handler must support the caller's concurrency; neither
// gets automatic timeouts or bounded recursion from Dispatcher.
type Middleware func(Handler) Handler

// ErrorMapper converts an ordinary application error into public JSON-RPC data.
// It runs synchronously, may be invoked concurrently and owns the public message/
// data disclosure policy. Dispatcher contains its panic or invalid/nil result;
// a directly returned or wrapped *Error bypasses this mapper.
type ErrorMapper func(error) *Error

// Hooks observe the complete dispatcher lifecycle, including protocol errors
// that occur before a Handler or Middleware can run. A nil Request represents
// an unparseable or invalid request. Notifications provide an internal outcome
// to OnResponse even though that response is never placed on the wire.
// Callbacks run synchronously without registry locks and may run concurrently
// across dispatches. Panics are contained, not blocking or unbounded work.
type Hooks struct {
	// OnRequest observes a copied validated request or nil for an invalid
	// envelope and may return a derived context for subsequent processing.
	// Params/ID raw bytes are copied for this callback. Nil or panic preserves ctx;
	// any other returned context is used without enforcing the original deadline.
	// Derive it from ctx to preserve cancellation and caller context values.
	OnRequest func(context.Context, *Request) context.Context
	// OnResponse observes copied response fields/raw buffers, including notification
	// outcomes, before final wire encoding. Request is borrowed, not copied: treat
	// it and its ID/Params storage as read-only. Mutating/re-decoding Request.ID can
	// alter shared response ID bytes. A local Error cause also remains borrowed
	// and may carry sensitive error or panic-stack data. Copy before retention.
	// Observed outcomes may differ from final aggregate/encoding-limit fallbacks.
	OnResponse func(context.Context, *Request, *Response)
}

type requestContextKey struct{}

// RequestFromContext returns the validated request made available to the
// active middleware or handler. ctx must be nonnil. The returned value is a
// shallow copy sharing Params and ID storage; treat it as read-only and create
// independent copies before retention or mutation.
func RequestFromContext(ctx context.Context) (Request, bool) {
	request, ok := ctx.Value(requestContextKey{}).(Request)
	return request, ok
}

// Registry is a concurrency-safe method registry. Its zero value is ready for
// use. A Registry must not be copied after first use.
type Registry struct {
	mu      sync.RWMutex
	methods map[string]Handler
}

// NewRegistry constructs an empty registry.
func NewRegistry() *Registry { return &Registry{methods: make(map[string]Handler)} }

// Register adds handler under name. Names beginning with rpc. are reserved,
// and an existing name cannot be replaced. Empty application names are valid.
// The registry borrows the function/captured state; it does not run the handler
// under its lock or provide synchronization for that state.
func (r *Registry) Register(name string, handler Handler) error {
	if strings.HasPrefix(name, "rpc.") {
		return ErrInvalidMethodName
	}
	return r.register(name, handler)
}

// RegisterSystem adds an explicitly reserved rpc.* handler. This method is
// intended for protocol extensions such as OpenRPC's rpc.discover and rejects
// application method names.
func (r *Registry) RegisterSystem(name string, handler Handler) error {
	if !strings.HasPrefix(name, "rpc.") {
		return ErrInvalidMethodName
	}
	return r.register(name, handler)
}

func (r *Registry) register(name string, handler Handler) error {
	if handler == nil {
		return ErrNilHandler
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.methods == nil {
		r.methods = make(map[string]Handler)
	}
	if _, exists := r.methods[name]; exists {
		return fmt.Errorf("%w: %s", ErrMethodAlreadyRegistered, name)
	}
	r.methods[name] = handler
	return nil
}

// Lookup returns the borrowed handler under name after releasing the registry
// read lock. The caller owns its invocation and any captured mutable state.
func (r *Registry) Lookup(name string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.methods[name]
	return handler, ok
}

// DispatcherOption configures a Dispatcher during construction.
type DispatcherOption func(*Dispatcher)

// WithMiddleware appends function values in outermost-to-innermost order when
// the option runs. Captured state is not cloned; nil entries are skipped at execution.
func WithMiddleware(middleware ...Middleware) DispatcherOption {
	return func(dispatcher *Dispatcher) {
		dispatcher.middleware = append(dispatcher.middleware, middleware...)
	}
}

// WithErrorMapper replaces the default internal-error mapper. Returning nil or
// an invalid error is contained as an internal error.
func WithErrorMapper(mapper ErrorMapper) DispatcherOption {
	return func(dispatcher *Dispatcher) { dispatcher.errorMapper = mapper }
}

// WithHooks installs lifecycle observers. Hook panics are contained and hook
// response fields are copied, but OnResponse's request/cause are borrowed and
// must be treated as read-only to avoid changing shared state or wire IDs.
func WithHooks(hooks Hooks) DispatcherOption {
	return func(dispatcher *Dispatcher) { dispatcher.hooks = hooks }
}

// WithMaxDispatchBytes changes the dispatcher's four-MiB payload limit.
// Non-positive values leave the existing bound unchanged.
func WithMaxDispatchBytes(limit int64) DispatcherOption {
	return func(dispatcher *Dispatcher) {
		if limit > 0 {
			dispatcher.maxDispatchBytes = limit
		}
	}
}

// WithMaxDispatchResponseBytes changes the dispatcher's four-MiB encoded
// response limit. Values smaller than the minimal protocol error use that
// minimal size so a valid bounded failure can still be returned. Non-positive
// values leave the limit unchanged. Encoding can allocate before this check;
// callers must bound handler values and custom marshaling (JSONRPC-DEC-011).
func WithMaxDispatchResponseBytes(limit int64) DispatcherOption {
	return func(dispatcher *Dispatcher) {
		if limit <= 0 {
			return
		}
		minimum := int64(len(internalErrorBatchResponse))
		dispatcher.maxResponseBytes = max(limit, minimum)
	}
}

// WithMaxBatchItems changes the dispatcher's default limit of 1,024 members.
// Non-positive values leave the existing bound unchanged.
func WithMaxBatchItems(limit int) DispatcherOption {
	return func(dispatcher *Dispatcher) {
		if limit > 0 {
			dispatcher.maxBatchItems = limit
		}
	}
}

// WithMaxNestingDepth changes the maximum number of nested JSON arrays and
// objects accepted before dispatch. The default matches encoding/json's
// built-in maximum depth of 10,000. Non-positive values leave the bound
// unchanged; raising it does not raise the standard JSON parser's own ceiling.
func WithMaxNestingDepth(limit int) DispatcherOption {
	return func(dispatcher *Dispatcher) {
		if limit > 0 {
			dispatcher.maxNestingDepth = limit
		}
	}
}

// Dispatcher validates and executes JSON-RPC requests and batches. Its
// configuration is immutable after construction; its Registry may be updated
// concurrently. Use NewDispatcher; the zero value is not ready for dispatch.
// It borrows the registry and callback state, which must be concurrency-safe.
// Options are construction-only; do not mutate configuration after publication.
type Dispatcher struct {
	registry         *Registry
	middleware       []Middleware
	errorMapper      ErrorMapper
	hooks            Hooks
	maxDispatchBytes int64
	maxResponseBytes int64
	maxBatchItems    int
	maxNestingDepth  int
}

// NewDispatcher constructs a bounded dispatcher. A nil registry is replaced
// by an empty registry, and nil options are ignored.
func NewDispatcher(registry *Registry, options ...DispatcherOption) *Dispatcher {
	if registry == nil {
		registry = NewRegistry()
	}
	dispatcher := &Dispatcher{
		registry:         registry,
		maxDispatchBytes: defaultMaxDispatchBytes,
		maxResponseBytes: defaultMaxDispatchResponseBytes,
		maxBatchItems:    defaultMaxBatchItems,
		maxNestingDepth:  defaultMaxNestingDepth,
		errorMapper: func(err error) *Error {
			return InternalError().WithCause(err)
		},
	}
	for _, option := range options {
		if option != nil {
			option(dispatcher)
		}
	}
	return dispatcher
}

// Dispatch processes one JSON-RPC message. The boolean reports whether the
// caller must send the returned response; notifications intentionally return
// no response for valid notifications, even on handler failure. Invalid id-less
// envelopes still return null-ID errors (JSONRPC-DEC-001); explicit null IDs are
// requests (JSONRPC-DEC-002). ctx must be nonnil; cancellation is propagated to
// callbacks, not preflight admission or forced preemption.
//
// Payload is borrowed for this synchronous call; returned bytes are caller-owned.
// Batches currently execute sequentially, but ordering is not a public guarantee
// (JSONRPC-DEC-006). Aggregate response overflow still runs remaining handlers
// and returns one null-ID internal-error member, so side effects may complete
// without correlated outcomes (JSONRPC-DEC-011); do not blindly retry.
//
// Notification and batch behavior follow:
//   - https://www.jsonrpc.org/specification#notification
//   - https://www.jsonrpc.org/specification#batch
func (d *Dispatcher) Dispatch(ctx context.Context, payload []byte) ([]byte, bool) {
	if int64(len(payload)) > d.maxDispatchBytes {
		return d.failure(ctx, RequestLimitExceeded())
	}
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 || !utf8.Valid(trimmed) {
		return d.failure(ctx, ParseError())
	}
	if exceedsNestingDepth(trimmed, d.maxNestingDepth) {
		return d.failure(ctx, RequestLimitExceeded())
	}
	if !json.Valid(trimmed) {
		return d.failure(ctx, ParseError())
	}
	if trimmed[0] == '[' {
		if batchExceedsLimit(trimmed, d.maxBatchItems) {
			return d.failure(ctx, RequestLimitExceeded())
		}
		return d.dispatchBatch(ctx, trimmed)
	}
	if trimmed[0] != '{' {
		return d.failure(ctx, InvalidRequest())
	}
	response, ok := d.dispatchSingleValidated(ctx, trimmed)
	if !ok {
		return nil, false
	}
	return d.marshalResponse(response), true
}

// DispatchSingle processes one non-batch JSON-RPC message and returns its
// typed response before final wire encoding. The boolean is false for valid
// notifications, which return a zero Response; batches return Invalid Request.
// ctx/payload ownership and cancellation match Dispatch. Result shaping still
// encodes to enforce the response-size bound; this is not a zero-encoding API.
// The returned Response can retain a handler-owned Error/cause, not a deep copy.
func (d *Dispatcher) DispatchSingle(
	ctx context.Context,
	payload []byte,
) (Response, bool) {
	if int64(len(payload)) > d.maxDispatchBytes {
		return d.failureResponse(ctx, RequestLimitExceeded())
	}
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 || !utf8.Valid(trimmed) {
		return d.failureResponse(ctx, ParseError())
	}
	if exceedsNestingDepth(trimmed, d.maxNestingDepth) {
		return d.failureResponse(ctx, RequestLimitExceeded())
	}
	if !json.Valid(trimmed) {
		return d.failureResponse(ctx, ParseError())
	}
	if trimmed[0] != '{' {
		return d.failureResponse(ctx, InvalidRequest())
	}

	return d.dispatchSingleValidated(ctx, trimmed)
}

func (d *Dispatcher) dispatchSingleValidated(
	ctx context.Context,
	payload []byte,
) (Response, bool) {
	return d.dispatchItem(ctx, payload)
}

func exceedsNestingDepth(payload []byte, limit int) bool {
	depth := 0
	inString := false
	escaped := false
	for _, character := range payload {
		if inString {
			switch {
			case escaped:
				escaped = false
			case character == '\\':
				escaped = true
			case character == '"':
				inString = false
			}
			continue
		}
		if character == '"' {
			inString = true
			continue
		}
		if character == '{' || character == '[' {
			depth++
			if depth > limit {
				return true
			}
			continue
		}
		if character == '}' || character == ']' {
			depth--
		}
	}
	return false
}

func batchExceedsLimit(payload []byte, limit int) bool {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	_, _ = decoder.Token()
	count := 0
	var item json.RawMessage
	for decoder.More() {
		count++
		if count > limit {
			return true
		}
		_ = decoder.Decode(&item)
	}
	return false
}

func (d *Dispatcher) dispatchBatch(ctx context.Context, payload []byte) ([]byte, bool) {
	var items []json.RawMessage
	_ = json.Unmarshal(payload, &items)
	if len(items) == 0 {
		return d.failure(ctx, InvalidRequest())
	}
	var encoded bytes.Buffer
	encoded.WriteByte('[')
	replies := 0
	overflow := false
	for _, item := range items {
		response, ok := d.dispatchItem(ctx, item)
		if ok {
			replies++
			member := d.marshalResponse(response)
			required := int64(encoded.Len() + len(member) + 1)
			if replies > 1 {
				required++
			}
			if overflow || required > d.maxResponseBytes {
				overflow = true
				continue
			}
			if replies > 1 {
				encoded.WriteByte(',')
			}
			encoded.Write(member)
		}
	}
	if replies == 0 {
		return nil, false
	}
	if overflow {
		return []byte(internalErrorBatchResponse), true
	}
	encoded.WriteByte(']')
	return encoded.Bytes(), true
}

func (d *Dispatcher) dispatchItem(ctx context.Context, payload []byte) (response Response, reply bool) {
	if len(payload) == 0 || payload[0] != '{' {
		response = errorResponse(NullID(), InvalidRequest())
		ctx = d.begin(ctx, nil)
		d.finish(ctx, nil, &response)
		return response, true
	}
	var request Request
	if err := json.Unmarshal(payload, &request); err != nil {
		response = errorResponse(NullID(), InvalidRequest().WithCause(err))
		ctx = d.begin(ctx, nil)
		d.finish(ctx, nil, &response)
		return response, true
	}
	if rpcErr := request.Validate(); rpcErr != nil {
		response = errorResponse(NullID(), rpcErr)
		ctx = d.begin(ctx, nil)
		d.finish(ctx, nil, &response)
		return response, true
	}
	ctx = d.begin(ctx, &request)
	if request.IsNotification() {
		outcome := d.execute(ctx, request)
		d.finish(ctx, &request, &outcome)
		return Response{}, false
	}
	response = d.execute(ctx, request)
	d.finish(ctx, &request, &response)
	return response, true
}

func (d *Dispatcher) execute(ctx context.Context, request Request) (response Response) {
	ctx = context.WithValue(ctx, requestContextKey{}, request)
	response = Response{JSONRPC: Version, ID: request.ID, idSet: true}
	defer func() {
		if recovered := recover(); recovered != nil {
			response = errorResponse(request.ID, InternalError().WithCause(fmt.Errorf("panic: %v\n%s", recovered, debug.Stack())))
		}
		response = d.boundResponse(response)
	}()

	handler, ok := d.registry.Lookup(request.Method)
	if !ok {
		return errorResponse(request.ID, MethodNotFound())
	}
	for index := len(d.middleware) - 1; index >= 0; index-- {
		if d.middleware[index] != nil {
			handler = d.middleware[index](handler)
		}
	}
	result, err := handler(ctx, request.Params)
	if err != nil {
		var rpcErr *Error
		if errors.As(err, &rpcErr) {
			return errorResponse(request.ID, rpcErr)
		}
		mapped := d.errorMapper(err)
		if mapped == nil {
			mapped = InternalError().WithCause(err)
		}
		return errorResponse(request.ID, mapped)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return errorResponse(request.ID, InternalError().WithCause(err))
	}
	response.Result = encoded
	response.resultSet = true
	return response
}

func (d *Dispatcher) failure(ctx context.Context, rpcErr *Error) ([]byte, bool) {
	response, reply := d.failureResponse(ctx, rpcErr)

	return d.marshalResponse(response), reply
}

func (d *Dispatcher) failureResponse(
	ctx context.Context,
	rpcErr *Error,
) (Response, bool) {
	response := errorResponse(NullID(), rpcErr)
	ctx = d.begin(ctx, nil)
	d.finish(ctx, nil, &response)

	return response, true
}

func (d *Dispatcher) begin(ctx context.Context, request *Request) (observed context.Context) {
	observed = ctx
	if d.hooks.OnRequest == nil {
		return observed
	}
	defer func() {
		if recover() != nil || observed == nil {
			observed = ctx
		}
	}()
	if request == nil {
		return d.hooks.OnRequest(ctx, nil)
	}
	requestCopy := *request
	requestCopy.Params = append(json.RawMessage(nil), request.Params...)
	requestCopy.ID.raw = append(json.RawMessage(nil), request.ID.raw...)
	return d.hooks.OnRequest(ctx, &requestCopy)
}

func (d *Dispatcher) finish(ctx context.Context, request *Request, response *Response) {
	if d.hooks.OnResponse == nil {
		return
	}
	defer func() { _ = recover() }()
	observed := *response
	observed.Result = append(json.RawMessage(nil), response.Result...)
	observed.ID.raw = append(json.RawMessage(nil), response.ID.raw...)
	if response.Error != nil {
		rpcErr := *response.Error
		rpcErr.Data = append(json.RawMessage(nil), response.Error.Data...)
		observed.Error = &rpcErr
	}
	d.hooks.OnResponse(ctx, request, &observed)
}

func errorResponse(id ID, rpcErr *Error) Response {
	if rpcErr == nil || !rpcErr.valid() || (len(rpcErr.Data) > 0 && !json.Valid(rpcErr.Data)) {
		rpcErr = InternalError().WithCause(rpcErr)
	}
	return Response{
		JSONRPC:  Version,
		Error:    rpcErr,
		ID:       id,
		errorSet: true,
		idSet:    true,
	}
}

func marshalResponse(response Response) []byte {
	encoded, err := json.Marshal(response)
	if err == nil {
		return encoded
	}
	return []byte(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"},"id":null}`)
}

const internalErrorResponse = `{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"},"id":null}`
const internalErrorBatchResponse = `[` + internalErrorResponse + `]`

func (d *Dispatcher) boundResponse(response Response) Response {
	encoded, err := json.Marshal(response)
	if err == nil && int64(len(encoded)) <= d.maxResponseBytes {
		return response
	}
	return errorResponse(response.ID, InternalError().WithCause(errDispatchResponseTooLarge))
}

func (d *Dispatcher) marshalResponse(response Response) []byte {
	encoded := marshalResponse(d.boundResponse(response))
	if int64(len(encoded)) <= d.maxResponseBytes {
		return encoded
	}
	return []byte(internalErrorResponse)
}

// DecodeParams strictly decodes params as T. Duplicate or unknown named
// members, invalid UTF-8, malformed JSON and trailing data return InvalidParams.
// Exact top-level field names apply when T is a struct (including pointer forms);
// nested field case matching and custom unmarshaling retain encoding/json rules.
// Duplicate object names are rejected at every depth (JSONRPC-DEC-008).
// Direct callers must bound params before parsing. On error T may be partially
// decoded; use it only after checking the returned protocol error.
func DecodeParams[T any](params json.RawMessage) (T, *Error) {
	var value T
	trimmed := bytes.TrimSpace(params)
	if len(trimmed) == 0 {
		return value, InvalidParams()
	}
	if err := rejectDuplicateMembers(trimmed); err != nil {
		return value, InvalidParams().WithCause(err)
	}
	if !namedParameterNamesMatch[T](params) {
		return value, InvalidParams()
	}
	decoder := json.NewDecoder(bytes.NewReader(params))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, InvalidParams().WithCause(err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return value, InvalidParams()
	}
	return value, nil
}

func namedParameterNamesMatch[T any](params json.RawMessage) bool {
	trimmed := bytes.TrimSpace(params)
	if trimmed[0] != '{' {
		return true
	}
	names, structured := parameterNameSet(reflect.TypeFor[T]())
	if !structured {
		return true
	}
	var object map[string]json.RawMessage
	if json.Unmarshal(trimmed, &object) != nil {
		return true
	}
	for name := range object {
		if _, ok := names[name]; !ok {
			return false
		}
	}
	return true
}

func parameterNameSet(parameterType reflect.Type) (map[string]struct{}, bool) {
	for parameterType.Kind() == reflect.Pointer {
		parameterType = parameterType.Elem()
	}
	if parameterType.Kind() != reflect.Struct {
		return nil, false
	}
	if cached, ok := parameterNames.Load(parameterType); ok {
		return cached.(map[string]struct{}), true
	}
	names := make(map[string]struct{})
	for _, field := range reflect.VisibleFields(parameterType) {
		if !field.IsExported() {
			continue
		}
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "-" {
			continue
		}
		fieldType := field.Type
		for fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		if name == "" && field.Anonymous && fieldType.Kind() == reflect.Struct {
			continue
		}
		if name == "" {
			name = field.Name
		}
		names[name] = struct{}{}
	}
	parameterNames.Store(parameterType, names)
	return names, true
}
