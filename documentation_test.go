package jsonrpc

import (
	"os"
	"strings"
	"testing"
)

func TestCoreDocumentationContract(t *testing.T) {
	t.Parallel()

	required := map[string][]string{
		"README.md": {
			"JSON-RPC 2.0",
			"go get github.com/shipit-dev/go-jsonrpc",
			"Protocol guarantees",
		},
		"docs/quickstart.md":   {"Server", "Client", "Notification", "Batch"},
		"docs/architecture.md": {"Transport", "Protocol", "Dispatch", "Execution"},
		"docs/api.md": {
			"## Protocol",
			"## Server",
			"## Client",
			"## HTTP",
		},
		"examples/server/main.go": {"NewHTTPHandler", "Register"},
		"examples/client/main.go": {"NewHTTPTransport", "Call"},
		"examples/e2e/main.go":    {"httptest.NewServer", "NewClient"},
	}

	for path, fragments := range required {
		path, fragments := path, fragments
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			contents, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile(%q) error = %v", path, err)
			}
			for _, fragment := range fragments {
				if !strings.Contains(string(contents), fragment) {
					t.Errorf("%s does not contain %q", path, fragment)
				}
			}
		})
	}
}
