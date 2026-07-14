package jsonrpc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type fixedIDGenerator struct{ id ID }

func (generator fixedIDGenerator) NextID() ID { return generator.id }

func TestClientOptionsAndDefensivePaths(t *testing.T) {
	t.Parallel()

	client := NewClient(TransportFunc(func(_ context.Context, payload []byte) ([]byte, error) {
		return []byte(`{"jsonrpc":"2.0","result":null,"id":"fixed"}`), nil
	}), WithIDGenerator(nil), WithIDGenerator(fixedIDGenerator{id: StringID("fixed")}))
	if err := client.Call(context.Background(), "discard", nil, nil); err != nil {
		t.Fatalf("Call(discard) error = %v", err)
	}
	if err := client.Call(context.Background(), "", nil, nil); !errors.Is(err, ErrInvalidMethodName) {
		t.Errorf("Call(invalid method) error = %v", err)
	}
	if err := client.Call(context.Background(), "method", make(chan int), nil); err == nil {
		t.Error("Call(unencodable params) unexpectedly succeeded")
	}
	if err := client.Notify(context.Background(), "", nil); !errors.Is(err, ErrInvalidMethodName) {
		t.Errorf("Notify(invalid method) error = %v", err)
	}
	if err := client.Notify(context.Background(), "method", make(chan int)); err == nil {
		t.Error("Notify(unencodable params) unexpectedly succeeded")
	}

	nilTransport := NewClient(nil)
	if err := nilTransport.Notify(context.Background(), "notice", nil); !errors.Is(err, ErrTransport) {
		t.Errorf("Notify(nil transport) error = %v", err)
	}
}

func TestClientBatchDefensivePaths(t *testing.T) {
	t.Parallel()

	client := NewClient(TransportFunc(func(context.Context, []byte) ([]byte, error) { return nil, nil }))
	if err := client.Batch(context.Background(), nil); err == nil {
		t.Error("Batch(nil call) unexpectedly succeeded")
	}
	if err := client.Batch(context.Background(), &BatchCall{Method: ""}); !errors.Is(err, ErrInvalidMethodName) {
		t.Errorf("Batch(invalid request) error = %v", err)
	}
	if err := client.Batch(context.Background(), &BatchCall{Method: "", Notification: true}); !errors.Is(err, ErrInvalidMethodName) {
		t.Errorf("Batch(invalid notification) error = %v", err)
	}
	if err := client.Batch(context.Background(), &BatchCall{Method: "notice", Notification: true}); err != nil {
		t.Errorf("notification-only Batch() error = %v", err)
	}

	responses := []struct {
		response string
		want     error
		call     *BatchCall
	}{
		{response: `[`, want: ErrInvalidResponse, call: &BatchCall{Method: "one"}},
		{response: `[{"jsonrpc":"1.0","result":1,"id":1}]`, want: ErrInvalidResponse, call: &BatchCall{Method: "one"}},
		{response: `[{"jsonrpc":"2.0","result":"bad","id":1}]`, want: ErrInvalidResponse, call: &BatchCall{Method: "one", Result: new(int)}},
	}
	for _, test := range responses {
		client := NewClient(TransportFunc(func(context.Context, []byte) ([]byte, error) {
			return []byte(test.response), nil
		}))
		if err := client.Batch(context.Background(), test.call); !errors.Is(err, test.want) {
			t.Errorf("Batch(response %q) error = %v, want %v", test.response, err, test.want)
		}
	}

	unexpected := NewClient(TransportFunc(func(context.Context, []byte) ([]byte, error) {
		return []byte(`[]`), nil
	}))
	if err := unexpected.Batch(context.Background(), &BatchCall{Method: "notice", Notification: true}); !errors.Is(err, ErrUnexpectedResponse) {
		t.Errorf("notification Batch(response) error = %v", err)
	}

	offline := NewClient(TransportFunc(func(context.Context, []byte) ([]byte, error) {
		return nil, io.EOF
	}))
	if err := offline.Batch(context.Background(), &BatchCall{Method: "one"}); !errors.Is(err, ErrTransport) {
		t.Errorf("Batch(transport error) = %v", err)
	}
}

func TestProtocolDefensivePaths(t *testing.T) {
	t.Parallel()

	missing, err := json.Marshal(ID{})
	if err != nil || string(missing) != "null" {
		t.Errorf("Marshal(missing ID) = %s, %v", missing, err)
	}
	var id ID
	if err := json.Unmarshal([]byte(`"unterminated`), &id); err == nil {
		t.Error("Unmarshal(malformed ID) unexpectedly succeeded")
	}
	if err := id.UnmarshalJSON([]byte(`"unterminated`)); err == nil {
		t.Error("ID.UnmarshalJSON(malformed) unexpectedly succeeded")
	}
	var request Request
	if err := json.Unmarshal([]byte(`{`), &request); err == nil {
		t.Error("Unmarshal(malformed request) unexpectedly succeeded")
	}

	for _, input := range []string{
		`{`,
		`{"jsonrpc":"2.0","error":null,"id":1}`,
		`{"jsonrpc":"2.0","error":1,"id":1}`,
		`{"jsonrpc":"2.0","result":1,"id":true}`,
	} {
		var response Response
		if err := json.Unmarshal([]byte(input), &response); err == nil {
			t.Errorf("Unmarshal(invalid response %s) unexpectedly succeeded", input)
		}
	}
	var malformedResponse Response
	if err := malformedResponse.UnmarshalJSON([]byte(`{`)); err == nil {
		t.Error("Response.UnmarshalJSON(malformed) unexpectedly succeeded")
	}

	response := Response{JSONRPC: Version, ID: NullID(), idSet: true, resultSet: true}
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, encoded, []byte(`{"jsonrpc":"2.0","result":null,"id":null}`))

	invalidError := Response{
		JSONRPC:  Version,
		Error:    &Error{Code: 1},
		ID:       StringID("x"),
		errorSet: true,
		idSet:    true,
	}
	if err := invalidError.Validate(); err == nil {
		t.Error("Validate(error without message) unexpectedly succeeded")
	}

	rpcErr := NewError(1, "bad").WithData(make(chan int))
	if rpcErr.Unwrap() == nil {
		t.Error("WithData(unencodable) did not retain marshal error")
	}
}

func TestServerDefensivePaths(t *testing.T) {
	t.Parallel()

	dispatcher := NewDispatcher(nil)
	response, _ := dispatcher.Dispatch(context.Background(), []byte(`{"jsonrpc":"2.0","method":"missing","id":1}`))
	assertJSONEqual(t, response, []byte(`{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":1}`))

	registry := NewRegistry()
	_ = registry.Register("failure", func(context.Context, json.RawMessage) (any, error) {
		return nil, errors.New("failure")
	})
	_ = registry.Register("encode", func(context.Context, json.RawMessage) (any, error) {
		return make(chan int), nil
	})
	dispatcher = NewDispatcher(registry, WithMiddleware(nil))
	response, _ = dispatcher.Dispatch(context.Background(), []byte(`{"jsonrpc":"2.0","method":"failure","id":1}`))
	assertJSONEqual(t, response, []byte(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"},"id":1}`))
	response, _ = dispatcher.Dispatch(context.Background(), []byte(`{"jsonrpc":"2.0","method":"encode","id":2}`))
	assertJSONEqual(t, response, []byte(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"},"id":2}`))

	nilMapper := NewDispatcher(registry, WithErrorMapper(func(error) *Error { return nil }))
	response, _ = nilMapper.Dispatch(context.Background(), []byte(`{"jsonrpc":"2.0","method":"failure","id":3}`))
	assertJSONEqual(t, response, []byte(`{"jsonrpc":"2.0","error":{"code":-32603,"message":"Internal error"},"id":3}`))
}

func TestHTTPDefensivePaths(t *testing.T) {
	t.Parallel()

	if NewHTTPHandler(nil) == nil {
		t.Fatal("NewHTTPHandler(nil) returned nil")
	}
	withBody := (&HTTPStatusError{StatusCode: 500, Body: "failure"}).Error()
	withoutBody := (&HTTPStatusError{StatusCode: 500}).Error()
	if !strings.Contains(withBody, "failure") || strings.Contains(withoutBody, "failure") {
		t.Errorf("HTTPStatusError strings = %q and %q", withBody, withoutBody)
	}

	transport, _ := NewHTTPTransport("http://example.test", WithHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       failingBody{},
			}, nil
		}),
	}))
	if _, err := transport.RoundTrip(context.Background(), []byte(`{}`)); err == nil {
		t.Error("RoundTrip(read error) unexpectedly succeeded")
	}
}

type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (failingBody) Close() error             { return nil }
