package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRegistryRegistration(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	handler := func(context.Context, json.RawMessage) (any, error) { return "ok", nil }
	if err := registry.Register("ping", handler); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if _, ok := registry.Lookup("ping"); !ok {
		t.Error("Lookup() did not find registered method")
	}
	if err := registry.Register("ping", handler); !errors.Is(err, ErrMethodAlreadyRegistered) {
		t.Errorf("duplicate Register() error = %v", err)
	}
	for _, name := range []string{"", "rpc.internal"} {
		if err := registry.Register(name, handler); !errors.Is(err, ErrInvalidMethodName) {
			t.Errorf("Register(%q) error = %v", name, err)
		}
	}
	if err := registry.Register("nil", nil); !errors.Is(err, ErrNilHandler) {
		t.Errorf("Register(nil) error = %v", err)
	}
}

func TestDispatcherSingleRequests(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	_ = registry.Register("subtract", func(_ context.Context, params json.RawMessage) (any, error) {
		var values []int
		if err := json.Unmarshal(params, &values); err != nil || len(values) != 2 {
			return nil, InvalidParams()
		}
		return values[0] - values[1], nil
	})
	dispatcher := NewDispatcher(registry)

	tests := []struct {
		name     string
		input    string
		response string
		hasReply bool
	}{
		{
			name:     "positional parameters",
			input:    `{"jsonrpc":"2.0","method":"subtract","params":[42,23],"id":1}`,
			response: `{"jsonrpc":"2.0","result":19,"id":1}`,
			hasReply: true,
		},
		{
			name:     "method not found",
			input:    `{"jsonrpc":"2.0","method":"missing","id":"a"}`,
			response: `{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":"a"}`,
			hasReply: true,
		},
		{
			name:     "notification",
			input:    `{"jsonrpc":"2.0","method":"subtract","params":[42,23]}`,
			hasReply: false,
		},
		{
			name:     "null id is a request",
			input:    `{"jsonrpc":"2.0","method":"subtract","params":[42,23],"id":null}`,
			response: `{"jsonrpc":"2.0","result":19,"id":null}`,
			hasReply: true,
		},
		{
			name:     "invalid params",
			input:    `{"jsonrpc":"2.0","method":"subtract","params":[1],"id":1}`,
			response: `{"jsonrpc":"2.0","error":{"code":-32602,"message":"Invalid params"},"id":1}`,
			hasReply: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			response, hasReply := dispatcher.Dispatch(context.Background(), []byte(tt.input))
			if hasReply != tt.hasReply {
				t.Fatalf("Dispatch() hasReply = %v, want %v", hasReply, tt.hasReply)
			}
			assertJSONEqual(t, response, []byte(tt.response))
		})
	}
}

func TestDispatcherProtocolErrors(t *testing.T) {
	t.Parallel()

	dispatcher := NewDispatcher(NewRegistry())
	tests := []struct {
		name     string
		input    string
		response string
	}{
		{name: "parse error", input: `{`, response: `{"jsonrpc":"2.0","error":{"code":-32700,"message":"Parse error"},"id":null}`},
		{name: "scalar", input: `1`, response: `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null}`},
		{name: "wrong version", input: `{"jsonrpc":"1.0","method":"x","id":1}`, response: `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null}`},
		{name: "invalid id", input: `{"jsonrpc":"2.0","method":"x","id":true}`, response: `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			response, hasReply := dispatcher.Dispatch(context.Background(), []byte(tt.input))
			if !hasReply {
				t.Fatal("Dispatch() unexpectedly omitted protocol error")
			}
			assertJSONEqual(t, response, []byte(tt.response))
		})
	}
}

func TestDispatcherBatch(t *testing.T) {
	t.Parallel()

	var notifications atomic.Int64
	registry := NewRegistry()
	_ = registry.Register("sum", func(_ context.Context, params json.RawMessage) (any, error) {
		var values []int
		if err := json.Unmarshal(params, &values); err != nil {
			return nil, InvalidParams()
		}
		total := 0
		for _, value := range values {
			total += value
		}
		return total, nil
	})
	_ = registry.Register("notify", func(context.Context, json.RawMessage) (any, error) {
		notifications.Add(1)
		return nil, nil
	})
	dispatcher := NewDispatcher(registry)

	input := `[
		{"jsonrpc":"2.0","method":"sum","params":[1,2,4],"id":"1"},
		{"jsonrpc":"2.0","method":"notify","params":[7]},
		{"jsonrpc":"2.0","method":"missing","id":"2"},
		{"foo":"boo"},
		1
	]`
	want := `[
		{"jsonrpc":"2.0","result":7,"id":"1"},
		{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":"2"},
		{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null},
		{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null}
	]`
	response, hasReply := dispatcher.Dispatch(context.Background(), []byte(input))
	if !hasReply {
		t.Fatal("Dispatch() omitted batch response")
	}
	assertJSONEqual(t, response, []byte(want))
	if notifications.Load() != 1 {
		t.Errorf("notifications executed = %d, want 1", notifications.Load())
	}
}

func TestDispatcherBatchEdgeCases(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	_ = registry.Register("notify", func(context.Context, json.RawMessage) (any, error) {
		return nil, nil
	})
	dispatcher := NewDispatcher(registry)

	response, hasReply := dispatcher.Dispatch(context.Background(), []byte(`[]`))
	if !hasReply {
		t.Fatal("empty batch unexpectedly omitted response")
	}
	assertJSONEqual(t, response, []byte(`{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null}`))

	response, hasReply = dispatcher.Dispatch(context.Background(), []byte(`[
		{"jsonrpc":"2.0","method":"notify"},
		{"jsonrpc":"2.0","method":"notify"}
	]`))
	if hasReply || response != nil {
		t.Errorf("notification-only batch = (%s, %v), want no response", response, hasReply)
	}
}

func TestDispatcherErrorMappingPanicRecoveryAndMiddleware(t *testing.T) {
	t.Parallel()

	type contextKey string
	const key contextKey = "trace"
	registry := NewRegistry()
	_ = registry.Register("failure", func(ctx context.Context, _ json.RawMessage) (any, error) {
		if ctx.Value(key) != "present" {
			t.Error("middleware context did not reach handler")
		}
		return nil, errors.New("secret database detail")
	})
	_ = registry.Register("panic", func(context.Context, json.RawMessage) (any, error) {
		panic("boom")
	})

	var events []string
	middleware := func(next Handler) Handler {
		return func(ctx context.Context, params json.RawMessage) (any, error) {
			events = append(events, "before")
			result, err := next(context.WithValue(ctx, key, "present"), params)
			events = append(events, "after")
			return result, err
		}
	}
	dispatcher := NewDispatcher(
		registry,
		WithMiddleware(middleware),
		WithErrorMapper(func(err error) *Error {
			return NewError(-32001, "Dependency unavailable").WithCause(err)
		}),
	)

	response, _ := dispatcher.Dispatch(context.Background(), []byte(`{"jsonrpc":"2.0","method":"failure","id":1}`))
	assertJSONEqual(t, response, []byte(`{"jsonrpc":"2.0","error":{"code":-32001,"message":"Dependency unavailable"},"id":1}`))
	if !reflect.DeepEqual(events, []string{"before", "after"}) {
		t.Errorf("middleware events = %v", events)
	}

	response, _ = dispatcher.Dispatch(context.Background(), []byte(`{"jsonrpc":"2.0","method":"panic","id":2}`))
	assertJSONEqual(t, response, []byte(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"},"id":2}`))
}

func TestDecodeParams(t *testing.T) {
	t.Parallel()

	type input struct {
		Name string `json:"name"`
	}
	decoded, rpcErr := DecodeParams[input](json.RawMessage(`{"name":"Ada"}`))
	if rpcErr != nil || decoded.Name != "Ada" {
		t.Fatalf("DecodeParams() = (%#v, %v)", decoded, rpcErr)
	}
	if _, rpcErr = DecodeParams[input](json.RawMessage(`{"name":`)); rpcErr == nil || rpcErr.Code != CodeInvalidParams {
		t.Errorf("DecodeParams(invalid) = %v", rpcErr)
	}
	if _, rpcErr = DecodeParams[input](nil); rpcErr == nil || rpcErr.Code != CodeInvalidParams {
		t.Errorf("DecodeParams(nil) = %v", rpcErr)
	}
	if _, rpcErr = DecodeParams[input](json.RawMessage(`{"name":"Ada"} {}`)); rpcErr == nil || rpcErr.Code != CodeInvalidParams {
		t.Errorf("DecodeParams(trailing JSON) = %v", rpcErr)
	}
}

func TestRequestIsAvailableFromHandlerContext(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	_ = registry.Register("inspect", func(ctx context.Context, _ json.RawMessage) (any, error) {
		request, ok := RequestFromContext(ctx)
		if !ok {
			t.Fatal("RequestFromContext() did not find request")
		}
		return request.Method, nil
	})
	response, _ := NewDispatcher(registry).Dispatch(
		context.Background(),
		[]byte(`{"jsonrpc":"2.0","method":"inspect","id":1}`),
	)
	assertJSONEqual(t, response, []byte(`{"jsonrpc":"2.0","result":"inspect","id":1}`))
}

func assertJSONEqual(t *testing.T, got, want []byte) {
	t.Helper()
	if len(want) == 0 {
		if len(got) != 0 {
			t.Fatalf("got JSON %s, want no JSON", got)
		}
		return
	}
	var gotValue, wantValue any
	gotDecoder := json.NewDecoder(strings.NewReader(string(got)))
	gotDecoder.UseNumber()
	if err := gotDecoder.Decode(&gotValue); err != nil {
		t.Fatalf("invalid got JSON %q: %v", got, err)
	}
	wantDecoder := json.NewDecoder(strings.NewReader(string(want)))
	wantDecoder.UseNumber()
	if err := wantDecoder.Decode(&wantValue); err != nil {
		t.Fatalf("invalid want JSON %q: %v", want, err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf("JSON mismatch\n got: %s\nwant: %s", got, want)
	}
}
