package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
)

var (
	ErrTransport          = errors.New("jsonrpc: transport error")
	ErrInvalidResponse    = errors.New("jsonrpc: invalid response")
	ErrMismatchedID       = errors.New("jsonrpc: mismatched response id")
	ErrUnexpectedResponse = errors.New("jsonrpc: unexpected response")
	ErrMissingResponse    = errors.New("jsonrpc: missing batch response")
	ErrDuplicateResponse  = errors.New("jsonrpc: duplicate batch response")
	ErrEmptyBatch         = errors.New("jsonrpc: empty client batch")
)

type Transport interface {
	RoundTrip(context.Context, []byte) ([]byte, error)
}

type TransportFunc func(context.Context, []byte) ([]byte, error)

func (function TransportFunc) RoundTrip(ctx context.Context, payload []byte) ([]byte, error) {
	return function(ctx, payload)
}

type IDGenerator interface {
	NextID() ID
}

type AtomicIDGenerator struct{ value atomic.Int64 }

func NewAtomicIDGenerator(start int64) *AtomicIDGenerator {
	generator := &AtomicIDGenerator{}
	generator.value.Store(start)
	return generator
}

func (generator *AtomicIDGenerator) NextID() ID {
	return NumberID(json.Number(strconv.FormatInt(generator.value.Add(1), 10)))
}

type ClientOption func(*Client)

func WithIDGenerator(generator IDGenerator) ClientOption {
	return func(client *Client) {
		if generator != nil {
			client.ids = generator
		}
	}
}

type Client struct {
	transport Transport
	ids       IDGenerator
}

func NewClient(transport Transport, options ...ClientOption) *Client {
	client := &Client{transport: transport, ids: NewAtomicIDGenerator(0)}
	for _, option := range options {
		option(client)
	}
	return client
}

func (client *Client) Call(ctx context.Context, method string, params, result any) error {
	request, err := NewRequest(method, params, client.ids.NextID())
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(request)
	reply, err := client.roundTrip(ctx, payload)
	if err != nil {
		return err
	}
	var response Response
	if err := json.Unmarshal(reply, &response); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	if err := response.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	if !request.ID.Equal(response.ID) {
		return ErrMismatchedID
	}
	if response.Error != nil {
		return response.Error
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(response.Result, result); err != nil {
		return fmt.Errorf("%w: result: %v", ErrInvalidResponse, err)
	}
	return nil
}

func Call[T any](ctx context.Context, client *Client, method string, params any) (T, error) {
	var result T
	err := client.Call(ctx, method, params, &result)
	return result, err
}

func (client *Client) Notify(ctx context.Context, method string, params any) error {
	request, err := NewNotification(method, params)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(request)
	reply, err := client.roundTrip(ctx, payload)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(reply)) != 0 {
		return ErrUnexpectedResponse
	}
	return nil
}

type BatchCall struct {
	Method       string
	Params       any
	Result       any
	Notification bool
	Error        *Error

	id ID
}

func (client *Client) Batch(ctx context.Context, calls ...*BatchCall) error {
	if len(calls) == 0 {
		return ErrEmptyBatch
	}
	requests := make([]Request, 0, len(calls))
	pending := make(map[string]*BatchCall, len(calls))
	for _, call := range calls {
		if call == nil {
			return errors.New("jsonrpc: nil batch call")
		}
		call.Error = nil
		var request Request
		var err error
		if call.Notification {
			request, err = NewNotification(call.Method, call.Params)
		} else {
			call.id = client.ids.NextID()
			request, err = NewRequest(call.Method, call.Params, call.id)
			pending[idKey(call.id)] = call
		}
		if err != nil {
			return err
		}
		requests = append(requests, request)
	}
	payload, _ := json.Marshal(requests)
	reply, err := client.roundTrip(ctx, payload)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		if len(bytes.TrimSpace(reply)) != 0 {
			return ErrUnexpectedResponse
		}
		return nil
	}
	trimmed := bytes.TrimSpace(reply)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return ErrInvalidResponse
	}
	var responses []Response
	if err := json.Unmarshal(trimmed, &responses); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	seen := make(map[string]struct{}, len(responses))
	for _, response := range responses {
		if err := response.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidResponse, err)
		}
		key := idKey(response.ID)
		call, ok := pending[key]
		if !ok {
			return ErrMismatchedID
		}
		if _, duplicate := seen[key]; duplicate {
			return ErrDuplicateResponse
		}
		seen[key] = struct{}{}
		if response.Error != nil {
			call.Error = response.Error
			continue
		}
		if call.Result != nil {
			if err := json.Unmarshal(response.Result, call.Result); err != nil {
				return fmt.Errorf("%w: result: %v", ErrInvalidResponse, err)
			}
		}
	}
	if len(seen) != len(pending) {
		return ErrMissingResponse
	}
	return nil
}

func (client *Client) roundTrip(ctx context.Context, payload []byte) ([]byte, error) {
	if client.transport == nil {
		return nil, fmt.Errorf("%w: nil transport", ErrTransport)
	}
	reply, err := client.transport.RoundTrip(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTransport, err)
	}
	return reply, nil
}

func NewRequest(method string, params any, id ID) (Request, error) {
	if err := validateClientMethod(method); err != nil {
		return Request{}, err
	}
	if id.Kind() == IDMissing {
		return Request{}, errors.New("jsonrpc: request id is required")
	}
	encoded, err := encodeParams(params)
	if err != nil {
		return Request{}, err
	}
	return Request{JSONRPC: Version, Method: method, Params: encoded, ID: id, idSet: true}, nil
}

func NewNotification(method string, params any) (Request, error) {
	if err := validateClientMethod(method); err != nil {
		return Request{}, err
	}
	encoded, err := encodeParams(params)
	if err != nil {
		return Request{}, err
	}
	return Request{JSONRPC: Version, Method: method, Params: encoded}, nil
}

func validateClientMethod(method string) error {
	if method == "" || strings.HasPrefix(method, "rpc.") {
		return ErrInvalidMethodName
	}
	return nil
}

func encodeParams(params any) (json.RawMessage, error) {
	if params == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("jsonrpc: encode params: %w", err)
	}
	trimmed := bytes.TrimSpace(encoded)
	if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
		return nil, errors.New("jsonrpc: params must encode as an object or array")
	}
	return encoded, nil
}

func idKey(id ID) string { return strconv.Itoa(int(id.Kind())) + ":" + string(id.raw) }
