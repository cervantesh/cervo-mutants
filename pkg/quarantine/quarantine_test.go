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
		Issue:     "cervantesh/cervo-mutants#1",
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
		Issue:     "cervantesh/cervo-mutants#1",
		ExpiresAt: now.Add(-time.Second),
	}}, policy, now); err == nil {
		t.Fatal("Validate accepted expired quarantine")
	}
}

func TestValidateRejectsEachRequiredFieldAndRenewalLimit(t *testing.T) {
	now := time.Date(2026, 5, 26, 12, 0, 0, 0, time.UTC)
	policy := Policy{RequireReason: true, RequireOwner: true, RequireIssue: true, FailOnExpired: true, MaxRenewals: 1}
	valid := Entry{
		MutantID:  "m1",
		Reason:    "temporary",
		Owner:     "qa",
		Issue:     "cervantesh/cervo-mutants#31",
		ExpiresAt: now.Add(time.Hour),
	}
	cases := []struct {
		name  string
		entry Entry
	}{
		{name: "missing mutant", entry: withEntry(valid, func(entry *Entry) { entry.MutantID = " " })},
		{name: "missing reason", entry: withEntry(valid, func(entry *Entry) { entry.Reason = " " })},
		{name: "missing owner", entry: withEntry(valid, func(entry *Entry) { entry.Owner = " " })},
		{name: "missing issue", entry: withEntry(valid, func(entry *Entry) { entry.Issue = " " })},
		{name: "missing expiry", entry: withEntry(valid, func(entry *Entry) { entry.ExpiresAt = time.Time{} })},
		{name: "too many renewals", entry: withEntry(valid, func(entry *Entry) { entry.Renewals = 2 })},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := Validate([]Entry{tc.entry}, policy, now); err == nil {
				t.Fatal("Validate accepted invalid quarantine entry")
			}
		})
	}
}

func withEntry(entry Entry, mutate func(*Entry)) Entry {
	mutate(&entry)
	return entry
}
