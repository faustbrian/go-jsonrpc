package jsonrpc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

var (
	ErrHTTPStatus       = errors.New("jsonrpc: unexpected HTTP status")
	ErrHTTPContentType  = errors.New("jsonrpc: invalid HTTP response content type")
	ErrResponseTooLarge = errors.New("jsonrpc: HTTP response too large")
)

const defaultMaxResponseBytes int64 = 4 << 20

type HTTPStatusError struct {
	StatusCode int
	Body       string
}

func (err *HTTPStatusError) Error() string {
	if err.Body == "" {
		return fmt.Sprintf("%s: %d", ErrHTTPStatus, err.StatusCode)
	}
	return fmt.Sprintf("%s: %d: %s", ErrHTTPStatus, err.StatusCode, err.Body)
}

func (err *HTTPStatusError) Unwrap() error { return ErrHTTPStatus }

type HTTPTransportOption func(*HTTPTransport)

func WithHTTPClient(client *http.Client) HTTPTransportOption {
	return func(transport *HTTPTransport) {
		if client != nil {
			transport.client = client
		}
	}
}

func WithHTTPHeader(name, value string) HTTPTransportOption {
	return func(transport *HTTPTransport) { transport.headers.Set(name, value) }
}

func WithMaxResponseBytes(limit int64) HTTPTransportOption {
	return func(transport *HTTPTransport) {
		if limit > 0 {
			transport.maxResponseBytes = limit
		}
	}
}

type HTTPTransport struct {
	endpoint         string
	client           *http.Client
	headers          http.Header
	maxResponseBytes int64
}

func NewHTTPTransport(endpoint string, options ...HTTPTransportOption) (*HTTPTransport, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("jsonrpc: invalid HTTP endpoint %q", endpoint)
	}
	transport := &HTTPTransport{
		endpoint:         parsed.String(),
		client:           http.DefaultClient,
		headers:          make(http.Header),
		maxResponseBytes: defaultMaxResponseBytes,
	}
	for _, option := range options {
		option(transport)
	}
	return transport, nil
}

func (transport *HTTPTransport) RoundTrip(ctx context.Context, payload []byte) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, transport.endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header = transport.headers.Clone()
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	response, err := transport.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNoContent {
		return nil, nil
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, transport.maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, &HTTPStatusError{
			StatusCode: response.StatusCode,
			Body:       strings.TrimSpace(string(body)),
		}
	}
	if !IsJSONContentType(response.Header.Get("Content-Type")) {
		return nil, ErrHTTPContentType
	}
	if int64(len(body)) > transport.maxResponseBytes {
		return nil, ErrResponseTooLarge
	}
	return body, nil
}
