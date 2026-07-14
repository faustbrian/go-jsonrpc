package jsonrpc

import (
	"context"
	"encoding/json"
	"testing"
)

func FuzzDispatcher(f *testing.F) {
	seeds := []string{
		`{"jsonrpc":"2.0","method":"ping","id":1}`,
		`[{"jsonrpc":"2.0","method":"ping"}]`,
		`{`,
		`null`,
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}
	dispatcher := NewDispatcher(NewRegistry())
	f.Fuzz(func(t *testing.T, payload []byte) {
		response, ok := dispatcher.Dispatch(context.Background(), payload)
		if ok && !json.Valid(response) {
			t.Fatalf("Dispatch() returned invalid JSON: %q", response)
		}
	})
}

func FuzzRequestUnmarshal(f *testing.F) {
	for _, seed := range []string{
		`{"jsonrpc":"2.0","method":"ping","id":1}`,
		`{"jsonrpc":"2.0","method":"ping"}`,
		`{`,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, payload []byte) {
		var request Request
		if json.Unmarshal(payload, &request) == nil {
			if _, err := json.Marshal(request); err != nil {
				t.Fatalf("valid request failed to marshal: %v", err)
			}
		}
	})
}
