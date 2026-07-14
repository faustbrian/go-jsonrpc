package jsonrpc

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestIDRoundTripAndEquality(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		kind  IDKind
	}{
		{name: "string", input: `"request-1"`, kind: IDString},
		{name: "integer", input: `1`, kind: IDNumber},
		{name: "fractional", input: `1.25`, kind: IDNumber},
		{name: "null", input: `null`, kind: IDNull},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var id ID
			if err := json.Unmarshal([]byte(tt.input), &id); err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if id.Kind() != tt.kind {
				t.Fatalf("Kind() = %v, want %v", id.Kind(), tt.kind)
			}
			encoded, err := json.Marshal(id)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}
			if string(encoded) != tt.input {
				t.Errorf("Marshal() = %s, want %s", encoded, tt.input)
			}
			if !id.Equal(id) {
				t.Error("Equal() = false for same ID")
			}
		})
	}

	if StringID("1").Equal(NumberID(json.Number("1"))) {
		t.Error("string and number IDs must not compare equal")
	}
}

func TestIDRejectsInvalidJSONTypes(t *testing.T) {
	t.Parallel()

	for _, input := range []string{`true`, `false`, `{}`, `[]`} {
		var id ID
		if err := json.Unmarshal([]byte(input), &id); err == nil {
			t.Errorf("Unmarshal(%s) unexpectedly succeeded", input)
		}
	}
}

func TestRequestValidation(t *testing.T) {
	t.Parallel()

	valid := []string{
		`{"jsonrpc":"2.0","method":"subtract","params":[42,23],"id":1}`,
		`{"jsonrpc":"2.0","method":"subtract","params":{"minuend":42},"id":"a"}`,
		`{"jsonrpc":"2.0","method":"update"}`,
		`{"jsonrpc":"2.0","method":"update","id":null}`,
	}
	for _, input := range valid {
		var request Request
		if err := json.Unmarshal([]byte(input), &request); err != nil {
			t.Errorf("Unmarshal(valid %s) error = %v", input, err)
			continue
		}
		if rpcErr := request.Validate(); rpcErr != nil {
			t.Errorf("Validate(valid %s) = %v", input, rpcErr)
		}
	}

	invalid := []string{
		`{"jsonrpc":"1.0","method":"x","id":1}`,
		`{"jsonrpc":"2.0","method":"","id":1}`,
		`{"jsonrpc":"2.0","method":"x","params":"bad","id":1}`,
		`{"jsonrpc":"2.0","method":"x","params":null,"id":1}`,
	}
	for _, input := range invalid {
		var request Request
		if err := json.Unmarshal([]byte(input), &request); err != nil {
			continue
		}
		if rpcErr := request.Validate(); rpcErr == nil || rpcErr.Code != CodeInvalidRequest {
			t.Errorf("Validate(invalid %s) = %v, want invalid request", input, rpcErr)
		}
	}
}

func TestRequestDistinguishesNotificationFromNullID(t *testing.T) {
	t.Parallel()

	var notification Request
	if err := json.Unmarshal([]byte(`{"jsonrpc":"2.0","method":"update"}`), &notification); err != nil {
		t.Fatal(err)
	}
	if !notification.IsNotification() {
		t.Error("request without id is not recognized as notification")
	}

	var nullID Request
	if err := json.Unmarshal([]byte(`{"jsonrpc":"2.0","method":"update","id":null}`), &nullID); err != nil {
		t.Fatal(err)
	}
	if nullID.IsNotification() {
		t.Error("request with explicit null id must not be a notification")
	}
}

func TestResponseValidation(t *testing.T) {
	t.Parallel()

	valid := []string{
		`{"jsonrpc":"2.0","result":19,"id":1}`,
		`{"jsonrpc":"2.0","result":null,"id":"a"}`,
		`{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":1}`,
	}
	for _, input := range valid {
		var response Response
		if err := json.Unmarshal([]byte(input), &response); err != nil {
			t.Errorf("Unmarshal(valid %s) error = %v", input, err)
			continue
		}
		if err := response.Validate(); err != nil {
			t.Errorf("Validate(valid %s) error = %v", input, err)
		}
	}

	invalid := []string{
		`{"jsonrpc":"1.0","result":19,"id":1}`,
		`{"jsonrpc":"2.0","id":1}`,
		`{"jsonrpc":"2.0","result":19,"error":{"code":-32603,"message":"Internal error"},"id":1}`,
		`{"jsonrpc":"2.0","result":19}`,
	}
	for _, input := range invalid {
		var response Response
		if err := json.Unmarshal([]byte(input), &response); err != nil {
			continue
		}
		if err := response.Validate(); err == nil {
			t.Errorf("Validate(invalid %s) unexpectedly succeeded", input)
		}
	}
}

func TestRPCErrorModel(t *testing.T) {
	t.Parallel()

	cause := errors.New("database unavailable")
	err := NewError(42, "application failure").WithData(map[string]string{"field": "name"}).WithCause(cause)
	if err.Code != 42 || err.Message != "application failure" {
		t.Fatalf("NewError() = %#v", err)
	}
	if !errors.Is(err, cause) {
		t.Error("RPC error does not preserve its internal cause")
	}
	encoded, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if string(encoded) != `{"code":42,"message":"application failure","data":{"field":"name"}}` {
		t.Errorf("Marshal() = %s", encoded)
	}
	if got := err.Error(); got != "jsonrpc error 42: application failure" {
		t.Errorf("Error() = %q", got)
	}
}

func TestStandardErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err     *Error
		code    int
		message string
	}{
		{ParseError(), CodeParseError, "Parse error"},
		{InvalidRequest(), CodeInvalidRequest, "Invalid Request"},
		{MethodNotFound(), CodeMethodNotFound, "Method not found"},
		{InvalidParams(), CodeInvalidParams, "Invalid params"},
		{InternalError(), CodeInternalError, "Internal error"},
	}
	for _, tt := range tests {
		if tt.err.Code != tt.code || tt.err.Message != tt.message {
			t.Errorf("error = %#v, want code %d and message %q", tt.err, tt.code, tt.message)
		}
	}
}
