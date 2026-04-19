package tui

import (
	"context"
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/ai"
)

type appTestProvider struct {
	emailResponse string
	emailErr      error
}

func (p appTestProvider) StructureNotes(context.Context, string, ai.MeetingMeta) (string, error) {
	return "", nil
}

func (p appTestProvider) GenerateEmailSummary(context.Context, string, string) (string, error) {
	if p.emailErr != nil {
		return "", p.emailErr
	}

	return p.emailResponse, nil
}

func TestAppMeetingCreatedMsgCreatesEditorAndSwitchesScreen(t *testing.T) {
	store := newTestStore(t)
	meeting := createProjectAndMeeting(t, store)
	m := NewAppModel(nil, store, nil, "", false, "")

	updated, cmd := appUpdate(t, m, MeetingCreatedMsg{Meeting: meeting})

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

func TestAppAIStructureDoneMsgSetsStructuredState(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false, "")
	m.aiRunning = true

	const structuredBody = "# Structured Notes\n- Action 1"
	updated, _ := appUpdate(t, m, AIStructureDoneMsg{StructuredMD: structuredBody})

	if updated.aiRunning {
		t.Fatalf("expected aiRunning false, got true")
	}
	if updated.structuredMD != structuredBody {
		t.Fatalf("expected structuredMD %q, got %q", structuredBody, updated.structuredMD)
	}
}

func TestAppAIStructureErrMsgStopsAIRunAndSetsStatusError(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false, "")
	m.aiRunning = true

	updateErr := errors.New("structure failed")
	updated, _ := appUpdate(t, m, AIStructureErrMsg{Err: updateErr})

	if updated.aiRunning {
		t.Fatalf("expected aiRunning false, got true")
	}
	if updated.err == nil {
		t.Fatalf("expected err to be set")
	}
	if updated.err.Error() != updateErr.Error() {
		t.Fatalf("expected err %q, got %q", updateErr.Error(), updated.err.Error())
	}
	if updated.statusMsg != "" {
		t.Fatalf("expected statusMsg to be empty on AI structure error, got %q", updated.statusMsg)
	}
}

func TestAppAIEmailDoneMsgSwitchesToEmailWithModel(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false, "")
	m.aiRunning = true

	updated, _ := appUpdate(t, m, AIEmailDoneMsg{Subject: "Summary", Body: "Body text"})

	if updated.screen != ScreenEmail {
		t.Fatalf("expected screen %v, got %v", ScreenEmail, updated.screen)
	}
	if !updated.hasEmailModel {
		t.Fatalf("expected hasEmailModel true, got false")
	}
	if updated.aiRunning {
		t.Fatalf("expected aiRunning false, got true")
	}
}

func TestAppAIEmailErrMsgStopsAIRunAndSetsStatusError(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false, "")
	m.aiRunning = true

	updateErr := errors.New("email generation failed")
	updated, _ := appUpdate(t, m, AIEmailErrMsg{Err: updateErr})

	if updated.aiRunning {
		t.Fatalf("expected aiRunning false, got true")
	}
	if updated.err == nil {
		t.Fatalf("expected err to be set")
	}
	if updated.err.Error() != updateErr.Error() {
		t.Fatalf("expected err %q, got %q", updateErr.Error(), updated.err.Error())
	}
	if updated.statusMsg != "" {
		t.Fatalf("expected statusMsg to be empty on AI email error, got %q", updated.statusMsg)
	}
}

func TestAppSaveDoneMsgClearsEditorErrorState(t *testing.T) {
	store := newTestStore(t)
	meeting := createProjectAndMeeting(t, store)
	m := NewAppModel(nil, store, nil, "", false, "")
	m.editorModel = NewEditorModel(meeting, store, 80, 24, "", "", "")
	m.hasEditorModel = true

	initialSaveErr := errors.New("save failed once")
	withSaveErr, _ := appUpdate(t, m, SaveErrMsg{Err: initialSaveErr})
	if withSaveErr.editorModel.err == nil {
		t.Fatalf("expected editor save error after SaveErrMsg")
	}

	afterSaveDone, _ := appUpdate(t, withSaveErr, SaveDoneMsg{})
	if afterSaveDone.editorModel.statusMsg != "Saved" {
		t.Fatalf("expected editor statusMsg %q, got %q", "Saved", afterSaveDone.editorModel.statusMsg)
	}
	if afterSaveDone.editorModel.statusMsg == "Save error: save failed once" {
		t.Fatalf("expected save error status to be replaced by saved status")
	}
}

func TestAppSaveErrMsgSetsEditorErrorStatus(t *testing.T) {
	store := newTestStore(t)
	meeting := createProjectAndMeeting(t, store)
	m := NewAppModel(nil, store, nil, "", false, "")
	m.editorModel = NewEditorModel(meeting, store, 80, 24, "", "", "")
	m.hasEditorModel = true

	saveErr := errors.New("disk full")
	updated, _ := appUpdate(t, m, SaveErrMsg{Err: saveErr})

	if updated.editorModel.err == nil {
		t.Fatalf("expected editor err to be set")
	}
	if updated.editorModel.err.Error() != saveErr.Error() {
		t.Fatalf("expected editor err %q, got %q", saveErr.Error(), updated.editorModel.err.Error())
	}
	if updated.editorModel.statusMsg != "Save error: disk full" {
		t.Fatalf("expected editor statusMsg %q, got %q", "Save error: disk full", updated.editorModel.statusMsg)
	}
}

func TestAppRegenerateEmailMsgStartsAIWhenConfigured(t *testing.T) {
	store := newTestStore(t)
	meeting := createProjectAndMeeting(t, store)
	provider := appTestProvider{emailResponse: "Subject: Follow-up\n\nThanks"}

	m := NewAppModel(nil, store, provider, "", false, "")
	m.screen = ScreenEmail
	m.currentMeeting = meeting
	m.structuredMD = "# Structured"

	updated, cmd := appUpdate(t, m, RegenerateEmailMsg{})

	if !updated.aiRunning {
		t.Fatalf("expected aiRunning true after RegenerateEmailMsg")
	}
	if cmd == nil {
		t.Fatalf("expected non-nil command for RegenerateEmailMsg")
	}

	emitted := cmd()
	if _, ok := emitted.(AIEmailDoneMsg); !ok {
		t.Fatalf("expected AIEmailDoneMsg from regenerate command, got %T", emitted)
	}
}

func TestAppEscFromNewMeetingReturnsToMeetingList(t *testing.T) {
	store := newTestStore(t)
	m := NewAppModel(nil, store, nil, "", false, "")
	m.screen = ScreenMeetingList

	// Open new-meeting as overlay
	opened, _ := appUpdate(t, m, NavigateMsg{Screen: ScreenNewMeeting})
	if !opened.showNewMeeting {
		t.Fatalf("expected showNewMeeting true, got false")
	}

	// Esc → ModalModel emits DismissModalMsg via cmd
	afterEsc, cmd := appUpdate(t, opened, tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatalf("expected non-nil cmd from esc")
	}

	// Execute cmd to get DismissModalMsg
	dismissMsg := cmd()
	if _, ok := dismissMsg.(DismissModalMsg); !ok {
		t.Fatalf("expected DismissModalMsg, got %T", dismissMsg)
	}

	// Process DismissModalMsg
	dismissed, _ := appUpdate(t, afterEsc, dismissMsg)
	if dismissed.showNewMeeting {
		t.Fatalf("expected showNewMeeting false after dismiss")
	}
	if dismissed.screen != ScreenMeetingList {
		t.Fatalf("expected screen %v, got %v", ScreenMeetingList, dismissed.screen)
	}
}

func TestAppEscFromEmailReturnsToMeetingList(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false, "")
	m.screen = ScreenEmail
	m.emailModel = NewEmailModel("Summary", "Body", 80, 24, "pl")
	m.hasEmailModel = true

	updated, _ := appUpdate(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	if updated.screen != ScreenMeetingList {
		t.Fatalf("expected screen %v, got %v", ScreenMeetingList, updated.screen)
	}
}

func TestAutoNewMeeting_InitEmitsNavigateMsg(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", true, "")

	cmd := m.Init()
	if cmd == nil {
		t.Fatalf("expected non-nil init command")
	}

	msg := cmd()
	foundNavigate := false

	switch typed := msg.(type) {
	case NavigateMsg:
		foundNavigate = typed.Screen == ScreenNewMeeting
	case tea.BatchMsg:
		for _, batchedCmd := range typed {
			if batchedCmd == nil {
				continue
			}

			emitted := batchedCmd()
			navMsg, ok := emitted.(NavigateMsg)
			if ok && navMsg.Screen == ScreenNewMeeting {
				foundNavigate = true
				break
			}
		}
	}

	if foundNavigate != true {
		t.Fatalf("expected init command to emit NavigateMsg to ScreenNewMeeting, got %T", msg)
	}

	updatedModel, _ := m.Update(NavigateMsg{Screen: ScreenNewMeeting})
	updated, ok := updatedModel.(AppModel)
	if ok != true {
		t.Fatalf("expected AppModel after update, got %T", updatedModel)
	}
	if updated.showNewMeeting != true {
		t.Fatalf("expected showNewMeeting=true after NavigateMsg to ScreenNewMeeting, got %v", updated.showNewMeeting)
	}
}

func TestAutoNewMeeting_FalseDoesNotTrigger(t *testing.T) {
	m := NewAppModel(nil, nil, nil, "", false, "")

	updatedModel, _ := m.Update(NavigateMsg{Screen: ScreenMeetingList})
	updated, ok := updatedModel.(AppModel)
	if ok != true {
		t.Fatalf("expected AppModel after update, got %T", updatedModel)
	}

	if updated.showNewMeeting != false {
		t.Fatalf("expected showNewMeeting=false for ScreenMeetingList navigate, got %v", updated.showNewMeeting)
	}
	if updated.screen != ScreenMeetingList {
		t.Fatalf("expected screen=%v after navigate, got %v", ScreenMeetingList, updated.screen)
	}
}

func screenName(s Screen) string {
	switch s {
	case ScreenMeetingList:
		return "meeting_list"
	case ScreenNewMeeting:
		return "new_meeting"
	case ScreenEditor:
		return "editor"
	case ScreenEmail:
		return "email"
	case ScreenHelp:
		return "help"
	default:
		return "unknown"
	}
}
