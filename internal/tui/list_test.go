package tui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/storage"
)

var (
	_ = os.ModePerm
	_ = filepath.Clean
)

func keyMsg(r string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: r}
}

func keyEnter() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyEnter}
}

func updateListModel(t *testing.T, m ListModel, msg tea.Msg) (ListModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	listModel, ok := updated.(ListModel)
	if !ok {
		t.Fatalf("expected ListModel from Update, got %T", updated)
	}

	return listModel, cmd
}

func createMeetingForTest(
	t *testing.T,
	store *storage.Store,
	projectName, projectPrefix, title string,
	date time.Time,
	tags []string,
) *storage.Meeting {
	t.Helper()

	_, err := store.CreateProject(projectName, projectPrefix)
	if err != nil && !errors.Is(err, storage.ErrPrefixExists) {
		t.Fatalf("CreateProject(%s): %v", projectPrefix, err)
	}

	meeting, err := store.CreateMeeting(title, date, []string{"alice"}, projectPrefix, tags, "")
	if err != nil {
		t.Fatalf("CreateMeeting(%s): %v", title, err)
	}

	return meeting
}

func TestMeetingItemDisplayAndFilterValue(t *testing.T) {
	meeting := storage.Meeting{
		Title:    "Sprint Planning",
		TicketID: "INFRA-003",
		Date:     time.Date(2026, 4, 16, 9, 30, 0, 0, time.UTC),
		Project:  "INFRA",
		Status:   storage.MeetingStatusDraft,
		Tags:     []string{"ops", "planning"},
	}

	item := MeetingItem{meeting: meeting}

	if got := item.Title(); got != "[INFRA-003] Sprint Planning" {
		t.Fatalf("Title(): expected %q, got %q", "[INFRA-003] Sprint Planning", got)
	}

	if got := item.Description(); got != "2026-04-16 | INFRA | ○ DRAFT" {
		t.Fatalf("Description() draft: expected %q, got %q", "2026-04-16 | INFRA | ○ DRAFT", got)
	}

	if got := item.FilterValue(); got != "Sprint Planning INFRA-003 ops planning" {
		t.Fatalf("FilterValue(): expected %q, got %q", "Sprint Planning INFRA-003 ops planning", got)
	}

	meeting.Status = storage.MeetingStatusStructured
	item = MeetingItem{meeting: meeting}
	if got := item.Description(); got != "2026-04-16 | INFRA | ✓ STRUCTURED" {
		t.Fatalf("Description() structured: expected %q, got %q", "2026-04-16 | INFRA | ✓ STRUCTURED", got)
	}
}

func TestNewListModelLoadsMeetingsAndEnablesFiltering(t *testing.T) {
	store := storage.NewStore(t.TempDir())
	meeting := createMeetingForTest(t, store, "Infrastructure", "INFRA", "Weekly Infra", time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC), []string{"ops"})

	m := NewListModel(store, 80, 20)

	if m.store != store {
		t.Fatalf("expected model store pointer to match input store")
	}

	if m.width != 80 || m.height != 20 {
		t.Fatalf("expected width=80 height=20, got width=%d height=%d", m.width, m.height)
	}

	if m.list.Title != "Meeting Notes" {
		t.Fatalf("expected list title %q, got %q", "Meeting Notes", m.list.Title)
	}

	if !m.list.FilteringEnabled() {
		t.Fatalf("expected filtering to be enabled")
	}

	if len(m.allMeetings) != 1 {
		t.Fatalf("expected allMeetings length 1, got %d", len(m.allMeetings))
	}

	if len(m.list.Items()) != 1 {
		t.Fatalf("expected list items length 1, got %d", len(m.list.Items()))
	}

	item, ok := m.list.Items()[0].(MeetingItem)
	if !ok {
		t.Fatalf("expected first list item type MeetingItem, got %T", m.list.Items()[0])
	}

	if item.meeting.ID != meeting.ID {
		t.Fatalf("expected first item meeting ID %q, got %q", meeting.ID, item.meeting.ID)
	}
}

func TestListModelInitReturnsNil(t *testing.T) {
	m := NewListModel(nil, 60, 10)
	if cmd := m.Init(); cmd != nil {
		t.Fatalf("expected Init to return nil cmd")
	}
}

func TestViewShowsEmptyStateWhenNoMeetingsAndNoFilters(t *testing.T) {
	store := storage.NewStore(t.TempDir())
	m := NewListModel(store, 80, 20)

	if got := m.View().Content; !strings.Contains(got, "No meetings yet. Press 'n' to create one.") {
		t.Fatalf("expected empty state in view, got %q", got)
	}
}

func TestUpdateInBuiltInFilterModePassesKeysToList(t *testing.T) {
	store := storage.NewStore(t.TempDir())
	createMeetingForTest(t, store, "Infrastructure", "INFRA", "Filter Test", time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC), []string{"ops"})

	m := NewListModel(store, 80, 20)
	m.list.SetFilterState(list.Filtering)

	updated, cmd := updateListModel(t, m, keyMsg("n"))

	if updated.list.FilterState() != list.Filtering {
		t.Fatalf("expected filter state to remain filtering, got %v", updated.list.FilterState())
	}

	if cmd != nil {
		if msg := cmd(); msg != nil {
			if _, isNavigate := msg.(NavigateMsg); isNavigate {
				t.Fatalf("expected built-in filtering mode to intercept key and not emit NavigateMsg")
			}
		}
	}
}

func TestUpdateKeyNReturnsNavigateMsg(t *testing.T) {
	m := NewListModel(nil, 80, 20)

	_, cmd := updateListModel(t, m, keyMsg("n"))
	if cmd == nil {
		t.Fatalf("expected non-nil command for 'n' key")
	}

	msg := cmd()
	nav, ok := msg.(NavigateMsg)
	if !ok {
		t.Fatalf("expected NavigateMsg, got %T", msg)
	}

	if nav.Screen != ScreenNewMeeting {
		t.Fatalf("expected navigation to ScreenNewMeeting, got %v", nav.Screen)
	}
}

func TestUpdateEnterEmitsMeetingSelectedMsg(t *testing.T) {
	store := storage.NewStore(t.TempDir())
	meeting := createMeetingForTest(t, store, "Infrastructure", "INFRA", "Select Me", time.Date(2026, 4, 12, 14, 0, 0, 0, time.UTC), []string{"ops"})

	m := NewListModel(store, 80, 20)
	m.list.Select(0)

	updated, cmd := updateListModel(t, m, keyEnter())
	if cmd == nil {
		t.Fatalf("expected non-nil command for enter key")
	}

	msg := cmd()
	selected, ok := msg.(MeetingSelectedMsg)
	if !ok {
		t.Fatalf("expected MeetingSelectedMsg, got %T", msg)
	}

	if selected.Meeting.ID != meeting.ID {
		t.Fatalf("expected selected meeting ID %q, got %q", meeting.ID, selected.Meeting.ID)
	}

	if len(updated.list.Items()) != 1 {
		t.Fatalf("expected one list item after enter, got %d", len(updated.list.Items()))
	}
}

func TestDeleteKeyEmitsRequestDeleteMsg(t *testing.T) {
	store := storage.NewStore(t.TempDir())
	meeting := createMeetingForTest(t, store, "Infrastructure", "INFRA", "Delete Candidate", time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC), []string{"ops"})

	m := NewListModel(store, 80, 20)
	m.list.Select(0)

	_, cmd := updateListModel(t, m, keyMsg("d"))
	if cmd == nil {
		t.Fatalf("expected non-nil command for 'd' key")
	}

	msg := cmd()
	requestMsg, ok := msg.(RequestDeleteMsg)
	if !ok {
		t.Fatalf("expected RequestDeleteMsg, got %T", msg)
	}

	if requestMsg.Meeting.ID != meeting.ID {
		t.Fatalf("expected requested meeting ID %q, got %q", meeting.ID, requestMsg.Meeting.ID)
	}
}

func TestUpdateFilterKeysCycleProjectAndTag(t *testing.T) {
	store := storage.NewStore(t.TempDir())
	createMeetingForTest(t, store, "Infrastructure", "INFRA", "Infra Alpha", time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC), []string{"alpha", "ops"})
	createMeetingForTest(t, store, "Application", "APP", "App Beta", time.Date(2026, 4, 11, 9, 0, 0, 0, time.UTC), []string{"beta"})

	m := NewListModel(store, 80, 20)
	if len(m.list.Items()) != 2 {
		t.Fatalf("expected two initial items, got %d", len(m.list.Items()))
	}

	updated, cmd := updateListModel(t, m, keyMsg("f"))
	if cmd != nil {
		_ = cmd()
	}

	if updated.projectFilter != "APP" {
		t.Fatalf("expected first project filter to be APP, got %q", updated.projectFilter)
	}

	if len(updated.list.Items()) != 1 {
		t.Fatalf("expected one item after project filter, got %d", len(updated.list.Items()))
	}

	updated, cmd = updateListModel(t, updated, keyMsg("t"))
	if cmd != nil {
		_ = cmd()
	}

	if updated.tagFilter != "alpha" {
		t.Fatalf("expected first tag filter to be alpha, got %q", updated.tagFilter)
	}

	if len(updated.list.Items()) != 0 {
		t.Fatalf("expected zero items for APP + alpha filter combination, got %d", len(updated.list.Items()))
	}
}

func TestRefreshMeetingsAppliesProjectAndTagFilters(t *testing.T) {
	store := storage.NewStore(t.TempDir())
	createMeetingForTest(t, store, "Infrastructure", "INFRA", "Infra Ops", time.Date(2026, 4, 10, 9, 0, 0, 0, time.UTC), []string{"ops", "alpha"})
	createMeetingForTest(t, store, "Infrastructure", "INFRA", "Infra Security", time.Date(2026, 4, 11, 9, 0, 0, 0, time.UTC), []string{"security"})
	createMeetingForTest(t, store, "Application", "APP", "App Ops", time.Date(2026, 4, 12, 9, 0, 0, 0, time.UTC), []string{"ops"})

	m := NewListModel(store, 80, 20)
	m.projectFilter = "INFRA"
	m.tagFilter = "ops"

	cmd := m.RefreshMeetings()
	if cmd != nil {
		_ = cmd()
	}

	if len(m.allMeetings) != 3 {
		t.Fatalf("expected allMeetings to contain 3 meetings, got %d", len(m.allMeetings))
	}

	if len(m.list.Items()) != 1 {
		t.Fatalf("expected exactly one item after INFRA+ops filtering, got %d", len(m.list.Items()))
	}

	onlyItem, ok := m.list.Items()[0].(MeetingItem)
	if !ok {
		t.Fatalf("expected filtered item type MeetingItem, got %T", m.list.Items()[0])
	}

	if onlyItem.meeting.Project != "INFRA" {
		t.Fatalf("expected filtered meeting project INFRA, got %q", onlyItem.meeting.Project)
	}

	if !meetingHasTag(onlyItem.meeting, "ops") {
		t.Fatalf("expected filtered meeting to have ops tag")
	}
}

func TestRefreshMeetingsNilStoreClearsItems(t *testing.T) {
	m := NewListModel(nil, 80, 20)

	cmd := m.RefreshMeetings()
	if cmd != nil {
		_ = cmd()
	}

	if len(m.allMeetings) != 0 {
		t.Fatalf("expected allMeetings to be empty with nil store, got %d", len(m.allMeetings))
	}

	if len(m.list.Items()) != 0 {
		t.Fatalf("expected list items to be empty with nil store, got %d", len(m.list.Items()))
	}
}

func TestCycleFilterValueSequence(t *testing.T) {
	values := []string{"APP", "INFRA"}

	if got := cycleFilterValue("", values); got != "APP" {
		t.Fatalf("expected first cycle from empty to APP, got %q", got)
	}

	if got := cycleFilterValue("APP", values); got != "INFRA" {
		t.Fatalf("expected second cycle to INFRA, got %q", got)
	}

	if got := cycleFilterValue("INFRA", values); got != "" {
		t.Fatalf("expected cycle past last to empty string, got %q", got)
	}

	if got := cycleFilterValue("UNKNOWN", values); got != "APP" {
		t.Fatalf("expected unknown current value to reset to first option APP, got %q", got)
	}

	if got := cycleFilterValue("anything", nil); got != "" {
		t.Fatalf("expected empty cycle values to return empty string, got %q", got)
	}
}

func TestCycleFilterValuePropertyRoundTrip(t *testing.T) {
	prop := func(size uint8) bool {
		n := int(size%10) + 1
		values := make([]string, n)
		for i := 0; i < n; i++ {
			values[i] = fmt.Sprintf("v%d", i)
		}

		current := ""
		for i := 0; i < n+1; i++ {
			current = cycleFilterValue(current, values)
		}

		return current == ""
	}

	if err := quick.Check(prop, nil); err != nil {
		t.Fatalf("cycleFilterValue round-trip property failed: %v", err)
	}
}

func TestUniqueProjectsFromMeetingsSortedDeduplicated(t *testing.T) {
	meetings := []storage.Meeting{
		{Project: "INFRA"},
		{Project: "APP"},
		{Project: "INFRA"},
		{Project: ""},
	}

	got := uniqueProjectsFromMeetings(meetings)
	want := []string{"APP", "INFRA"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uniqueProjectsFromMeetings(): expected %v, got %v", want, got)
	}

	if !sort.StringsAreSorted(got) {
		t.Fatalf("expected projects to be sorted, got %v", got)
	}
}

func TestUniqueTagsFromMeetingsSortedDeduplicatedTrimmed(t *testing.T) {
	meetings := []storage.Meeting{
		{Tags: []string{"ops", " alpha "}},
		{Tags: []string{"ops", "security", ""}},
		{Tags: []string{"  "}},
	}

	got := uniqueTagsFromMeetings(meetings)
	want := []string{"alpha", "ops", "security"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("uniqueTagsFromMeetings(): expected %v, got %v", want, got)
	}

	if !sort.StringsAreSorted(got) {
		t.Fatalf("expected tags to be sorted, got %v", got)
	}
}

func TestMeetingHasTagTrimsWhitespace(t *testing.T) {
	meeting := storage.Meeting{Tags: []string{"ops", " alpha "}}

	if !meetingHasTag(meeting, "alpha") {
		t.Fatalf("expected meetingHasTag to match trimmed tag alpha")
	}

	if meetingHasTag(meeting, "beta") {
		t.Fatalf("expected meetingHasTag to be false for missing tag beta")
	}
}

func TestMeetingsToItemsPreservesMeetingContent(t *testing.T) {
	meetings := []storage.Meeting{
		{ID: "a", Title: "One", Project: "INFRA"},
		{ID: "b", Title: "Two", Project: "APP"},
	}

	items := meetingsToItems(meetings)
	if len(items) != 2 {
		t.Fatalf("expected two items, got %d", len(items))
	}

	first, ok := items[0].(MeetingItem)
	if !ok {
		t.Fatalf("expected first converted item type MeetingItem, got %T", items[0])
	}

	if first.meeting.ID != "a" || first.meeting.Title != "One" {
		t.Fatalf("expected first item meeting ID/title a/One, got %s/%s", first.meeting.ID, first.meeting.Title)
	}
}
