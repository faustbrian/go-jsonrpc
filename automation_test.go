package jsonrpc

import (
	"os"
	"strings"
	"testing"
)

func TestRepositoryAutomationContract(t *testing.T) {
	t.Parallel()

	required := map[string][]string{
		"CLAUDE.md": {"@AGENTS.md"},
		"Makefile": {
			"release-patch",
			"release-minor",
			"release-major",
			"scripts/release.sh",
		},
		"llms.txt": {
			"# go-jsonrpc",
			"llms-full.txt",
			"docs/quickstart.md",
		},
		"llms-full.txt": {
			"# go-jsonrpc",
			"# Quickstart",
		},
		"README.md": {"llms.txt", "llms-full.txt"},
		".github/workflows/ci.yml": {
			"go test -race ./...",
			"scripts/check-coverage.sh",
			"staticcheck ./...",
			"go vet ./...",
			"gofmt",
			"scripts/check-docs.sh",
		},
		".github/workflows/fuzz.yml": {
			"FuzzDispatcher",
			"FuzzRequestUnmarshal",
			"FuzzResponseUnmarshal",
			"FuzzErrorUnmarshal",
			"FuzzIDRoundTrip",
			"FuzzClientCorrelation",
			"FuzzClientBatchCorrelation",
		},
		".github/workflows/benchmark.yml": {"-bench=."},
		".github/workflows/security.yml":  {"govulncheck"},
		".github/workflows/release.yml": {
			"tags:",
			`"v*"`,
			"go run ./cmd/semvercheck",
			"merge-base --is-ancestor",
			"gh release create",
		},
		".github/dependabot.yml":    {"gomod", "github-actions"},
		"scripts/check-coverage.sh": {"100.0%"},
		"scripts/check-docs.sh": {
			"go test ./...",
			"examples/e2e",
			"Markdown link",
			"generate-llms.py --check",
		},
		"scripts/generate-llms.py": {"README.md", "--check"},
		"scripts/release.sh": {
			"git tag -a",
			"origin/main",
			"scripts/check-coverage.sh",
			"FuzzDispatcher",
			"FuzzRequestUnmarshal",
			"FuzzResponseUnmarshal",
			"FuzzErrorUnmarshal",
			"FuzzIDRoundTrip",
			"FuzzClientCorrelation",
			"FuzzClientBatchCorrelation",
		},
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

func TestRepositoryRequiresGo12512(t *testing.T) {
	t.Parallel()

	required := map[string]string{
		"go.mod":          "go 1.25.12",
		"README.md":       "Go 1.25.12 or newer",
		"CONTRIBUTING.md": "Go 1.25.12 or newer",
	}
	for path, fragment := range required {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, err)
		}
		if !strings.Contains(string(contents), fragment) {
			t.Errorf("%s does not contain %q", path, fragment)
		}
	}
}
