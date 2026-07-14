package jsonrpc

import (
	"context"
	"encoding/json"
	"testing"
)

func benchmarkDispatcher() *Dispatcher {
	registry := NewRegistry()
	_ = registry.Register("sum", func(_ context.Context, params json.RawMessage) (any, error) {
		var values []int
		if err := json.Unmarshal(params, &values); err != nil {
			return nil, InvalidParams()
		}
		return values[0] + values[1], nil
	})
	return NewDispatcher(registry)
}

func BenchmarkDispatchSingle(b *testing.B) {
	dispatcher := benchmarkDispatcher()
	payload := []byte(`{"jsonrpc":"2.0","method":"sum","params":[1,2],"id":1}`)
	b.ReportAllocs()
	for b.Loop() {
		dispatcher.Dispatch(context.Background(), payload)
	}
}

func BenchmarkDispatchBatch(b *testing.B) {
	dispatcher := benchmarkDispatcher()
	payload := []byte(`[
		{"jsonrpc":"2.0","method":"sum","params":[1,2],"id":1},
		{"jsonrpc":"2.0","method":"sum","params":[3,4],"id":2},
		{"jsonrpc":"2.0","method":"sum","params":[5,6]}
	]`)
	b.ReportAllocs()
	for b.Loop() {
		dispatcher.Dispatch(context.Background(), payload)
	}
}
