package jsonrpc_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"

	jsonrpc "github.com/faustbrian/go-jsonrpc"
)

func Example() {
	registry := jsonrpc.NewRegistry()
	if err := registry.Register("add", func(_ context.Context, raw json.RawMessage) (any, error) {
		values, rpcErr := jsonrpc.DecodeParams[[]int](raw)
		if rpcErr != nil || len(values) != 2 {
			return nil, jsonrpc.InvalidParams()
		}
		return values[0] + values[1], nil
	}); err != nil {
		fmt.Println(err)
		return
	}
	server := httptest.NewServer(jsonrpc.NewHTTPHandler(jsonrpc.NewDispatcher(registry)))
	defer server.Close()
	transport, err := jsonrpc.NewHTTPTransport(server.URL)
	if err != nil {
		fmt.Println(err)
		return
	}
	result, err := jsonrpc.Call[int](context.Background(), jsonrpc.NewClient(transport), "add", []int{2, 3})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(result)
	// Output: 5
}

func ExampleClient_Batch() {
	transport := jsonrpc.TransportFunc(func(context.Context, []byte) ([]byte, error) {
		return []byte(`[
			{"jsonrpc":"2.0","result":4,"id":2},
			{"jsonrpc":"2.0","result":2,"id":1}
		]`), nil
	})
	client := jsonrpc.NewClient(transport)
	var first, second int
	calls := []*jsonrpc.BatchCall{
		{Method: "double", Params: []int{1}, Result: &first},
		{Method: "double", Params: []int{2}, Result: &second},
	}
	if err := client.Batch(context.Background(), calls...); err != nil {
		fmt.Println(err)
		return
	}
	for _, call := range calls {
		if call.Error != nil {
			fmt.Println(call.Error)
			return
		}
	}
	fmt.Println(first, second)
	// Output: 2 4
}

func ExampleWithMiddleware() {
	registry := jsonrpc.NewRegistry()
	if err := registry.Register("ping", func(context.Context, json.RawMessage) (any, error) {
		return "pong", nil
	}); err != nil {
		fmt.Println(err)
		return
	}
	logging := func(next jsonrpc.Handler) jsonrpc.Handler {
		return func(ctx context.Context, params json.RawMessage) (any, error) {
			request, ok := jsonrpc.RequestFromContext(ctx)
			if !ok {
				return nil, jsonrpc.InternalError()
			}
			fmt.Println(request.Method)
			return next(ctx, params)
		}
	}
	dispatcher := jsonrpc.NewDispatcher(registry, jsonrpc.WithMiddleware(logging))
	response, reply := dispatcher.DispatchSingle(context.Background(), []byte(`{"jsonrpc":"2.0","method":"ping","id":1}`))
	if reply && response.Error != nil {
		fmt.Println(response.Error)
	}
	// Output: ping
}
