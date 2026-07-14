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
	var escapedString, decimalNumber, exponentNumber ID
	for input, target := range map[string]*ID{
		`"\u0061"`: &escapedString,
		`1.0`:      &decimalNumber,
		`1e0`:      &exponentNumber,
	} {
		if err := json.Unmarshal([]byte(input), target); err != nil {
			t.Fatal(err)
		}
	}
	if !StringID("a").Equal(escapedString) {
		t.Error("equivalent escaped string IDs do not compare equal")
	}
	if !NumberID("1").Equal(decimalNumber) || !NumberID("1").Equal(exponentNumber) {
		t.Error("equivalent numeric IDs do not compare equal")
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
		`{"jsonrpc":"2.0","method":"","id":1}`,
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
		`{"jsonrpc":"2.0","id":1}`,
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

func TestRequestUnmarshalClearsReusedState(t *testing.T) {
	t.Parallel()

	var request Request
	if err := json.Unmarshal([]byte(`{"jsonrpc":"2.0","method":"first","id":1}`), &request); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"jsonrpc":"2.0","id":2}`), &request); err != nil {
		t.Fatal(err)
	}
	if request.Method != "" || request.Validate() == nil {
		t.Errorf("reused request retained stale method: %#v", request)
	}
}

func TestErrorUnmarshalClearsReusedState(t *testing.T) {
	t.Parallel()

	var rpcErr Error
	if err := json.Unmarshal([]byte(`{"code":1,"message":"first","data":{"safe":true}}`), &rpcErr); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"message":"second"}`), &rpcErr); err != nil {
		t.Fatal(err)
	}
	if rpcErr.Code != 0 || rpcErr.Data != nil || rpcErr.valid() {
		t.Errorf("reused error retained stale state: %#v", rpcErr)
	}
}

func TestResponseValidation(t *testing.T) {
	t.Parallel()

	valid := []string{
		`{"jsonrpc":"2.0","result":19,"id":1}`,
		`{"jsonrpc":"2.0","result":null,"id":"a"}`,
		`{"jsonrpc":"2.0","error":{"code":-32601,"message":"Method not found"},"id":1}`,
		`{"jsonrpc":"2.0","error":{"code":0,"message":""},"id":1}`,
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
		`{"jsonrpc":"2.0","error":{"message":"missing code"},"id":1}`,
		`{"jsonrpc":"2.0","error":{"code":1},"id":1}`,
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

func TestResponseUnmarshalClearsReusedState(t *testing.T) {
	t.Parallel()

	var response Response
	if err := json.Unmarshal([]byte(`{"jsonrpc":"2.0","result":1,"id":1}`), &response); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"jsonrpc":"2.0","error":{"code":1,"message":"bad"},"id":2}`), &response); err != nil {
		t.Fatal(err)
	}
	if response.Result != nil || response.Error == nil || !response.ID.Equal(NumberID("2")) {
		t.Errorf("reused error response retained stale state: %#v", response)
	}
	if err := json.Unmarshal([]byte(`{"jsonrpc":"2.0","result":3,"id":3}`), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error != nil || !response.ID.Equal(NumberID("3")) {
		t.Errorf("reused result response retained stale state: %#v", response)
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
	err.WithData(make(chan int))
	if err.Data != nil {
		t.Error("WithData(unencodable) retained stale public data")
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
