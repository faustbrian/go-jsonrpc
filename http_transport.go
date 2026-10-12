package jsonrpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
)

// HTTP transport errors distinguish status, content-type, and body-limit
// failures and support errors.Is through direct return or wrapping.
var (
	// ErrHTTPStatus identifies a non-200, non-204 response after bounded body acquisition.
	ErrHTTPStatus = errors.New("jsonrpc: unexpected HTTP status")
	// ErrHTTPContentType identifies an unsupported content type on an HTTP 200 reply.
	ErrHTTPContentType = errors.New("jsonrpc: invalid HTTP response content type")
	// ErrResponseTooLarge identifies an HTTP reply body above the acquisition limit.
	ErrResponseTooLarge = errors.New("jsonrpc: HTTP response too large")
)

const (
	defaultMaxResponseBytes       int64 = 4 << 20
	defaultMaxHTTPDiagnosticBytes       = 4 << 10
	defaultHTTPTimeout                  = 30 * time.Second
)

var defaultHTTPTransport = func() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return transport
}()

var defaultHTTPClient = &http.Client{
	Transport: defaultHTTPTransport,
	Timeout:   defaultHTTPTimeout,
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

type httpRequestError struct{ cause error }

// Error returns a credential-safe request-failure description.
func (*httpRequestError) Error() string { return "jsonrpc: HTTP request failed" }

// Unwrap exposes the original transport failure, which may contain an endpoint
// or other sensitive data even though Error omits it.
func (err *httpRequestError) Unwrap() error { return err.cause }

// HTTPStatusError reports a non-200 HTTP response. Body is empty by default and
// populated only when the transport enables a bounded diagnostic preview.
type HTTPStatusError struct {
	// StatusCode is the peer's HTTP response status.
	StatusCode int
	// Body is an untrusted, control-character-sanitized preview bounded to four
	// KiB. It is empty by default, and Error deliberately omits it.
	Body string
}

// Error returns only the status code. Body is excluded because it is
// controlled by the remote peer and may contain secrets.
func (err *HTTPStatusError) Error() string {
	return fmt.Sprintf("%s: %d", ErrHTTPStatus, err.StatusCode)
}

// Unwrap returns ErrHTTPStatus.
func (err *HTTPStatusError) Unwrap() error { return ErrHTTPStatus }

// HTTPTransportOption configures an HTTPTransport during construction.
type HTTPTransportOption func(*HTTPTransport)

// WithHTTPClient installs a non-nil HTTP client. Its timeout, redirect, proxy,
// DNS, and dial policies remain caller-owned. The client is borrowed, not
// cloned or closed; nil leaves the existing client unchanged.
func WithHTTPClient(client *http.Client) HTTPTransportOption {
	return func(transport *HTTPTransport) {
		switch client {
		case nil:
		default:
			transport.client = client
		}
	}
}

// WithHTTPHeader adds a header to each request. Content-Type and Accept are
// always overwritten with JSON values during RoundTrip.
func WithHTTPHeader(name, value string) HTTPTransportOption {
	return func(transport *HTTPTransport) { transport.headers.Set(name, value) }
}

// WithMaxResponseBytes changes the four-MiB HTTP acquisition limit for positive
// values; non-positive values leave it unchanged. Client parsing has its own limit.
func WithMaxResponseBytes(limit int64) HTTPTransportOption {
	return func(transport *HTTPTransport) {
		if limit > 0 {
			transport.maxResponseBytes = limit
		}
	}
}

// WithHTTPDiagnosticPreviewBytes opts into retaining up to limit bytes of a
// non-success response body in HTTPStatusError.Body. Values above four KiB are
// clamped to four KiB. Non-positive values leave an existing preview setting
// unchanged. Control sanitation is not secret redaction; Error never prints Body.
func WithHTTPDiagnosticPreviewBytes(limit int64) HTTPTransportOption {
	return func(transport *HTTPTransport) {
		if limit > 0 {
			transport.maxDiagnosticBytes = min(limit, int64(defaultMaxHTTPDiagnosticBytes))
		}
	}
}

// HTTPTransport exchanges JSON-RPC payloads over HTTP POST. Construct it with
// NewHTTPTransport; the zero value is not usable. Concurrent RoundTrip calls are
// supported with a concurrency-safe borrowed client and construction-only options.
// It starts no owned background worker and does not close client idle connections.
type HTTPTransport struct {
	endpoint           string
	client             *http.Client
	headers            http.Header
	maxResponseBytes   int64
	maxDiagnosticBytes int64
}

// NewHTTPTransport validates an HTTP(S) endpoint without URL user information
// and constructs a transport. The default client has a 30-second timeout and
// does not follow redirects or use environment proxies. Nil options are ignored.
// HTTP(S), loopback/private destinations and query strings are allowed; endpoint
// validation is not SSRF protection. Callers own origin/DNS/TLS/egress policy.
func NewHTTPTransport(endpoint string, options ...HTTPTransportOption) (*HTTPTransport, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || parsed.User != nil ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("jsonrpc: invalid HTTP endpoint")
	}
	transport := &HTTPTransport{
		endpoint:         parsed.String(),
		client:           defaultHTTPClient,
		headers:          make(http.Header),
		maxResponseBytes: defaultMaxResponseBytes,
	}
	for _, option := range options {
		switch option {
		case nil:
		default:
			option(transport)
		}
	}
	return transport, nil
}

// RoundTrip posts payload and returns a bounded JSON response. A 204 response
// returns nil without reading/validating a JSON body (JSONRPC-DEC-009); other
// non-200 statuses return *HTTPStatusError after bounded body acquisition. Size
// or read failures can precede status classification. A 200 response needs an
// accepted content type, but envelope validation belongs to Client.
// Payload is borrowed until return and returned body bytes are caller-owned.
// Every received body is closed. Context cancellation and the configured HTTP
// client bound network work; no retry or extra deadline is imposed here.
func (transport *HTTPTransport) RoundTrip(ctx context.Context, payload []byte) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, transport.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("jsonrpc: create HTTP request: %w", err)
	}
	request.Header = transport.headers.Clone()
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := transport.client.Do(request)
	if err != nil {
		return nil, &httpRequestError{cause: err}
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	readLimit := transport.maxResponseBytes
	if readLimit < math.MaxInt64 {
		readLimit++
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, readLimit))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > transport.maxResponseBytes {
		return nil, ErrResponseTooLarge
	}
	if response.StatusCode != http.StatusOK {
		return nil, &HTTPStatusError{
			StatusCode: response.StatusCode,
			Body:       sanitizeHTTPDiagnostic(body, transport.maxDiagnosticBytes),
		}
	}
	if !IsJSONContentType(response.Header.Get("Content-Type")) {
		return nil, ErrHTTPContentType
	}
	return body, nil
}

func sanitizeHTTPDiagnostic(body []byte, limit int64) string {
	if limit == 0 {
		return ""
	}
	body = body[:min(int64(len(body)), limit)]
	text := strings.ToValidUTF8(string(body), "�")
	sanitized := strings.Map(func(character rune) rune {
		if unicode.IsControl(character) {
			return ' '
		}
		return character
	}, text)
	sanitized = sanitized[:min(int64(len(sanitized)), limit)]
	return strings.TrimSpace(strings.ToValidUTF8(sanitized, ""))
}
