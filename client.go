package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync/atomic"
)

// Client errors identify transport, envelope, correlation, batch, and size
// failures and support errors.Is through direct return or wrapping.
var (
	// ErrTransport identifies a missing transport or a failure returned by it.
	ErrTransport = errors.New("jsonrpc: transport error")
	// ErrInvalidResponse identifies malformed envelopes or failed result decoding.
	ErrInvalidResponse = errors.New("jsonrpc: invalid response")
	// ErrMismatchedID identifies a response ID not matching a pending request.
	ErrMismatchedID = errors.New("jsonrpc: mismatched response id")
	// ErrUnexpectedResponse identifies notification replies or excess batch reply members.
	ErrUnexpectedResponse = errors.New("jsonrpc: unexpected response")
	// ErrMissingResponse identifies an incomplete set of batch responses.
	ErrMissingResponse = errors.New("jsonrpc: missing batch response")
	// ErrDuplicateResponse identifies repeated response IDs within a batch.
	ErrDuplicateResponse = errors.New("jsonrpc: duplicate batch response")
	// ErrDuplicateRequestID identifies ambiguous generated request IDs before batch transport.
	ErrDuplicateRequestID = errors.New("jsonrpc: duplicate batch request id")
	// ErrEmptyBatch identifies a client batch with no calls.
	ErrEmptyBatch = errors.New("jsonrpc: empty client batch")
	// ErrClientBatchTooLarge identifies a call count above the client batch bound.
	ErrClientBatchTooLarge = errors.New("jsonrpc: client batch too large")
	// ErrClientResponseTooLarge identifies a reply above the client parsing byte bound.
	ErrClientResponseTooLarge = errors.New("jsonrpc: client response too large")
)

const (
	defaultMaxClientResponseBytes int64 = 4 << 20
	defaultMaxClientBatchItems          = 1024
)

// Transport exchanges one complete JSON-RPC payload with a peer.
type Transport interface {
	// RoundTrip synchronously exchanges payload using the supplied context.
	// Implementations own cancellation and acquisition bounds; Client checks reply
	// size only after return. Treat payload as borrowed until return, and return
	// reply bytes safe for the caller to inspect without concurrent mutation.
	RoundTrip(context.Context, []byte) ([]byte, error)
}

// TransportFunc adapts a function to Transport.
type TransportFunc func(context.Context, []byte) ([]byte, error)

// RoundTrip calls function with ctx and payload.
func (function TransportFunc) RoundTrip(ctx context.Context, payload []byte) ([]byte, error) {
	return function(ctx, payload)
}

// IDGenerator supplies request IDs to a Client.
type IDGenerator interface {
	// NextID returns a valid non-missing ID for correlation. Client calls may invoke
	// it concurrently; a custom implementation must synchronize its own state.
	// Duplicate generated IDs within a batch are rejected before transport.
	NextID() ID
}

// AtomicIDGenerator generates concurrency-safe numeric IDs by int64 increments.
// Its zero value starts at 1. It must not be copied after use; arithmetic wraps
// at int64 overflow, so it is not an unlimited monotonic or unique-ID guarantee.
type AtomicIDGenerator struct{ value atomic.Int64 }

// NewAtomicIDGenerator creates a generator whose first ID is start+1 using
// wrapping int64 arithmetic.
func NewAtomicIDGenerator(start int64) *AtomicIDGenerator {
	generator := &AtomicIDGenerator{}
	generator.value.Store(start)
	return generator
}

// NextID atomically increments the generator and returns the new numeric ID.
func (generator *AtomicIDGenerator) NextID() ID {
	return NumberID(json.Number(strconv.FormatInt(generator.value.Add(1), 10)))
}

// ClientOption configures a Client during construction.
type ClientOption func(*Client)

// WithIDGenerator borrows a non-nil request ID generator; nil leaves the
// existing generator unchanged. Typed-nil implementations are not filtered and
// must not be supplied. It does not validate future IDs in advance.
func WithIDGenerator(generator IDGenerator) ClientOption {
	return func(client *Client) {
		if generator != nil {
			client.ids = generator
		}
	}
}

// WithMaxClientResponseBytes changes the four-MiB parsing limit for positive
// values. Non-positive values leave it unchanged. Transport acquisition occurs
// before this check and needs its own bounds.
func WithMaxClientResponseBytes(limit int64) ClientOption {
	return func(client *Client) {
		if limit > 0 {
			client.maxResponseBytes = limit
		}
	}
}

// WithMaxClientBatchItems changes the default limit of 1,024 calls sent and
// response members inspected for one client batch. Non-positive values leave
// the existing bound unchanged.
func WithMaxClientBatchItems(limit int) ClientOption {
	return func(client *Client) {
		if limit > 0 {
			client.maxBatchItems = limit
		}
	}
}

// Client validates requests and correlates responses over a Transport. Its
// default AtomicIDGenerator is safe for concurrent calls; custom transports,
// generators, and BatchCall values retain their own concurrency contracts.
// Construct it with NewClient; the zero value is not ready for calls. The client
// borrows collaborators, performs no automatic retries and does not close them.
type Client struct {
	transport        Transport
	ids              IDGenerator
	maxResponseBytes int64
	maxBatchItems    int
}

// NewClient constructs a client. A nil transport is reported as ErrTransport
// when an operation is attempted. Nil options are ignored.
func NewClient(transport Transport, options ...ClientOption) *Client {
	client := &Client{
		transport:        transport,
		ids:              NewAtomicIDGenerator(0),
		maxResponseBytes: defaultMaxClientResponseBytes,
		maxBatchItems:    defaultMaxClientBatchItems,
	}
	for _, option := range options {
		if option != nil {
			option(client)
		}
	}
	return client
}

// Call sends one request, validates and correlates its response, and decodes a
// successful result into result when result is non-nil. The target must be a
// suitable json.Unmarshal destination and is caller-owned; decoding errors may
// leave it partially updated. A valid peer failure is returned as *Error; other
// failures preserve their category/causes. The transport enforces cancellation;
// Client does not preflight ctx.Err or impose a separate deadline.
func (client *Client) Call(ctx context.Context, method string, params, result any) error {
	request, err := NewRequest(method, params, client.ids.NextID())
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(request)
	reply, err := client.roundTrip(ctx, payload)
	if err != nil {
		return err
	}
	var response Response
	if err := json.Unmarshal(reply, &response); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}
	if err := response.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}
	if !request.ID.Equal(response.ID) {
		return ErrMismatchedID
	}
	if response.Error != nil {
		return response.Error
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(response.Result, result); err != nil {
		return fmt.Errorf("%w: result: %w", ErrInvalidResponse, err)
	}
	return nil
}

// Call sends one request through a constructed, non-nil client and decodes T.
// On failure the returned value may contain partial decoding; consult the error.
func Call[T any](ctx context.Context, client *Client, method string, params any) (T, error) {
	var result T
	err := client.Call(ctx, method, params, &result)
	return result, err
}

// Notify sends a notification and requires an empty or whitespace-only reply.
// Success proves transport acceptance, not remote method completion. JSON-RPC
// notifications have no correlated outcome; callers own replay/observability.
func (client *Client) Notify(ctx context.Context, method string, params any) error {
	request, err := NewNotification(method, params)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(request)
	reply, err := client.roundTrip(ctx, payload)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(reply)) != 0 {
		return ErrUnexpectedResponse
	}
	return nil
}

// BatchCall describes one member of a client batch. Batch writes a decoded
// success into Result and assigns a protocol failure to Error. These fields and
// internal correlation state are mutated; do not share calls or result targets
// concurrently. Earlier members may be updated even if Batch later returns an error.
type BatchCall struct {
	// Method is the JSON-RPC method name.
	Method string
	// Params must encode as an object or array when non-nil.
	Params any
	// Result is a caller-owned json.Unmarshal destination for success, when non-nil.
	Result any
	// Notification omits the request ID and expects no response member.
	Notification bool
	// Error receives a per-call protocol failure and is reset as the batch is built.
	// A nil Batch error does not mean every call succeeded; inspect each Error.
	Error *Error

	id ID
}

// Batch sends calls together and correlates every non-notification response by
// ID. It rejects empty batches, nil calls, duplicate generated IDs, and
// malformed response membership. Replies may arrive in any order. Valid per-call
// protocol failures populate BatchCall.Error without making Batch itself fail;
// envelope/transport/decoding failures return an error. Result mutation is not
// transactional and is not rolled back when a later reply fails validation.
func (client *Client) Batch(ctx context.Context, calls ...*BatchCall) error {
	if len(calls) == 0 {
		return ErrEmptyBatch
	}
	if len(calls) > client.maxBatchItems {
		return ErrClientBatchTooLarge
	}
	requests := make([]Request, 0, len(calls))
	pending := make(map[string]*BatchCall, len(calls))
	for _, call := range calls {
		if call == nil {
			return errors.New("jsonrpc: nil batch call")
		}
		call.Error = nil
		var request Request
		var err error
		if call.Notification {
			request, err = NewNotification(call.Method, call.Params)
		} else {
			call.id = client.ids.NextID()
			request, err = NewRequest(call.Method, call.Params, call.id)
			key := idKey(call.id)
			if _, duplicate := pending[key]; duplicate {
				return ErrDuplicateRequestID
			}
			pending[key] = call
		}
		if err != nil {
			return err
		}
		requests = append(requests, request)
	}
	payload, _ := json.Marshal(requests)
	reply, err := client.roundTrip(ctx, payload)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		if len(bytes.TrimSpace(reply)) != 0 {
			return ErrUnexpectedResponse
		}
		return nil
	}
	trimmed := bytes.TrimSpace(reply)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return ErrInvalidResponse
	}
	if err := validateBatchResponseCount(trimmed, client.maxBatchItems); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	_, _ = decoder.Token()
	seen := make(map[string]struct{}, len(pending))
	for decoder.More() {
		var response Response
		if err := decoder.Decode(&response); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidResponse, err)
		}
		if err := response.Validate(); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidResponse, err)
		}
		key := idKey(response.ID)
		call, ok := pending[key]
		if !ok {
			return ErrMismatchedID
		}
		if _, duplicate := seen[key]; duplicate {
			return ErrDuplicateResponse
		}
		seen[key] = struct{}{}
		if response.Error != nil {
			call.Error = response.Error
			continue
		}
		if call.Result != nil {
			if err := json.Unmarshal(response.Result, call.Result); err != nil {
				return fmt.Errorf("%w: result: %w", ErrInvalidResponse, err)
			}
		}
	}
	if len(seen) != len(pending) {
		return ErrMissingResponse
	}
	return nil
}

func validateBatchResponseCount(payload []byte, limit int) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('[') {
		return ErrInvalidResponse
	}
	count := 0
	for decoder.More() {
		count++
		if count > limit {
			return ErrUnexpectedResponse
		}
		var response json.RawMessage
		if err := decoder.Decode(&response); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidResponse, err)
		}
	}
	if _, err := decoder.Token(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return ErrInvalidResponse
	}
	return nil
}

func (client *Client) roundTrip(ctx context.Context, payload []byte) ([]byte, error) {
	if client.transport == nil {
		return nil, fmt.Errorf("%w: nil transport", ErrTransport)
	}
	reply, err := client.transport.RoundTrip(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTransport, err)
	}
	if int64(len(reply)) > client.maxResponseBytes {
		return nil, ErrClientResponseTooLarge
	}
	return reply, nil
}

// NewRequest validates a non-missing ID and rejects reserved rpc. application
// method names (empty names are permitted). Nil params omit the member; nonnil
// params must JSON-encode as an object or array. Encoding allocates owned Params;
// the supplied ID is a value copy sharing its raw storage. Explicit NullID is
// accepted as a request per JSONRPC-DEC-002, not treated as a notification.
func NewRequest(method string, params any, id ID) (Request, error) {
	if err := validateClientMethod(method); err != nil {
		return Request{}, err
	}
	if id.Kind() == IDMissing {
		return Request{}, errors.New("jsonrpc: request id is required")
	}
	if !id.valid() {
		return Request{}, errors.New("jsonrpc: request id is invalid")
	}
	encoded, err := encodeParams(params)
	if err != nil {
		return Request{}, err
	}
	return Request{JSONRPC: Version, Method: method, Params: encoded, ID: id, idSet: true, methodSet: true}, nil
}

// NewNotification validates method/params as NewRequest does, omits the ID
// member and therefore cannot receive a correlated protocol response.
func NewNotification(method string, params any) (Request, error) {
	if err := validateClientMethod(method); err != nil {
		return Request{}, err
	}
	encoded, err := encodeParams(params)
	if err != nil {
		return Request{}, err
	}
	return Request{JSONRPC: Version, Method: method, Params: encoded, methodSet: true}, nil
}

func validateClientMethod(method string) error {
	if strings.HasPrefix(method, "rpc.") {
		return ErrInvalidMethodName
	}
	return nil
}

func encodeParams(params any) (json.RawMessage, error) {
	if params == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("jsonrpc: encode params: %w", err)
	}
	trimmed := bytes.TrimSpace(encoded)
	if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
		return nil, errors.New("jsonrpc: params must encode as an object or array")
	}
	return encoded, nil
}

func idKey(id ID) string { return strconv.Itoa(int(id.Kind())) + ":" + id.canonical }
