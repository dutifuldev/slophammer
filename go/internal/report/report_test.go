package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dutifuldev/slophammer/go/internal/rules"
)

func TestWriteJSON(t *testing.T) {
	var out bytes.Buffer
	input := rules.Report{
		OK: false,
		Findings: []rules.Finding{{
			RuleID:   "repo.agents-required",
			Severity: rules.SeverityError,
			Path:     "AGENTS.md",
			Message:  "AGENTS.md is required",
		}},
	}

	if err := WriteJSON(&out, input); err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}
	var decoded rules.Report
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if len(decoded.Findings) != 1 || decoded.Findings[0].RuleID != "repo.agents-required" {
		t.Fatalf("decoded report = %#v", decoded)
	}
}

func TestWriteText(t *testing.T) {
	var out bytes.Buffer
	input := rules.Report{
		OK: false,
		Findings: []rules.Finding{{
			RuleID:   "repo.readme-required",
			Severity: rules.SeverityError,
			Path:     "README.md",
			Message:  "README.md is required",
		}},
	}

	if err := WriteText(&out, input); err != nil {
		t.Fatalf("WriteText returned error: %v", err)
	}
	if !strings.Contains(out.String(), "repo.readme-required") {
		t.Fatalf("text output missing rule ID: %q", out.String())
	}
}

func TestWriteTextOK(t *testing.T) {
	var out bytes.Buffer
	if err := WriteText(&out, rules.Report{OK: true}); err != nil {
		t.Fatalf("WriteText returned error: %v", err)
	}
	if strings.TrimSpace(out.String()) != "OK: no findings" {
		t.Fatalf("output = %q", out.String())
	}
}

func TestWriteSARIF(t *testing.T) {
	var out bytes.Buffer
	input := rules.Report{
		OK: false,
		Findings: []rules.Finding{{
			RuleID:   "repo.agents-required",
			Severity: rules.SeverityError,
			Path:     "AGENTS.md",
			Message:  "AGENTS.md is required",
		}},
	}

	if err := WriteSARIF(&out, input); err != nil {
		t.Fatalf("WriteSARIF returned error: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(out.Bytes(), &decoded); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if decoded["version"] != "2.1.0" {
		t.Fatalf("version = %#v, want 2.1.0", decoded["version"])
	}
	if !strings.Contains(out.String(), `"ruleId": "repo.agents-required"`) {
		t.Fatalf("SARIF output missing rule ID: %s", out.String())
	}
}
