package quarantine

import (
	"testing"
	"time"
)

func TestValidateRequiresAuditedTemporaryEntries(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	policy := Policy{RequireReason: true, RequireOwner: true, RequireIssue: true, FailOnExpired: true}

	err := Validate([]Entry{{
		MutantID:  "pkg/foo.go:1:conditionals:eq-to-ne",
		Reason:    "Equivalent defensive branch",
		Owner:     "platform",
		Issue:     "CervoSoft/cervo-mutant#1",
		CreatedAt: now.Add(-24 * time.Hour),
		ExpiresAt: now.Add(24 * time.Hour),
	}}, policy, now)
	if err != nil {
		t.Fatalf("Validate rejected valid entry: %v", err)
	}

	if err := Validate([]Entry{{MutantID: "m1", ExpiresAt: now.Add(24 * time.Hour)}}, policy, now); err == nil {
		t.Fatal("Validate accepted entry without reason, owner, and issue")
	}
	if err := Validate([]Entry{{
		MutantID:  "m1",
		Reason:    "temporary",
		Owner:     "platform",
		Issue:     "CervoSoft/cervo-mutant#1",
		ExpiresAt: now.Add(-time.Second),
	}}, policy, now); err == nil {
		t.Fatal("Validate accepted expired quarantine")
	}
}
