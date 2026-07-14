package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	ErrInvalidMethodName       = errors.New("jsonrpc: invalid method name")
	ErrMethodAlreadyRegistered = errors.New("jsonrpc: method already registered")
	ErrNilHandler              = errors.New("jsonrpc: nil handler")
)

type Handler func(context.Context, json.RawMessage) (any, error)
type Middleware func(Handler) Handler
type ErrorMapper func(error) *Error

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
	if name == "" || strings.HasPrefix(name, "rpc.") {
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

type Dispatcher struct {
	registry    *Registry
	middleware  []Middleware
	errorMapper ErrorMapper
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
		return marshalResponse(errorResponse(NullID(), ParseError())), true
	}
	if trimmed[0] == '[' {
		return d.dispatchBatch(ctx, trimmed)
	}
	if trimmed[0] != '{' {
		return marshalResponse(errorResponse(NullID(), InvalidRequest())), true
	}
	response, ok := d.dispatchItem(ctx, trimmed)
	if !ok {
		return nil, false
	}
	return marshalResponse(response), true
}

func (d *Dispatcher) dispatchBatch(ctx context.Context, payload []byte) ([]byte, bool) {
	var items []json.RawMessage
	if err := json.Unmarshal(payload, &items); err != nil {
		return marshalResponse(errorResponse(NullID(), ParseError())), true
	}
	if len(items) == 0 {
		return marshalResponse(errorResponse(NullID(), InvalidRequest())), true
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
	encoded, err := json.Marshal(responses)
	if err != nil {
		return marshalResponse(errorResponse(NullID(), InternalError().WithCause(err))), true
	}
	return encoded, true
}

func (d *Dispatcher) dispatchItem(ctx context.Context, payload []byte) (response Response, reply bool) {
	if len(payload) == 0 || payload[0] != '{' {
		return errorResponse(NullID(), InvalidRequest()), true
	}
	var request Request
	if err := json.Unmarshal(payload, &request); err != nil {
		return errorResponse(NullID(), InvalidRequest().WithCause(err)), true
	}
	if rpcErr := request.Validate(); rpcErr != nil {
		return errorResponse(NullID(), rpcErr), true
	}
	if request.IsNotification() {
		d.execute(ctx, request)
		return Response{}, false
	}
	return d.execute(ctx, request), true
}

func (d *Dispatcher) execute(ctx context.Context, request Request) (response Response) {
	ctx = context.WithValue(ctx, requestContextKey{}, request)
	response = Response{JSONRPC: Version, ID: request.ID, idSet: true}
	defer func() {
		if recovered := recover(); recovered != nil {
			response = errorResponse(request.ID, InternalError().WithCause(fmt.Errorf("panic: %v", recovered)))
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

func errorResponse(id ID, rpcErr *Error) Response {
	return Response{
		JSONRPC:  Version,
		Error:    rpcErr,
		ID:       id,
		errorSet: true,
		idSet:    true,
	}
}

func marshalResponse(response Response) []byte {
	encoded, _ := json.Marshal(response)
	return encoded
}

func DecodeParams[T any](params json.RawMessage) (T, *Error) {
	var value T
	if len(params) == 0 {
		return value, InvalidParams()
	}
	decoder := json.NewDecoder(bytes.NewReader(params))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, InvalidParams().WithCause(err)
	}
	if decoder.More() {
		return value, InvalidParams()
	}
	return value, nil
}
