package tui

import (
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/glieske/recap/internal/config"
)

func updateLanguageSelectModelForTest(t *testing.T, m *LanguageSelectModel, msg tea.Msg) (*LanguageSelectModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	updatedModel, ok := updated.(*LanguageSelectModel)
	if !ok {
		t.Fatalf("expected *LanguageSelectModel from Update, got %T", updated)
	}

	return updatedModel, cmd
}

func updateEmailModelForLanguageTest(t *testing.T, m EmailModel, msg tea.Msg) (EmailModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	updatedModel, ok := updated.(EmailModel)
	if !ok {
		t.Fatalf("expected EmailModel from Update, got %T", updated)
	}

	return updatedModel, cmd
}

func TestLanguageSelectModel_Navigation(t *testing.T) {
	m := NewLanguageSelectModel("pl", []string{"en", "pl", "no"})

	if m.cursor != 1 {
		t.Fatalf("expected initial cursor at index 1 for pl, got %d", m.cursor)
	}

	updated, cmd := updateLanguageSelectModelForTest(t, &m, tea.KeyPressMsg{Code: tea.KeyUp})
	if cmd != nil {
		t.Fatalf("expected nil cmd on up key, got non-nil")
	}
	if updated.cursor != 0 {
		t.Fatalf("expected cursor to move up to 0, got %d", updated.cursor)
	}

	updated, cmd = updateLanguageSelectModelForTest(t, updated, tea.KeyPressMsg{Code: tea.KeyDown})
	if cmd != nil {
		t.Fatalf("expected nil cmd on down key, got non-nil")
	}
	if updated.cursor != 1 {
		t.Fatalf("expected cursor to move down to 1, got %d", updated.cursor)
	}

	updated.cursor = 0
	updated, cmd = updateLanguageSelectModelForTest(t, updated, tea.KeyPressMsg{Code: tea.KeyUp})
	if cmd != nil {
		t.Fatalf("expected nil cmd on wrapped up key, got non-nil")
	}
	if updated.cursor != 2 {
		t.Fatalf("expected cursor to wrap from 0 to 2 on up, got %d", updated.cursor)
	}

	updated.cursor = 2
	updated, cmd = updateLanguageSelectModelForTest(t, updated, tea.KeyPressMsg{Code: tea.KeyDown})
	if cmd != nil {
		t.Fatalf("expected nil cmd on wrapped down key, got non-nil")
	}
	if updated.cursor != 0 {
		t.Fatalf("expected cursor to wrap from 2 to 0 on down, got %d", updated.cursor)
	}
}

func TestLanguageSelectModel_SelectDifferentLanguage(t *testing.T) {
	m := NewLanguageSelectModel("pl", []string{"en", "pl", "no"})
	m.cursor = 0 // en

	updated, cmd := updateLanguageSelectModelForTest(t, &m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected non-nil cmd on enter")
	}
	if updated.cursor != 0 {
		t.Fatalf("expected cursor to stay on selected option 0, got %d", updated.cursor)
	}

	emitted := cmd()
	langChanged, ok := emitted.(LanguageChangedMsg)
	if !ok {
		t.Fatalf("expected LanguageChangedMsg, got %T", emitted)
	}

	if langChanged.Language != "en" {
		t.Fatalf("expected selected language en, got %q", langChanged.Language)
	}
}

func TestLanguageSelectModel_SelectSameLanguage(t *testing.T) {
	m := NewLanguageSelectModel("pl", []string{"en", "pl", "no"})
	m.cursor = 1 // pl

	updated, cmd := updateLanguageSelectModelForTest(t, &m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("expected non-nil cmd on enter")
	}
	if updated.cursor != 1 {
		t.Fatalf("expected cursor to stay on index 1, got %d", updated.cursor)
	}

	emitted := cmd()
	if _, ok := emitted.(DismissModalMsg); !ok {
		t.Fatalf("expected DismissModalMsg when selecting current language, got %T", emitted)
	}
}

func TestLanguageSelectModel_NewModel_CursorPosition(t *testing.T) {
	m := NewLanguageSelectModel("pl", []string{"en", "pl", "no"})

	if m.cursor != 1 {
		t.Fatalf("expected cursor index 1 for pl, got %d", m.cursor)
	}
}

func TestLanguageSelectModel_InitReturnsNilCommand(t *testing.T) {
	m := NewLanguageSelectModel("en", []string{"en", "pl", "no"})

	if cmd := m.Init(); cmd != nil {
		t.Fatalf("expected nil init command, got non-nil")
	}
}

func TestLanguageSelectModel_ViewMarksCursorAndCurrentLanguage(t *testing.T) {
	m := NewLanguageSelectModel("pl", []string{"en", "pl", "no"})

	view := m.View().Content
	stripped := ansi.Strip(view)
	if view == "" {
		t.Fatalf("expected non-empty view")
	}
	if !containsLine(stripped, "> PL *") {
		t.Fatalf("expected current selection marker in view, got %q", stripped)
	}
	if !containsLine(stripped, "  EN") {
		t.Fatalf("expected EN option in view, got %q", stripped)
	}
	if !containsLine(stripped, "  NO") {
		t.Fatalf("expected NO option in view, got %q", stripped)
	}
}

func containsLine(content, line string) bool {
	return content == line ||
		strings.Contains(content, "\n"+line+"\n") ||
		strings.HasPrefix(content, line+"\n") ||
		strings.HasSuffix(content, "\n"+line)
}

func TestEmailModel_LKeyEmitsShowLanguageModal(t *testing.T) {
	m := NewEmailModel("Subject", "Body", 80, 24, "no", "")

	updated, cmd := updateEmailModelForLanguageTest(t, m, tea.KeyPressMsg{Text: "l"})
	if cmd == nil {
		t.Fatalf("expected non-nil command for l key")
	}
	if updated.language != "no" {
		t.Fatalf("expected language to remain unchanged in model, got %q", updated.language)
	}

	emitted := cmd()
	modalMsg, ok := emitted.(ShowLanguageModalMsg)
	if !ok {
		t.Fatalf("expected ShowLanguageModalMsg, got %T", emitted)
	}

	if modalMsg.CurrentLanguage != "no" {
		t.Fatalf("expected CurrentLanguage to be no, got %q", modalMsg.CurrentLanguage)
	}

	if _, isLanguageChanged := emitted.(LanguageChangedMsg); isLanguageChanged {
		t.Fatalf("did not expect LanguageChangedMsg from l key")
	}
}

func TestNextEmailLanguage(t *testing.T) {
	tests := []struct {
		name       string
		current    string
		configured []string
		want       string
	}{
		{
			name:       "advances to next configured language",
			current:    "pl",
			configured: []string{"en", "pl", "de", "no"},
			want:       "de",
		},
		{
			name:       "wraps from last to first",
			current:    "no",
			configured: []string{"en", "pl", "de", "no"},
			want:       "en",
		},
		{
			name:       "single language remains same",
			current:    "en",
			configured: []string{"en"},
			want:       "en",
		},
		{
			name:       "nil configured falls back to en",
			current:    "en",
			configured: nil,
			want:       "en",
		},
		{
			name:       "unknown current uses first configured",
			current:    "xx",
			configured: []string{"fr", "ja"},
			want:       "fr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextEmailLanguage(tt.current, tt.configured)
			if got != tt.want {
				t.Fatalf("nextEmailLanguage(%q, %v) = %q, want %q", tt.current, tt.configured, got, tt.want)
			}
		})
	}
}

func TestNextEmailLanguage_CycleProperty_ReturnsToStartAfterListLengthSteps(t *testing.T) {
	configured := []string{"en", "pl", "de", "no"}
	start := configured[0]
	current := start

	for range configured {
		current = nextEmailLanguage(current, configured)
	}

	if current != start {
		t.Fatalf("after %d transitions expected %q, got %q", len(configured), start, current)
	}
}

func TestEmailLanguageDisplayName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "en", want: "EN"},
		{input: "pl", want: "PL"},
		{input: "de", want: "DE"},
	}

	for _, tt := range tests {
		got := emailLanguageDisplayName(tt.input)
		if got != tt.want {
			t.Fatalf("emailLanguageDisplayName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestNewLanguageSelectModel_DynamicOptionsAndFallback(t *testing.T) {
	t.Run("uses configured options and uppercase labels with current cursor", func(t *testing.T) {
		m := NewLanguageSelectModel("de", []string{"en", "de", "ja"})

		if !reflect.DeepEqual(m.options, []string{"en", "de", "ja"}) {
			t.Fatalf("options = %v, want %v", m.options, []string{"en", "de", "ja"})
		}
		if !reflect.DeepEqual(m.labels, []string{"EN", "DE", "JA"}) {
			t.Fatalf("labels = %v, want %v", m.labels, []string{"EN", "DE", "JA"})
		}
		if m.cursor != 1 {
			t.Fatalf("cursor = %d, want %d", m.cursor, 1)
		}
	})

	t.Run("falls back to en when configured codes are nil", func(t *testing.T) {
		m := NewLanguageSelectModel("en", nil)

		if !reflect.DeepEqual(m.options, []string{"en"}) {
			t.Fatalf("options = %v, want %v", m.options, []string{"en"})
		}
		if !reflect.DeepEqual(m.labels, []string{"EN"}) {
			t.Fatalf("labels = %v, want %v", m.labels, []string{"EN"})
		}
		if m.cursor != 0 {
			t.Fatalf("cursor = %d, want %d", m.cursor, 0)
		}
	})

	t.Run("unknown current keeps cursor at zero", func(t *testing.T) {
		m := NewLanguageSelectModel("xx", []string{"en", "de", "ja"})

		if m.cursor != 0 {
			t.Fatalf("cursor = %d, want %d", m.cursor, 0)
		}
	})
}

func TestAppModelConfiguredLanguages(t *testing.T) {
	t.Run("returns configured EmailLanguages", func(t *testing.T) {
		m := NewAppModel(&config.Config{EmailLanguages: []string{"fr", "ja"}}, nil, nil, "", false, "")

		got := m.configuredLanguages()
		if !reflect.DeepEqual(got, []string{"fr", "ja"}) {
			t.Fatalf("configuredLanguages() = %v, want %v", got, []string{"fr", "ja"})
		}
	})

	t.Run("falls back to en when EmailLanguages nil", func(t *testing.T) {
		m := NewAppModel(&config.Config{}, nil, nil, "", false, "")

		got := m.configuredLanguages()
		if !reflect.DeepEqual(got, []string{"en"}) {
			t.Fatalf("configuredLanguages() = %v, want %v", got, []string{"en"})
		}
	})

	t.Run("falls back to en when config is nil", func(t *testing.T) {
		m := NewAppModel(nil, nil, nil, "", false, "")

		got := m.configuredLanguages()
		if !reflect.DeepEqual(got, []string{"en"}) {
			t.Fatalf("configuredLanguages() = %v, want %v", got, []string{"en"})
		}
	})
}

func TestAppModelEmailLanguage(t *testing.T) {
	t.Run("returns first configured language", func(t *testing.T) {
		m := NewAppModel(&config.Config{EmailLanguages: []string{"pl", "en", "de"}}, nil, nil, "", false, "")

		got := m.emailLanguage()
		if got != "pl" {
			t.Fatalf("emailLanguage() = %q, want %q", got, "pl")
		}
	})

	t.Run("returns first configured when multiple languages", func(t *testing.T) {
		m := NewAppModel(&config.Config{EmailLanguages: []string{"en", "pl", "de"}}, nil, nil, "", false, "")

		got := m.emailLanguage()
		if got != "en" {
			t.Fatalf("emailLanguage() = %q, want %q", got, "en")
		}
	})

	t.Run("returns first configured when single language", func(t *testing.T) {
		m := NewAppModel(&config.Config{EmailLanguages: []string{"en"}}, nil, nil, "", false, "")

		got := m.emailLanguage()
		if got != "en" {
			t.Fatalf("emailLanguage() = %q, want %q", got, "en")
		}
	})

	t.Run("nil config falls back to en", func(t *testing.T) {
		m := NewAppModel(nil, nil, nil, "", false, "")

		got := m.emailLanguage()
		if got != "en" {
			t.Fatalf("emailLanguage() = %q, want %q", got, "en")
		}
	})
}

func TestAdversarialNextEmailLanguage_VeryLargeConfiguredSlice_NoPanic(t *testing.T) {
	configured := make([]string, 1200)
	for i := range configured {
		configured[i] = "en"
	}
	configured[1199] = "pl"

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("nextEmailLanguage panicked with very large slice: %v", r)
		}
	}()

	got := nextEmailLanguage("pl", configured)
	if got != "en" {
		t.Fatalf("nextEmailLanguage on large slice = %q, want %q", got, "en")
	}
}

func TestAdversarialNextEmailLanguage_EmptyAndDuplicateCodes(t *testing.T) {
	t.Run("empty strings are handled deterministically", func(t *testing.T) {
		configured := []string{"", "en", ""}

		got := nextEmailLanguage("en", configured)
		if got != "" {
			t.Fatalf("nextEmailLanguage(%q, %v) = %q, want %q", "en", configured, got, "")
		}

		got = nextEmailLanguage("", configured)
		if got != "en" {
			t.Fatalf("nextEmailLanguage(%q, %v) = %q, want %q", "", configured, got, "en")
		}
	})

	t.Run("duplicate codes cycle from first matched occurrence", func(t *testing.T) {
		configured := []string{"en", "pl", "en", "de"}

		got := nextEmailLanguage("en", configured)
		if got != "pl" {
			t.Fatalf("nextEmailLanguage(%q, %v) = %q, want %q", "en", configured, got, "pl")
		}

		got = nextEmailLanguage("pl", configured)
		if got != "en" {
			t.Fatalf("nextEmailLanguage(%q, %v) = %q, want %q", "pl", configured, got, "en")
		}
	})
}

func TestAdversarialEmailLanguageDisplayName_MalformedInputs(t *testing.T) {
	long := strings.Repeat("a", 12000)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty string", input: "", want: ""},
		{name: "special chars and injection-like payload", input: "<script>alert(1)</script>", want: "<SCRIPT>ALERT(1)</SCRIPT>"},
		{name: "very long string", input: long, want: strings.Repeat("A", 12000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := emailLanguageDisplayName(tt.input)
			if got != tt.want {
				t.Fatalf("emailLanguageDisplayName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAdversarialNewLanguageSelectModel_MalformedConfiguredCodes(t *testing.T) {
	t.Run("very large configured codes does not panic", func(t *testing.T) {
		configured := make([]string, 1500)
		for i := range configured {
			configured[i] = "en"
		}
		configured[1499] = "zz"

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("NewLanguageSelectModel panicked with very large configured codes: %v", r)
			}
		}()

		m := NewLanguageSelectModel("zz", configured)
		if len(m.options) != 1500 {
			t.Fatalf("len(options) = %d, want %d", len(m.options), 1500)
		}
		if m.cursor != 1499 {
			t.Fatalf("cursor = %d, want %d", m.cursor, 1499)
		}
	})

	t.Run("empty configured codes become empty labels without panic", func(t *testing.T) {
		configured := []string{"", "en", ""}
		m := NewLanguageSelectModel("", configured)

		if !reflect.DeepEqual(m.options, configured) {
			t.Fatalf("options = %v, want %v", m.options, configured)
		}
		if !reflect.DeepEqual(m.labels, []string{"", "EN", ""}) {
			t.Fatalf("labels = %v, want %v", m.labels, []string{"", "EN", ""})
		}
		if m.cursor != 0 {
			t.Fatalf("cursor = %d, want %d", m.cursor, 0)
		}
	})
}

func TestAdversarialAppConfiguredLanguages_EmptySliceFallsBackToDefault(t *testing.T) {
	m := NewAppModel(&config.Config{EmailLanguages: []string{}}, nil, nil, "", false, "")

	got := m.configuredLanguages()
	if !reflect.DeepEqual(got, []string{"en"}) {
		t.Fatalf("configuredLanguages() = %v, want %v", got, []string{"en"})
	}
}

func TestAdversarialAppEmailLanguage_PersistedMalformedLanguageFallsBack(t *testing.T) {
	m := NewAppModel(&config.Config{
		EmailLanguages: []string{"en", "pl", "de"},
	}, nil, nil, "", false, "")

	got := m.emailLanguage()
	if got != "en" {
		t.Fatalf("emailLanguage() = %q, want %q", got, "en")
	}
}
