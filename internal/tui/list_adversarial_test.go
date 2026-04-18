package tui

import (
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/storage"
)

func assertNoPanic(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	fn()
}

func adversarialKey(r string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: r}
}

func adversarialEnter() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyEnter}
}

func TestAdversarialNewListModelNilStoreShowsEmptyState(t *testing.T) {
	assertNoPanic(t, func() {
		m := NewListModel(nil, 80, 20)

		if m.store != nil {
			t.Fatalf("expected nil store, got non-nil")
		}

		if len(m.list.Items()) != 0 {
			t.Fatalf("expected 0 list items, got %d", len(m.list.Items()))
		}

		if got := m.View().Content; !strings.Contains(got, "No meetings yet. Press 'n' to create one.") {
			t.Fatalf("expected empty state view, got %q", got)
		}
	})
}

func TestAdversarialUpdateEnterWithNilSelectedItemDoesNotPanic(t *testing.T) {
	m := NewListModel(nil, 80, 20)

	assertNoPanic(t, func() {
		updatedModel, cmd := m.Update(adversarialEnter())
		_, ok := updatedModel.(ListModel)
		if !ok {
			t.Fatalf("expected ListModel, got %T", updatedModel)
		}

		if cmd != nil {
			t.Fatalf("expected nil command for enter with nil selected item")
		}
	})
}

func TestAdversarialMeetingItemZeroValueFieldsDoNotPanic(t *testing.T) {
	item := MeetingItem{meeting: storage.Meeting{}}

	assertNoPanic(t, func() {
		if got := item.Title(); got != "[] " {
			t.Fatalf("expected zero-value title %q, got %q", "[] ", got)
		}

		if got := item.FilterValue(); got != "  " {
			t.Fatalf("expected zero-value filter value %q, got %q", "  ", got)
		}

		if got := item.Description(); got != "0001-01-01 |  | ○ DRAFT" {
			t.Fatalf("expected zero-value description %q, got %q", "0001-01-01 |  | ○ DRAFT", got)
		}
	})
}

func TestAdversarialCycleFilterValueEmptyValuesReturnsEmptyString(t *testing.T) {
	if got := cycleFilterValue("anything", []string{}); got != "" {
		t.Fatalf("expected empty string for empty values, got %q", got)
	}
}

func TestAdversarialCycleFilterValueUnknownCurrentReturnsFirstValue(t *testing.T) {
	values := []string{"ALPHA", "BETA"}
	if got := cycleFilterValue("MISSING", values); got != "ALPHA" {
		t.Fatalf("expected first value %q for unknown current value, got %q", "ALPHA", got)
	}
}

func TestAdversarialRefreshMeetingsNilStoreClearsStateWithoutPanic(t *testing.T) {
	m := NewListModel(nil, 80, 20)
	m.allMeetings = []storage.Meeting{{ID: "stale"}}
	m.list = list.New(
		[]list.Item{MeetingItem{meeting: storage.Meeting{ID: "stale", Date: time.Now()}}},
		list.NewDefaultDelegate(),
		80,
		20,
	)

	assertNoPanic(t, func() {
		cmd := m.RefreshMeetings()
		if cmd != nil {
			_ = cmd()
		}

		if len(m.allMeetings) != 0 {
			t.Fatalf("expected allMeetings to be cleared, got %d items", len(m.allMeetings))
		}

		if len(m.list.Items()) != 0 {
			t.Fatalf("expected list items to be cleared, got %d", len(m.list.Items()))
		}
	})
}

func TestAdversarialUpdateUnknownMsgTypeDoesNotPanic(t *testing.T) {
	type unknownMsg struct{ marker string }

	m := NewListModel(nil, 80, 20)

	assertNoPanic(t, func() {
		updatedModel, cmd := m.Update(unknownMsg{marker: "unexpected"})
		if _, ok := updatedModel.(ListModel); !ok {
			t.Fatalf("expected ListModel for unknown message, got %T", updatedModel)
		}

		if cmd != nil {
			_ = cmd()
		}
	})
}

func TestAdversarialRapidFilterCyclingDoesNotCorruptState(t *testing.T) {
	m := NewListModel(nil, 80, 20)

	assertNoPanic(t, func() {
		for i := 0; i < 250; i++ {
			updatedModel, cmd := m.Update(adversarialKey("f"))
			updated, ok := updatedModel.(ListModel)
			if !ok {
				t.Fatalf("expected ListModel on iteration %d, got %T", i, updatedModel)
			}
			m = updated

			if cmd != nil {
				_ = cmd()
			}
		}

		if m.projectFilter != "" {
			t.Fatalf("expected empty project filter with no projects, got %q", m.projectFilter)
		}

		if m.tagFilter != "" {
			t.Fatalf("expected empty tag filter, got %q", m.tagFilter)
		}

		if len(m.list.Items()) != 0 {
			t.Fatalf("expected 0 items after rapid cycling on nil store, got %d", len(m.list.Items()))
		}
	})
}
