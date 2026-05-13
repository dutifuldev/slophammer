package gotools

import (
	"reflect"
	"testing"
)

func TestGoRunArgsUsesVersionedPackage(t *testing.T) {
	got := Dry4Go.GoRunArgs(Latest, "--format", "json", ".")
	want := []string{"run", "github.com/unclebob/dry4go/cmd/dry4go@latest", "--format", "json", "."}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GoRunArgs() = %#v, want %#v", got, want)
	}
}

func TestGoRunLineUsesVersionedPackage(t *testing.T) {
	got := Mutate4Go.GoRunLine(Latest, "main.go", "--scan")
	want := "go run github.com/unclebob/mutate4go/cmd/mutate4go@latest main.go --scan"
	if got != want {
		t.Fatalf("GoRunLine() = %q, want %q", got, want)
	}
}
