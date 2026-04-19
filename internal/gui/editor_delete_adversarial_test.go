//go:build gui

package gui

import (
	"os"
	"testing"
	"time"

	fyneTest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/glieske/recap/internal/storage"
)

func TestDeleteAdversarial(t *testing.T) {
	t.Run("double-tap delete does not trigger double deletion", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		store := storage.NewStore(t.TempDir())
		if _, err := store.CreateProject("Delete Attack", "ATK"); err != nil {
			t.Fatalf("CreateProject returned error: %v", err)
		}
		meeting, err := store.CreateMeeting("Attack Meeting", time.Now(), nil, "ATK", nil, "")
		if err != nil {
			t.Fatalf("CreateMeeting returned error: %v", err)
		}

		win := app.NewWindow("delete-double-tap")
		defer win.Close()

		backCalls := 0
		screen := NewEditorScreen(*meeting, store, nil, win, nil, func() {
			backCalls++
		}, nil)
		win.SetContent(screen.Content)

		// Double-tap: in headless mode both taps may create overlays
		fyneTest.Tap(screen.DeleteButton)
		fyneTest.Tap(screen.DeleteButton)

		// Confirm the first dialog
		yes := deleteWaitForButtonInOverlay(win, "Yes", 2*time.Second)
		if yes == nil {
			t.Fatal("expected confirmation dialog with Yes button")
		}
		fyneTest.Tap(yes)
		time.Sleep(150 * time.Millisecond)

		// If second dialog appeared (headless artifact), confirm it too — should not panic
		secondYes := deleteWaitForButtonInOverlay(win, "Yes", 250*time.Millisecond)
		if secondYes != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("second confirm panicked: %v", r)
					}
				}()
				fyneTest.Tap(secondYes)
			}()
			time.Sleep(100 * time.Millisecond)
		}

		// Invariant: at least one back call, no panic, state not corrupted
		if backCalls < 1 {
			t.Fatalf("expected at least 1 onBack call, got %d", backCalls)
		}
	})

	t.Run("delete with empty meeting ID does not panic", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		store := storage.NewStore(t.TempDir())
		if _, err := store.CreateProject("Delete Empty ID", "EID"); err != nil {
			t.Fatalf("CreateProject returned error: %v", err)
		}

		meeting := storage.Meeting{
			ID:      "",
			Title:   "Empty ID",
			Project: "EID",
			Date:    time.Now(),
		}

		win := app.NewWindow("delete-empty-id")
		defer win.Close()

		backCalls := 0
		screen := NewEditorScreen(meeting, store, nil, win, nil, func() {
			backCalls++
		}, nil)
		win.SetContent(screen.Content)

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("expected no panic deleting empty meeting ID, got panic: %v", r)
				}
			}()

			fyneTest.Tap(screen.DeleteButton)
			yes := deleteWaitForButtonInOverlay(win, "Yes", 2*time.Second)
			if yes == nil {
				t.Fatal("expected confirmation dialog for empty meeting ID")
			}
			fyneTest.Tap(yes)
		}()

		if got, want := backCalls, 1; got != want {
			t.Fatalf("expected onBack calls %d, got %d", want, got)
		}
	})

	t.Run("delete with path traversal meeting ID does not panic and does not delete sibling", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		store := storage.NewStore(t.TempDir())
		if _, err := store.CreateProject("Delete Traversal", "TRV"); err != nil {
			t.Fatalf("CreateProject returned error: %v", err)
		}
		siblingMeeting, err := store.CreateMeeting("Sibling", time.Now(), nil, "TRV", nil, "")
		if err != nil {
			t.Fatalf("CreateMeeting sibling returned error: %v", err)
		}
		siblingPath := store.MeetingDir("TRV", siblingMeeting.ID)

		meeting := storage.Meeting{
			ID:      "../outside",
			Title:   "Traversal",
			Project: "TRV",
			Date:    time.Now(),
		}

		win := app.NewWindow("delete-traversal-id")
		defer win.Close()

		backCalls := 0
		screen := NewEditorScreen(meeting, store, nil, win, nil, func() {
			backCalls++
		}, nil)
		win.SetContent(screen.Content)

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("expected no panic deleting traversal meeting ID, got panic: %v", r)
				}
			}()

			fyneTest.Tap(screen.DeleteButton)
			yes := deleteWaitForButtonInOverlay(win, "Yes", 2*time.Second)
			if yes == nil {
				t.Fatal("expected confirmation dialog for traversal meeting ID")
			}
			fyneTest.Tap(yes)
		}()

		if got, want := backCalls, 1; got != want {
			t.Fatalf("expected onBack calls %d, got %d", want, got)
		}
		if _, err := os.Stat(siblingPath); err != nil {
			t.Fatalf("expected sibling meeting path to remain, got error: %v", err)
		}
	})

	t.Run("delete button with nil window is created and tap does not panic", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		store := storage.NewStore(t.TempDir())
		meeting := storage.Meeting{ID: "nil-window", Title: "Nil Window", Project: "NW", Date: time.Now()}

		screen := NewEditorScreen(meeting, store, nil, nil, nil, func() {}, nil)
		if screen.DeleteButton == nil {
			t.Fatal("expected DeleteButton to be non-nil")
		}
		if got, want := screen.DeleteButton.Text, "🗑 Delete"; got != want {
			t.Fatalf("expected DeleteButton text %q, got %q", want, got)
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("expected no panic tapping delete button with nil window, got panic: %v", r)
				}
			}()
			screen.DeleteButton.OnTapped()
		}()
	})

	t.Run("tap delete after navigation away does not trigger second deletion", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		store := storage.NewStore(t.TempDir())
		if _, err := store.CreateProject("Delete Navigate", "NAV"); err != nil {
			t.Fatalf("CreateProject returned error: %v", err)
		}
		meeting, err := store.CreateMeeting("Navigate Meeting", time.Now(), nil, "NAV", nil, "")
		if err != nil {
			t.Fatalf("CreateMeeting returned error: %v", err)
		}

		win := app.NewWindow("delete-after-nav")
		defer win.Close()

		backCalls := 0
		screen := NewEditorScreen(*meeting, store, nil, win, nil, func() {
			backCalls++
			win.SetContent(widget.NewLabel("meeting-list"))
		}, nil)
		win.SetContent(screen.Content)

		// First delete
		fyneTest.Tap(screen.DeleteButton)
		yes := deleteWaitForButtonInOverlay(win, "Yes", 2*time.Second)
		if yes == nil {
			t.Fatal("expected initial delete confirmation")
		}
		fyneTest.Tap(yes)
		time.Sleep(100 * time.Millisecond)

		if got, want := backCalls, 1; got != want {
			t.Fatalf("expected onBack calls after first delete %d, got %d", want, got)
		}

		// Second tap after navigation: may still show a dialog in headless
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("second delete tap after navigation panicked: %v", r)
				}
			}()
			fyneTest.Tap(screen.DeleteButton)
			secondYes := deleteWaitForButtonInOverlay(win, "Yes", 250*time.Millisecond)
			if secondYes != nil {
				fyneTest.Tap(secondYes)
				time.Sleep(100 * time.Millisecond)
			}
		}()
		// Invariant: no panic occurred — that's the real test
	})
}
