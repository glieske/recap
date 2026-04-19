package tui

import (
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/glieske/recap/internal/storage"
)

func createEditorTestStoreAndMeeting(t *testing.T) (*storage.Store, *storage.Meeting) {
	t.Helper()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("Test Project", "TEST"); err != nil {
		t.Fatalf("CreateProject(TEST): %v", err)
	}

	meeting, err := store.CreateMeeting("Editor Meeting", time.Now(), nil, "TEST", nil, "")
	if err != nil {
		t.Fatalf("CreateMeeting: %v", err)
	}

	return store, meeting
}

func updateEditorModelForTest(t *testing.T, m EditorModel, msg tea.Msg) (EditorModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	updatedModel, ok := updated.(EditorModel)
	if !ok {
		t.Fatalf("expected EditorModel from Update, got %T", updated)
	}

	return updatedModel, cmd
}

func TestEditorNewEditorModelNilInputs(t *testing.T) {
	m := NewEditorModel(nil, nil, 80, 20, "", "", "")

	if m.meeting != nil {
		t.Fatalf("expected meeting=nil")
	}

	if m.store != nil {
		t.Fatalf("expected store=nil")
	}

	if m.width != 80 || m.height != 20 {
		t.Fatalf("expected width=80 height=20, got width=%d height=%d", m.width, m.height)
	}

	if m.dirty {
		t.Fatalf("expected dirty=false on construction")
	}
}

func TestEditorNewEditorModelNilMeetingOrStore(t *testing.T) {
	store, meeting := createEditorTestStoreAndMeeting(t)

	withNilMeeting := NewEditorModel(nil, store, 70, 10, "", "", "")
	if withNilMeeting.meeting != nil {
		t.Fatalf("expected nil meeting when input meeting is nil")
	}

	withNilStore := NewEditorModel(meeting, nil, 70, 10, "", "", "")
	if withNilStore.store != nil {
		t.Fatalf("expected nil store when input store is nil")
	}
}

func TestEditorNewEditorModelLoadsExistingRawNotes(t *testing.T) {
	store, meeting := createEditorTestStoreAndMeeting(t)
	if err := store.SaveRawNotes(meeting.Project, meeting.ID, "existing raw notes"); err != nil {
		t.Fatalf("SaveRawNotes: %v", err)
	}

	m := NewEditorModel(meeting, store, 80, 20, "", "", "")

	if got := m.textarea.Value(); got != "existing raw notes" {
		t.Fatalf("expected textarea value %q, got %q", "existing raw notes", got)
	}
}

func TestEditorInitReturnsNonNilCommand(t *testing.T) {
	m := NewEditorModel(nil, nil, 80, 20, "", "", "")

	if cmd := m.Init(); cmd == nil {
		t.Fatalf("expected non-nil init command")
	}
}

func TestEditorUpdateWindowSizeMsgResizesEditor(t *testing.T) {
	m := NewEditorModel(nil, nil, 40, 8, "", "", "")

	updated, cmd := updateEditorModelForTest(t, m, tea.WindowSizeMsg{Width: 120, Height: 42})
	if cmd != nil {
		t.Fatalf("expected nil command for window size update")
	}

	if updated.width != 120 || updated.height != 42 {
		t.Fatalf("expected width=120 height=42, got width=%d height=%d", updated.width, updated.height)
	}
}

func TestEditorUpdateCtrlTInsertsTimestampAndMarksDirty(t *testing.T) {
	m := NewEditorModel(nil, nil, 80, 20, "", "", "")

	updated, cmd := updateEditorModelForTest(t, m, tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl})
	if cmd != nil {
		t.Fatalf("expected nil command for ctrl+t")
	}

	if !updated.dirty {
		t.Fatalf("expected dirty=true after ctrl+t")
	}

	value := updated.textarea.Value()
	if !regexp.MustCompile(`^\[\d{2}:\d{2}\]\s$`).MatchString(value) {
		t.Fatalf("expected timestamp format [HH:MM] , got %q", value)
	}
}

func TestEditorUpdateCtrlSReturnsSaveCmdAndClearsDirty(t *testing.T) {
	store, meeting := createEditorTestStoreAndMeeting(t)
	m := NewEditorModel(meeting, store, 80, 20, "", "", "")
	m.textarea.SetValue("save this")
	m.dirty = true

	updated, cmd := updateEditorModelForTest(t, m, tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatalf("expected non-nil save command for ctrl+s")
	}

	if updated.dirty {
		t.Fatalf("expected dirty=false immediately after ctrl+s")
	}

	msg := cmd()
	if _, ok := msg.(SaveDoneMsg); !ok {
		t.Fatalf("expected SaveDoneMsg from ctrl+s save command, got %T", msg)
	}
}

func TestEditorUpdateCtrlAEMessagesAndTogglePreview(t *testing.T) {
	cases := []struct {
		name    string
		msg     tea.KeyPressMsg
		expectT any
	}{
		{name: "ctrl+a", msg: tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl}, expectT: TriggerAIMsg{}},
		{name: "ctrl+e", msg: tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl}, expectT: TriggerEmailMsg{}},
		{name: "ctrl+p", msg: tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl}, expectT: TogglePreviewMsg{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewEditorModel(nil, nil, 80, 20, "", "", "")
			updated, cmd := updateEditorModelForTest(t, m, tc.msg)
			if cmd == nil {
				t.Fatalf("expected non-nil command for %s", tc.name)
			}

			if updated.dirty {
				t.Fatalf("expected dirty to remain false for %s", tc.name)
			}

			msg := cmd()
			switch tc.expectT.(type) {
			case TriggerAIMsg:
				if _, ok := msg.(TriggerAIMsg); !ok {
					t.Fatalf("expected TriggerAIMsg, got %T", msg)
				}
			case TriggerEmailMsg:
				if _, ok := msg.(TriggerEmailMsg); !ok {
					t.Fatalf("expected TriggerEmailMsg, got %T", msg)
				}
			case TogglePreviewMsg:
				if _, ok := msg.(TogglePreviewMsg); !ok {
					t.Fatalf("expected TogglePreviewMsg, got %T", msg)
				}
			}
		})
	}
}

func TestEditorUpdateAutoSaveTickDirtyAndCleanBranches(t *testing.T) {
	store, meeting := createEditorTestStoreAndMeeting(t)

	t.Run("dirty triggers save path", func(t *testing.T) {
		m := NewEditorModel(meeting, store, 80, 20, "", "", "")
		m.textarea.SetValue("autosave content")
		m.dirty = true

		updated, cmd := updateEditorModelForTest(t, m, AutoSaveTickMsg{})
		if cmd == nil {
			t.Fatalf("expected non-nil command for dirty autosave tick")
		}

		if updated.dirty {
			t.Fatalf("expected dirty=false after dirty autosave tick")
		}
	})

	t.Run("clean skips save and schedules next tick", func(t *testing.T) {
		m := NewEditorModel(meeting, store, 80, 20, "", "", "")
		m.dirty = false

		updated, cmd := updateEditorModelForTest(t, m, AutoSaveTickMsg{})
		if cmd == nil {
			t.Fatalf("expected non-nil tick command when clean")
		}

		if updated.dirty {
			t.Fatalf("expected dirty=false to remain unchanged for clean autosave tick")
		}
	})
}

func TestEditorUpdateSaveDoneMsgSetsStatusAndLastSaved(t *testing.T) {
	m := NewEditorModel(nil, nil, 80, 20, "", "", "")
	before := m.lastSaved

	updated, cmd := updateEditorModelForTest(t, m, SaveDoneMsg{})
	if cmd != nil {
		t.Fatalf("expected nil command for SaveDoneMsg")
	}

	if updated.statusMsg != "Saved" {
		t.Fatalf("expected status message %q, got %q", "Saved", updated.statusMsg)
	}

	if !updated.lastSaved.After(before) && !updated.lastSaved.Equal(before) {
		t.Fatalf("expected lastSaved to be updated, before=%v after=%v", before, updated.lastSaved)
	}

	if time.Until(updated.statusExpiry) <= 0 {
		t.Fatalf("expected statusExpiry to be in the future")
	}
}

func TestEditorUpdateSaveErrMsgSetsErrorAndDirty(t *testing.T) {
	m := NewEditorModel(nil, nil, 80, 20, "", "", "")
	m.dirty = false

	errMsg := SaveErrMsg{Err: errors.New("disk full")}

	updated, cmd := updateEditorModelForTest(t, m, errMsg)
	if cmd != nil {
		t.Fatalf("expected nil command for SaveErrMsg")
	}

	if !updated.dirty {
		t.Fatalf("expected dirty=true after SaveErrMsg")
	}

	if updated.err == nil {
		t.Fatalf("expected model error to be set")
	}

	if !strings.Contains(updated.statusMsg, "Save error:") {
		t.Fatalf("expected status to contain save error prefix, got %q", updated.statusMsg)
	}
}

func TestEditorSaveCmdNilStoreOrMeetingReturnsSaveErrMsg(t *testing.T) {
	m := NewEditorModel(nil, nil, 80, 20, "", "", "")
	msg := m.saveCmd()()
	errMsg, ok := msg.(SaveErrMsg)
	if !ok {
		t.Fatalf("expected SaveErrMsg, got %T", msg)
	}

	if errMsg.Err == nil || errMsg.Err.Error() != "editor not configured" {
		t.Fatalf("expected editor not configured error, got %v", errMsg.Err)
	}
}

func TestEditorSaveCmdValidStoreSavesRawNotesAndReturnsDone(t *testing.T) {
	store, meeting := createEditorTestStoreAndMeeting(t)
	m := NewEditorModel(meeting, store, 80, 20, "", "", "")
	m.textarea.SetValue("persisted text")

	msg := m.saveCmd()()
	if _, ok := msg.(SaveDoneMsg); !ok {
		t.Fatalf("expected SaveDoneMsg, got %T", msg)
	}

	raw, err := store.LoadRawNotes(meeting.Project, meeting.ID)
	if err != nil {
		t.Fatalf("LoadRawNotes: %v", err)
	}

	if raw != "persisted text" {
		t.Fatalf("expected saved raw notes %q, got %q", "persisted text", raw)
	}
}

func TestEditorViewContainsTextareaAndStatusBar(t *testing.T) {
	store, meeting := createEditorTestStoreAndMeeting(t)
	m := NewEditorModel(meeting, store, 80, 20, "", "", "")
	m.textarea.SetValue("editor body")
	m.statusMsg = "Saved"
	m.statusExpiry = time.Now().Add(2 * time.Second)

	view := m.View().Content
	if !strings.Contains(view, "editor body") {
		t.Fatalf("expected textarea content in view, got %q", view)
	}

	if !strings.Contains(view, meeting.TicketID) || !strings.Contains(view, meeting.Title) {
		t.Fatalf("expected status bar to include ticket/title, got %q", view)
	}

	if !strings.Contains(view, "Saved") {
		t.Fatalf("expected status message in view, got %q", view)
	}
}

func TestEditorRenderStatusBarShowsDirtyAndCharCount(t *testing.T) {
	m := NewEditorModel(nil, nil, 40, 10, "", "", "")
	m.textarea.SetValue("abcd")
	m.dirty = true

	bar := m.renderStatusBar()
	if !strings.Contains(bar, "4 chars") {
		t.Fatalf("expected char count in status bar, got %q", bar)
	}

	if !strings.Contains(bar, "•") {
		t.Fatalf("expected dirty marker in status bar, got %q", bar)
	}
}

func TestRenderStatusBarStylingBoldAndHighContrast(t *testing.T) {
	input := textarea.New()
	input.SetValue("hello")

	m := EditorModel{
		meeting:   &storage.Meeting{TicketID: "TEST-001", Title: "Test Meeting"},
		textarea:  input,
		width:     80,
		dirty:     true,
		startTime: time.Now(),
	}

	bar := m.renderStatusBar()
	if bar == "" {
		t.Fatalf("expected non-empty status bar output")
	}

	if !strings.Contains(bar, "TEST-001") {
		t.Fatalf("expected ticket id in status bar, got %q", bar)
	}

	if !strings.Contains(bar, "Test Meeting") {
		t.Fatalf("expected title in status bar, got %q", bar)
	}

	if !strings.Contains(bar, "5 chars") {
		t.Fatalf("expected char count in status bar, got %q", bar)
	}

	if len([]rune(ansi.Strip(bar))) != m.width {
		t.Fatalf("expected rendered bar width %d, got %d", m.width, len([]rune(bar)))
	}
}

func TestEditorRenderStatusBarProviderDisplayEmpty(t *testing.T) {
	m := NewEditorModel(nil, nil, 80, 10, "", "", "")
	m.textarea.SetValue("abcd")
	m.dirty = true

	bar := m.renderStatusBar()
	if strings.Contains(bar, " / ") {
		t.Fatalf("expected no provider separator when provider fields are empty, got %q", bar)
	}

	if strings.Contains(bar, " | 4 chars") {
		t.Fatalf("expected no provider prefix before char count when provider fields are empty, got %q", bar)
	}

	if !strings.Contains(bar, "4 chars •") {
		t.Fatalf("expected char count and dirty marker %q in status bar, got %q", "4 chars •", bar)
	}
}

func TestEditorRenderStatusBarProviderDisplayNameOnly(t *testing.T) {
	m := NewEditorModel(nil, nil, 90, 10, "GitHub Models", "", "")
	m.textarea.SetValue("abcde")
	m.dirty = true

	bar := m.renderStatusBar()
	if !strings.Contains(bar, "GitHub Models | 5 chars •") {
		t.Fatalf("expected provider name-only prefix with char count, got %q", bar)
	}

	if strings.Contains(bar, "GitHub Models / ") {
		t.Fatalf("expected no slash model separator for name-only provider, got %q", bar)
	}
}

func TestEditorRenderStatusBarProviderDisplayNameAndModel(t *testing.T) {
	m := NewEditorModel(nil, nil, 110, 10, "OpenAI", "gpt-4.1", "")
	m.textarea.SetValue("abcdef")
	m.dirty = true

	bar := m.renderStatusBar()
	if !strings.Contains(bar, "OpenAI / gpt-4.1 | 6 chars •") {
		t.Fatalf("expected provider name/model prefix with char count, got %q", bar)
	}
}

func TestFormatElapsed(t *testing.T) {
	if got := formatElapsed(0); got != "00:00:00" {
		t.Fatalf("0s: expected %q, got %q", "00:00:00", got)
	}

	if got := formatElapsed(65 * time.Second); got != "00:01:05" {
		t.Fatalf("65s: expected %q, got %q", "00:01:05", got)
	}

	if got := formatElapsed(3661 * time.Second); got != "01:01:01" {
		t.Fatalf("3661s: expected %q, got %q", "01:01:01", got)
	}
}

func TestComposeStatusLine(t *testing.T) {
	t.Run("normal width", func(t *testing.T) {
		got := composeStatusLine("L", "C", "R", 9)
		if got != "L   C   R" {
			t.Fatalf("expected %q, got %q", "L   C   R", got)
		}
	})

	t.Run("zero width", func(t *testing.T) {
		got := composeStatusLine("left", "center", "right", 0)
		if got != "" {
			t.Fatalf("expected empty string for zero width, got %q", got)
		}
	})

	t.Run("narrow width truncates", func(t *testing.T) {
		got := composeStatusLine("LEFT", "MID", "RIGHT", 5)
		if got != "LEFT " {
			t.Fatalf("expected truncated line %q, got %q", "LEFT ", got)
		}
	})

	t.Run("unicode chars", func(t *testing.T) {
		got := composeStatusLine("Ł", "中", "🚀", 7)
		if got != "Ł  中  🚀" {
			t.Fatalf("expected unicode-safe line %q, got %q", "Ł  中  🚀", got)
		}
	})
}

func TestMaxEditorHeight(t *testing.T) {
	if got := maxEditorHeight(0); got != 1 {
		t.Fatalf("height=0: expected 1, got %d", got)
	}

	if got := maxEditorHeight(3); got != 1 {
		t.Fatalf("height=3: expected 1, got %d", got)
	}

	if got := maxEditorHeight(10); got != 6 {
		t.Fatalf("height=10: expected 6, got %d", got)
	}
}

func TestEditorLegendRenderNormalWidthContainsShortcuts(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 20, "", "", "")

	legend := m.renderLegend()
	if !strings.Contains(legend, "ctrl+s") {
		t.Fatalf("expected legend to contain %q, got %q", "ctrl+s", legend)
	}

	if !strings.Contains(legend, "ctrl+/ Help") {
		t.Fatalf("expected legend to contain %q, got %q", "ctrl+/ Help", legend)
	}
}

func TestEditorLegendRenderNarrowWidthTruncatesWithEllipsis(t *testing.T) {
	m := NewEditorModel(nil, nil, 20, 20, "", "", "")

	legend := m.renderLegend()
	if len([]rune(ansi.Strip(legend))) != 20 {
		t.Fatalf("expected rendered legend width %d, got %d", 20, len([]rune(legend)))
	}

	if !strings.HasSuffix(ansi.Strip(legend), "…") {
		t.Fatalf("expected truncated legend to end with ellipsis, got %q", legend)
	}
}

func TestEditorLegendRenderWidthOneReturnsOnlyEllipsis(t *testing.T) {
	m := NewEditorModel(nil, nil, 1, 20, "", "", "")

	legend := m.renderLegend()
	if ansi.Strip(legend) != "…" {
		t.Fatalf("expected width=1 legend to be %q, got %q", "…", legend)
	}
}

func TestEditorLegendRenderWidthGuardBelowOneUsesOne(t *testing.T) {
	m := NewEditorModel(nil, nil, 0, 20, "", "", "")

	legend := m.renderLegend()
	if ansi.Strip(legend) != "…" {
		t.Fatalf("expected width<1 guard to render %q, got %q", "…", legend)
	}
}

func TestEditorLegendMaxEditorHeightAdjustedByFour(t *testing.T) {
	if got := maxEditorHeight(20); got != 16 {
		t.Fatalf("height=20: expected 16, got %d", got)
	}
}

func TestEditorLegendMaxEditorHeightSmallHeightsClampToOne(t *testing.T) {
	if got := maxEditorHeight(4); got != 1 {
		t.Fatalf("height=4: expected 1, got %d", got)
	}

	if got := maxEditorHeight(3); got != 1 {
		t.Fatalf("height=3: expected 1, got %d", got)
	}
}

func TestEditorLegendViewContainsLegendText(t *testing.T) {
	m := NewEditorModel(nil, nil, 120, 20, "", "", "")
	m.textarea.SetValue("body")

	view := m.View().Content
	if !strings.Contains(view, "ctrl+s Save") {
		t.Fatalf("expected View to include legend text %q, got %q", "ctrl+s Save", view)
	}

	if !strings.Contains(view, "ctrl+/ Help") {
		t.Fatalf("expected View to include legend text %q, got %q", "ctrl+/ Help", view)
	}
}
