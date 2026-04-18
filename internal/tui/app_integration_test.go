package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/storage"
)

func appKeyRunes(r string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: r}
}

func appUpdate(t *testing.T, m AppModel, msg tea.Msg) (AppModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	app, ok := updated.(AppModel)
	if !ok {
		t.Fatalf("expected Update to return AppModel, got %T", updated)
	}

	return app, cmd
}

func assertQuitCmd(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Fatalf("expected quit command, got nil")
	}

	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

func assertNotQuitCmd(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		return
	}

	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); ok {
		t.Fatalf("expected non-quit command")
	}
}

func newTestStore(t *testing.T) *storage.Store {
	t.Helper()
	tmpDir := t.TempDir()
	return storage.NewStore(tmpDir)
}

func createProjectAndMeeting(t *testing.T, store *storage.Store) *storage.Meeting {
	t.Helper()

	_, err := store.CreateProject("Infrastructure", "INFRA")
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	meeting, err := store.CreateMeeting(
		"Integration test meeting",
		time.Date(2026, time.January, 15, 10, 0, 0, 0, time.UTC),
		[]string{"Alice", "Bob"},
		"INFRA",
		[]string{"sync"},
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting failed: %v", err)
	}

	return meeting
}

func TestAppNewAppModelStartsOnWelcome(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "")

	if m.screen != ScreenWelcome {
		t.Fatalf("expected initial screen %v, got %v", ScreenWelcome, m.screen)
	}
}

func TestAppInitReturnsCommand(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "")

	got := m.Init()
	want := m.listModel.Init()
	if (got == nil) != (want == nil) {
		t.Fatalf("expected Init nil-ness %v, got %v", want == nil, got == nil)
	}
}

func TestAppWindowSizeUpdatesModel(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "")

	updated, _ := appUpdate(t, m, tea.WindowSizeMsg{Width: 120, Height: 35})

	if updated.width != 120 {
		t.Fatalf("expected width 120, got %d", updated.width)
	}
	if updated.height != 35 {
		t.Fatalf("expected height 35, got %d", updated.height)
	}
}

func TestAppNavigateToNewMeeting(t *testing.T) {
	store := newTestStore(t)
	m := NewAppModel(nil, store, nil, "")
	m.screen = ScreenMeetingList

	updated, _ := appUpdate(t, m, NavigateMsg{Screen: ScreenNewMeeting})

	if updated.screen != ScreenMeetingList {
		t.Fatalf("expected screen %v, got %v", ScreenMeetingList, updated.screen)
	}
	if !updated.showNewMeeting {
		t.Fatalf("expected showNewMeeting true, got false")
	}
	if updated.newMeetingModal.Title != "New Meeting" {
		t.Fatalf("expected newMeetingModal title %q, got %q", "New Meeting", updated.newMeetingModal.Title)
	}
	if !updated.hasNewMeetingModel {
		t.Fatalf("expected hasNewMeetingModel true, got false")
	}
}

func TestAppQuitOnlyFromListScreen(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "")
	m.screen = ScreenMeetingList

	fromList, cmd := appUpdate(t, m, appKeyRunes("q"))
	if fromList.screen != ScreenMeetingList {
		t.Fatalf("expected list screen to remain %v, got %v", ScreenMeetingList, fromList.screen)
	}
	assertQuitCmd(t, cmd)

	fromEditor := NewAppModel(nil, nil, nil, "")
	fromEditor.screen = ScreenEditor
	fromEditor.editorModel = NewEditorModel(nil, nil, 80, 24, "", "")
	fromEditor.hasEditorModel = true

	updatedEditor, editorCmd := appUpdate(t, fromEditor, appKeyRunes("q"))
	if updatedEditor.screen != ScreenEditor {
		t.Fatalf("expected editor screen %v, got %v", ScreenEditor, updatedEditor.screen)
	}
	assertNotQuitCmd(t, editorCmd)
}

func TestAppCtrlCQuitsFromAnyScreen(t *testing.T) {
	testScreens := []Screen{
		ScreenMeetingList,
		ScreenNewMeeting,
		ScreenEditor,
		ScreenEmail,
		ScreenHelp,
	}

	for _, screen := range testScreens {
		t.Run(screenName(screen), func(t *testing.T) {
			m := NewAppModel(nil, nil, nil, "")
			m.screen = screen

			if screen == ScreenEditor {
				m.editorModel = NewEditorModel(nil, nil, 80, 24, "", "")
				m.hasEditorModel = true
			}

			_, cmd := appUpdate(t, m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
			assertQuitCmd(t, cmd)
		})
	}
}

func TestAppHelpToggle(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "")
	m.screen = ScreenMeetingList

	toHelp, _ := appUpdate(t, m, appKeyRunes("?"))
	if !toHelp.showHelp {
		t.Fatalf("expected showHelp true, got false")
	}
	if toHelp.screen != ScreenMeetingList {
		t.Fatalf("expected screen %v, got %v", ScreenMeetingList, toHelp.screen)
	}

	backToList, _ := appUpdate(t, toHelp, appKeyRunes("?"))
	if backToList.showHelp {
		t.Fatalf("expected showHelp false, got true")
	}
	if backToList.screen != ScreenMeetingList {
		t.Fatalf("expected screen %v, got %v", ScreenMeetingList, backToList.screen)
	}
}

func TestAppQuestionMarkDelegatesToEditorScreen(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "")
	m.screen = ScreenEditor
	m.editorModel = NewEditorModel(nil, nil, 80, 24, "", "")
	m.hasEditorModel = true

	updated, _ := appUpdate(t, m, appKeyRunes("?"))
	if updated.screen != ScreenEditor {
		t.Fatalf("expected screen to remain %v, got %v", ScreenEditor, updated.screen)
	}
}

func TestAppEscFromEditorSequencesSaveAndRefresh(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "")
	m.screen = ScreenEditor
	m.editorModel = NewEditorModel(nil, nil, 80, 24, "", "")
	m.hasEditorModel = true

	updated, cmd := appUpdate(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if updated.screen != ScreenMeetingList {
		t.Fatalf("expected screen %v, got %v", ScreenMeetingList, updated.screen)
	}
	if cmd == nil {
		t.Fatalf("expected non-nil command for editor esc sequence")
	}
}

func TestAppEscFromHelpReturnsToPreviousScreen(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "")
	m.screen = ScreenEditor
	m.showHelp = true

	updated, cmd := appUpdate(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatalf("expected non-nil command for help dismiss")
	}

	dismissMsg, ok := cmd().(tea.Msg)
	if !ok {
		t.Fatalf("expected tea.Msg from dismiss command")
	}

	final, _ := appUpdate(t, updated, dismissMsg)
	if final.showHelp {
		t.Fatalf("expected showHelp false, got true")
	}
	if final.screen != ScreenEditor {
		t.Fatalf("expected screen %v, got %v", ScreenEditor, final.screen)
	}
}

func TestAppMeetingSelectedMsgCreatesEditor(t *testing.T) {
	store := newTestStore(t)
	meeting := createProjectAndMeeting(t, store)
	m := NewAppModel(nil, store, nil, "")

	updated, cmd := appUpdate(t, m, MeetingSelectedMsg{Meeting: *meeting})

	if updated.screen != ScreenEditor {
		t.Fatalf("expected screen %v, got %v", ScreenEditor, updated.screen)
	}
	if !updated.hasEditorModel {
		t.Fatalf("expected hasEditorModel true, got false")
	}
	if updated.currentMeeting == nil {
		t.Fatalf("expected currentMeeting to be set")
	}
	if updated.currentMeeting.ID != meeting.ID {
		t.Fatalf("expected current meeting ID %q, got %q", meeting.ID, updated.currentMeeting.ID)
	}
	if cmd == nil {
		t.Fatalf("expected editor init command, got nil")
	}
}

func TestAppTriggerAIMsgWithoutProviderIsNoOp(t *testing.T) {
	store := newTestStore(t)
	meeting := createProjectAndMeeting(t, store)
	m := NewAppModel(nil, store, nil, "")
	m.currentMeeting = meeting
	m.editorModel = NewEditorModel(meeting, store, 80, 24, "", "")
	m.hasEditorModel = true

	updated, cmd := appUpdate(t, m, TriggerAIMsg{})
	if cmd != nil {
		t.Fatalf("expected nil command when provider is nil")
	}
	if updated.aiRunning {
		t.Fatalf("expected aiRunning false, got true")
	}
}

func TestAppTogglePreviewFromEditor(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "")
	m.screen = ScreenEditor
	m.editorModel = NewEditorModel(nil, nil, 80, 24, "", "")
	m.hasEditorModel = true

	updated, _ := appUpdate(t, m, TogglePreviewMsg{})
	if updated.screen != ScreenEditor {
		t.Fatalf("expected screen %v, got %v", ScreenEditor, updated.screen)
	}
}

func TestAppViewUnknownScreen(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "")
	m.screen = Screen(999)

	got := m.View().Content
	if got != "Unknown screen" {
		t.Fatalf("expected %q, got %q", "Unknown screen", got)
	}
}
