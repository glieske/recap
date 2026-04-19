//go:build gui

package gui

import (
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	fyneTest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/glieske/recap/internal/storage"
)

func TestNewMeetingAdversarial(t *testing.T) {
	t.Run("nil window should not panic", func(t *testing.T) {
		store := storage.NewStore(filepath.Join(t.TempDir(), "notes"))

		assertNotPanicsNewMeetingAdversarial(t, func() {
			ShowNewMeetingDialog(nil, store, nil)
		})
	})

	t.Run("nil store and nil callback should not panic", func(t *testing.T) {
		testApp := fyneTest.NewApp()
		t.Cleanup(testApp.Quit)

		win := testApp.NewWindow("Test")

		assertNotPanicsNewMeetingAdversarial(t, func() {
			ShowNewMeetingDialog(win, nil, nil)
		})
	})

	t.Run("nil onCreate callback with valid store should not panic", func(t *testing.T) {
		store := storage.NewStore(filepath.Join(t.TempDir(), "notes"))
		if _, err := store.CreateProject("Ops", "OPS"); err != nil {
			t.Fatalf("CreateProject returned error: %v", err)
		}

		testApp := fyneTest.NewApp()
		t.Cleanup(testApp.Quit)

		win := testApp.NewWindow("Test")

		assertNotPanicsNewMeetingAdversarial(t, func() {
			ShowNewMeetingDialog(win, store, nil)
		})
	})

	t.Run("malformed date is rejected without creating meeting", func(t *testing.T) {
		store := storage.NewStore(filepath.Join(t.TempDir(), "notes"))
		if _, err := store.CreateProject("Ops", "OPS"); err != nil {
			t.Fatalf("CreateProject returned error: %v", err)
		}

		testApp := fyneTest.NewApp()
		t.Cleanup(testApp.Quit)

		win := testApp.NewWindow("Test")
		ShowNewMeetingDialog(win, store, nil)

		overlay := win.Canvas().Overlays().Top()
		if overlay == nil {
			t.Fatal("expected dialog overlay to be present")
		}

		titleEntry := findEntryByPlaceholder(overlay, "Meeting title")
		if titleEntry == nil {
			t.Fatal("expected title entry in dialog")
		}
		titleEntry.SetText("Adversarial Date")

		dateEntry := findEntryByPlaceholder(overlay, "YYYY-MM-DD")
		if dateEntry == nil {
			t.Fatal("expected date entry in dialog")
		}
		dateEntry.SetText("2026-99-99")

		projectSelect := findSelectInObject(overlay)
		if projectSelect == nil {
			t.Fatal("expected project select in dialog")
		}
		projectSelect.SetSelected("OPS")

		createButton := findButtonByTextNewMeetingAdversarial(overlay, "Create")
		if createButton == nil {
			t.Fatal("expected Create button in dialog")
		}

		fyneTest.Tap(createButton)

		meetings, err := store.ListMeetings("")
		if err != nil {
			t.Fatalf("ListMeetings returned error: %v", err)
		}
		if got, want := len(meetings), 0; got != want {
			t.Fatalf("meeting count after malformed date submit = %d, want %d", got, want)
		}
	})

	t.Run("whitespace-only title is rejected without creating meeting", func(t *testing.T) {
		store := storage.NewStore(filepath.Join(t.TempDir(), "notes"))
		if _, err := store.CreateProject("Ops", "OPS"); err != nil {
			t.Fatalf("CreateProject returned error: %v", err)
		}

		testApp := fyneTest.NewApp()
		t.Cleanup(testApp.Quit)

		win := testApp.NewWindow("Test")
		ShowNewMeetingDialog(win, store, nil)

		overlay := win.Canvas().Overlays().Top()
		if overlay == nil {
			t.Fatal("expected dialog overlay to be present")
		}

		titleEntry := findEntryByPlaceholder(overlay, "Meeting title")
		if titleEntry == nil {
			t.Fatal("expected title entry in dialog")
		}
		titleEntry.SetText("  \t  ")

		dateEntry := findEntryByPlaceholder(overlay, "YYYY-MM-DD")
		if dateEntry == nil {
			t.Fatal("expected date entry in dialog")
		}
		dateEntry.SetText("2026-02-02")

		projectSelect := findSelectInObject(overlay)
		if projectSelect == nil {
			t.Fatal("expected project select in dialog")
		}
		projectSelect.SetSelected("OPS")

		createButton := findButtonByTextNewMeetingAdversarial(overlay, "Create")
		if createButton == nil {
			t.Fatal("expected Create button in dialog")
		}

		fyneTest.Tap(createButton)

		meetings, err := store.ListMeetings("")
		if err != nil {
			t.Fatalf("ListMeetings returned error: %v", err)
		}
		if got, want := len(meetings), 0; got != want {
			t.Fatalf("meeting count after whitespace title submit = %d, want %d", got, want)
		}
	})

	t.Run("multiple rapid calls should not panic", func(t *testing.T) {
		store := storage.NewStore(filepath.Join(t.TempDir(), "notes"))

		testApp := fyneTest.NewApp()
		t.Cleanup(testApp.Quit)

		win := testApp.NewWindow("Test")

		assertNotPanicsNewMeetingAdversarial(t, func() {
			for i := 0; i < 200; i++ {
				ShowNewMeetingDialog(win, store, nil)
			}
		})
	})
}

func findEntryByPlaceholder(obj fyne.CanvasObject, placeholder string) *widget.Entry {
	if obj == nil {
		return nil
	}

	if entry, ok := obj.(*widget.Entry); ok {
		if entry.PlaceHolder == placeholder {
			return entry
		}
	}

	if c, ok := obj.(*fyne.Container); ok {
		for _, child := range c.Objects {
			if found := findEntryByPlaceholder(child, placeholder); found != nil {
				return found
			}
		}
	}

	if w, ok := obj.(fyne.Widget); ok {
		r := fyneTest.WidgetRenderer(w)
		if r != nil {
			for _, child := range r.Objects() {
				if found := findEntryByPlaceholder(child, placeholder); found != nil {
					return found
				}
			}
		}
	}

	return nil
}

func findButtonByTextNewMeetingAdversarial(obj fyne.CanvasObject, label string) *widget.Button {
	if obj == nil {
		return nil
	}

	if btn, ok := obj.(*widget.Button); ok {
		if btn.Text == label {
			return btn
		}
	}

	if c, ok := obj.(*fyne.Container); ok {
		for _, child := range c.Objects {
			if found := findButtonByTextNewMeetingAdversarial(child, label); found != nil {
				return found
			}
		}
	}

	if w, ok := obj.(fyne.Widget); ok {
		r := fyneTest.WidgetRenderer(w)
		if r != nil {
			for _, child := range r.Objects() {
				if found := findButtonByTextNewMeetingAdversarial(child, label); found != nil {
					return found
				}
			}
		}
	}

	return nil
}

func assertNotPanicsNewMeetingAdversarial(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	fn()
}
