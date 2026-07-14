package main

import (
	"fmt"
	"os"

	"github.com/shipit-dev/go-jsonrpc/internal/semver"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: semvercheck vX.Y.Z")
		os.Exit(2)
	}
	if err := semver.ValidateTag(os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "invalid release tag %q: %v\n", os.Args[1], err)
		os.Exit(1)
	}
}
