package jsonrpc

import (
	"bytes"
	"encoding/json"
	"errors"
)

const Version = "2.0"

type IDKind uint8

const (
	IDMissing IDKind = iota
	IDString
	IDNumber
	IDNull
)

// ID preserves the exact JSON representation of a string, number, or null ID.
type ID struct {
	kind IDKind
	raw  json.RawMessage
}

func StringID(value string) ID {
	raw, _ := json.Marshal(value)
	return ID{kind: IDString, raw: raw}
}

func NumberID(value json.Number) ID {
	return ID{kind: IDNumber, raw: json.RawMessage(value.String())}
}

func NullID() ID { return ID{kind: IDNull, raw: json.RawMessage("null")} }

func (id ID) Kind() IDKind { return id.kind }

func (id ID) Equal(other ID) bool {
	return id.kind == other.kind && bytes.Equal(id.raw, other.raw)
}

func (id ID) MarshalJSON() ([]byte, error) {
	if id.kind == IDMissing {
		return []byte("null"), nil
	}
	return append([]byte(nil), id.raw...), nil
}

func (id *ID) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	switch value.(type) {
	case string:
		id.kind = IDString
	case json.Number:
		id.kind = IDNumber
	case nil:
		id.kind = IDNull
	default:
		return errors.New("jsonrpc: id must be a string, number, or null")
	}
	id.raw = append(id.raw[:0], data...)
	return nil
}

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      ID              `json:"-"`
	idSet   bool
}

func (r *Request) UnmarshalJSON(data []byte) error {
	type wireRequest struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
		ID      json.RawMessage `json:"id"`
	}
	var wire wireRequest
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	r.JSONRPC, r.Method, r.Params = wire.JSONRPC, wire.Method, wire.Params
	r.idSet = wire.ID != nil
	if r.idSet {
		return json.Unmarshal(wire.ID, &r.ID)
	}
	r.ID = ID{}
	return nil
}

func (r Request) MarshalJSON() ([]byte, error) {
	object := map[string]any{"jsonrpc": r.JSONRPC, "method": r.Method}
	if r.Params != nil {
		object["params"] = r.Params
	}
	if r.idSet || r.ID.Kind() != IDMissing {
		object["id"] = r.ID
	}
	return json.Marshal(object)
}

func (r Request) IsNotification() bool { return !r.idSet && r.ID.Kind() == IDMissing }

func (r Request) Validate() *Error {
	if r.JSONRPC != Version || r.Method == "" {
		return InvalidRequest()
	}
	if r.Params != nil {
		trimmed := bytes.TrimSpace(r.Params)
		if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') || !json.Valid(trimmed) {
			return InvalidRequest()
		}
	}
	return nil
}

type Response struct {
	JSONRPC   string          `json:"jsonrpc"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *Error          `json:"error,omitempty"`
	ID        ID              `json:"id"`
	resultSet bool
	errorSet  bool
	idSet     bool
}

func (r *Response) UnmarshalJSON(data []byte) error {
	type wireResponse struct {
		JSONRPC string          `json:"jsonrpc"`
		Result  json.RawMessage `json:"result"`
		Error   json.RawMessage `json:"error"`
		ID      json.RawMessage `json:"id"`
	}
	var wire wireResponse
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	r.JSONRPC, r.Result = wire.JSONRPC, wire.Result
	r.resultSet, r.errorSet, r.idSet = wire.Result != nil, wire.Error != nil, wire.ID != nil
	if r.errorSet {
		if bytes.Equal(bytes.TrimSpace(wire.Error), []byte("null")) {
			return errors.New("jsonrpc: error must be an object")
		}
		if err := json.Unmarshal(wire.Error, &r.Error); err != nil {
			return err
		}
	}
	if r.idSet {
		if err := json.Unmarshal(wire.ID, &r.ID); err != nil {
			return err
		}
	}
	return nil
}

func (r Response) MarshalJSON() ([]byte, error) {
	object := map[string]any{"jsonrpc": r.JSONRPC, "id": r.ID}
	if r.Error != nil || r.errorSet {
		object["error"] = r.Error
	} else {
		result := r.Result
		if result == nil {
			result = json.RawMessage("null")
		}
		object["result"] = result
	}
	return json.Marshal(object)
}

func (r Response) Validate() error {
	if r.JSONRPC != Version || !r.idSet || r.ID.Kind() == IDMissing {
		return errors.New("jsonrpc: invalid response envelope")
	}
	if r.resultSet == r.errorSet {
		return errors.New("jsonrpc: response must contain exactly one of result or error")
	}
	if r.errorSet && (r.Error == nil || r.Error.Message == "") {
		return errors.New("jsonrpc: invalid error object")
	}
	return nil
}
