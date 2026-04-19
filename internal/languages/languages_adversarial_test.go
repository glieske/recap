package languages

import (
	"strings"
	"sync"
	"testing"
)

func TestAdversarialValidCodeVeryLongString(t *testing.T) {
	veryLong := strings.Repeat("a", 1000)

	if got, want := ValidCode(veryLong), false; got != want {
		t.Fatalf("ValidCode(veryLong[1000 chars]) = %v, want %v", got, want)
	}
}

func TestAdversarialValidCodeSpecialCharacters(t *testing.T) {
	inputs := []string{
		"\n",
		"\t",
		"\x00",
		"en\n",
		"pl\t",
		"\u200b",       // zero-width space
		"\u202e",       // right-to-left override
		"👾",            // emoji
		"e\u0301",      // combining character sequence
		"en\x00suffix", // null-byte injection style
	}

	for _, input := range inputs {
		if got, want := ValidCode(input), false; got != want {
			t.Fatalf("ValidCode(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestAdversarialValidCodeSQLInjectionAttempt(t *testing.T) {
	injection := "'; DROP TABLE--"

	if got, want := ValidCode(injection), false; got != want {
		t.Fatalf("ValidCode(%q) = %v, want %v", injection, got, want)
	}
}

func TestAdversarialDisplayNameEmptyStringReturnsEnglish(t *testing.T) {
	if got, want := DisplayName(""), "English"; got != want {
		t.Fatalf("DisplayName(empty) = %q, want %q", got, want)
	}
}

func TestAdversarialDisplayNameWhitespaceOnlyReturnsEnglish(t *testing.T) {
	inputs := []string{" ", "\t", "\n", "\r\n", "\t \n"}

	for _, input := range inputs {
		if got, want := DisplayName(input), "English"; got != want {
			t.Fatalf("DisplayName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestAdversarialConcurrentReadAccess(t *testing.T) {
	t.Parallel()

	testInputs := []struct {
		code            string
		wantValid       bool
		wantDisplayName string
	}{
		{code: "en", wantValid: true, wantDisplayName: "English"},
		{code: "pl", wantValid: true, wantDisplayName: "Polish"},
		{code: "xx", wantValid: false, wantDisplayName: "English"},
		{code: "", wantValid: false, wantDisplayName: "English"},
		{code: "\x00", wantValid: false, wantDisplayName: "English"},
		{code: "'; DROP TABLE--", wantValid: false, wantDisplayName: "English"},
	}

	const goroutines = 64
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()

			for i := 0; i < iterations; i++ {
				for _, tc := range testInputs {
					if got := ValidCode(tc.code); got != tc.wantValid {
						t.Errorf("ValidCode(%q) = %v, want %v", tc.code, got, tc.wantValid)
					}

					if got := DisplayName(tc.code); got != tc.wantDisplayName {
						t.Errorf("DisplayName(%q) = %q, want %q", tc.code, got, tc.wantDisplayName)
					}
				}
			}
		}()
	}

	wg.Wait()
}
