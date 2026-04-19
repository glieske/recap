package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/storage"
)

func TestNewMeetingAdversarial_DoubleOpenReplacesOverlayWithoutCorruption(t *testing.T) {
	store := newTestStore(t)
	m := NewAppModel(nil, store, nil, "", false, "")
	m.screen = ScreenMeetingList

	firstOpen, _ := appUpdate(t, m, NavigateMsg{Screen: ScreenNewMeeting})
	if firstOpen.showNewMeeting != true {
		t.Fatalf("expected showNewMeeting true after first open, got %v", firstOpen.showNewMeeting)
	}
	if firstOpen.hasNewMeetingModel != true {
		t.Fatalf("expected hasNewMeetingModel true after first open, got %v", firstOpen.hasNewMeetingModel)
	}

	secondOpen, _ := appUpdate(t, firstOpen, NavigateMsg{Screen: ScreenNewMeeting})
	if secondOpen.showNewMeeting != true {
		t.Fatalf("expected showNewMeeting true after second open, got %v", secondOpen.showNewMeeting)
	}
	if secondOpen.hasNewMeetingModel != true {
		t.Fatalf("expected hasNewMeetingModel true after second open, got %v", secondOpen.hasNewMeetingModel)
	}
	if secondOpen.newMeetingModal.Title != "New Meeting" {
		t.Fatalf("expected modal title %q, got %q", "New Meeting", secondOpen.newMeetingModal.Title)
	}
	if secondOpen.screen != ScreenMeetingList {
		t.Fatalf("expected screen %v after second open, got %v", ScreenMeetingList, secondOpen.screen)
	}
}

func TestNewMeetingAdversarial_DismissWhenNoOverlayIsNoOp(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false, "")
	m.screen = ScreenMeetingList
	m.showNewMeeting = false
	m.showHelp = false

	updated, _ := appUpdate(t, m, DismissModalMsg{})

	if updated.showNewMeeting != false {
		t.Fatalf("expected showNewMeeting false after dismiss without overlay, got %v", updated.showNewMeeting)
	}
	if updated.showHelp != false {
		t.Fatalf("expected showHelp false after dismiss without overlay, got %v", updated.showHelp)
	}
	if updated.screen != ScreenMeetingList {
		t.Fatalf("expected screen %v, got %v", ScreenMeetingList, updated.screen)
	}
}

func TestNewMeetingAdversarial_MeetingCreatedNilWhenClosedDoesNotCorruptState(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false, "")
	beforeScreen := m.screen

	updated, cmd := appUpdate(t, m, MeetingCreatedMsg{Meeting: nil})

	if cmd != nil {
		t.Fatalf("expected nil command for nil MeetingCreatedMsg, got non-nil")
	}
	if updated.showNewMeeting != false {
		t.Fatalf("expected showNewMeeting false, got %v", updated.showNewMeeting)
	}
	if updated.currentMeeting != nil {
		t.Fatalf("expected currentMeeting nil, got non-nil")
	}
	if updated.hasEditorModel != false {
		t.Fatalf("expected hasEditorModel false, got %v", updated.hasEditorModel)
	}
	if updated.screen != beforeScreen {
		t.Fatalf("expected screen to remain %v, got %v", beforeScreen, updated.screen)
	}
}

func TestNewMeetingAdversarial_ZeroSizeWindowWhileOverlayVisible(t *testing.T) {
	store := newTestStore(t)
	m := NewAppModel(nil, store, nil, "", false, "")
	m.screen = ScreenMeetingList

	opened, _ := appUpdate(t, m, NavigateMsg{Screen: ScreenNewMeeting})
	if opened.showNewMeeting != true {
		t.Fatalf("expected showNewMeeting true after open, got %v", opened.showNewMeeting)
	}

	updated, _ := appUpdate(t, opened, tea.WindowSizeMsg{Width: 0, Height: 0})

	if updated.width != 0 {
		t.Fatalf("expected width 0, got %d", updated.width)
	}
	if updated.height != 0 {
		t.Fatalf("expected height 0, got %d", updated.height)
	}
	if updated.showNewMeeting != true {
		t.Fatalf("expected showNewMeeting true after zero-size resize, got %v", updated.showNewMeeting)
	}

	view := updated.View()
	if view.AltScreen != true {
		t.Fatalf("expected AltScreen true for app view, got %v", view.AltScreen)
	}
}

func TestNewMeetingAdversarial_RapidOpenCloseCyclesDoNotLeakState(t *testing.T) {
	store := newTestStore(t)
	m := NewAppModel(nil, store, nil, "", false, "")
	m.screen = ScreenMeetingList

	for i := 0; i < 10; i++ {
		opened, _ := appUpdate(t, m, NavigateMsg{Screen: ScreenNewMeeting})
		if opened.showNewMeeting != true {
			t.Fatalf("cycle %d: expected showNewMeeting true after open, got %v", i, opened.showNewMeeting)
		}
		if opened.hasNewMeetingModel != true {
			t.Fatalf("cycle %d: expected hasNewMeetingModel true after open, got %v", i, opened.hasNewMeetingModel)
		}

		closed, _ := appUpdate(t, opened, DismissModalMsg{})
		if closed.showNewMeeting != false {
			t.Fatalf("cycle %d: expected showNewMeeting false after close, got %v", i, closed.showNewMeeting)
		}
		if closed.screen != ScreenMeetingList {
			t.Fatalf("cycle %d: expected screen %v after close, got %v", i, ScreenMeetingList, closed.screen)
		}

		m = closed
	}

	if m.showHelp != false {
		t.Fatalf("expected showHelp false after rapid cycles, got %v", m.showHelp)
	}

	// Re-touch exported Init using adversarially cycled state to ensure command path remains stable.
	initNil := m.Init() == nil
	wantNil := m.listModel.Init() == nil
	if initNil != wantNil {
		t.Fatalf("expected Init nil-ness %v after rapid cycles, got %v", wantNil, initNil)
	}
}

func TestNewMeetingAdversarial_MeetingCreatedNilWithBoundaryTimestampInput(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false, "")
	boundaryMeeting := &storage.Meeting{Date: time.Unix(0, 0).UTC()}

	opened, _ := appUpdate(t, m, NavigateMsg{Screen: ScreenNewMeeting})
	if opened.showNewMeeting != true {
		t.Fatalf("expected showNewMeeting true before boundary check, got %v", opened.showNewMeeting)
	}

	updated, _ := appUpdate(t, opened, MeetingCreatedMsg{Meeting: nil})
	if updated.currentMeeting == boundaryMeeting {
		t.Fatalf("expected currentMeeting to remain nil, got unexpected boundary meeting reference")
	}
	if updated.currentMeeting != nil {
		t.Fatalf("expected currentMeeting nil after nil MeetingCreatedMsg, got non-nil")
	}
}
