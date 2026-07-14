package jsonrpc

import (
	"os"
	"strings"
	"testing"
)

func TestRepositoryAutomationContract(t *testing.T) {
	t.Parallel()

	required := map[string][]string{
		".github/workflows/ci.yml": {
			"go test -race ./...",
			"scripts/check-coverage.sh",
			"staticcheck ./...",
			"go vet ./...",
			"gofmt",
		},
		".github/workflows/fuzz.yml": {
			"FuzzDispatcher",
			"FuzzRequestUnmarshal",
		},
		".github/workflows/benchmark.yml": {"-bench=."},
		".github/workflows/security.yml":  {"govulncheck"},
		".github/workflows/release.yml": {
			"tags:",
			`"v*"`,
			"gh release create",
		},
		".github/dependabot.yml":    {"gomod", "github-actions"},
		"scripts/check-coverage.sh": {"100.0%"},
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
