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
)

var (
	ErrInvalidMethodName       = errors.New("jsonrpc: invalid method name")
	ErrMethodAlreadyRegistered = errors.New("jsonrpc: method already registered")
	ErrNilHandler              = errors.New("jsonrpc: nil handler")
	parameterNames             sync.Map
)

type Handler func(context.Context, json.RawMessage) (any, error)
type Middleware func(Handler) Handler
type ErrorMapper func(error) *Error

// Hooks observe the complete dispatcher lifecycle, including protocol errors
// that occur before a Handler or Middleware can run. A nil Request represents
// an unparseable or invalid request. A nil Response represents a notification.
type Hooks struct {
	OnRequest  func(context.Context, *Request) context.Context
	OnResponse func(context.Context, *Request, *Response)
}

type requestContextKey struct{}

func RequestFromContext(ctx context.Context) (Request, bool) {
	request, ok := ctx.Value(requestContextKey{}).(Request)
	return request, ok
}

type Registry struct {
	mu      sync.RWMutex
	methods map[string]Handler
}

func NewRegistry() *Registry { return &Registry{methods: make(map[string]Handler)} }

func (r *Registry) Register(name string, handler Handler) error {
	if strings.HasPrefix(name, "rpc.") {
		return ErrInvalidMethodName
	}
	if handler == nil {
		return ErrNilHandler
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.methods[name]; exists {
		return fmt.Errorf("%w: %s", ErrMethodAlreadyRegistered, name)
	}
	r.methods[name] = handler
	return nil
}

func (r *Registry) Lookup(name string) (Handler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	handler, ok := r.methods[name]
	return handler, ok
}

type DispatcherOption func(*Dispatcher)

func WithMiddleware(middleware ...Middleware) DispatcherOption {
	return func(dispatcher *Dispatcher) {
		dispatcher.middleware = append(dispatcher.middleware, middleware...)
	}
}

func WithErrorMapper(mapper ErrorMapper) DispatcherOption {
	return func(dispatcher *Dispatcher) { dispatcher.errorMapper = mapper }
}

func WithHooks(hooks Hooks) DispatcherOption {
	return func(dispatcher *Dispatcher) { dispatcher.hooks = hooks }
}

type Dispatcher struct {
	registry    *Registry
	middleware  []Middleware
	errorMapper ErrorMapper
	hooks       Hooks
}

func NewDispatcher(registry *Registry, options ...DispatcherOption) *Dispatcher {
	if registry == nil {
		registry = NewRegistry()
	}
	dispatcher := &Dispatcher{
		registry: registry,
		errorMapper: func(err error) *Error {
			return InternalError().WithCause(err)
		},
	}
	for _, option := range options {
		option(dispatcher)
	}
	return dispatcher
}

// Dispatch processes one JSON-RPC message. The boolean reports whether the
// caller must send the returned response; notifications intentionally return
// no response.
func (d *Dispatcher) Dispatch(ctx context.Context, payload []byte) ([]byte, bool) {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 || !json.Valid(trimmed) {
		return d.failure(ctx, ParseError())
	}
	if trimmed[0] == '[' {
		return d.dispatchBatch(ctx, trimmed)
	}
	if trimmed[0] != '{' {
		return d.failure(ctx, InvalidRequest())
	}
	response, ok := d.dispatchItem(ctx, trimmed)
	if !ok {
		return nil, false
	}
	return marshalResponse(response), true
}

func (d *Dispatcher) dispatchBatch(ctx context.Context, payload []byte) ([]byte, bool) {
	var items []json.RawMessage
	_ = json.Unmarshal(payload, &items)
	if len(items) == 0 {
		return d.failure(ctx, InvalidRequest())
	}
	responses := make([]Response, 0, len(items))
	for _, item := range items {
		response, ok := d.dispatchItem(ctx, item)
		if ok {
			responses = append(responses, response)
		}
	}
	if len(responses) == 0 {
		return nil, false
	}
	encoded, _ := json.Marshal(responses)
	return encoded, true
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
		d.execute(ctx, request)
		d.finish(ctx, &request, nil)
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
	response := errorResponse(NullID(), rpcErr)
	ctx = d.begin(ctx, nil)
	d.finish(ctx, nil, &response)
	return marshalResponse(response), true
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
	return d.hooks.OnRequest(ctx, request)
}

func (d *Dispatcher) finish(ctx context.Context, request *Request, response *Response) {
	if d.hooks.OnResponse == nil {
		return
	}
	defer func() { _ = recover() }()
	d.hooks.OnResponse(ctx, request, response)
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

func DecodeParams[T any](params json.RawMessage) (T, *Error) {
	var value T
	if len(params) == 0 {
		return value, InvalidParams()
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
