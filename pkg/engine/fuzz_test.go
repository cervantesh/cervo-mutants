package engine

import (
	"strings"
	"testing"
	"testing/quick"
)

func FuzzNormalizeGoFlags(f *testing.F) {
	for _, seed := range []string{
		"",
		"-count=1 -p 4 ./...",
		"-count=1 -p=4 ./...",
		"-run TestThing -count=1",
		"-shuffle=on -cpu 1,2 -p 8 ./pkg",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		got := normalizeGoFlags(input)
		if strings.Count(got, "-p=1") != 1 {
			t.Fatalf("normalizeGoFlags should emit exactly one -p=1, got %q", got)
		}
		fields := strings.Fields(got)
		for i, field := range fields {
			if field == "-p" {
				t.Fatalf("normalizeGoFlags should remove split -p flags, got %q", got)
			}
			if strings.HasPrefix(field, "-p=") && field != "-p=1" {
				t.Fatalf("normalizeGoFlags should normalize inline -p flags, got %q", got)
			}
			if field == "-p=1" && i != len(fields)-1 && strings.HasPrefix(fields[i+1], "=") {
				t.Fatalf("normalizeGoFlags emitted malformed field sequence: %q", got)
			}
		}
	})
}

func TestNormalizeGoFlagsIdempotentQuick(t *testing.T) {
	if err := quick.Check(func(input string) bool {
		got := normalizeGoFlags(input)
		return got == normalizeGoFlags(got) && strings.Count(got, "-p=1") == 1 && !strings.Contains(got, " -p ")
	}, nil); err != nil {
		t.Fatalf("normalizeGoFlags property failed: %v", err)
	}
}
