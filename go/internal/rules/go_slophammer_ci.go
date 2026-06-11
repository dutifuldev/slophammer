package rules

import (
	"context"
	"strings"

	"github.com/dutifuldev/slophammer/go/internal/repo"
)

// slophammerInvocationWindow bounds how far after a checker binary name the
// check subcommand may appear and still count as one invocation.
const slophammerInvocationWindow = 160

// slophammerCIRule enforces that config without enforcement is decoration:
// when slophammer.yml is present, binding CI evidence must invoke a
// Slophammer checker.
type slophammerCIRule struct {
	definition Definition
}

func newSlophammerCIRule(definition Definition) Rule {
	return slophammerCIRule{definition: definition}
}

func (r slophammerCIRule) Metadata() Metadata {
	return r.definition.Metadata()
}

func (r slophammerCIRule) Check(_ context.Context, snapshot repo.Snapshot) []Finding {
	if !hasModuleLocalSlophammerConfig(snapshot, "") {
		return nil
	}
	if slophammerInvocation(commandEvidenceText(snapshot)) {
		return nil
	}
	return []Finding{finding(r.definition)}
}

// commandEvidenceText joins the binding command evidence: workflow run
// scripts plus the scripts and runner files they reach.
func commandEvidenceText(snapshot repo.Snapshot) string {
	var evidence strings.Builder
	for _, file := range commandFiles(snapshot) {
		evidence.WriteString(file.Content)
		evidence.WriteString("\n")
	}
	return strings.ToLower(evidence.String())
}

func slophammerInvocation(evidence string) bool {
	if strings.Contains(evidence, "uses: dutifuldev/slophammer@") {
		return true
	}
	for _, binary := range []string{"slophammer-go", "slophammer-ts", "slophammer-rs", "slophammer-py"} {
		if invocationWithCheck(evidence, binary) {
			return true
		}
	}
	return false
}

func invocationWithCheck(evidence string, binary string) bool {
	for index := strings.Index(evidence, binary); index >= 0; {
		end := min(index+slophammerInvocationWindow, len(evidence))
		if strings.Contains(evidence[index:end], " check") {
			return true
		}
		next := strings.Index(evidence[index+1:], binary)
		if next < 0 {
			return false
		}
		index += 1 + next
	}
	return false
}
