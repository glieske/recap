package tui

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/quick"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestAdversarialEditor(t *testing.T) {
	t.Run("saveCmd concurrent empty and oversized payload", func(t *testing.T) {
		store, meeting := createEditorTestStoreAndMeeting(t)
		m := NewEditorModel(meeting, store, 80, 20, "", "", "")

		empty := ""
		oversized := strings.Repeat("A", 12*1024)

		m.textarea.SetValue(empty)
		emptyCmd := m.saveCmd()

		m.textarea.SetValue(oversized)
		largeCmd := m.saveCmd()

		results := make(chan tea.Msg, 2)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			results <- emptyCmd()
		}()
		go func() {
			defer wg.Done()
			results <- largeCmd()
		}()
		wg.Wait()
		close(results)

		var successes int
		for msg := range results {
			switch msg.(type) {
			case SaveDoneMsg:
				successes++
			case SaveErrMsg:
				// acceptable — concurrent writes to same file may conflict
			default:
				t.Fatalf("unexpected message type %T for concurrent save", msg)
			}
		}
		if successes == 0 {
			t.Fatalf("expected at least one successful save, got zero")
		}

		raw, err := store.LoadRawNotes(meeting.Project, meeting.ID)
		if err != nil {
			t.Fatalf("LoadRawNotes: %v", err)
		}
		if raw != empty && raw != oversized {
			t.Fatalf("expected final raw notes to be empty or oversized payload, got len=%d", len(raw))
		}
	})

	t.Run("composeStatusLine boundary widths unicode and long sections", func(t *testing.T) {
		if got := composeStatusLine("left", "center", "right", -5); got != "" {
			t.Fatalf("negative width: expected empty string, got %q", got)
		}

		if got := composeStatusLine("ŁŁŁ", "中中", "🚀🚀", 1); got != "Ł" {
			t.Fatalf("width=1 unicode truncation: expected %q, got %q", "Ł", got)
		}

		left := strings.Repeat("L", 7000)
		center := strings.Repeat("中", 3000)
		right := strings.Repeat("🚀", 2000)
		line := composeStatusLine(left, center, right, 97)
		if got := len([]rune(line)); got != 97 {
			t.Fatalf("expected rune length 97, got %d", got)
		}

		cfg := &quick.Config{MaxCount: 200}
		propErr := quick.Check(func(a, b, c string, w uint8) bool {
			width := int(w) + 1 // 1..256
			out := composeStatusLine(a, b, c, width)
			return len([]rune(out)) == width
		}, cfg)
		if propErr != nil {
			t.Fatalf("composeStatusLine width preservation invariant failed: %v", propErr)
		}
	})

	t.Run("formatElapsed negative zero and huge duration", func(t *testing.T) {
		if got := formatElapsed(-1 * time.Second); got != "00:00:-1" {
			t.Fatalf("negative duration: expected %q, got %q", "00:00:-1", got)
		}

		if got := formatElapsed(0); got != "00:00:00" {
			t.Fatalf("zero duration: expected %q, got %q", "00:00:00", got)
		}

		if got := formatElapsed(1000*time.Hour + 2*time.Minute + 3*time.Second); got != "1000:02:03" {
			t.Fatalf("large duration: expected %q, got %q", "1000:02:03", got)
		}
	})

	t.Run("maxEditorHeight negative zero and int max", func(t *testing.T) {
		if got := maxEditorHeight(-100); got != 1 {
			t.Fatalf("negative totalHeight: expected 1, got %d", got)
		}
		if got := maxEditorHeight(0); got != 1 {
			t.Fatalf("zero totalHeight: expected 1, got %d", got)
		}

		maxInt := int(^uint(0) >> 1)
		if got := maxEditorHeight(maxInt); got != maxInt-4 {
			t.Fatalf("max int boundary: expected %d, got %d", maxInt-4, got)
		}
	})

	t.Run("NewEditorModel nil and negative dimensions", func(t *testing.T) {
		mZero := NewEditorModel(nil, nil, 0, 0, "", "", "")
		if mZero.width != 0 || mZero.height != 0 {
			t.Fatalf("zero dimensions: expected width=0 height=0, got width=%d height=%d", mZero.width, mZero.height)
		}
		if mZero.meeting != nil || mZero.store != nil {
			t.Fatalf("nil dependencies: expected meeting/store nil")
		}

		mNeg := NewEditorModel(nil, nil, -10, -20, "", "", "")
		if mNeg.width != -10 || mNeg.height != -20 {
			t.Fatalf("negative dimensions: expected width=-10 height=-20, got width=%d height=%d", mNeg.width, mNeg.height)
		}
		if mNeg.View().Content == "" {
			t.Fatalf("expected non-empty view with negative dimensions")
		}
	})

	t.Run("Init returns command for malformed model dimensions", func(t *testing.T) {
		m := NewEditorModel(nil, nil, -1, -1, "", "", "")
		if cmd := m.Init(); cmd == nil {
			t.Fatalf("expected non-nil init command for malformed dimensions")
		}
	})

	t.Run("Update rapid autosave ticks while clean", func(t *testing.T) {
		store, meeting := createEditorTestStoreAndMeeting(t)
		m := NewEditorModel(meeting, store, 80, 20, "", "", "")
		m.dirty = false

		for i := 0; i < 250; i++ {
			updatedModel, cmd := m.Update(AutoSaveTickMsg{})
			updated, ok := updatedModel.(EditorModel)
			if !ok {
				t.Fatalf("expected EditorModel from Update, got %T", updatedModel)
			}
			if cmd == nil {
				t.Fatalf("iteration %d: expected non-nil tick command", i)
			}
			if updated.dirty {
				t.Fatalf("iteration %d: expected dirty=false, got true", i)
			}
			m = updated
		}
	})

	t.Run("renderStatusBar nil meeting zero width and long identifiers", func(t *testing.T) {
		mNil := NewEditorModel(nil, nil, 0, 10, "", "", "")
		mNil.textarea.SetValue("x")
		barNil := mNil.renderStatusBar()
		if !strings.Contains(barNil, "-") {
			t.Fatalf("expected placeholder content for nil meeting, got %q", barNil)
		}

		store, meeting := createEditorTestStoreAndMeeting(t)
		meeting.TicketID = strings.Repeat("TICKET-", 300)
		meeting.Title = strings.Repeat("非常に長いタイトル🚀", 200)
		mLong := NewEditorModel(meeting, store, 40, 10, "", "", "")
		mLong.textarea.SetValue(strings.Repeat("Z", 2048))
		barLong := mLong.renderStatusBar()
		if barLong == "" {
			t.Fatalf("expected non-empty status bar for long metadata")
		}
	})

	t.Run("Update SaveErrMsg after SaveDoneMsg sequence", func(t *testing.T) {
		m := NewEditorModel(nil, nil, 80, 20, "", "", "")

		updatedAfterDoneModel, cmd := m.Update(SaveDoneMsg{})
		if cmd != nil {
			t.Fatalf("SaveDoneMsg: expected nil command, got non-nil")
		}
		updatedAfterDone, ok := updatedAfterDoneModel.(EditorModel)
		if !ok {
			t.Fatalf("SaveDoneMsg: expected EditorModel, got %T", updatedAfterDoneModel)
		}
		if updatedAfterDone.statusMsg != "Saved" {
			t.Fatalf("SaveDoneMsg: expected status %q, got %q", "Saved", updatedAfterDone.statusMsg)
		}

		errPayload := errors.New("disk failure after success")
		updatedAfterErrModel, cmd := updatedAfterDone.Update(SaveErrMsg{Err: errPayload})
		if cmd != nil {
			t.Fatalf("SaveErrMsg: expected nil command, got non-nil")
		}
		updatedAfterErr, ok := updatedAfterErrModel.(EditorModel)
		if !ok {
			t.Fatalf("SaveErrMsg: expected EditorModel, got %T", updatedAfterErrModel)
		}
		if !updatedAfterErr.dirty {
			t.Fatalf("SaveErrMsg: expected dirty=true after error")
		}
		if !reflect.DeepEqual(updatedAfterErr.err, errPayload) {
			t.Fatalf("SaveErrMsg: expected stored error %v, got %v", errPayload, updatedAfterErr.err)
		}
		if updatedAfterErr.statusMsg != "Save error: disk failure after success" {
			t.Fatalf("SaveErrMsg: expected overwritten error status, got %q", updatedAfterErr.statusMsg)
		}
	})

	t.Run("Update SaveErrMsg with nil error panics", func(t *testing.T) {
		m := NewEditorModel(nil, nil, 80, 20, "", "", "")
		defer func() {
			recovered := recover()
			if recovered == nil {
				t.Fatalf("expected panic for SaveErrMsg with nil Err")
			}
			if got := fmt.Sprint(recovered); got != "runtime error: invalid memory address or nil pointer dereference" {
				t.Fatalf("expected nil-pointer panic, got %v", got)
			}
		}()

		_, _ = m.Update(SaveErrMsg{Err: nil})
	})

	t.Run("formatElapsed monotonicity for non-negative durations", func(t *testing.T) {
		cfg := &quick.Config{MaxCount: 200}
		propErr := quick.Check(func(a, b uint16) bool {
			da := time.Duration(a) * time.Second
			db := time.Duration(b) * time.Second
			if da > db {
				da, db = db, da
			}

			toSeconds := func(s string) int {
				var h, m, sec int
				_, _ = fmtSscanfNoErr(s, &h, &m, &sec)
				return h*3600 + m*60 + sec
			}

			return toSeconds(formatElapsed(da)) <= toSeconds(formatElapsed(db))
		}, cfg)
		if propErr != nil {
			t.Fatalf("formatElapsed monotonicity failed: %v", propErr)
		}
	})
}

func fmtSscanfNoErr(input string, h, m, s *int) (int, error) {
	return fmt.Sscanf(input, "%d:%d:%d", h, m, s)
}
