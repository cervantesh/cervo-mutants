package mutator

import (
	"encoding/hex"
	"strings"
	"testing"
	"testing/quick"
)

func FuzzParseInlineIgnore(f *testing.F) {
	for _, seed := range []string{
		`// cervomut:ignore conditionals reason="covered by generated contract"`,
		`// cervomut:ignore reason="all operators explained"`,
		`// cervomut:ignore logical reason=no_quotes_needed`,
		`fmt.Println("no comment marker")`,
		`// cervomut:ignore`,
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, line string) {
		ignore, ok, err := parseInlineIgnore("fuzz.go", 0, line, false)
		if err != nil {
			t.Fatalf("parseInlineIgnore(requireReason=false) returned error for %q: %v", line, err)
		}
		if !strings.Contains(line, "cervomut:ignore") {
			if ok {
				t.Fatalf("parseInlineIgnore unexpectedly matched %q: %+v", line, ignore)
			}
			return
		}
		if ok {
			if ignore.line != 2 {
				t.Fatalf("inline ignore line = %d, want 2", ignore.line)
			}
			if strings.TrimSpace(ignore.operator) == "" {
				t.Fatalf("inline ignore operator should not be empty for %q", line)
			}
		}

		_, _, _ = parseInlineIgnore("fuzz.go", 0, line, true)
	})
}

func FuzzValidateInlineIgnores(f *testing.F) {
	for _, seed := range []string{
		"package fuzz\n// cervomut:ignore conditionals reason=\"documented\"\nfunc example() {}\n",
		"package fuzz\nfunc example() {}\n",
		"package fuzz\n// cervomut:ignore reason=all\nvar value = 1\n",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, src string) {
		_, _ = ValidateInlineIgnores("fuzz.go", []byte(src), false)
		_, _ = ValidateInlineIgnores("fuzz.go", []byte(src), true)
	})
}

func TestFingerprintDeterministicQuick(t *testing.T) {
	if err := quick.Check(func(parts []string) bool {
		got := fingerprint(parts...)
		if got != fingerprint(parts...) {
			return false
		}
		if len(got) != 64 {
			return false
		}
		_, err := hex.DecodeString(got)
		return err == nil
	}, nil); err != nil {
		t.Fatalf("fingerprint property failed: %v", err)
	}
}

func TestNormalizeExprIdempotentQuick(t *testing.T) {
	if err := quick.Check(func(input string) bool {
		want := strings.Join(strings.Fields(input), " ")
		got := normalizeExpr(input)
		return got == want && normalizeExpr(got) == got
	}, nil); err != nil {
		t.Fatalf("normalizeExpr property failed: %v", err)
	}
}

func TestNormalizeTagQuick(t *testing.T) {
	if err := quick.Check(func(input string) bool {
		got := normalizeTag(input)
		return got == normalizeTag(got) && got == strings.ToLower(strings.ReplaceAll(strings.TrimSpace(got), " ", "-"))
	}, nil); err != nil {
		t.Fatalf("normalizeTag property failed: %v", err)
	}
}
