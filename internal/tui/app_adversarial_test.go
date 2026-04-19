package tui

import (
	"context"
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/ai"
	"github.com/glieske/recap/internal/storage"
)

func updateAppNoPanic(t *testing.T, m AppModel, msg tea.Msg) (AppModel, tea.Cmd) {
	t.Helper()

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("Update panicked for msg %T: %v", msg, recovered)
		}
	}()

	updatedModel, cmd := m.Update(msg)
	updated, ok := updatedModel.(AppModel)
	if !ok {
		t.Fatalf("expected AppModel from Update, got %T", updatedModel)
	}

	return updated, cmd
}

type adversarialProvider struct{}

func (adversarialProvider) StructureNotes(context.Context, string, ai.MeetingMeta) (string, error) {
	return "# Structured", nil
}

func (adversarialProvider) GenerateEmailSummary(context.Context, string, string) (string, error) {
	return "Subject: Summary\n\nBody", nil
}

func TestAppAdversarialNilStore(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false)

	for _, screen := range []Screen{
		ScreenMeetingList,
		ScreenNewMeeting,
		ScreenEditor,
		ScreenEmail,
		ScreenHelp,
	} {
		m, _ = updateAppNoPanic(t, m, NavigateMsg{Screen: screen})
	}

	msgs := []tea.Msg{
		tea.WindowSizeMsg{Width: 80, Height: 24},
		NavigateMsg{Screen: ScreenMeetingList},
		MeetingSelectedMsg{Meeting: storage.Meeting{Title: "adversarial"}},
		MeetingCreatedMsg{Meeting: nil},
		TriggerAIMsg{},
		TriggerEmailMsg{},
		AIStructureDoneMsg{StructuredMD: "# Structured"},
		AIStructureErrMsg{Err: errors.New("ai structure failed")},
		AIEmailDoneMsg{Subject: "Summary", Body: "Body"},
		AIEmailErrMsg{Err: errors.New("ai email failed")},
		TogglePreviewMsg{},
		RegenerateEmailMsg{},
		SaveDoneMsg{},
		SaveErrMsg{Err: errors.New("save failed")},
		tea.KeyPressMsg{Text: "?"},
		tea.KeyPressMsg{Code: tea.KeyEscape},
		tea.KeyPressMsg{Text: "q"},
	}

	for _, msg := range msgs {
		m, _ = updateAppNoPanic(t, m, msg)
	}

	if m.width != 80 || m.height != 24 {
		t.Fatalf("expected width=80 and height=24 after window resize, got width=%d height=%d", m.width, m.height)
	}
}

func TestAppAdversarialRapidScreenSwitching(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false)

	sequence := []Screen{
		ScreenMeetingList,
		ScreenNewMeeting,
		ScreenEditor,
		ScreenEmail,
		ScreenHelp,
		Screen(-1),
		Screen(999),
	}

	const switches = 160
	for i := 0; i < switches; i++ {
		target := sequence[i%len(sequence)]
		m, _ = updateAppNoPanic(t, m, NavigateMsg{Screen: target})
	}

	expected := sequence[(switches-1)%len(sequence)]
	if m.screen != expected {
		t.Fatalf("expected final screen %v, got %v", expected, m.screen)
	}
}

func TestAppAdversarialAIMsgWithNoEditor(t *testing.T) {
	m := NewAppModel(nil, nil, adversarialProvider{}, "", false)
	m.currentMeeting = &storage.Meeting{Title: "meeting"}
	m.hasEditorModel = false

	updated, cmd := updateAppNoPanic(t, m, TriggerAIMsg{})
	if cmd != nil {
		t.Fatalf("expected nil command when hasEditorModel=false, got non-nil")
	}
	if updated.aiRunning {
		t.Fatalf("expected aiRunning=false when hasEditorModel=false")
	}
}

func TestAppAdversarialAIMsgWithNoProvider(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false)
	m.currentMeeting = &storage.Meeting{Title: "meeting"}
	m.hasEditorModel = true

	updated, cmd := updateAppNoPanic(t, m, TriggerAIMsg{})
	if cmd != nil {
		t.Fatalf("expected nil command when provider=nil, got non-nil")
	}
	if updated.aiRunning {
		t.Fatalf("expected aiRunning=false when provider=nil")
	}
}

func TestAppAdversarialMeetingCreatedNilMeeting(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false)

	updated, cmd := updateAppNoPanic(t, m, MeetingCreatedMsg{Meeting: nil})
	if cmd != nil {
		t.Fatalf("expected nil command for MeetingCreatedMsg with nil meeting")
	}
	if updated.hasEditorModel {
		t.Fatalf("expected hasEditorModel=false for nil meeting")
	}
}

func TestAppAdversarialZeroSizeWindow(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false)

	updated, _ := updateAppNoPanic(t, m, tea.WindowSizeMsg{Width: 0, Height: 0})
	if updated.width != 0 || updated.height != 0 {
		t.Fatalf("expected width=0 height=0, got width=%d height=%d", updated.width, updated.height)
	}
}

func TestAppAdversarialNegativeSizeWindow(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false)

	updated, _ := updateAppNoPanic(t, m, tea.WindowSizeMsg{Width: -1, Height: -1})
	if updated.width != -1 || updated.height != -1 {
		t.Fatalf("expected width=-1 height=-1, got width=%d height=%d", updated.width, updated.height)
	}
}

func TestAppAdversarialTogglePreviewWithNoEditor(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false)
	m.screen = ScreenMeetingList

	updated, cmd := updateAppNoPanic(t, m, TogglePreviewMsg{})
	if cmd != nil {
		t.Fatalf("expected nil command when toggling preview outside editor")
	}
	if updated.screen != ScreenMeetingList {
		t.Fatalf("expected screen to remain %v, got %v", ScreenMeetingList, updated.screen)
	}
}
