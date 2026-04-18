package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/glieske/recap/internal/config"
	"github.com/glieske/recap/internal/storage"
)

func TestNewMeetingOverlayModalOpensFromNavigateWithoutScreenSwitch(t *testing.T) {
	m := NewAppModel(&config.Config{}, nil, nil, "")
	m.screen = ScreenMeetingList

	updated, cmd := appUpdate(t, m, NavigateMsg{Screen: ScreenNewMeeting})

	if cmd == nil {
		t.Fatalf("expected modal init cmd, got nil")
	}
	if updated.showNewMeeting != true {
		t.Fatalf("expected showNewMeeting true, got %v", updated.showNewMeeting)
	}
	if updated.hasNewMeetingModel != true {
		t.Fatalf("expected hasNewMeetingModel true, got %v", updated.hasNewMeetingModel)
	}
	if updated.newMeetingModal.Title != "New Meeting" {
		t.Fatalf("expected newMeetingModal title %q, got %q", "New Meeting", updated.newMeetingModal.Title)
	}
	if updated.screen != ScreenMeetingList {
		t.Fatalf("expected screen to remain %v, got %v", ScreenMeetingList, updated.screen)
	}
}

func TestNewMeetingOverlayEscapeEmitsDismissThenClosesModal(t *testing.T) {
	m := NewAppModel(&config.Config{}, nil, nil, "")
	opened, _ := appUpdate(t, m, NavigateMsg{Screen: ScreenNewMeeting})

	afterEsc, cmd := appUpdate(t, opened, tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatalf("expected non-nil cmd from new-meeting modal esc")
	}
	if afterEsc.showNewMeeting != true {
		t.Fatalf("expected showNewMeeting true before DismissModalMsg, got %v", afterEsc.showNewMeeting)
	}

	emitted := cmd()
	if _, ok := emitted.(DismissModalMsg); !ok {
		t.Fatalf("expected DismissModalMsg, got %T", emitted)
	}

	dismissed, dismissCmd := appUpdate(t, afterEsc, emitted)
	if dismissCmd != nil {
		t.Fatalf("expected nil cmd when processing DismissModalMsg, got non-nil")
	}
	if dismissed.showNewMeeting != false {
		t.Fatalf("expected showNewMeeting false after DismissModalMsg, got %v", dismissed.showNewMeeting)
	}
}

func TestNewMeetingOverlayMeetingCreatedClosesModal(t *testing.T) {
	m := NewAppModel(&config.Config{}, nil, nil, "")
	opened, _ := appUpdate(t, m, NavigateMsg{Screen: ScreenNewMeeting})

	meeting := &storage.Meeting{
		ID:      "meeting-1",
		Title:   "Overlay Test",
		Date:    time.Date(2026, time.January, 20, 9, 0, 0, 0, time.UTC),
		Project: "INFRA",
	}

	updated, _ := appUpdate(t, opened, MeetingCreatedMsg{Meeting: meeting})

	if updated.showNewMeeting != false {
		t.Fatalf("expected showNewMeeting false after MeetingCreatedMsg, got %v", updated.showNewMeeting)
	}
	if updated.screen != ScreenEditor {
		t.Fatalf("expected screen %v after MeetingCreatedMsg, got %v", ScreenEditor, updated.screen)
	}
}

func TestNewMeetingOverlayZOrderPrefersNewMeetingOverHelp(t *testing.T) {
	m := NewAppModel(&config.Config{}, nil, nil, "")
	opened, _ := appUpdate(t, m, NavigateMsg{Screen: ScreenNewMeeting})
	opened.showHelp = true
	opened.helpModal = NewModalModel("Help", opened.helpModel, opened.width, opened.height)

	afterEsc, cmd := appUpdate(t, opened, tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatalf("expected non-nil cmd from esc while overlays are stacked")
	}

	emitted := cmd()
	if _, ok := emitted.(DismissModalMsg); !ok {
		t.Fatalf("expected DismissModalMsg, got %T", emitted)
	}

	dismissed, _ := appUpdate(t, afterEsc, emitted)
	if dismissed.showNewMeeting != false {
		t.Fatalf("expected new-meeting modal dismissed first, got showNewMeeting=%v", dismissed.showNewMeeting)
	}
	if dismissed.showHelp != true {
		t.Fatalf("expected help overlay to remain open after dismissing new-meeting first, got showHelp=%v", dismissed.showHelp)
	}
}

func TestNewMeetingOverlayViewRendersModalOnTopOfMeetingList(t *testing.T) {
	m := NewAppModel(&config.Config{}, nil, nil, "")
	m.screen = ScreenMeetingList
	opened, _ := appUpdate(t, m, NavigateMsg{Screen: ScreenNewMeeting})

	view := ansi.Strip(opened.View().Content)
	if !strings.Contains(view, "New Meeting") {
		t.Fatalf("expected overlay view to contain modal title")
	}
	if !strings.Contains(view, "No meetings yet. Press 'n' to create one.") {
		t.Fatalf("expected overlay view to retain underlying list content")
	}
	if !strings.Contains(view, "╭") || !strings.Contains(view, "╯") {
		t.Fatalf("expected overlay view to contain modal border characters")
	}
}

func TestNewMeetingOverlayCtrlCQuitsWhenModalIsShowing(t *testing.T) {
	m := NewAppModel(&config.Config{}, nil, nil, "")
	opened, _ := appUpdate(t, m, NavigateMsg{Screen: ScreenNewMeeting})

	_, cmd := appUpdate(t, opened, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatalf("expected quit command, got nil")
	}

	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}
