package checkout

import "testing"

func TestReadyToSubmit(t *testing.T) {
	if !ReadyToSubmit(3, 3, 1000, 20) {
		t.Fatal("billable order with non-zero total should be ready")
	}
	if ReadyToSubmit(2, 3, 1000, 20) {
		t.Fatal("order below threshold should not be ready")
	}
	if ReadyToSubmit(3, 3, 1000, 100) {
		t.Fatal("fully discounted order should not be ready")
	}
}
