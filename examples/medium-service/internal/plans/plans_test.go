package plans

import "testing"

func TestEffectiveTier(t *testing.T) {
	if got := EffectiveTier("pro", true); got != "trial-pro" {
		t.Fatalf("EffectiveTier(pro, true) = %q", got)
	}
	if got := EffectiveTier("pro", false); got != "pro" {
		t.Fatalf("EffectiveTier(pro, false) = %q", got)
	}
}

func TestCanArchive(t *testing.T) {
	if !CanArchive(3, 3) {
		t.Fatal("equal limit should be archivable")
	}
	if CanArchive(4, 3) {
		t.Fatal("over limit should not be archivable")
	}
}
