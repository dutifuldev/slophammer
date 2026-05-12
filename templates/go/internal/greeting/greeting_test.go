package greeting

import "testing"

func TestCreateTrimsName(t *testing.T) {
	got, err := Create(Input{Name: " Ada "})
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}

	want := "Hello, Ada."
	if got != want {
		t.Fatalf("Create() = %q, want %q", got, want)
	}
}

func TestCreateRejectsEmptyName(t *testing.T) {
	_, err := Create(Input{Name: " "})
	if err == nil {
		t.Fatal("Create returned nil error")
	}
}
