package jsonrpc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	// CodeRequestLimitExceeded is the implementation-defined server error used
	// when a dispatcher payload or batch exceeds its configured bound.
	CodeRequestLimitExceeded = -32000
	// CodeParseError indicates invalid JSON.
	CodeParseError = -32700
	// CodeInvalidRequest indicates a structurally invalid request.
	CodeInvalidRequest = -32600
	// CodeMethodNotFound indicates that no handler is registered for a method.
	CodeMethodNotFound = -32601
	// CodeInvalidParams indicates invalid method parameters.
	CodeInvalidParams = -32602
	// CodeInternalError indicates an internal JSON-RPC failure.
	CodeInternalError = -32603
)

// Error is a mutable JSON-RPC error object. Message and Data are public wire
// content, not sanitized secrets. WithCause retains an unredacted local error
// exposed by Unwrap but excluded from JSON serialization. Do not share mutable
// errors across concurrent calls; even copies share Data and the cause.
// See https://www.jsonrpc.org/specification#error_object.
type Error struct {
	// Code is the integer JSON-RPC error code.
	Code int `json:"code"`
	// Message is the public error description.
	Message string `json:"message"`
	// Data contains optional public JSON details. Direct assignments borrow the
	// supplied slice; use WithData to encode a new owned buffer.
	Data       json.RawMessage `json:"data,omitempty"`
	cause      error
	codeSet    bool
	messageSet bool
	decoded    bool
}

// NewError constructs a JSON-RPC error with code and public message. Application
// codes should avoid the protocol-reserved range -32768 through -32000. This
// constructor preserves code/message without enforcing that range or redacting them.
func NewError(code int, message string) *Error {
	return &Error{Code: code, Message: message, codeSet: true, messageSet: true}
}

// UnmarshalJSON decodes error fields, rejecting invalid UTF-8, duplicate envelope
// names and case variants of reserved names. Presence of code/message is checked
// later by Response.Validate; successful decoding alone is not validation.
// It clears old fields after preliminary decoding, but errors may leave partial
// receiver state. Use a separate value when rollback/alias isolation is needed.
func (e *Error) UnmarshalJSON(data []byte) error {
	if err := rejectDuplicateMembers(data, "code", "message", "data"); err != nil {
		return err
	}
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

// Error returns a textual representation containing the public code and
// message.
func (e *Error) Error() string {
	return fmt.Sprintf("jsonrpc error %d: %s", e.Code, e.Message)
}

// Unwrap returns the local cause, which is never serialized.
func (e *Error) Unwrap() error { return e.cause }

// WithData JSON-encodes public error details. An encoding failure clears Data
// and is retained as the local cause. It mutates and returns the same receiver;
// successful encoding replaces Data but does not clear a previous cause.
func (e *Error) WithData(value any) *Error {
	data, err := json.Marshal(value)
	if err != nil {
		e.Data = nil
		return e.WithCause(err)
	}
	e.Data = data
	return e
}

// WithCause mutates and returns the receiver, borrowing cause without wire
// serialization. Local inspection/formatting of the cause remains caller-owned.
func (e *Error) WithCause(cause error) *Error {
	e.cause = cause
	return e
}

// ParseError constructs the standard parse error.
func ParseError() *Error { return NewError(CodeParseError, "Parse error") }

// InvalidRequest constructs the standard invalid-request error.
func InvalidRequest() *Error { return NewError(CodeInvalidRequest, "Invalid Request") }

// MethodNotFound constructs the standard method-not-found error.
func MethodNotFound() *Error { return NewError(CodeMethodNotFound, "Method not found") }

// InvalidParams constructs the standard invalid-params error.
func InvalidParams() *Error { return NewError(CodeInvalidParams, "Invalid params") }

// InternalError constructs the standard internal error.
func InternalError() *Error { return NewError(CodeInternalError, "Internal error") }

// RequestLimitExceeded constructs the dispatcher resource-limit error.
func RequestLimitExceeded() *Error {
	return NewError(CodeRequestLimitExceeded, "Request limit exceeded")
}
