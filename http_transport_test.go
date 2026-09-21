package jsonrpc

import (
	"context"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPTransportMaxInt64ResponseLimitDoesNotOverflow(t *testing.T) {
	t.Parallel()

	const payload = `{"jsonrpc":"2.0","result":true,"id":1}`
	transport, err := NewHTTPTransport("https://example.test/rpc", WithHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(payload)),
			}, nil
		}),
	}), WithMaxResponseBytes(math.MaxInt64))
	if err != nil {
		t.Fatal(err)
	}

	reply, err := transport.RoundTrip(context.Background(), []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(reply) != payload {
		t.Fatalf("RoundTrip() reply = %q, want %q", reply, payload)
	}
}

func TestHTTPTransportRoundTrip(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s", request.Method)
		}
		if request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type = %q", request.Header.Get("Content-Type"))
		}
		if request.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q", request.Header.Get("Accept"))
		}
		if request.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("Authorization = %q", request.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(request.Body)
		assertJSONEqual(t, body, []byte(`{"jsonrpc":"2.0","method":"ping","id":1}`))
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"jsonrpc":"2.0","result":"pong","id":1}`))
	}))
	defer server.Close()

	transport, err := NewHTTPTransport(server.URL, WithHTTPHeader("Authorization", "Bearer token"))
	if err != nil {
		t.Fatal(err)
	}
	reply, err := transport.RoundTrip(context.Background(), []byte(`{"jsonrpc":"2.0","method":"ping","id":1}`))
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	assertJSONEqual(t, reply, []byte(`{"jsonrpc":"2.0","result":"pong","id":1}`))
}

func TestHTTPTransportNoContent(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	transport, _ := NewHTTPTransport(server.URL)
	reply, err := transport.RoundTrip(context.Background(), []byte(`{}`))
	if err != nil || reply != nil {
		t.Errorf("RoundTrip() = (%q, %v), want nil, nil", reply, err)
	}
}

func TestHTTPTransportDefaultClientIsolatedFromProcessGlobalTransport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"jsonrpc":"2.0","result":null,"id":1}`))
	}))
	defer server.Close()

	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("process-global transport used")
	})
	defer func() { http.DefaultTransport = originalTransport }()

	transport, err := NewHTTPTransport(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transport.RoundTrip(context.Background(), []byte(`{}`)); err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
}

func TestHTTPTransportDoesNotFollowRedirectsByDefault(t *testing.T) {
	t.Parallel()

	targetCalled := false
	target := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		targetCalled = true
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"jsonrpc":"2.0","result":null,"id":1}`))
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Redirect(writer, request, target.URL, http.StatusTemporaryRedirect)
	}))
	defer source.Close()

	transport, _ := NewHTTPTransport(source.URL, WithHTTPHeader("X-API-Key", "secret"))
	_, err := transport.RoundTrip(context.Background(), []byte(`{}`))
	if !errors.Is(err, ErrHTTPStatus) {
		t.Fatalf("RoundTrip(redirect) error = %v, want HTTP status error", err)
	}
	if targetCalled {
		t.Fatal("default transport followed a redirect and forwarded the request")
	}

	optedIn, _ := NewHTTPTransport(source.URL, WithHTTPClient(&http.Client{}))
	if _, err := optedIn.RoundTrip(context.Background(), []byte(`{}`)); err != nil {
		t.Fatalf("RoundTrip(explicit redirect client) error = %v", err)
	}
	if !targetCalled {
		t.Fatal("explicit client redirect policy was not honored")
	}
}

func TestHTTPTransportValidation(t *testing.T) {
	t.Parallel()

	if _, err := NewHTTPTransport("://invalid"); err == nil {
		t.Error("NewHTTPTransport(invalid) unexpectedly succeeded")
	}
	if _, err := NewHTTPTransport("ftp://example.com/rpc"); err == nil {
		t.Error("NewHTTPTransport(ftp) unexpectedly succeeded")
	}

	tests := []struct {
		name        string
		status      int
		contentType string
		body        string
		limit       int64
		want        error
	}{
		{name: "status", status: http.StatusBadGateway, contentType: "text/plain", body: "upstream failed", want: ErrHTTPStatus},
		{name: "media type", status: http.StatusOK, contentType: "text/plain", body: `{}`, want: ErrHTTPContentType},
		{name: "response too large", status: http.StatusOK, contentType: "application/json", body: "12345", limit: 4, want: ErrResponseTooLarge},
		{name: "error response too large", status: http.StatusBadGateway, contentType: "text/plain", body: "12345", limit: 4, want: ErrResponseTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", tt.contentType)
				writer.WriteHeader(tt.status)
				_, _ = writer.Write([]byte(tt.body))
			}))
			defer server.Close()
			options := []HTTPTransportOption{}
			if tt.limit > 0 {
				options = append(options, WithMaxResponseBytes(tt.limit))
			}
			transport, _ := NewHTTPTransport(server.URL, options...)
			_, err := transport.RoundTrip(context.Background(), []byte(`{}`))
			if !errors.Is(err, tt.want) {
				t.Errorf("RoundTrip() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestHTTPTransportDiagnosticsDoNotExposeUntrustedSecrets(t *testing.T) {
	t.Parallel()

	const endpointSecret = "endpoint-password"
	if _, err := NewHTTPTransport("https://user:" + endpointSecret + "@example.test/rpc"); err == nil {
		t.Fatal("NewHTTPTransport(credential-bearing endpoint) unexpectedly succeeded")
	} else if strings.Contains(err.Error(), endpointSecret) {
		t.Fatalf("endpoint validation error exposed credentials: %v", err)
	}

	bodySecret := "response-secret"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusBadGateway)
		_, _ = writer.Write([]byte(bodySecret + "\r\nforged-log" + strings.Repeat("x", 4<<10)))
	}))
	defer server.Close()

	transport, err := NewHTTPTransport(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = transport.RoundTrip(context.Background(), []byte(`{}`))
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("RoundTrip() error = %v, want *HTTPStatusError", err)
	}
	if strings.Contains(statusErr.Error(), bodySecret) || strings.ContainsAny(statusErr.Error(), "\r\n") {
		t.Fatalf("HTTPStatusError.Error() exposed untrusted body: %q", statusErr.Error())
	}
	if statusErr.Body != "" {
		t.Fatalf("HTTPStatusError.Body retained peer-controlled text by default: %q", statusErr.Body)
	}
}

func TestHTTPTransportDiagnosticPreviewRequiresBoundedOptIn(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusBadGateway)
		_, _ = writer.Write([]byte("diagnostic\r\n" + strings.Repeat("x", 128)))
	}))
	defer server.Close()

	transport, err := NewHTTPTransport(server.URL, WithHTTPDiagnosticPreviewBytes(32))
	if err != nil {
		t.Fatal(err)
	}
	_, err = transport.RoundTrip(context.Background(), []byte(`{}`))
	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("RoundTrip() error = %v, want *HTTPStatusError", err)
	}
	if statusErr.Body == "" || len(statusErr.Body) > 32 {
		t.Fatalf("HTTPStatusError.Body length = %d, want 1..32", len(statusErr.Body))
	}
	if strings.ContainsAny(statusErr.Body, "\r\n") {
		t.Fatalf("HTTPStatusError.Body contains raw line controls: %q", statusErr.Body)
	}
}

func TestHTTPTransportDiagnosticPreviewOptionBounds(t *testing.T) {
	t.Parallel()

	transport, err := NewHTTPTransport(
		"https://example.test/rpc",
		WithHTTPDiagnosticPreviewBytes(1),
		WithHTTPDiagnosticPreviewBytes(0),
		WithHTTPDiagnosticPreviewBytes(-1),
		WithHTTPDiagnosticPreviewBytes(defaultMaxHTTPDiagnosticBytes+1),
	)
	if err != nil {
		t.Fatal(err)
	}
	if transport.maxDiagnosticBytes != defaultMaxHTTPDiagnosticBytes {
		t.Fatalf("diagnostic preview limit = %d, want %d", transport.maxDiagnosticBytes, defaultMaxHTTPDiagnosticBytes)
	}
	unchanged, err := NewHTTPTransport(
		"https://example.test/rpc",
		WithHTTPDiagnosticPreviewBytes(1),
		WithHTTPDiagnosticPreviewBytes(0),
	)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.maxDiagnosticBytes != 1 {
		t.Fatalf("zero diagnostic option reset limit to %d", unchanged.maxDiagnosticBytes)
	}
	if got := sanitizeHTTPDiagnostic([]byte{0xff, 0xff}, 2); got != "" {
		t.Fatalf("sanitizeHTTPDiagnostic(invalid UTF-8) = %q, want empty bounded value", got)
	}
}

func TestHTTPTransportDefaultClientHasTimeout(t *testing.T) {
	t.Parallel()

	transport, err := NewHTTPTransport("https://example.test/rpc")
	if err != nil {
		t.Fatal(err)
	}
	if transport.client.Timeout <= 0 {
		t.Fatalf("default HTTP client timeout = %s, want positive", transport.client.Timeout)
	}
}

func TestHTTPTransportDefaultClientDoesNotUseEnvironmentProxy(t *testing.T) {
	t.Parallel()

	transport, err := NewHTTPTransport("https://example.test/rpc")
	if err != nil {
		t.Fatal(err)
	}
	httpTransport, ok := transport.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("default HTTP transport type = %T, want *http.Transport", transport.client.Transport)
	}
	if httpTransport.Proxy != nil {
		t.Fatal("default HTTP transport consults process proxy configuration")
	}
}

func TestHTTPTransportNetworkErrorOmitsEndpointSecrets(t *testing.T) {
	t.Parallel()

	const endpointSecret = "query-secret"
	underlying := errors.New("dial failed")
	transport, err := NewHTTPTransport(
		"https://example.test/rpc?access_token="+endpointSecret,
		WithHTTPClient(&http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, underlying
		})}),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = transport.RoundTrip(context.Background(), []byte(`{}`))
	if err == nil {
		t.Fatal("RoundTrip(network failure) unexpectedly succeeded")
	}
	if strings.Contains(err.Error(), endpointSecret) {
		t.Fatalf("RoundTrip(network failure) exposed endpoint credentials: %v", err)
	}
	if !errors.Is(err, underlying) {
		t.Fatalf("RoundTrip(network failure) lost underlying error: %v", err)
	}
}

func TestHTTPTransportNetworkError(t *testing.T) {
	t.Parallel()

	underlying := errors.New("network down")
	transport, _ := NewHTTPTransport("http://example.invalid", WithHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, underlying
		}),
	}))
	_, err := transport.RoundTrip(context.Background(), []byte(`{}`))
	if err == nil || !errors.Is(err, underlying) {
		t.Errorf("RoundTrip() error = %v, want wrapped network failure", err)
	}
	if strings.Contains(err.Error(), underlying.Error()) {
		t.Errorf("RoundTrip() error exposed transport diagnostic: %v", err)
	}
}

func TestHTTPTransportRejectsNilContextWithoutNetworkIO(t *testing.T) {
	t.Parallel()

	called := false
	transport, _ := NewHTTPTransport("http://example.test", WithHTTPClient(&http.Client{
		Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
			called = true
			return nil, errors.New("unexpected network call")
		}),
	}))
	//lint:ignore SA1012 Public boundary must reject nil context before network I/O.
	if _, err := transport.RoundTrip(nil, []byte(`{}`)); err == nil { //nolint:staticcheck // verifies defensive nil handling
		t.Fatal("RoundTrip(nil context) unexpectedly succeeded")
	}
	if called {
		t.Fatal("RoundTrip(nil context) performed network I/O")
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (function roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}
