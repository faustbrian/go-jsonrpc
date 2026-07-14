package jsonrpc

import (
	"context"
	"encoding/json"
	"testing"
)

// TestSpecificationExamples covers the normative request/response examples in
// the JSON-RPC 2.0 specification, including named parameters and mixed batches.
func TestSpecificationExamples(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	_ = registry.Register("subtract", func(_ context.Context, params json.RawMessage) (any, error) {
		var positional []int
		if len(params) > 0 && params[0] == '[' {
			if err := json.Unmarshal(params, &positional); err != nil || len(positional) != 2 {
				return nil, InvalidParams()
			}
			return positional[0] - positional[1], nil
		}
		var named struct {
			Minuend    int `json:"minuend"`
			Subtrahend int `json:"subtrahend"`
		}
		if err := json.Unmarshal(params, &named); err != nil {
			return nil, InvalidParams()
		}
		return named.Minuend - named.Subtrahend, nil
	})
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
	_ = registry.Register("get_data", func(context.Context, json.RawMessage) (any, error) {
		return []any{"hello", 5}, nil
	})
	dispatcher := NewDispatcher(registry)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "positional one", input: `{"jsonrpc":"2.0","method":"subtract","params":[42,23],"id":1}`, want: `{"jsonrpc":"2.0","result":19,"id":1}`},
		{name: "positional two", input: `{"jsonrpc":"2.0","method":"subtract","params":[23,42],"id":2}`, want: `{"jsonrpc":"2.0","result":-19,"id":2}`},
		{name: "named one", input: `{"jsonrpc":"2.0","method":"subtract","params":{"subtrahend":23,"minuend":42},"id":3}`, want: `{"jsonrpc":"2.0","result":19,"id":3}`},
		{name: "named two", input: `{"jsonrpc":"2.0","method":"subtract","params":{"minuend":42,"subtrahend":23},"id":4}`, want: `{"jsonrpc":"2.0","result":19,"id":4}`},
		{name: "invalid request", input: `{"jsonrpc":"2.0","method":1,"params":"bar"}`, want: `{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, ok := dispatcher.Dispatch(context.Background(), []byte(test.input))
			if !ok {
				t.Fatal("specification request unexpectedly omitted response")
			}
			assertJSONEqual(t, response, []byte(test.want))
		})
	}

	batch := `[
		{"jsonrpc":"2.0","method":"sum","params":[1,2,4],"id":"1"},
		{"jsonrpc":"2.0","method":"notify_hello","params":[7]},
		{"jsonrpc":"2.0","method":"subtract","params":[42,23],"id":"2"},
		{"foo":"boo"},
		{"jsonrpc":"2.0","method":"foo.get","params":{"name":"myself"},"id":"5"},
		{"jsonrpc":"2.0","method":"get_data","id":"9"}
	]`
	want := `[
		{"jsonrpc":"2.0","result":7,"id":"1"},
		{"jsonrpc":"2.0","result":19,"id":"2"},
		{"jsonrpc":"2.0","error":{"code":-32600,"message":"Invalid Request"},"id":null},
		{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":"5"},
		{"jsonrpc":"2.0","result":["hello",5],"id":"9"}
	]`
	response, ok := dispatcher.Dispatch(context.Background(), []byte(batch))
	if !ok {
		t.Fatal("specification batch unexpectedly omitted response")
	}
	assertJSONEqual(t, response, []byte(want))
}
