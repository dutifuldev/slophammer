package rules

import "testing"

func TestLineHasGoCommandSignal(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{name: "go test", line: "go test ./...", want: true},
		{name: "go global flag", line: "go -C tools vet ./...", want: true},
		{name: "env prefix", line: "GOFLAGS=-mod=readonly go build ./...", want: true},
		{name: "unsupported command", line: "go version", want: false},
		{name: "after shell separator", line: "cd go && go test ./...", want: true},
		{name: "not command token", line: "echo go test ./...", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := lineHasGoCommandSignal(commandTokens(tt.line)); got != tt.want {
				t.Fatalf("lineHasGoCommandSignal(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestPriorCDWorkingDirectory(t *testing.T) {
	tests := []struct {
		name   string
		line   string
		want   string
		wantOK bool
	}{
		{name: "none", line: "go run ./cmd/slophammer go crap", wantOK: false},
		{name: "simple", line: "cd go && go run ./cmd/slophammer go crap ..", want: "go", wantOK: true},
		{name: "last wins", line: "cd services && cd api && go test ./...", want: "api", wantOK: true},
		{name: "flag ignored", line: "cd - && go test ./...", wantOK: false},
		{name: "separator ignored", line: "cd && go test ./...", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := priorCDWorkingDirectory(commandTokens(tt.line))
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("priorCDWorkingDirectory(%q) = %q, %v; want %q, %v", tt.line, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestFirstSlophammerGoPathArgument(t *testing.T) {
	tests := []struct {
		name   string
		line   string
		want   string
		wantOK bool
	}{
		{name: "default", line: "--scan", wantOK: false},
		{name: "path", line: ".. --scan", want: "..", wantOK: true},
		{name: "flag value", line: "--target internal/rules/rules.go ..", want: "..", wantOK: true},
		{name: "equals flag", line: "--target=internal/rules/rules.go ..", want: "..", wantOK: true},
		{name: "separator stops", line: "&& ..", wantOK: false},
		{name: "blank skipped", line: "   ..", want: "..", wantOK: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := firstSlophammerGoPathArgument(commandTokens(tt.line))
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("firstSlophammerGoPathArgument(%q) = %q, %v; want %q, %v", tt.line, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}
