package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func updateSummaryModelForTest(t *testing.T, m SummaryModel, msg tea.Msg) (SummaryModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	updatedModel, ok := updated.(SummaryModel)
	if !ok {
		t.Fatalf("expected SummaryModel from Update, got %T", updated)
	}

	return updatedModel, cmd
}

func TestSummaryNewSummaryModelEmptyContent(t *testing.T) {
	m := NewSummaryModel("", "PRJ", "meeting-1", nil, 80, 20)

	if m.width != 80 || m.height != 20 {
		t.Fatalf("expected width=80 height=20, got width=%d height=%d", m.width, m.height)
	}

	if got := m.Value(); got != "" {
		t.Fatalf("expected empty content, got %q", got)
	}

	if m.IsDirty() {
		t.Fatalf("expected dirty=false on construction")
	}

	if m.IsOverwritePending() {
		t.Fatalf("expected overwriteConfirmPending=false on construction")
	}

	if m.baselineContent != "" {
		t.Fatalf("expected baselineContent to match empty initial content, got %q", m.baselineContent)
	}
}

func TestSummaryNewSummaryModelWithContentAndDimensions(t *testing.T) {
	m := NewSummaryModel("existing summary", "PRJ", "meeting-1", nil, 72, 11)

	if got := m.Value(); got != "existing summary" {
		t.Fatalf("expected initial content %q, got %q", "existing summary", got)
	}

	if m.baselineContent != "existing summary" {
		t.Fatalf("expected baselineContent %q, got %q", "existing summary", m.baselineContent)
	}

	if m.width != 72 || m.height != 11 {
		t.Fatalf("expected width=72 height=11, got width=%d height=%d", m.width, m.height)
	}
}

func TestSummaryInitReturnsBlinkCmd(t *testing.T) {
	m := NewSummaryModel("", "PRJ", "meeting-1", nil, 80, 20)

	if cmd := m.Init(); cmd == nil {
		t.Fatalf("expected non-nil init command")
	}
}

func TestSummaryDirtyTrackingStartsFalseThenTrueOnFocusedInputAndResetsAfterSaveDone(t *testing.T) {
	m := NewSummaryModel("", "PRJ", "meeting-1", nil, 80, 20)

	if m.IsDirty() {
		t.Fatalf("expected initial dirty=false")
	}

	m.Focus()
	updated, _ := updateSummaryModelForTest(t, m, tea.KeyPressMsg{Text: "a"})

	if got := updated.Value(); got != "a" {
		t.Fatalf("expected content %q after typing, got %q", "a", got)
	}

	if !updated.IsDirty() {
		t.Fatalf("expected dirty=true after focused key input")
	}

	updated, cmd := updateSummaryModelForTest(t, updated, SummarySaveDoneMsg{})
	if cmd != nil {
		t.Fatalf("expected nil cmd for SummarySaveDoneMsg")
	}

	if updated.IsDirty() {
		t.Fatalf("expected dirty=false after SummarySaveDoneMsg")
	}

	if updated.baselineContent != "a" {
		t.Fatalf("expected baselineContent updated to saved content %q, got %q", "a", updated.baselineContent)
	}
}

func TestSummarySetContentNotDirtyReplacesContentWithoutPending(t *testing.T) {
	m := NewSummaryModel("old", "PRJ", "meeting-1", nil, 80, 20)
	m.SetContent("new")

	if got := m.Value(); got != "new" {
		t.Fatalf("expected replaced content %q, got %q", "new", got)
	}

	if m.IsDirty() {
		t.Fatalf("expected dirty=false after SetContent when clean")
	}

	if m.IsOverwritePending() {
		t.Fatalf("expected overwrite pending=false after direct replace")
	}
}

func TestSummarySetContentDirtyStartsPendingAndAcceptWithY(t *testing.T) {
	m := NewSummaryModel("baseline", "PRJ", "meeting-1", nil, 80, 20)
	m.Focus()
	updated, _ := updateSummaryModelForTest(t, m, tea.KeyPressMsg{Text: "x"})

	if !updated.IsDirty() {
		t.Fatalf("expected dirty=true after typing")
	}

	updated.SetContent("ai replacement")
	if !updated.IsOverwritePending() {
		t.Fatalf("expected overwrite pending=true when setting content while dirty")
	}

	if got := updated.Value(); got != "baselinex" {
		t.Fatalf("expected user-edited content retained before confirmation, got %q", got)
	}

	updatedAfterConfirm, cmd := updateSummaryModelForTest(t, updated, tea.KeyPressMsg{Text: "y"})
	if cmd != nil {
		t.Fatalf("expected nil command for overwrite y confirmation")
	}

	if got := updatedAfterConfirm.Value(); got != "ai replacement" {
		t.Fatalf("expected accepted overwrite content %q, got %q", "ai replacement", got)
	}

	if updatedAfterConfirm.IsDirty() {
		t.Fatalf("expected dirty=false after accepting overwrite")
	}

	if updatedAfterConfirm.IsOverwritePending() {
		t.Fatalf("expected overwrite pending=false after accepting overwrite")
	}
}

func TestSummarySetContentDirtyRejectWithNAndEscKeepsUserContent(t *testing.T) {
	t.Run("n rejects pending overwrite", func(t *testing.T) {
		m := NewSummaryModel("base", "PRJ", "meeting-1", nil, 80, 20)
		m.Focus()
		updated, _ := updateSummaryModelForTest(t, m, tea.KeyPressMsg{Text: "u"})
		updated.SetContent("new ai")

		if !updated.IsOverwritePending() {
			t.Fatalf("expected overwrite pending=true before rejection")
		}

		rejected, cmd := updateSummaryModelForTest(t, updated, tea.KeyPressMsg{Text: "n"})
		if cmd != nil {
			t.Fatalf("expected nil command for n rejection")
		}

		if got := rejected.Value(); got != "baseu" {
			t.Fatalf("expected user content preserved after n reject, got %q", got)
		}

		if rejected.IsOverwritePending() {
			t.Fatalf("expected overwrite pending=false after n rejection")
		}

		if !rejected.IsDirty() {
			t.Fatalf("expected dirty=true to remain after rejecting overwrite")
		}
	})

	t.Run("esc rejects pending overwrite", func(t *testing.T) {
		m := NewSummaryModel("base", "PRJ", "meeting-1", nil, 80, 20)
		m.Focus()
		updated, _ := updateSummaryModelForTest(t, m, tea.KeyPressMsg{Text: "u"})
		updated.SetContent("new ai")

		rejected, cmd := updateSummaryModelForTest(t, updated, tea.KeyPressMsg{Code: tea.KeyEscape})
		if cmd != nil {
			t.Fatalf("expected nil command for esc rejection")
		}

		if got := rejected.Value(); got != "baseu" {
			t.Fatalf("expected user content preserved after esc reject, got %q", got)
		}

		if rejected.IsOverwritePending() {
			t.Fatalf("expected overwrite pending=false after esc rejection")
		}
	})
}

func TestSummaryViewShowsHeaderDirtyDotAndOverwritePrompt(t *testing.T) {
	m := NewSummaryModel("", "PRJ", "meeting-1", nil, 80, 20)
	m.Focus()
	updated, _ := updateSummaryModelForTest(t, m, tea.KeyPressMsg{Text: "a"})
	updated.SetContent("incoming ai")

	view := updated.View().Content
	if !strings.Contains(view, "AI Summary") {
		t.Fatalf("expected view header to contain %q, got %q", "AI Summary", view)
	}

	if !strings.Contains(view, "•") {
		t.Fatalf("expected dirty indicator bullet in view, got %q", view)
	}

	if !strings.Contains(view, "y=accept, n=keep yours") {
		t.Fatalf("expected overwrite confirmation prompt in view, got %q", view)
	}
}

func TestSummarySaveCmdReturnsErrorWithNilStore(t *testing.T) {
	t.Run("configured identifiers but nil store", func(t *testing.T) {
		m := NewSummaryModel("content", "PRJ", "meeting-1", nil, 80, 20)
		msg := m.saveCmd()()

		errMsg, ok := msg.(SummarySaveErrMsg)
		if !ok {
			t.Fatalf("expected SummarySaveErrMsg, got %T", msg)
		}

		if errMsg.Err == nil {
			t.Fatalf("expected non-nil save error")
		}

		if got := errMsg.Err.Error(); got != "summary store is not configured" {
			t.Fatalf("expected nil-store error message, got %q", got)
		}
	})

	t.Run("empty project still returns nil-store error when store is nil", func(t *testing.T) {
		m := NewSummaryModel("content", "", "meeting-1", nil, 80, 20)
		msg := m.saveCmd()()

		errMsg, ok := msg.(SummarySaveErrMsg)
		if !ok {
			t.Fatalf("expected SummarySaveErrMsg, got %T", msg)
		}

		if got := errMsg.Err.Error(); got != "summary store is not configured" {
			t.Fatalf("expected nil-store error precedence, got %q", got)
		}
	})

	t.Run("empty meeting ID still returns nil-store error when store is nil", func(t *testing.T) {
		m := NewSummaryModel("content", "PRJ", "", nil, 80, 20)
		msg := m.saveCmd()()

		errMsg, ok := msg.(SummarySaveErrMsg)
		if !ok {
			t.Fatalf("expected SummarySaveErrMsg, got %T", msg)
		}

		if got := errMsg.Err.Error(); got != "summary store is not configured" {
			t.Fatalf("expected nil-store error precedence, got %q", got)
		}
	})
}

func TestSummaryUpdateWindowSizeMsgUpdatesDimensions(t *testing.T) {
	m := NewSummaryModel("", "PRJ", "meeting-1", nil, 40, 8)

	updated, cmd := updateSummaryModelForTest(t, m, tea.WindowSizeMsg{Width: 120, Height: 42})
	if cmd != nil {
		t.Fatalf("expected nil command for WindowSizeMsg")
	}

	if updated.width != 120 || updated.height != 42 {
		t.Fatalf("expected width=120 height=42, got width=%d height=%d", updated.width, updated.height)
	}
}

func TestSummaryValueDirtyAndOverwritePendingAccessors(t *testing.T) {
	m := NewSummaryModel("hello", "PRJ", "meeting-1", nil, 80, 20)

	if got := m.Value(); got != "hello" {
		t.Fatalf("expected Value() to return %q, got %q", "hello", got)
	}

	if m.IsDirty() {
		t.Fatalf("expected IsDirty() false at start")
	}

	if m.IsOverwritePending() {
		t.Fatalf("expected IsOverwritePending() false at start")
	}

	m.Focus()
	updated, _ := updateSummaryModelForTest(t, m, tea.KeyPressMsg{Text: "!"})
	updated.SetContent("incoming")

	if !updated.IsDirty() {
		t.Fatalf("expected IsDirty() true after edit")
	}

	if !updated.IsOverwritePending() {
		t.Fatalf("expected IsOverwritePending() true after SetContent while dirty")
	}
}

func TestSummaryFocusAndBlurToggleTextareaFocusState(t *testing.T) {
	m := NewSummaryModel("", "PRJ", "meeting-1", nil, 80, 20)

	if m.textarea.Focused() {
		t.Fatalf("expected textarea to start blurred")
	}

	m.Focus()
	if !m.textarea.Focused() {
		t.Fatalf("expected textarea focused after Focus()")
	}

	m.Blur()
	if m.textarea.Focused() {
		t.Fatalf("expected textarea blurred after Blur()")
	}
}

func TestSummaryAcceptAndRejectOverwriteMethods(t *testing.T) {
	t.Run("accept overwrite replaces content and resets pending", func(t *testing.T) {
		m := NewSummaryModel("user", "PRJ", "meeting-1", nil, 80, 20)
		m.pendingContent = "ai"
		m.overwriteConfirmPending = true
		m.dirty = true

		m.AcceptOverwrite()

		if got := m.Value(); got != "ai" {
			t.Fatalf("expected content %q after AcceptOverwrite, got %q", "ai", got)
		}

		if m.IsDirty() {
			t.Fatalf("expected dirty=false after AcceptOverwrite")
		}

		if m.IsOverwritePending() {
			t.Fatalf("expected pending=false after AcceptOverwrite")
		}
	})

	t.Run("reject overwrite keeps content and resets pending only", func(t *testing.T) {
		m := NewSummaryModel("user", "PRJ", "meeting-1", nil, 80, 20)
		m.pendingContent = "ai"
		m.overwriteConfirmPending = true
		m.dirty = true

		m.RejectOverwrite()

		if got := m.Value(); got != "user" {
			t.Fatalf("expected content unchanged after RejectOverwrite, got %q", got)
		}

		if !m.IsDirty() {
			t.Fatalf("expected dirty to remain true after RejectOverwrite")
		}

		if m.IsOverwritePending() {
			t.Fatalf("expected pending=false after RejectOverwrite")
		}
	})
}

func TestSummaryUpdateSummarySaveErrMsgSetsStatusMessage(t *testing.T) {
	m := NewSummaryModel("", "PRJ", "meeting-1", nil, 80, 20)

	updated, cmd := updateSummaryModelForTest(t, m, SummarySaveErrMsg{Err: errors.New("disk full")})
	if cmd != nil {
		t.Fatalf("expected nil cmd for SummarySaveErrMsg update")
	}

	view := updated.View().Content
	if !strings.Contains(view, "Save error: disk full") {
		t.Fatalf("expected view to contain save error status, got %q", view)
	}
}

func TestSummaryTextareaHeightBoundaryAndMonotonicProperty(t *testing.T) {
	if got := summaryTextareaHeight(-100); got != 1 {
		t.Fatalf("expected height clamp to 1 for negative total height, got %d", got)
	}

	if got := summaryTextareaHeight(0); got != 1 {
		t.Fatalf("expected height 1 for totalHeight=0, got %d", got)
	}

	if got := summaryTextareaHeight(2); got != 1 {
		t.Fatalf("expected height 1 for totalHeight=2, got %d", got)
	}

	if got := summaryTextareaHeight(6); got != 4 {
		t.Fatalf("expected height 4 for totalHeight=6, got %d", got)
	}

	previous := summaryTextareaHeight(-100)
	for h := -99; h <= 200; h++ {
		current := summaryTextareaHeight(h)
		if current < previous {
			t.Fatalf("expected monotonic non-decreasing result, h=%d prev=%d curr=%d", h, previous, current)
		}
		previous = current
	}
}
