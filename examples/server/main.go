package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	jsonrpc "github.com/faustbrian/go-jsonrpc"
)

func main() {
	registry := jsonrpc.NewRegistry()
	if err := registry.Register("greet", greet); err != nil {
		log.Fatal(err)
	}

	handler := jsonrpc.NewHTTPHandler(jsonrpc.NewDispatcher(registry))
	log.Println("JSON-RPC listening on http://localhost:8080/rpc")
	log.Fatal(http.ListenAndServe(":8080", http.StripPrefix("/rpc", handler)))
}

func greet(_ context.Context, raw json.RawMessage) (any, error) {
	params, rpcErr := jsonrpc.DecodeParams[struct {
		Name string `json:"name"`
	}](raw)
	if rpcErr != nil || params.Name == "" {
		return nil, jsonrpc.InvalidParams()
	}
	return map[string]string{"message": "Hello " + params.Name}, nil
}
