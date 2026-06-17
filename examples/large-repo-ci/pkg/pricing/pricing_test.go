package pricing

import "testing"

func TestDiscountedCents(t *testing.T) {
	if got := DiscountedCents(1000, 0); got != 1000 {
		t.Fatalf("DiscountedCents(1000, 0) = %d, want 1000", got)
	}
	if got := DiscountedCents(1000, 20); got != 800 {
		t.Fatalf("DiscountedCents(1000, 20) = %d, want 800", got)
	}
	if got := DiscountedCents(1000, 100); got != 0 {
		t.Fatalf("DiscountedCents(1000, 100) = %d, want 0", got)
	}
}

func TestBillable(t *testing.T) {
	if !Billable(3, 3) {
		t.Fatal("equal threshold should be billable")
	}
	if Billable(2, 3) {
		t.Fatal("below threshold should not be billable")
	}
}
