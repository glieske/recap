package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/storage"
)

func appUpdateNoPanic(t *testing.T, m AppModel, msg tea.Msg) (AppModel, tea.Cmd) {
	t.Helper()

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("Update panicked for %T: %v", msg, recovered)
		}
	}()

	updated, cmd := appUpdate(t, m, msg)
	return updated, cmd
}

func TestAppSplitWiringAdversarial(t *testing.T) {
	t.Run("AIStructureDoneMsg with no editor does not panic", func(t *testing.T) {
		m := NewAppModel(nil, nil, nil, "", false)

		updated, cmd := appUpdateNoPanic(t, m, AIStructureDoneMsg{StructuredMD: "# generated"})

		if updated.hasEditorModel != false {
			t.Fatalf("expected hasEditorModel=false, got %v", updated.hasEditorModel)
		}
		if updated.structuredMD != "# generated" {
			t.Fatalf("expected structuredMD %q, got %q", "# generated", updated.structuredMD)
		}
		if updated.statusMsg != "Structured notes updated" {
			t.Fatalf("expected statusMsg %q, got %q", "Structured notes updated", updated.statusMsg)
		}
		if cmd != nil {
			t.Fatalf("expected nil command, got non-nil")
		}
	})

	t.Run("AIStructureDoneMsg with empty StructuredMD updates summary to empty", func(t *testing.T) {
		store := newTestStore(t)
		meeting := createProjectAndMeeting(t, store)
		m := NewAppModel(nil, store, nil, "", false)

		updated, _ := appUpdateNoPanic(t, m, MeetingSelectedMsg{Meeting: *meeting})
		updated, cmd := appUpdateNoPanic(t, updated, AIStructureDoneMsg{StructuredMD: ""})

		if updated.structuredMD != "" {
			t.Fatalf("expected structuredMD to be empty, got %q", updated.structuredMD)
		}
		summary := updated.editorModel.GetSummaryModel()
		if summary.Value() != "" {
			t.Fatalf("expected summary content to be empty, got %q", summary.Value())
		}
		if updated.statusMsg != "Structured notes updated" {
			t.Fatalf("expected statusMsg %q, got %q", "Structured notes updated", updated.statusMsg)
		}
		if cmd != nil {
			t.Fatalf("expected nil command, got non-nil")
		}
	})

	t.Run("TogglePreviewMsg from non-editor screens is no-op", func(t *testing.T) {
		screens := []Screen{ScreenMeetingList, ScreenEmail, ScreenHelp, ScreenNewMeeting, ScreenProviderSelector}

		for _, screen := range screens {
			m := NewAppModel(nil, nil, nil, "", false)
			m.screen = screen

			updated, cmd := appUpdateNoPanic(t, m, TogglePreviewMsg{})

			if updated.screen != screen {
				t.Fatalf("expected screen %v to remain unchanged, got %v", screen, updated.screen)
			}
			if cmd != nil {
				t.Fatalf("expected nil command for screen %v, got non-nil", screen)
			}
		}
	})

	t.Run("MeetingSelectedMsg with nil store still wires SummaryModel", func(t *testing.T) {
		m := NewAppModel(nil, nil, nil, "", false)
		meeting := storage.Meeting{ID: "meeting-nil-store", Project: "INFRA", Title: "Nil store"}

		updated, _ := appUpdateNoPanic(t, m, MeetingSelectedMsg{Meeting: meeting})

		if updated.hasEditorModel != true {
			t.Fatalf("expected hasEditorModel=true, got %v", updated.hasEditorModel)
		}
		if updated.editorModel.hasSummaryModel != true {
			t.Fatalf("expected hasSummaryModel=true, got %v", updated.editorModel.hasSummaryModel)
		}
		summary := updated.editorModel.GetSummaryModel()
		if summary.store != nil {
			t.Fatalf("expected SummaryModel store=nil, got non-nil")
		}
		if summary.project != "INFRA" {
			t.Fatalf("expected summary project %q, got %q", "INFRA", summary.project)
		}
		if summary.meetingID != "meeting-nil-store" {
			t.Fatalf("expected summary meetingID %q, got %q", "meeting-nil-store", summary.meetingID)
		}
	})

	t.Run("Rapid meeting switching replaces editor and summary state", func(t *testing.T) {
		store := newTestStore(t)
		meetingA := createProjectAndMeeting(t, store)
		meetingB, err := store.CreateMeeting(
			"Second meeting",
			time.Date(2026, time.February, 20, 10, 0, 0, 0, time.UTC),
			[]string{"Carol"},
			"INFRA",
			[]string{"switch"},
			"",
		)
		if err != nil {
			t.Fatalf("CreateMeeting for meetingB failed: %v", err)
		}

		m := NewAppModel(nil, store, nil, "", false)
		updated, _ := appUpdateNoPanic(t, m, MeetingSelectedMsg{Meeting: *meetingA})
		updated, _ = appUpdateNoPanic(t, updated, MeetingSelectedMsg{Meeting: *meetingB})

		if updated.currentMeeting == nil {
			t.Fatalf("expected currentMeeting to be set")
		}
		if updated.currentMeeting.ID != meetingB.ID {
			t.Fatalf("expected currentMeeting ID %q, got %q", meetingB.ID, updated.currentMeeting.ID)
		}
		if updated.editorModel.meeting == nil {
			t.Fatalf("expected editor meeting to be set")
		}
		if updated.editorModel.meeting.ID != meetingB.ID {
			t.Fatalf("expected editor meeting ID %q, got %q", meetingB.ID, updated.editorModel.meeting.ID)
		}
		summary := updated.editorModel.GetSummaryModel()
		if summary.meetingID != meetingB.ID {
			t.Fatalf("expected summary meetingID %q, got %q", meetingB.ID, summary.meetingID)
		}
		if summary.Value() != "" {
			t.Fatalf("expected empty summary for meetingB, got %q", summary.Value())
		}
	})

	t.Run("AIStructureDoneMsg followed by MeetingSelectedMsg does not leak old AI result", func(t *testing.T) {
		store := newTestStore(t)
		meetingA := createProjectAndMeeting(t, store)
		meetingB, err := store.CreateMeeting(
			"Third meeting",
			time.Date(2026, time.March, 11, 9, 30, 0, 0, time.UTC),
			[]string{"Dana"},
			"INFRA",
			[]string{"handoff"},
			"",
		)
		if err != nil {
			t.Fatalf("CreateMeeting for meetingB failed: %v", err)
		}

		m := NewAppModel(nil, store, nil, "", false)
		updated, _ := appUpdateNoPanic(t, m, MeetingSelectedMsg{Meeting: *meetingA})

		const oldAIResult = "# Meeting A structured"
		updated, _ = appUpdateNoPanic(t, updated, AIStructureDoneMsg{StructuredMD: oldAIResult})
		if updated.structuredMD != oldAIResult {
			t.Fatalf("expected structuredMD %q before switch, got %q", oldAIResult, updated.structuredMD)
		}

		updated, _ = appUpdateNoPanic(t, updated, MeetingSelectedMsg{Meeting: *meetingB})

		if updated.structuredMD != "" {
			t.Fatalf("expected structuredMD to reset for meetingB, got %q", updated.structuredMD)
		}
		summary := updated.editorModel.GetSummaryModel()
		if summary.meetingID != meetingB.ID {
			t.Fatalf("expected summary meetingID %q, got %q", meetingB.ID, summary.meetingID)
		}
		if summary.Value() != "" {
			t.Fatalf("expected meetingB summary to be empty, got %q", summary.Value())
		}
	})
}
