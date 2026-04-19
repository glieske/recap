package languages

import (
	"reflect"
	"testing"
)

func TestAllLanguagesHasExactly13Entries(t *testing.T) {
	if got, want := len(AllLanguages), 13; got != want {
		t.Fatalf("AllLanguages length = %d, want %d", got, want)
	}
}

func TestAllLanguagesCodesAreUnique(t *testing.T) {
	seen := make(map[string]struct{}, len(AllLanguages))

	for _, language := range AllLanguages {
		if _, exists := seen[language.Code]; exists {
			t.Fatalf("duplicate language code found: %q", language.Code)
		}
		seen[language.Code] = struct{}{}
	}

	if got, want := len(seen), len(AllLanguages); got != want {
		t.Fatalf("unique code count = %d, want %d", got, want)
	}
}

func TestDefaultEnabledCodesAreExactlyEnPlDeNo(t *testing.T) {
	expected := []string{"en", "pl", "de", "no"}

	if got, want := len(DefaultEnabledCodes), 4; got != want {
		t.Fatalf("DefaultEnabledCodes length = %d, want %d", got, want)
	}

	if !reflect.DeepEqual(DefaultEnabledCodes, expected) {
		t.Fatalf("DefaultEnabledCodes = %v, want %v", DefaultEnabledCodes, expected)
	}
}

func TestDefaultEnabledCodesAreAllValid(t *testing.T) {
	for _, code := range DefaultEnabledCodes {
		if got, want := ValidCode(code), true; got != want {
			t.Fatalf("ValidCode(%q) = %v, want %v", code, got, want)
		}
	}
}

func TestSelectionLimits(t *testing.T) {
	if got, want := MaxSelected, 5; got != want {
		t.Fatalf("MaxSelected = %d, want %d", got, want)
	}

	if got, want := MinSelected, 1; got != want {
		t.Fatalf("MinSelected = %d, want %d", got, want)
	}
}

func TestValidCodeReturnsTrueForAllKnownCodes(t *testing.T) {
	knownCodes := []string{"en", "pl", "de", "no", "zh", "hi", "es", "fr", "ar", "bn", "pt", "ru", "ja"}

	for _, code := range knownCodes {
		if got, want := ValidCode(code), true; got != want {
			t.Fatalf("ValidCode(%q) = %v, want %v", code, got, want)
		}
	}
}

func TestValidCodeReturnsFalseForUnknownCodes(t *testing.T) {
	unknownCodes := []string{"xx", "", "EN", "english"}

	for _, code := range unknownCodes {
		if got, want := ValidCode(code), false; got != want {
			t.Fatalf("ValidCode(%q) = %v, want %v", code, got, want)
		}
	}
}

func TestDisplayNameReturnsCorrectNameForKnownCodes(t *testing.T) {
	expectedByCode := map[string]string{
		"en": "English",
		"pl": "Polish",
		"de": "German",
		"no": "Norwegian",
		"zh": "Chinese",
		"hi": "Hindi",
		"es": "Spanish",
		"fr": "French",
		"ar": "Arabic",
		"bn": "Bengali",
		"pt": "Portuguese",
		"ru": "Russian",
		"ja": "Japanese",
	}

	for code, want := range expectedByCode {
		if got := DisplayName(code); got != want {
			t.Fatalf("DisplayName(%q) = %q, want %q", code, got, want)
		}
	}
}

func TestDisplayNameReturnsEnglishForUnknownCodes(t *testing.T) {
	unknownCodes := []string{"xx", "", "unknown"}

	for _, code := range unknownCodes {
		if got, want := DisplayName(code), "English"; got != want {
			t.Fatalf("DisplayName(%q) = %q, want %q", code, got, want)
		}
	}
}
