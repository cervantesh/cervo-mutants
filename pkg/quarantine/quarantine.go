package quarantine

import (
	"errors"
	"strings"
	"time"
)

type Entry struct {
	MutantID  string    `json:"mutant_id"`
	Reason    string    `json:"reason"`
	Owner     string    `json:"owner"`
	Issue     string    `json:"issue"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Renewals  int       `json:"renewals"`
}

type Policy struct {
	RequireReason bool `json:"require_reason"`
	RequireOwner  bool `json:"require_owner"`
	RequireIssue  bool `json:"require_issue"`
	FailOnExpired bool `json:"fail_on_expired"`
	MaxRenewals   int  `json:"max_renewals"`
}

func Validate(entries []Entry, policy Policy, now time.Time) error {
	for _, entry := range entries {
		if strings.TrimSpace(entry.MutantID) == "" {
			return errors.New("quarantine entry requires mutant_id")
		}
		if policy.RequireReason && strings.TrimSpace(entry.Reason) == "" {
			return errors.New("quarantine entry requires reason")
		}
		if policy.RequireOwner && strings.TrimSpace(entry.Owner) == "" {
			return errors.New("quarantine entry requires owner")
		}
		if policy.RequireIssue && strings.TrimSpace(entry.Issue) == "" {
			return errors.New("quarantine entry requires issue")
		}
		if entry.ExpiresAt.IsZero() {
			return errors.New("quarantine entry requires expires_at")
		}
		if policy.FailOnExpired && !entry.ExpiresAt.After(now) {
			return errors.New("quarantine entry expired")
		}
		if policy.MaxRenewals > 0 && entry.Renewals > policy.MaxRenewals {
			return errors.New("quarantine entry exceeded max renewals")
		}
	}
	return nil
}
