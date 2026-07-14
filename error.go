package jsonrpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// Error is a JSON-RPC error object. Cause is retained locally and is never
// serialized, allowing callers to preserve diagnostic context safely.
type Error struct {
	Code       int             `json:"code"`
	Message    string          `json:"message"`
	Data       json.RawMessage `json:"data,omitempty"`
	cause      error
	codeSet    bool
	messageSet bool
	decoded    bool
}

func NewError(code int, message string) *Error {
	return &Error{Code: code, Message: message, codeSet: true, messageSet: true}
}

func (e *Error) UnmarshalJSON(data []byte) error {
	type wireError struct {
		Code    json.RawMessage `json:"code"`
		Message json.RawMessage `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	var wire wireError
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	e.Code, e.Message, e.Data, e.cause = 0, "", nil, nil
	e.decoded = true
	e.codeSet, e.messageSet = wire.Code != nil, wire.Message != nil
	if e.codeSet {
		if bytes.Equal(bytes.TrimSpace(wire.Code), []byte("null")) {
			return errors.New("jsonrpc: error code must be an integer")
		}
		if err := json.Unmarshal(wire.Code, &e.Code); err != nil {
			return err
		}
	}
	if e.messageSet {
		trimmed := bytes.TrimSpace(wire.Message)
		if len(trimmed) == 0 || trimmed[0] != '"' {
			return errors.New("jsonrpc: error message must be a string")
		}
		_ = json.Unmarshal(wire.Message, &e.Message)
	}
	e.Data = wire.Data
	return nil
}

func (e *Error) valid() bool {
	if e.decoded {
		return e.codeSet && e.messageSet
	}
	return true
}

func (e *Error) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.cause }

func (e *Error) WithData(value any) *Error {
	data, err := json.Marshal(value)
	if err != nil {
		e.Data = nil
		return e.WithCause(err)
	}
	e.Data = data
	return e
}

func (e *Error) WithCause(cause error) *Error {
	e.cause = cause
	return e
}

func ParseError() *Error     { return NewError(CodeParseError, "Parse error") }
func InvalidRequest() *Error { return NewError(CodeInvalidRequest, "Invalid Request") }
func MethodNotFound() *Error { return NewError(CodeMethodNotFound, "Method not found") }
func InvalidParams() *Error  { return NewError(CodeInvalidParams, "Invalid params") }
func InternalError() *Error  { return NewError(CodeInternalError, "Internal error") }
