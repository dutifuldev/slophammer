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
