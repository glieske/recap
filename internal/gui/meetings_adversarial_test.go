//go:build gui

package gui

import (
	"path/filepath"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"

	"github.com/glieske/recap/internal/storage"
)

func TestMeetingsAdversarial(t *testing.T) {
	t.Run("nil store does not panic", func(t *testing.T) {
		var screenObj any
		assertNotPanics(t, func() {
			screenObj = NewMeetingsScreen(nil, nil, nil)
		})

		if screenObj == nil {
			t.Fatal("expected NewMeetingsScreen to return non-nil object for nil store")
		}

		list := findMeetingsList(screenObj.(fyne.CanvasObject))
		if list == nil {
			t.Fatal("expected meetings list to be present")
		}
		if got, want := list.Length(), 0; got != want {
			t.Fatalf("expected list length %d with nil store, got %d", want, got)
		}
	})

	t.Run("nil callbacks on select and new do not panic", func(t *testing.T) {
		store := storage.NewStore(filepath.Join(t.TempDir(), "notes"))
		meetingDate := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)

		if _, err := store.CreateProject("Ops", "OPS"); err != nil {
			t.Fatalf("CreateProject returned error: %v", err)
		}
		if _, err := store.CreateMeeting("Adversarial", meetingDate, []string{"A"}, "OPS", nil, ""); err != nil {
			t.Fatalf("CreateMeeting returned error: %v", err)
		}

		screen := NewMeetingsScreen(store, nil, nil)
		list := findMeetingsList(screen)
		if list == nil {
			t.Fatal("expected meetings list to be present")
		}
		if got, want := list.Length(), 1; got != want {
			t.Fatalf("expected list length %d, got %d", want, got)
		}

		newButton := findButtonByText(screen, "+ New Meeting")
		if newButton == nil || newButton.OnTapped == nil {
			t.Fatal("expected + New Meeting button tap handler to be present")
		}

		assertNotPanics(t, func() {
			list.Select(0)
			newButton.OnTapped()
		})

		if got, want := list.Length(), 1; got != want {
			t.Fatalf("expected list length to remain %d after nil callbacks, got %d", want, got)
		}
	})

	t.Run("out of bounds selection does not panic", func(t *testing.T) {
		store := storage.NewStore(filepath.Join(t.TempDir(), "notes"))
		meetingDate := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)

		if _, err := store.CreateProject("Backend", "BE"); err != nil {
			t.Fatalf("CreateProject returned error: %v", err)
		}
		if _, err := store.CreateMeeting("Bounds", meetingDate, []string{"B"}, "BE", nil, ""); err != nil {
			t.Fatalf("CreateMeeting returned error: %v", err)
		}

		screen := NewMeetingsScreen(store, nil, nil)
		list := findMeetingsList(screen)
		if list == nil {
			t.Fatal("expected meetings list to be present")
		}

		assertNotPanics(t, func() {
			list.Select(-1)
			list.Select(9999)
		})

		if got, want := list.Length(), 1; got != want {
			t.Fatalf("expected list length %d after out-of-bounds selects, got %d", want, got)
		}
	})

	t.Run("rapid and concurrent refresh calls do not panic", func(t *testing.T) {
		store := storage.NewStore(filepath.Join(t.TempDir(), "notes"))
		meetingDate := time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC)

		if _, err := store.CreateProject("Infra", "INFRA"); err != nil {
			t.Fatalf("CreateProject returned error: %v", err)
		}
		if _, err := store.CreateMeeting("Concurrency", meetingDate, []string{"C"}, "INFRA", nil, ""); err != nil {
			t.Fatalf("CreateMeeting returned error: %v", err)
		}

		screen := NewMeetingsScreen(store, nil, nil)
		list := findMeetingsList(screen)
		if list == nil {
			t.Fatal("expected meetings list to be present")
		}

		refreshButton := findButtonByText(screen, "Refresh")
		if refreshButton == nil || refreshButton.OnTapped == nil {
			t.Fatal("expected Refresh button tap handler to be present")
		}

		assertNotPanics(t, func() {
			for i := 0; i < 200; i++ {
				refreshButton.OnTapped()
			}
		})

		panicCh := make(chan any, 32)
		var wg sync.WaitGroup
		for i := 0; i < 32; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer func() {
					if r := recover(); r != nil {
						panicCh <- r
					}
				}()
				refreshButton.OnTapped()
			}()
		}
		wg.Wait()
		close(panicCh)

		for p := range panicCh {
			t.Fatalf("unexpected panic during concurrent refresh: %v", p)
		}

		if got, want := list.Length(), 1; got != want {
			t.Fatalf("expected list length %d after rapid/concurrent refresh, got %d", want, got)
		}
	})

	t.Run("filter change with empty projects list does not panic", func(t *testing.T) {
		store := storage.NewStore(filepath.Join(t.TempDir(), "notes"))

		screen := NewMeetingsScreen(store, nil, nil)
		filter := findProjectFilter(screen)
		if filter == nil {
			t.Fatal("expected project filter to be present")
		}
		if got, want := len(filter.Options), 1; got != want {
			t.Fatalf("expected %d filter option for empty projects list, got %d", want, got)
		}

		assertNotPanics(t, func() {
			filter.OnChanged("INVALID")
		})

		if got, want := filter.Selected, "All"; got != want {
			t.Fatalf("expected selected filter %q after change on empty projects list, got %q", want, got)
		}
	})
}

func assertNotPanics(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	fn()
}
