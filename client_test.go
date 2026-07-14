package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestClientCallAndTypedCall(t *testing.T) {
	t.Parallel()

	transport := TransportFunc(func(_ context.Context, payload []byte) ([]byte, error) {
		assertJSONEqual(t, payload, []byte(`{"jsonrpc":"2.0","method":"greet","params":{"name":"Ada"},"id":1}`))
		return []byte(`{"jsonrpc":"2.0","result":{"message":"Hello Ada"},"id":1}`), nil
	})
	client := NewClient(transport)
	type greeting struct {
		Message string `json:"message"`
	}
	result, err := Call[greeting](context.Background(), client, "greet", map[string]string{"name": "Ada"})
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}
	if result.Message != "Hello Ada" {
		t.Errorf("Call() result = %#v", result)
	}
}

func TestClientCallErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		response  string
		transport error
		want      error
		wantRPC   int
	}{
		{name: "transport", transport: errors.New("offline"), want: ErrTransport},
		{name: "malformed", response: `{`, want: ErrInvalidResponse},
		{name: "wrong version", response: `{"jsonrpc":"1.0","result":1,"id":1}`, want: ErrInvalidResponse},
		{name: "wrong id", response: `{"jsonrpc":"2.0","result":1,"id":2}`, want: ErrMismatchedID},
		{name: "rpc error", response: `{"jsonrpc":"2.0","error":{"code":-32602,"message":"Invalid params"},"id":1}`, wantRPC: CodeInvalidParams},
		{name: "bad result", response: `{"jsonrpc":"2.0","result":"not an int","id":1}`, want: ErrInvalidResponse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := NewClient(TransportFunc(func(context.Context, []byte) ([]byte, error) {
				return []byte(tt.response), tt.transport
			}))
			var result int
			err := client.Call(context.Background(), "method", nil, &result)
			if tt.wantRPC != 0 {
				var rpcErr *Error
				if !errors.As(err, &rpcErr) || rpcErr.Code != tt.wantRPC {
					t.Fatalf("Call() error = %v, want RPC code %d", err, tt.wantRPC)
				}
				return
			}
			if !errors.Is(err, tt.want) {
				t.Errorf("Call() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestClientNotification(t *testing.T) {
	t.Parallel()

	transport := TransportFunc(func(_ context.Context, payload []byte) ([]byte, error) {
		assertJSONEqual(t, payload, []byte(`{"jsonrpc":"2.0","method":"update","params":[1,2]}`))
		return nil, nil
	})
	if err := NewClient(transport).Notify(context.Background(), "update", []int{1, 2}); err != nil {
		t.Fatalf("Notify() error = %v", err)
	}

	unexpected := NewClient(TransportFunc(func(context.Context, []byte) ([]byte, error) {
		return []byte(`{"jsonrpc":"2.0","result":null,"id":1}`), nil
	}))
	if err := unexpected.Notify(context.Background(), "update", nil); !errors.Is(err, ErrUnexpectedResponse) {
		t.Errorf("Notify() error = %v", err)
	}
}

func TestClientBatch(t *testing.T) {
	t.Parallel()

	transport := TransportFunc(func(_ context.Context, payload []byte) ([]byte, error) {
		assertJSONEqual(t, payload, []byte(`[
			{"jsonrpc":"2.0","method":"first","id":1},
			{"jsonrpc":"2.0","method":"notice","params":{"ok":true}},
			{"jsonrpc":"2.0","method":"second","params":[2],"id":2}
		]`))
		return []byte(`[
			{"jsonrpc":"2.0","error":{"code":-32001,"message":"failed"},"id":2},
			{"jsonrpc":"2.0","result":"one","id":1}
		]`), nil
	})
	client := NewClient(transport)
	var first string
	var second int
	calls := []*BatchCall{
		{Method: "first", Result: &first},
		{Method: "notice", Params: map[string]bool{"ok": true}, Notification: true},
		{Method: "second", Params: []int{2}, Result: &second},
	}
	if err := client.Batch(context.Background(), calls...); err != nil {
		t.Fatalf("Batch() error = %v", err)
	}
	if first != "one" {
		t.Errorf("first result = %q", first)
	}
	if calls[2].Error == nil || calls[2].Error.Code != -32001 {
		t.Errorf("second error = %v", calls[2].Error)
	}
}

func TestClientBatchValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response string
		want     error
	}{
		{name: "not an array", response: `{"jsonrpc":"2.0","result":1,"id":1}`, want: ErrInvalidResponse},
		{name: "missing response", response: `[]`, want: ErrMissingResponse},
		{name: "unexpected id", response: `[{"jsonrpc":"2.0","result":1,"id":9}]`, want: ErrMismatchedID},
		{name: "duplicate id", response: `[{"jsonrpc":"2.0","result":1,"id":1},{"jsonrpc":"2.0","result":2,"id":1}]`, want: ErrDuplicateResponse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := NewClient(TransportFunc(func(context.Context, []byte) ([]byte, error) {
				return []byte(tt.response), nil
			}))
			var result int
			err := client.Batch(context.Background(), &BatchCall{Method: "one", Result: &result})
			if !errors.Is(err, tt.want) {
				t.Errorf("Batch() error = %v, want %v", err, tt.want)
			}
		})
	}

	client := NewClient(TransportFunc(func(context.Context, []byte) ([]byte, error) { return nil, nil }))
	if err := client.Batch(context.Background()); !errors.Is(err, ErrEmptyBatch) {
		t.Errorf("empty Batch() error = %v", err)
	}
}

func TestRequestBuilders(t *testing.T) {
	t.Parallel()

	request, err := NewRequest("sum", []int{1, 2}, StringID("x"))
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(request)
	assertJSONEqual(t, encoded, []byte(`{"jsonrpc":"2.0","method":"sum","params":[1,2],"id":"x"}`))

	notification, err := NewNotification("ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ = json.Marshal(notification)
	assertJSONEqual(t, encoded, []byte(`{"jsonrpc":"2.0","method":"ping"}`))

	for _, input := range []struct {
		method string
		params any
		id     ID
	}{
		{method: "", id: StringID("x")},
		{method: "rpc.reserved", id: StringID("x")},
		{method: "method", params: "scalar", id: StringID("x")},
		{method: "method", id: ID{}},
	} {
		if _, err := NewRequest(input.method, input.params, input.id); err == nil {
			t.Errorf("NewRequest(%q, %#v, %#v) unexpectedly succeeded", input.method, input.params, input.id)
		}
	}
}

func TestAtomicIDGenerator(t *testing.T) {
	t.Parallel()

	generator := NewAtomicIDGenerator(40)
	got := []ID{generator.NextID(), generator.NextID()}
	want := []ID{NumberID("41"), NumberID("42")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("generated IDs = %#v, want %#v", got, want)
	}
}
