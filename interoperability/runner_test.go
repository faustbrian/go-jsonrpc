package main

import (
	"bytes"
	"os"
	"testing"
)

func TestObservationsMatchRecordedMatrix(t *testing.T) {
	t.Parallel()

	observations, err := observe()
	if err != nil {
		t.Fatal(err)
	}
	var actual bytes.Buffer
	if err := write(&actual, observations); err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile("../specification/interoperability.tsv")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual.Bytes(), expected) {
		t.Fatalf("interoperability matrix differs\nactual:\n%s\nexpected:\n%s", actual.Bytes(), expected)
	}
}
