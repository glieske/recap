package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/glieske/recap/internal/config"
)

func TestHelpOverlay(t *testing.T) {
	t.Run("app init delegates to list init", func(t *testing.T) {
		m := NewAppModel(&config.Config{}, nil, nil, "", false)

		got := m.Init()
		want := m.listModel.Init()
		if (got == nil) != (want == nil) {
			t.Fatalf("expected init nil-ness %v, got %v", want == nil, got == nil)
		}
	})

	t.Run("help toggle on creates modal on meeting list", func(t *testing.T) {
		m := NewAppModel(&config.Config{}, nil, nil, "", false)
		m.screen = ScreenMeetingList

		updated, cmd := appUpdate(t, m, tea.KeyPressMsg{Text: "?"})

		if cmd != nil {
			t.Fatalf("expected nil cmd, got non-nil")
		}
		if updated.showHelp != true {
			t.Fatalf("expected showHelp true, got %v", updated.showHelp)
		}
		if updated.helpModal.Title != "Help" {
			t.Fatalf("expected help modal title %q, got %q", "Help", updated.helpModal.Title)
		}
		if updated.helpModal.Content == nil {
			t.Fatalf("expected help modal content to be initialized")
		}
	})

	t.Run("help toggle off via question mark", func(t *testing.T) {
		m := NewAppModel(&config.Config{}, nil, nil, "", false)
		opened, _ := appUpdate(t, m, tea.KeyPressMsg{Text: "?"})
		if !opened.showHelp {
			t.Fatalf("expected showHelp true after opening, got %v", opened.showHelp)
		}

		closed, cmd := appUpdate(t, opened, tea.KeyPressMsg{Text: "?"})

		if cmd != nil {
			t.Fatalf("expected nil cmd when closing help with ?, got non-nil")
		}
		if closed.showHelp != false {
			t.Fatalf("expected showHelp false, got %v", closed.showHelp)
		}
	})

	t.Run("help dismiss via esc emits dismiss message then closes next cycle", func(t *testing.T) {
		m := NewAppModel(&config.Config{}, nil, nil, "", false)
		opened, _ := appUpdate(t, m, tea.KeyPressMsg{Text: "?"})

		afterEsc, cmd := appUpdate(t, opened, tea.KeyPressMsg{Code: tea.KeyEscape})
		if cmd == nil {
			t.Fatalf("expected non-nil cmd from modal esc")
		}
		if afterEsc.showHelp != true {
			t.Fatalf("expected showHelp to remain true before DismissModalMsg, got %v", afterEsc.showHelp)
		}

		emitted := cmd()
		if _, ok := emitted.(DismissModalMsg); !ok {
			t.Fatalf("expected DismissModalMsg, got %T", emitted)
		}

		dismissed, dismissCmd := appUpdate(t, afterEsc, emitted)
		if dismissCmd != nil {
			t.Fatalf("expected nil cmd when processing DismissModalMsg, got non-nil")
		}
		if dismissed.showHelp != false {
			t.Fatalf("expected showHelp false after DismissModalMsg, got %v", dismissed.showHelp)
		}
	})

	t.Run("ctrl+c always quits even when help is open", func(t *testing.T) {
		m := NewAppModel(&config.Config{}, nil, nil, "", false)
		opened, _ := appUpdate(t, m, tea.KeyPressMsg{Text: "?"})

		_, cmd := appUpdate(t, opened, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		if cmd == nil {
			t.Fatalf("expected quit command, got nil")
		}

		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Fatalf("expected tea.QuitMsg, got %T", msg)
		}
	})

	t.Run("input trap blocks underlying screen shortcuts while help is open", func(t *testing.T) {
		m := NewAppModel(&config.Config{}, nil, nil, "", false)
		m.screen = ScreenMeetingList
		opened, _ := appUpdate(t, m, tea.KeyPressMsg{Text: "?"})

		afterN, cmdN := appUpdate(t, opened, tea.KeyPressMsg{Code: -1, Text: "n"})
		if cmdN != nil {
			t.Fatalf("expected nil cmd for trapped 'n', got non-nil")
		}
		if afterN.screen != ScreenMeetingList {
			t.Fatalf("expected screen to remain %v, got %v", ScreenMeetingList, afterN.screen)
		}

		afterD, cmdD := appUpdate(t, afterN, tea.KeyPressMsg{Code: -1, Text: "d"})
		if cmdD != nil {
			t.Fatalf("expected nil cmd for trapped 'd', got non-nil")
		}
		if afterD.screen != ScreenMeetingList {
			t.Fatalf("expected screen to remain %v, got %v", ScreenMeetingList, afterD.screen)
		}
	})

	t.Run("view compositing overlays help content on top of base screen", func(t *testing.T) {
		m := NewAppModel(&config.Config{}, nil, nil, "", false)
		m.screen = ScreenMeetingList
		opened, _ := appUpdate(t, m, tea.KeyPressMsg{Text: "?"})

		view := ansi.Strip(opened.View().Content)
		if !strings.Contains(view, "Keybindings") {
			t.Fatalf("expected overlaid view to contain help content")
		}
		if !strings.Contains(view, "No meetings yet. Press 'n' to create one.") {
			t.Fatalf("expected overlaid view to retain underlying list content")
		}
	})

	t.Run("help content remains raw without modal border wrapper", func(t *testing.T) {
		h := NewHelpModel()
		raw := ansi.Strip(h.View().Content)

		if !strings.Contains(raw, "Keybindings") {
			t.Fatalf("expected raw help content to contain keybinding title")
		}

		borderChars := []string{"╭", "╮", "╰", "╯", "│"}
		for _, border := range borderChars {
			if strings.Contains(raw, border) {
				t.Fatalf("expected raw help view without modal border char %q", border)
			}
		}
	})

	t.Run("opening and closing help preserves underlying screen", func(t *testing.T) {
		m := NewAppModel(&config.Config{}, nil, nil, "", false)
		m.screen = ScreenMeetingList

		opened, _ := appUpdate(t, m, tea.KeyPressMsg{Text: "?"})
		if opened.screen != ScreenMeetingList {
			t.Fatalf("expected screen to remain %v after opening help, got %v", ScreenMeetingList, opened.screen)
		}

		closed, _ := appUpdate(t, opened, tea.KeyPressMsg{Text: "?"})
		if closed.screen != ScreenMeetingList {
			t.Fatalf("expected screen to remain %v after closing help, got %v", ScreenMeetingList, closed.screen)
		}
	})

	t.Run("help model init and window size update behavior", func(t *testing.T) {
		h := NewHelpModel()
		initCmd := h.Init()
		if initCmd != nil {
			t.Fatalf("expected nil init cmd, got non-nil")
		}

		updatedModel, updateCmd := h.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		if updateCmd != nil {
			t.Fatalf("expected nil update cmd, got non-nil")
		}

		updated, ok := updatedModel.(HelpModel)
		if !ok {
			t.Fatalf("expected HelpModel type, got %T", updatedModel)
		}
		if updated.width != 120 {
			t.Fatalf("expected updated width 120, got %d", updated.width)
		}
		if updated.height != 40 {
			t.Fatalf("expected updated height 40, got %d", updated.height)
		}
	})
}

func TestHelpView_WelcomeScreen(t *testing.T) {
	m := HelpModel{screen: ScreenWelcome}
	view := m.View().Content

	if !strings.Contains(view, "Welcome Screen") {
		t.Fatalf("expected help view to contain %q", "Welcome Screen")
	}
	if !strings.Contains(view, "Global") {
		t.Fatalf("expected help view to contain %q", "Global")
	}

	if strings.Contains(view, "Meeting List") {
		t.Fatalf("expected help view to not contain %q", "Meeting List")
	}
	if strings.Contains(view, "Editor") {
		t.Fatalf("expected help view to not contain %q", "Editor")
	}
	if strings.Contains(view, "Email") {
		t.Fatalf("expected help view to not contain %q", "Email")
	}
}

func TestHelpView_MeetingListScreen(t *testing.T) {
	m := HelpModel{screen: ScreenMeetingList}
	view := m.View().Content

	if !strings.Contains(view, "Meeting List") {
		t.Fatalf("expected help view to contain %q", "Meeting List")
	}
	if !strings.Contains(view, "Global") {
		t.Fatalf("expected help view to contain %q", "Global")
	}
	if !strings.Contains(view, "Confirmation Dialogs") {
		t.Fatalf("expected help view to contain %q", "Confirmation Dialogs")
	}

	if strings.Contains(view, "Welcome Screen") {
		t.Fatalf("expected help view to not contain %q", "Welcome Screen")
	}
	if strings.Contains(view, "Editor") {
		t.Fatalf("expected help view to not contain %q", "Editor")
	}
}

func TestHelpView_EditorScreen(t *testing.T) {
	m := HelpModel{screen: ScreenEditor}
	view := m.View().Content

	if !strings.Contains(view, "Editor") {
		t.Fatalf("expected help view to contain %q", "Editor")
	}
	if !strings.Contains(view, "Split Pane") {
		t.Fatalf("expected help view to contain %q", "Split Pane")
	}
	if !strings.Contains(view, "Global") {
		t.Fatalf("expected help view to contain %q", "Global")
	}
	if !strings.Contains(view, "Confirmation Dialogs") {
		t.Fatalf("expected help view to contain %q", "Confirmation Dialogs")
	}

	if strings.Contains(view, "Welcome Screen") {
		t.Fatalf("expected help view to not contain %q", "Welcome Screen")
	}
	if strings.Contains(view, "Meeting List") {
		t.Fatalf("expected help view to not contain %q", "Meeting List")
	}
	if strings.Contains(view, "Email") {
		t.Fatalf("expected help view to not contain %q", "Email")
	}
}

func TestHelpView_EmailScreen(t *testing.T) {
	m := HelpModel{screen: ScreenEmail}
	view := m.View().Content

	if !strings.Contains(view, "Email") {
		t.Fatalf("expected help view to contain %q", "Email")
	}
	if !strings.Contains(view, "Global") {
		t.Fatalf("expected help view to contain %q", "Global")
	}

	if strings.Contains(view, "Editor") {
		t.Fatalf("expected help view to not contain %q", "Editor")
	}
	if strings.Contains(view, "Meeting List") {
		t.Fatalf("expected help view to not contain %q", "Meeting List")
	}
}

func TestAppModel_HelpOpensWithCorrectScreen(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false)
	m.screen = ScreenMeetingList

	updated, _ := appUpdate(t, m, tea.KeyPressMsg{Code: '/', Mod: tea.ModCtrl})

	if updated.showHelp != true {
		t.Fatalf("expected showHelp true, got %v", updated.showHelp)
	}
	if updated.helpModel.screen != ScreenMeetingList {
		t.Fatalf("expected helpModel.screen %v, got %v", ScreenMeetingList, updated.helpModel.screen)
	}
}
