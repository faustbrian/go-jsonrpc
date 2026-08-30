package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime/debug"
	"strings"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"
	"github.com/creachadair/jrpc2/jhttp"
	jsonrpc "github.com/faustbrian/go-jsonrpc"
)

const (
	localName    = "golib-jsonrpc"
	localVersion = "workspace"
	peerName     = "creachadair/jrpc2"
	peerModule   = "github.com/creachadair/jrpc2"
)

type observation struct {
	decisionID     string
	caseName       string
	implementation string
	version        string
	outcome        string
	classification string
}

func main() {
	observations, err := observe()
	if err != nil {
		panic(err)
	}
	if err := write(os.Stdout, observations); err != nil {
		panic(err)
	}
}

func observe() (observations []observation, err error) {
	registry := jsonrpc.NewRegistry()
	if err := registry.Register("echo", func(context.Context, json.RawMessage) (any, error) {
		return map[string]bool{"ok": true}, nil
	}); err != nil {
		return nil, fmt.Errorf("register local echo handler: %w", err)
	}
	local := jsonrpc.NewHTTPHandler(jsonrpc.NewDispatcher(registry))
	peer := jhttp.NewBridge(handler.Map{
		"echo": func(context.Context, *jrpc2.Request) (any, error) {
			return map[string]bool{"ok": true}, nil
		},
	}, nil)
	defer func() { err = errors.Join(err, peer.Close()) }()

	peerVersion, err := moduleVersion(peerModule)
	if err != nil {
		return nil, err
	}
	add := func(decisionID, caseName, localOutcome, peerOutcome string) {
		classification := "deliberate policy difference"
		if localOutcome == peerOutcome {
			classification = "agreement"
		}
		observations = append(observations,
			observation{decisionID, caseName, localName, localVersion, localOutcome, classification},
			observation{decisionID, caseName, peerName, peerVersion, peerOutcome, classification},
		)
	}

	const nullRequest = `{"jsonrpc":"2.0","id":null,"method":"echo"}`
	localResponse := exchange(local, http.MethodPost, "application/json", nullRequest)
	peerResponse := exchange(peer, http.MethodPost, "application/json", nullRequest)
	if localResponse.status != http.StatusOK || responseID(localResponse.body) != "null" {
		return nil, fmt.Errorf("unexpected local explicit-null response: status=%d body=%q", localResponse.status, localResponse.body)
	}
	if peerResponse.status != http.StatusNoContent || len(peerResponse.body) != 0 {
		return nil, fmt.Errorf("unexpected peer explicit-null response: status=%d body=%q", peerResponse.status, peerResponse.body)
	}
	add("JSONRPC-DEC-002", "explicit-null-id", "request;status=200;response-id=null", "notification;status=204;body=empty")

	localNumeric, err := localNumericEquivalence()
	if err != nil {
		return nil, err
	}
	peerNumeric, err := peerNumericEquivalence()
	if err != nil {
		return nil, err
	}
	const largeNumericID = "9007199254740993"
	largeNumericRequest := `{"jsonrpc":"2.0","id":` + largeNumericID + `,"method":"echo"}`
	localResponse = exchange(local, http.MethodPost, "application/json", largeNumericRequest)
	peerResponse = exchange(peer, http.MethodPost, "application/json", largeNumericRequest)
	if responseID(localResponse.body) != largeNumericID || responseID(peerResponse.body) != largeNumericID {
		return nil, fmt.Errorf("large numeric ID was not preserved: local=%q peer=%q", localResponse.body, peerResponse.body)
	}
	localNumeric += ";" + largeNumericID + "=preserved"
	peerNumeric += ";" + largeNumericID + "=preserved"
	add("JSONRPC-DEC-003", "numeric-id-equivalence", localNumeric, peerNumeric)

	const notification = `{"jsonrpc":"2.0","method":"echo"}`
	localResponse = exchange(local, http.MethodPost, "application/json", notification)
	peerResponse = exchange(peer, http.MethodPost, "application/json", notification)
	if localResponse.status != http.StatusNoContent || len(localResponse.body) != 0 ||
		peerResponse.status != http.StatusNoContent || len(peerResponse.body) != 0 {
		return nil, fmt.Errorf("unexpected notification responses: local=%d/%q peer=%d/%q", localResponse.status, localResponse.body, peerResponse.status, peerResponse.body)
	}
	add("JSONRPC-DEC-009", "notification-only-http", "status=204;body=empty", "status=204;body=empty")

	const call = `{"jsonrpc":"2.0","id":1,"method":"echo"}`
	for _, testCase := range []struct {
		name             string
		method           string
		contentType      string
		payload          string
		localStatus      int
		peerStatus       int
		localOutcome     string
		peerOutcome      string
		responseSemantic string
	}{
		{"post-application-json", http.MethodPost, "application/json", call, 200, 200, "status=200;response=success", "status=200;response=success", "success"},
		{"get-application-json", http.MethodGet, "application/json", call, 405, 405, "status=405", "status=405", ""},
		{"post-application-json-rpc", http.MethodPost, "application/json-rpc", call, 200, 415, "status=200;response=success", "status=415", "local-success"},
		{"post-vendor-json", http.MethodPost, "application/vnd.example+json", call, 200, 415, "status=200;response=success", "status=415", "local-success"},
		{"protocol-error-http-status", http.MethodPost, "application/json", `{"jsonrpc":"2.0","id":1,"method":"missing"}`, 200, 200, "status=200;error=-32601", "status=200;error=-32601", "method-not-found"},
	} {
		localResponse = exchange(local, testCase.method, testCase.contentType, testCase.payload)
		peerResponse = exchange(peer, testCase.method, testCase.contentType, testCase.payload)
		if localResponse.status != testCase.localStatus || peerResponse.status != testCase.peerStatus {
			return nil, fmt.Errorf("unexpected %s statuses: local=%d peer=%d", testCase.name, localResponse.status, peerResponse.status)
		}
		switch testCase.responseSemantic {
		case "success":
			if !responseSucceeded(localResponse.body) || !responseSucceeded(peerResponse.body) {
				return nil, fmt.Errorf("unexpected %s success responses", testCase.name)
			}
		case "local-success":
			if !responseSucceeded(localResponse.body) {
				return nil, fmt.Errorf("unexpected %s local response", testCase.name)
			}
		case "method-not-found":
			if responseErrorCode(localResponse.body) != -32601 || responseErrorCode(peerResponse.body) != -32601 {
				return nil, fmt.Errorf("unexpected %s protocol responses", testCase.name)
			}
		}
		add("JSONRPC-DEC-010", testCase.name, testCase.localOutcome, testCase.peerOutcome)
	}

	return observations, nil
}

type httpResponse struct {
	status int
	body   []byte
}

func exchange(target http.Handler, method, contentType, payload string) httpResponse {
	request := httptest.NewRequestWithContext(context.Background(), method, "http://example.test/rpc", strings.NewReader(payload))
	request.Header.Set("Content-Type", contentType)
	recorder := httptest.NewRecorder()
	target.ServeHTTP(recorder, request)
	result := recorder.Result()
	defer func() { _ = result.Body.Close() }()
	body, _ := io.ReadAll(result.Body)
	return httpResponse{status: result.StatusCode, body: bytes.TrimSpace(body)}
}

func localNumericEquivalence() (string, error) {
	ids := make([]jsonrpc.ID, 3)
	for index, value := range []string{"1", "1.0", "1e0"} {
		if err := json.Unmarshal([]byte(value), &ids[index]); err != nil {
			return "", fmt.Errorf("decode local numeric ID %q: %w", value, err)
		}
	}
	if !ids[0].Equal(ids[1]) || !ids[1].Equal(ids[2]) {
		return "", errors.New("local numeric IDs are not mathematically equivalent")
	}
	return "1=1.0=1e0", nil
}

func peerNumericEquivalence() (string, error) {
	ids := make([]string, 3)
	for index, value := range []string{"1", "1.0", "1e0"} {
		requests, err := jrpc2.ParseRequests([]byte(`{"jsonrpc":"2.0","id":` + value + `,"method":"echo"}`))
		if err != nil {
			return "", fmt.Errorf("parse peer numeric ID %q: %w", value, err)
		}
		if len(requests) != 1 || requests[0].Error != nil {
			return "", fmt.Errorf("peer rejected numeric ID %q", value)
		}
		ids[index] = requests[0].ID
	}
	if ids[0] == ids[1] || ids[1] == ids[2] || ids[0] == ids[2] {
		return "", fmt.Errorf("peer unexpectedly normalized numeric IDs: %q", ids)
	}
	return "1!=1.0!=1e0", nil
}

func responseID(body []byte) string {
	var response struct {
		ID json.RawMessage `json:"id"`
	}
	if json.Unmarshal(body, &response) != nil {
		return ""
	}
	return string(response.ID)
}

func responseSucceeded(body []byte) bool {
	var response struct {
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	return json.Unmarshal(body, &response) == nil && len(response.Result) != 0 && len(response.Error) == 0
}

func responseErrorCode(body []byte) int {
	var response struct {
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &response) != nil {
		return 0
	}
	return response.Error.Code
}

func moduleVersion(path string) (string, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", errors.New("read interoperability build information")
	}
	for _, dependency := range info.Deps {
		if dependency.Path == path {
			return dependency.Version, nil
		}
	}
	return "", fmt.Errorf("missing interoperability module version for %s", path)
}

func write(target io.Writer, observations []observation) error {
	writer := csv.NewWriter(target)
	writer.Comma = '\t'
	if err := writer.Write([]string{"decision_id", "case", "implementation", "version", "outcome", "classification"}); err != nil {
		return err
	}
	for _, observation := range observations {
		if err := writer.Write([]string{
			observation.decisionID,
			observation.caseName,
			observation.implementation,
			observation.version,
			observation.outcome,
			observation.classification,
		}); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}
