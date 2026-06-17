package users

import "testing"

func TestCanAccessBilling(t *testing.T) {
	if !CanAccessBilling(User{ID: "u1", Verified: true, Quota: 1}) {
		t.Fatal("verified user with quota should access billing")
	}
	if CanAccessBilling(User{ID: "u2", Verified: false, Quota: 1}) {
		t.Fatal("unverified user should not access billing")
	}
	if CanAccessBilling(User{ID: "u3", Verified: true, Quota: 0}) {
		t.Fatal("user without quota should not access billing")
	}
}

func TestRemainingQuota(t *testing.T) {
	if got := RemainingQuota(User{Quota: 3}, 1); got != 2 {
		t.Fatalf("RemainingQuota(...) = %d, want 2", got)
	}
	if got := RemainingQuota(User{Quota: 3}, 5); got != 0 {
		t.Fatalf("RemainingQuota(...) = %d, want 0", got)
	}
}
