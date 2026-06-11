package clean

import "testing"

func TestMessage(t *testing.T) {
	if Message() != "ok" {
		t.Fatal("Message returned unexpected value")
	}
}
