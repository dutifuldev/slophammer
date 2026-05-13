package repo

import "testing"

func TestName(t *testing.T) {
	if Name() != "rules" {
		t.Fatal("Name returned unexpected value")
	}
}
