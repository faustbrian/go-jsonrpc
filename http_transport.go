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
	ErrHTTPStatus       = errors.New("jsonrpc: unexpected HTTP status")
	ErrHTTPContentType  = errors.New("jsonrpc: invalid HTTP response content type")
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

// Unwrap preserves programmatic inspection of the transport failure.
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
// DNS, and dial policies remain caller-owned.
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

// WithMaxResponseBytes changes the default four-MiB HTTP response-body limit.
func WithMaxResponseBytes(limit int64) HTTPTransportOption {
	return func(transport *HTTPTransport) {
		if limit > 0 {
			transport.maxResponseBytes = limit
		}
	}
}

// WithHTTPDiagnosticPreviewBytes opts into retaining up to limit bytes of a
// non-success response body in HTTPStatusError.Body. Values above four KiB are
// clamped to four KiB. The preview remains untrusted and Error never prints it.
func WithHTTPDiagnosticPreviewBytes(limit int64) HTTPTransportOption {
	return func(transport *HTTPTransport) {
		if limit > 0 {
			transport.maxDiagnosticBytes = min(limit, int64(defaultMaxHTTPDiagnosticBytes))
		}
	}
}

// HTTPTransport exchanges JSON-RPC payloads over HTTP POST.
type HTTPTransport struct {
	endpoint           string
	client             *http.Client
	headers            http.Header
	maxResponseBytes   int64
	maxDiagnosticBytes int64
}

// NewHTTPTransport validates an HTTP(S) endpoint without URL user information
// and constructs a transport. The default client has a 30-second timeout and
// does not follow redirects. Nil options are ignored.
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
// returns a nil payload, and non-200 statuses return *HTTPStatusError.
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
