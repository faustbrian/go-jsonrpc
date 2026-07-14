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
		"docs/middleware.md":      {"Observability", "Authentication", "correlation"},
		"docs/cookbook.md":        {"Custom application errors", "Batch", "Custom transport"},
		"docs/adoption.md":        {"Inventory", "Shadow", "Rollout", "Rollback"},
		"docs/faq.md":             {"notification", "batch", "WebSocket"},
		"docs/troubleshooting.md": {"Parse error", "Invalid Request", "Method not found", "Invalid params"},
		"docs/compatibility.md":   {"Semantic Versioning", "Wire compatibility", "Pre-v1"},
		"docs/releasing.md":       {"Release checklist", "semantic version", "tag"},
		"CHANGELOG.md":            {"Unreleased", "Keep a Changelog"},
		"ROADMAP.md":              {"v1.0.0", "WebSocket", "OpenRPC"},
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
