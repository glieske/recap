//go:build gui

package gui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/glieske/recap/internal/storage"
)

func TestMeetingsScreenReturnsCanvasObjectAndHandlesEmptyList(t *testing.T) {
	store := storage.NewStore(filepath.Join(t.TempDir(), "notes"))

	screen := NewMeetingsScreen(store, nil, nil)
	if screen == nil {
		t.Fatal("expected NewMeetingsScreen to return non-nil CanvasObject")
	}

	list := findMeetingsList(screen)
	if list == nil {
		t.Fatal("expected meetings screen to include a widget.List")
	}
	if got, want := list.Length(), 0; got != want {
		t.Fatalf("expected empty meetings list length %d, got %d", want, got)
	}

	filter := findProjectFilter(screen)
	if filter == nil {
		t.Fatal("expected meetings screen to include project filter")
	}
	if got, want := len(filter.Options), 1; got != want {
		t.Fatalf("expected %d filter option, got %d", want, got)
	}
	if got, want := filter.Options[0], "All"; got != want {
		t.Fatalf("expected first filter option %q, got %q", want, got)
	}
}

func TestMeetingsScreenPopulatesListAndProjectFilterOptions(t *testing.T) {
	store := storage.NewStore(filepath.Join(t.TempDir(), "notes"))
	meetingDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	if _, err := store.CreateProject("Alpha", "ALPHA"); err != nil {
		t.Fatalf("CreateProject ALPHA returned error: %v", err)
	}
	if _, err := store.CreateProject("Beta", "BETA"); err != nil {
		t.Fatalf("CreateProject BETA returned error: %v", err)
	}

	if _, err := store.CreateMeeting("Alpha sync", meetingDate, []string{"Alice"}, "ALPHA", []string{"infra"}, "EXT-1"); err != nil {
		t.Fatalf("CreateMeeting ALPHA returned error: %v", err)
	}
	if _, err := store.CreateMeeting("Beta sync", meetingDate.Add(24*time.Hour), []string{"Bob"}, "BETA", []string{"ops"}, "EXT-2"); err != nil {
		t.Fatalf("CreateMeeting BETA returned error: %v", err)
	}

	screen := NewMeetingsScreen(store, nil, nil)

	list := findMeetingsList(screen)
	if list == nil {
		t.Fatal("expected meetings list widget to be present")
	}
	if got, want := list.Length(), 2; got != want {
		t.Fatalf("expected list to contain %d meetings, got %d", want, got)
	}

	filter := findProjectFilter(screen)
	if filter == nil {
		t.Fatal("expected project filter widget to be present")
	}
	if got, want := filter.Options[0], "All"; got != want {
		t.Fatalf("expected first filter option %q, got %q", want, got)
	}
	if !containsString(filter.Options, "ALPHA") {
		t.Fatalf("expected filter options to include %q, got %v", "ALPHA", filter.Options)
	}
	if !containsString(filter.Options, "BETA") {
		t.Fatalf("expected filter options to include %q, got %v", "BETA", filter.Options)
	}
}

func TestMeetingsScreenOnSelectFiresWithCorrectMeeting(t *testing.T) {
	store := storage.NewStore(filepath.Join(t.TempDir(), "notes"))
	meetingDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	if _, err := store.CreateProject("Infrastructure", "INFRA"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	olderMeeting, err := store.CreateMeeting("Older", meetingDate, []string{"Alice"}, "INFRA", nil, "")
	if err != nil {
		t.Fatalf("CreateMeeting older returned error: %v", err)
	}
	newerMeeting, err := store.CreateMeeting("Newer", meetingDate.Add(24*time.Hour), []string{"Bob"}, "INFRA", nil, "")
	if err != nil {
		t.Fatalf("CreateMeeting newer returned error: %v", err)
	}

	var selected storage.Meeting
	callbackCount := 0
	screen := NewMeetingsScreen(store, func(m storage.Meeting) {
		selected = m
		callbackCount++
	}, nil)

	list := findMeetingsList(screen)
	if list == nil {
		t.Fatal("expected meetings list widget to be present")
	}
	if got, want := list.Length(), 2; got != want {
		t.Fatalf("expected list length %d, got %d", want, got)
	}

	list.Select(0)

	if got, want := callbackCount, 1; got != want {
		t.Fatalf("expected onSelect callback count %d, got %d", want, got)
	}
	if got, want := selected.ID, newerMeeting.ID; got != want {
		t.Fatalf("expected selected meeting ID %q, got %q", want, got)
	}
	if selected.ID == olderMeeting.ID {
		t.Fatalf("expected selected meeting to not be older meeting %q", olderMeeting.ID)
	}
}

func TestMeetingsScreenOnNewButtonFiresCallback(t *testing.T) {
	store := storage.NewStore(filepath.Join(t.TempDir(), "notes"))

	newCount := 0
	screen := NewMeetingsScreen(store, nil, func() {
		newCount++
	})

	newButton := findButtonByText(screen, "+ New Meeting")
	if newButton == nil {
		t.Fatal("expected + New Meeting button to be present")
	}
	if newButton.OnTapped == nil {
		t.Fatal("expected + New Meeting button to have tap handler")
	}

	newButton.OnTapped()

	if got, want := newCount, 1; got != want {
		t.Fatalf("expected onNew callback count %d, got %d", want, got)
	}
}

func TestMeetingsScreenRefreshHandlesStoreErrorsGracefully(t *testing.T) {
	notesDir := filepath.Join(t.TempDir(), "notes")
	store := storage.NewStore(notesDir)
	meetingDate := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)

	if _, err := store.CreateProject("Infrastructure", "INFRA"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	if _, err := store.CreateMeeting("Initial", meetingDate, []string{"Alice"}, "INFRA", nil, ""); err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	screen := NewMeetingsScreen(store, nil, nil)
	list := findMeetingsList(screen)
	if list == nil {
		t.Fatal("expected meetings list widget to be present")
	}
	if got, want := list.Length(), 1; got != want {
		t.Fatalf("expected initial list length %d, got %d", want, got)
	}

	if err := os.RemoveAll(notesDir); err != nil {
		t.Fatalf("RemoveAll returned error: %v", err)
	}
	if err := os.WriteFile(notesDir, []byte("not-a-directory"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	refreshButton := findButtonByText(screen, "Refresh")
	if refreshButton == nil {
		t.Fatal("expected Refresh button to be present")
	}
	if refreshButton.OnTapped == nil {
		t.Fatal("expected Refresh button to have tap handler")
	}

	refreshButton.OnTapped()

	if got, want := list.Length(), 0; got != want {
		t.Fatalf("expected list length %d after refresh error, got %d", want, got)
	}
}

func findMeetingsList(obj fyne.CanvasObject) *widget.List {
	switch v := obj.(type) {
	case *widget.List:
		return v
	case *fyne.Container:
		for _, child := range v.Objects {
			if found := findMeetingsList(child); found != nil {
				return found
			}
		}
	}

	return nil
}

func findProjectFilter(obj fyne.CanvasObject) *widget.Select {
	switch v := obj.(type) {
	case *widget.Select:
		return v
	case *fyne.Container:
		for _, child := range v.Objects {
			if found := findProjectFilter(child); found != nil {
				return found
			}
		}
	}

	return nil
}

func findButtonByText(obj fyne.CanvasObject, text string) *widget.Button {
	switch v := obj.(type) {
	case *widget.Button:
		if v.Text == text {
			return v
		}
	case *fyne.Container:
		for _, child := range v.Objects {
			if found := findButtonByText(child, text); found != nil {
				return found
			}
		}
	}

	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}

	return false
}
