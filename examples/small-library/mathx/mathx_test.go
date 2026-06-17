package mathx

import "testing"

func TestClampPort(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want int
	}{
		{name: "below minimum", in: 0, want: 1},
		{name: "inside range", in: 8080, want: 8080},
		{name: "above range", in: 70000, want: 65535},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClampPort(tc.in); got != tc.want {
				t.Fatalf("ClampPort(%d) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestRetryBudget(t *testing.T) {
	if got := RetryBudget(0, false); got != 0 {
		t.Fatalf("RetryBudget(0, false) = %d, want 0", got)
	}
	if got := RetryBudget(2, false); got != 2 {
		t.Fatalf("RetryBudget(2, false) = %d, want 2", got)
	}
	if got := RetryBudget(2, true); got != 3 {
		t.Fatalf("RetryBudget(2, true) = %d, want 3", got)
	}
}
