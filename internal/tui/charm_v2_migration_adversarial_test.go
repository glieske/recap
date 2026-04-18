package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/config"
)

func requireNoPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("%s panicked: %v", name, r)
		}
	}()
	fn()
}

func TestAdversarial_KeyPressMsg_Malformed_NoPanics(t *testing.T) {
	malformed := tea.KeyPressMsg(tea.Key{Text: "\x00\u200b😀", Code: 0})

	app := NewAppModel(&config.Config{}, nil, nil, "")
	requireNoPanic(t, "AppModel.Update malformed key", func() {
		updated, _ := app.Update(malformed)
		typed, ok := updated.(AppModel)
		if !ok {
			t.Fatalf("AppModel.Update type mismatch: got %T", updated)
		}
		if typed.screen != ScreenMeetingList {
			t.Fatalf("AppModel.Update screen mismatch: got %v want %v", typed.screen, ScreenMeetingList)
		}
	})

	editor := NewEditorModel(nil, nil, 80, 24, "", "")
	requireNoPanic(t, "EditorModel.Update malformed key", func() {
		updated, _ := editor.Update(malformed)
		typed, ok := updated.(EditorModel)
		if !ok {
			t.Fatalf("EditorModel.Update type mismatch: got %T", updated)
		}
		if typed.height != 24 {
			t.Fatalf("EditorModel.Update unexpected height mutation: got %d want %d", typed.height, 24)
		}
	})

	email := NewEmailModel("", "", 80, 24, "pl")
	requireNoPanic(t, "EmailModel.Update malformed key", func() {
		updated, _ := email.Update(malformed)
		typed, ok := updated.(EmailModel)
		if !ok {
			t.Fatalf("EmailModel.Update type mismatch: got %T", updated)
		}
		if typed.language != "pl" {
			t.Fatalf("EmailModel.Update language mismatch: got %q want %q", typed.language, "pl")
		}
	})

	help := NewHelpModel()
	requireNoPanic(t, "HelpModel.Update malformed key", func() {
		updated, _ := help.Update(malformed)
		typed, ok := updated.(HelpModel)
		if !ok {
			t.Fatalf("HelpModel.Update type mismatch: got %T", updated)
		}
		if typed.width != 0 {
			t.Fatalf("HelpModel.Update width mismatch: got %d want 0", typed.width)
		}
	})

	listModel := NewListModel(nil, 80, 24)
	requireNoPanic(t, "ListModel.Update malformed key", func() {
		updated, _ := listModel.Update(malformed)
		_, ok := updated.(ListModel)
		if !ok {
			t.Fatalf("ListModel.Update type mismatch: got %T", updated)
		}
	})

	newMeeting := NewNewMeetingModel(nil, 80, 24)
	requireNoPanic(t, "NewMeetingModel.Update malformed key", func() {
		updated, _ := newMeeting.Update(malformed)
		typed, ok := updated.(NewMeetingModel)
		if !ok {
			t.Fatalf("NewMeetingModel.Update type mismatch: got %T", updated)
		}
		if typed.cancelled {
			t.Fatalf("NewMeetingModel.Update cancelled mismatch: got %v want %v", typed.cancelled, false)
		}
	})

	preview := NewPreviewModel("", "title", 80, 24)
	requireNoPanic(t, "PreviewModel.Update malformed key", func() {
		updated, _ := preview.Update(malformed)
		typed, ok := updated.(PreviewModel)
		if !ok {
			t.Fatalf("PreviewModel.Update type mismatch: got %T", updated)
		}
		if typed.title != "title" {
			t.Fatalf("PreviewModel.Update title mismatch: got %q want %q", typed.title, "title")
		}
	})

	provider := NewProviderModel("", 80, 24)
	requireNoPanic(t, "ProviderModel.Update malformed key", func() {
		updated, _ := provider.Update(malformed)
		typed, ok := updated.(ProviderModel)
		if !ok {
			t.Fatalf("ProviderModel.Update type mismatch: got %T", updated)
		}
		if typed.height != 24 {
			t.Fatalf("ProviderModel.Update height mismatch: got %d want %d", typed.height, 24)
		}
	})

	summary := NewSummaryModel("", "", "", nil, 40, 20)
	requireNoPanic(t, "SummaryModel.Update malformed key", func() {
		updated, _ := summary.Update(malformed)
		typed, ok := updated.(SummaryModel)
		if !ok {
			t.Fatalf("SummaryModel.Update type mismatch: got %T", updated)
		}
		if typed.IsOverwritePending() {
			t.Fatalf("SummaryModel.Update overwrite pending mismatch: got %v want %v", typed.IsOverwritePending(), false)
		}
	})
}

func TestAdversarial_WindowSize_ZeroAndNegative_NoPanics(t *testing.T) {
	windowMsg := tea.WindowSizeMsg{Width: 0, Height: 0}
	negativeMsg := tea.WindowSizeMsg{Width: -1, Height: -1}

	app := NewAppModel(&config.Config{}, nil, nil, "")
	requireNoPanic(t, "AppModel.Update zero window", func() {
		updated, _ := app.Update(windowMsg)
		typed, ok := updated.(AppModel)
		if !ok {
			t.Fatalf("AppModel.Update type mismatch: got %T", updated)
		}
		if typed.width != 0 || typed.height != 0 {
			t.Fatalf("AppModel.Update size mismatch: got (%d,%d) want (0,0)", typed.width, typed.height)
		}
	})

	listModel := NewListModel(nil, 80, 24)
	requireNoPanic(t, "ListModel.Update negative window", func() {
		updated, _ := listModel.Update(negativeMsg)
		typed, ok := updated.(ListModel)
		if !ok {
			t.Fatalf("ListModel.Update type mismatch: got %T", updated)
		}
		if typed.width != -1 || typed.height != -1 {
			t.Fatalf("ListModel.Update size mismatch: got (%d,%d) want (-1,-1)", typed.width, typed.height)
		}
	})

	email := NewEmailModel("", "", 80, 24, "pl")
	requireNoPanic(t, "EmailModel.Update zero window", func() {
		updated, _ := email.Update(windowMsg)
		typed, ok := updated.(EmailModel)
		if !ok {
			t.Fatalf("EmailModel.Update type mismatch: got %T", updated)
		}
		if typed.width != 0 || typed.height != 0 {
			t.Fatalf("EmailModel.Update size mismatch: got (%d,%d) want (0,0)", typed.width, typed.height)
		}
	})

	editor := NewEditorModel(nil, nil, 80, 24, "", "")
	requireNoPanic(t, "EditorModel.Update zero window", func() {
		updated, _ := editor.Update(windowMsg)
		typed, ok := updated.(EditorModel)
		if !ok {
			t.Fatalf("EditorModel.Update type mismatch: got %T", updated)
		}
		if typed.width != 0 || typed.height != 0 {
			t.Fatalf("EditorModel.Update size mismatch: got (%d,%d) want (0,0)", typed.width, typed.height)
		}
	})

	summary := NewSummaryModel("", "", "", nil, 10, 10)
	requireNoPanic(t, "SummaryModel.Update negative window", func() {
		updated, _ := summary.Update(negativeMsg)
		typed, ok := updated.(SummaryModel)
		if !ok {
			t.Fatalf("SummaryModel.Update type mismatch: got %T", updated)
		}
		if typed.width != -1 || typed.height != -1 {
			t.Fatalf("SummaryModel.Update size mismatch: got (%d,%d) want (-1,-1)", typed.width, typed.height)
		}
	})
}

func TestAdversarial_ViewComposition_NilLikeChildren_NoPanics(t *testing.T) {
	// Test View() on properly constructed models at zero-size windows
	cfg := &config.Config{}

	app := NewAppModel(cfg, nil, nil, "")
	for _, screen := range []Screen{
		ScreenMeetingList,
		ScreenNewMeeting,
		ScreenEditor,
		ScreenEmail,
		ScreenHelp,
		ScreenProviderSelector,
		Screen(999),
	} {
		testApp := app
		testApp.screen = screen
		// For screens that need initialized models
		switch screen {
		case ScreenNewMeeting:
			testApp.hasNewMeetingModel = true
			testApp.newMeetingModel = NewNewMeetingModel(nil, 0, 0)
		case ScreenEditor:
			testApp.hasEditorModel = true
			testApp.editorModel = NewEditorModel(nil, nil, 0, 0, "", "")
		case ScreenEmail:
			testApp.hasEmailModel = true
			testApp.emailModel = NewEmailModel("", "", 0, 0, "en")
		case ScreenProviderSelector:
			testApp.hasProviderModel = true
			testApp.providerModel = NewProviderModel("", 0, 0)
		}
		requireNoPanic(t, fmt.Sprintf("AppModel.View screen=%d", screen), func() {
			view := testApp.View()
			if !view.AltScreen {
				t.Errorf("AppModel.View AltScreen mismatch: got %v want true", view.AltScreen)
			}
		})
	}

	// Test EditorModel split mode with constructed models at zero size
	splitEditor := NewEditorModel(nil, nil, 1, 1, "", "")
	splitEditor.splitMode = true
	splitEditor.hasSummaryModel = true
	splitEditor.summaryModel = NewSummaryModel("", "", "", nil, 1, 1)
	requireNoPanic(t, "EditorModel.View split with small summary", func() {
		view := splitEditor.View()
		if view.Content == "" {
			t.Errorf("EditorModel.View content mismatch: got empty content")
		}
	})
}

func TestAdversarial_KeyPressMsg_OversizedInjectionPayload_NoPanics(t *testing.T) {
	payloadUnit := "<script>alert(1)</script>${x}../\x00😀"
	payload := strings.Repeat(payloadUnit, 600)
	if len(payload) <= 10_000 {
		t.Fatalf("payload length mismatch: got %d want > 10000", len(payload))
	}

	oversized := tea.KeyPressMsg(tea.Key{Text: payload, Code: 0})

	editor := NewEditorModel(nil, nil, 80, 24, "", "")
	requireNoPanic(t, "EditorModel.Update oversized payload", func() {
		updated, _ := editor.Update(oversized)
		typed, ok := updated.(EditorModel)
		if !ok {
			t.Fatalf("EditorModel.Update type mismatch: got %T", updated)
		}
		if !typed.dirty {
			t.Fatalf("EditorModel.Update dirty mismatch: got %v want %v", typed.dirty, true)
		}
	})

	summary := NewSummaryModel("", "", "", nil, 80, 24)
	requireNoPanic(t, "SummaryModel.Update oversized payload", func() {
		updated, _ := summary.Update(oversized)
		typed, ok := updated.(SummaryModel)
		if !ok {
			t.Fatalf("SummaryModel.Update type mismatch: got %T", updated)
		}
		if typed.IsOverwritePending() {
			t.Fatalf("SummaryModel.Update overwrite pending mismatch: got %v want %v", typed.IsOverwritePending(), false)
		}
	})

	app := NewAppModel(&config.Config{}, nil, nil, "")
	requireNoPanic(t, "AppModel.Update oversized payload", func() {
		updated, _ := app.Update(oversized)
		typed, ok := updated.(AppModel)
		if !ok {
			t.Fatalf("AppModel.Update type mismatch: got %T", updated)
		}
		if typed.screen != ScreenMeetingList {
			t.Fatalf("AppModel.Update screen mismatch: got %v want %v", typed.screen, ScreenMeetingList)
		}
	})
}

func TestAdversarial_BoundaryInvariants_HeightHelpers(t *testing.T) {
	for i := -100; i <= 0; i++ {
		if got := maxEditorHeight(i); got != 1 {
			t.Fatalf("maxEditorHeight(%d) mismatch: got %d want %d", i, got, 1)
		}
		if got := emailViewportHeight(i); got != 1 {
			t.Fatalf("emailViewportHeight(%d) mismatch: got %d want %d", i, got, 1)
		}
		if got := previewViewportHeight(i); got != 1 {
			t.Fatalf("previewViewportHeight(%d) mismatch: got %d want %d", i, got, 1)
		}
		if got := summaryTextareaHeight(i); got != 1 {
			t.Fatalf("summaryTextareaHeight(%d) mismatch: got %d want %d", i, got, 1)
		}
	}

	prevEditor := maxEditorHeight(-1)
	prevEmail := emailViewportHeight(-1)
	prevPreview := previewViewportHeight(-1)
	prevSummary := summaryTextareaHeight(-1)

	for i := 0; i <= 200; i++ {
		currEditor := maxEditorHeight(i)
		currEmail := emailViewportHeight(i)
		currPreview := previewViewportHeight(i)
		currSummary := summaryTextareaHeight(i)

		if currEditor < prevEditor {
			t.Fatalf("maxEditorHeight monotonicity violated at %d: got %d prev %d", i, currEditor, prevEditor)
		}
		if currEmail < prevEmail {
			t.Fatalf("emailViewportHeight monotonicity violated at %d: got %d prev %d", i, currEmail, prevEmail)
		}
		if currPreview < prevPreview {
			t.Fatalf("previewViewportHeight monotonicity violated at %d: got %d prev %d", i, currPreview, prevPreview)
		}
		if currSummary < prevSummary {
			t.Fatalf("summaryTextareaHeight monotonicity violated at %d: got %d prev %d", i, currSummary, prevSummary)
		}

		prevEditor = currEditor
		prevEmail = currEmail
		prevPreview = currPreview
		prevSummary = currSummary
	}
}
