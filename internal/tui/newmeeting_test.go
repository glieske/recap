package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/glieske/recap/internal/storage"
)

func updateNewMeetingModel(t *testing.T, m NewMeetingModel, msg tea.Msg) (NewMeetingModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	updatedModel, ok := updated.(NewMeetingModel)
	if !ok {
		t.Fatalf("expected NewMeetingModel from Update, got %T", updated)
	}

	return updatedModel, cmd
}

func TestNewMeetingModelConstructionNoProjects(t *testing.T) {
	store := storage.NewStore(t.TempDir())

	m := NewNewMeetingModel(store, 100, 30)

	if m.store != store {
		t.Fatalf("expected model store to match input store")
	}

	if m.form == nil {
		t.Fatalf("expected form to be initialized")
	}

	if m.width != 100 || m.height != 30 {
		t.Fatalf("expected width=100 height=30, got width=%d height=%d", m.width, m.height)
	}

	if !m.creatingProject {
		t.Fatalf("expected creatingProject=true when no projects exist")
	}

	if m.values.project != newProjectOptionValue {
		t.Fatalf("expected default project %q, got %q", newProjectOptionValue, m.values.project)
	}

	if _, err := time.Parse(isoDateLayout, m.values.date); err != nil {
		t.Fatalf("expected default date in ISO format, got %q: %v", m.values.date, err)
	}
}

func TestNewMeetingModelConstructionWithExistingProjectsDefaultsToFirstPrefix(t *testing.T) {
	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("Infrastructure", "INFRA"); err != nil {
		t.Fatalf("CreateProject INFRA: %v", err)
	}
	if _, err := store.CreateProject("Application", "APP"); err != nil {
		t.Fatalf("CreateProject APP: %v", err)
	}

	m := NewNewMeetingModel(store, 80, 24)

	if m.creatingProject {
		t.Fatalf("expected creatingProject=false when projects exist")
	}

	if m.values.project != "APP" {
		t.Fatalf("expected first sorted project prefix APP, got %q", m.values.project)
	}
}

func TestNewMeetingModelInitReturnsCmdWhenFormExists(t *testing.T) {
	m := NewNewMeetingModel(storage.NewStore(t.TempDir()), 80, 24)

	if cmd := m.Init(); cmd == nil {
		t.Fatalf("expected non-nil Init command when form is present")
	}
}

func TestNewMeetingModelInitReturnsNilWhenFormMissing(t *testing.T) {
	m := NewMeetingModel{}

	if cmd := m.Init(); cmd != nil {
		t.Fatalf("expected nil Init command when form is nil")
	}
}

func TestNewMeetingModelUpdateWindowSizeMsg(t *testing.T) {
	m := NewNewMeetingModel(storage.NewStore(t.TempDir()), 10, 5)

	updated, cmd := updateNewMeetingModel(t, m, tea.WindowSizeMsg{Width: 120, Height: 42})
	if cmd != nil {
		t.Fatalf("expected nil command for WindowSizeMsg")
	}

	if updated.width != 120 || updated.height != 42 {
		t.Fatalf("expected width=120 height=42, got width=%d height=%d", updated.width, updated.height)
	}
}

func TestNewMeetingModelUpdateEscAndCtrlCEmitNavigateToList(t *testing.T) {
	keys := []struct {
		name string
		msg  tea.KeyPressMsg
	}{
		{name: "esc", msg: tea.KeyPressMsg{Code: tea.KeyEscape}},
		{name: "ctrl+c", msg: tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}},
	}

	for _, tt := range keys {
		t.Run(tt.name, func(t *testing.T) {
			m := NewNewMeetingModel(storage.NewStore(t.TempDir()), 80, 24)

			updated, cmd := updateNewMeetingModel(t, m, tt.msg)
			if !updated.cancelled {
				t.Fatalf("expected cancelled=true for key %s", tt.name)
			}

			if cmd == nil {
				t.Fatalf("expected navigate command for key %s", tt.name)
			}

			msg := cmd()
			nav, ok := msg.(NavigateMsg)
			if !ok {
				t.Fatalf("expected NavigateMsg, got %T", msg)
			}

			if nav.Screen != ScreenMeetingList {
				t.Fatalf("expected ScreenMeetingList, got %v", nav.Screen)
			}
		})
	}
}

func TestNewMeetingModelUpdateNewMeetingErrMsgSetsError(t *testing.T) {
	m := NewNewMeetingModel(storage.NewStore(t.TempDir()), 80, 24)
	expectedErr := errors.New("boom")

	updated, cmd := updateNewMeetingModel(t, m, NewMeetingErrMsg{Err: expectedErr})
	if cmd != nil {
		t.Fatalf("expected nil command for NewMeetingErrMsg")
	}

	if !errors.Is(updated.err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, updated.err)
	}
}

func TestNewMeetingModelView(t *testing.T) {
	t.Run("nil form", func(t *testing.T) {
		m := NewMeetingModel{}
		if got := m.View().Content; got != "Unable to render new meeting form" {
			t.Fatalf("expected nil-form message, got %q", got)
		}
	})

	t.Run("form with error and cancelled", func(t *testing.T) {
		m := NewNewMeetingModel(storage.NewStore(t.TempDir()), 80, 24)
		m.err = errors.New("submit failed")
		m.cancelled = true

		view := m.View().Content
		if !strings.Contains(view, "Error: submit failed") {
			t.Fatalf("expected error text in view, got %q", view)
		}

		if !strings.Contains(view, "Cancelled") {
			t.Fatalf("expected cancelled text in view, got %q", view)
		}
	})
}

func TestSubmitMeetingCmdNilStoreReturnsErrorMsg(t *testing.T) {
	m := NewMeetingModel{
		values: &newMeetingFormValues{},
	}

	msg := m.submitMeetingCmd()()
	errMsg, ok := msg.(NewMeetingErrMsg)
	if !ok {
		t.Fatalf("expected NewMeetingErrMsg, got %T", msg)
	}

	if errMsg.Err == nil || errMsg.Err.Error() != "store is not configured" {
		t.Fatalf("expected store-not-configured error, got %v", errMsg.Err)
	}
}

func TestSubmitMeetingCmdCreatesMeetingAndNewProject(t *testing.T) {
	store := storage.NewStore(t.TempDir())
	m := NewMeetingModel{
		store: store,
		values: &newMeetingFormValues{
			project:        newProjectOptionValue,
			newProjectName: "Infrastructure",
			newProjectPref: "infra",
			title:          "Weekly Infra",
			date:           "2026-04-16",
			participants:   " Alice, Bob , , Charlie ",
			tags:           " ops, backend , , infra ",
			externalTicket: "  https://jira.example/TST-123  ",
		},
	}

	msg := m.submitMeetingCmd()()
	created, ok := msg.(MeetingCreatedMsg)
	if !ok {
		t.Fatalf("expected MeetingCreatedMsg, got %T", msg)
	}

	if created.Meeting == nil {
		t.Fatalf("expected created meeting to be non-nil")
	}

	if created.Meeting.Project != "INFRA" {
		t.Fatalf("expected project INFRA, got %q", created.Meeting.Project)
	}

	if created.Meeting.TicketID != "INFRA-001" {
		t.Fatalf("expected ticket INFRA-001, got %q", created.Meeting.TicketID)
	}

	if got := created.Meeting.Participants; len(got) != 3 || got[0] != "Alice" || got[1] != "Bob" || got[2] != "Charlie" {
		t.Fatalf("expected trimmed participants [Alice Bob Charlie], got %v", got)
	}

	if got := created.Meeting.Tags; len(got) != 3 || got[0] != "ops" || got[1] != "backend" || got[2] != "infra" {
		t.Fatalf("expected trimmed tags [ops backend infra], got %v", got)
	}

	if created.Meeting.ExternalTicket != "https://jira.example/TST-123" {
		t.Fatalf("expected trimmed external ticket, got %q", created.Meeting.ExternalTicket)
	}

	project, err := store.GetProject("INFRA")
	if err != nil {
		t.Fatalf("expected created project INFRA to exist: %v", err)
	}

	if project.Name != "Infrastructure" {
		t.Fatalf("expected project name Infrastructure, got %q", project.Name)
	}
}

func TestSubmitMeetingCmdUsesExistingProject(t *testing.T) {
	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("Infrastructure", "INFRA"); err != nil {
		t.Fatalf("CreateProject INFRA: %v", err)
	}

	m := NewMeetingModel{
		store: store,
		values: &newMeetingFormValues{
			project:      "INFRA",
			title:        "Infra Sync",
			date:         "2026-04-17",
			participants: "Alice",
		},
	}

	msg := m.submitMeetingCmd()()
	created, ok := msg.(MeetingCreatedMsg)
	if !ok {
		t.Fatalf("expected MeetingCreatedMsg, got %T", msg)
	}

	if created.Meeting.TicketID != "INFRA-001" {
		t.Fatalf("expected ticket INFRA-001, got %q", created.Meeting.TicketID)
	}

	if created.Meeting.Project != "INFRA" {
		t.Fatalf("expected project INFRA, got %q", created.Meeting.Project)
	}
}

func TestSubmitMeetingCmdInvalidDateReturnsErrorMsg(t *testing.T) {
	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("Infrastructure", "INFRA"); err != nil {
		t.Fatalf("CreateProject INFRA: %v", err)
	}

	m := NewMeetingModel{
		store: store,
		values: &newMeetingFormValues{
			project: "INFRA",
			title:   "Bad Date",
			date:    "2026-99-99",
		},
	}

	msg := m.submitMeetingCmd()()
	errMsg, ok := msg.(NewMeetingErrMsg)
	if !ok {
		t.Fatalf("expected NewMeetingErrMsg, got %T", msg)
	}

	if errMsg.Err == nil || !strings.Contains(errMsg.Err.Error(), "invalid date") {
		t.Fatalf("expected invalid date error, got %v", errMsg.Err)
	}
}

func TestBuildNewMeetingFormOptionsAndStateTransitions(t *testing.T) {
	model := NewMeetingModel{
		values: &newMeetingFormValues{
			project: newProjectOptionValue,
		},
	}

	projects := []storage.Project{{Name: "Infrastructure", Prefix: "INFRA"}}
	form := buildNewMeetingForm(&model, projects)
	if form == nil {
		t.Fatalf("expected non-nil form")
	}

	model.form = form
	if model.form.State != huh.StateNormal {
		t.Fatalf("expected initial form state %v, got %v", huh.StateNormal, model.form.State)
	}

	model.values.project = "INFRA"
	updated, _ := updateNewMeetingModel(t, model, struct{}{})
	if updated.creatingProject {
		t.Fatalf("expected creatingProject=false for existing project selection")
	}

	updated.values.project = newProjectOptionValue
	updated.values.newProjectName = "  New Name  "
	updated.values.newProjectPref = " prj "
	updated2, _ := updateNewMeetingModel(t, updated, struct{}{})
	if !updated2.creatingProject {
		t.Fatalf("expected creatingProject=true for new project selection")
	}

	if updated2.newProjectName != "New Name" {
		t.Fatalf("expected trimmed newProjectName %q, got %q", "New Name", updated2.newProjectName)
	}

	if updated2.newProjectPrefix != "PRJ" {
		t.Fatalf("expected uppercased newProjectPrefix %q, got %q", "PRJ", updated2.newProjectPrefix)
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "empty", input: "", want: []string{}},
		{name: "single", input: "alice", want: []string{"alice"}},
		{name: "multiple with whitespace", input: " alice, bob ,carol ", want: []string{"alice", "bob", "carol"}},
		{name: "skips empty chunks", input: "a,, ,b,", want: []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitAndTrim(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("expected %d items, got %d (%v)", len(tt.want), len(got), got)
			}

			for idx := range tt.want {
				if got[idx] != tt.want[idx] {
					t.Fatalf("expected item %d to be %q, got %q", idx, tt.want[idx], got[idx])
				}
			}
		})
	}
}
