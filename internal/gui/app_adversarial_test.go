//go:build gui

package gui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	fyneTest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/glieske/recap/internal/storage"
)

func TestAppNavigationAdversarial(t *testing.T) {
	testApp := fyneTest.NewApp()
	t.Cleanup(testApp.Quit)

	win := testApp.NewWindow("recap-adversarial")
	meetingsContainer := container.NewStack()

	var showList func()
	var showEditor func(storage.Meeting)

	showEditor = func(meeting storage.Meeting) {
		backButton := widget.NewButton("← Back", func() {
			showList()
		})

		editorPlaceholder := container.NewVBox(
			backButton,
			container.NewCenter(widget.NewLabel("Editor: "+meeting.Title)),
		)

		meetingsContainer.Objects = []fyne.CanvasObject{editorPlaceholder}
		meetingsContainer.Refresh()
	}

	showList = func() {
		listView := NewMeetingsScreen(
			nil,
			func(meeting storage.Meeting) {
				showEditor(meeting)
			},
			func() {
				ShowNewMeetingDialog(win, nil, func(meeting storage.Meeting) {
					showEditor(meeting)
				})
			},
		)

		meetingsContainer.Objects = []fyne.CanvasObject{listView}
		meetingsContainer.Refresh()
	}

	t.Run("showEditor with zero-value meeting does not panic", func(t *testing.T) {
		assertNoPanicAppAdversarial(t, func() {
			showEditor(storage.Meeting{})
		})

		if got, want := len(meetingsContainer.Objects), 1; got != want {
			t.Fatalf("meetings container object count = %d, want %d", got, want)
		}

		label := findLabelByExactTextAppAdversarial(meetingsContainer, "Editor: ")
		if label == nil {
			t.Fatal("expected editor placeholder label for zero-value meeting")
		}
	})

	t.Run("rapid alternation showList and showEditor does not panic", func(t *testing.T) {
		assertNoPanicAppAdversarial(t, func() {
			for i := 0; i < 250; i++ {
				showList()
				showEditor(storage.Meeting{Title: "rapid-switch"})
				showList()
				showEditor(storage.Meeting{Title: "rapid-switch-2"})
			}
		})

		if got, want := len(meetingsContainer.Objects), 1; got != want {
			t.Fatalf("meetings container object count after rapid alternation = %d, want %d", got, want)
		}

		label := findLabelByExactTextAppAdversarial(meetingsContainer, "Editor: rapid-switch-2")
		if label == nil {
			t.Fatal("expected final editor label after rapid alternation")
		}
	})

	t.Run("showEditor with special and oversized title does not panic", func(t *testing.T) {
		longTitle := strings.Repeat("🚀<script>alert(1)</script>&amp;\x00", 400)

		assertNoPanicAppAdversarial(t, func() {
			showEditor(storage.Meeting{Title: longTitle})
		})

		if got, want := len(longTitle) > 10_000, true; got != want {
			t.Fatalf("oversized title condition = %t, want %t", got, want)
		}

		expected := "Editor: " + longTitle
		label := findLabelByExactTextAppAdversarial(meetingsContainer, expected)
		if label == nil {
			t.Fatal("expected editor label to preserve special/oversized title exactly")
		}
	})

	t.Run("back button from editor without prior selection returns list", func(t *testing.T) {
		showEditor(storage.Meeting{Title: "from-direct-call"})

		backButton := findButtonByTextAppAdversarial(meetingsContainer, "← Back")
		if backButton == nil {
			t.Fatal("expected back button in editor")
		}
		if backButton.OnTapped == nil {
			t.Fatal("expected back button tap handler")
		}

		assertNoPanicAppAdversarial(t, func() {
			backButton.OnTapped()
		})

		if got, want := len(meetingsContainer.Objects), 1; got != want {
			t.Fatalf("meetings container object count after back = %d, want %d", got, want)
		}

		list := findMeetingsList(meetingsContainer)
		if list == nil {
			t.Fatal("expected meetings list after tapping back")
		}
		if got, want := list.Length(), 0; got != want {
			t.Fatalf("meetings list length after tapping back = %d, want %d", got, want)
		}
	})

	t.Run("multiple showList calls create fresh list view each time", func(t *testing.T) {
		showList()
		if got, want := len(meetingsContainer.Objects), 1; got != want {
			t.Fatalf("meetings container object count after first showList = %d, want %d", got, want)
		}
		first := meetingsContainer.Objects[0]

		assertNoPanicAppAdversarial(t, func() {
			for i := 0; i < 100; i++ {
				showList()
			}
		})

		if got, want := len(meetingsContainer.Objects), 1; got != want {
			t.Fatalf("meetings container object count after repeated showList = %d, want %d", got, want)
		}

		last := meetingsContainer.Objects[0]
		if got, want := last == first, false; got != want {
			t.Fatalf("expected repeated showList to replace list view object: got same=%t want %t", got, want)
		}

		list := findMeetingsList(meetingsContainer)
		if list == nil {
			t.Fatal("expected meetings list after repeated showList")
		}
		if got, want := list.Length(), 0; got != want {
			t.Fatalf("meetings list length after repeated showList = %d, want %d", got, want)
		}
	})
}

func assertNoPanicAppAdversarial(t *testing.T, fn func()) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	fn()
}

func findLabelByExactTextAppAdversarial(obj fyne.CanvasObject, text string) *widget.Label {
	switch v := obj.(type) {
	case *widget.Label:
		if v.Text == text {
			return v
		}
	case *fyne.Container:
		for _, child := range v.Objects {
			if found := findLabelByExactTextAppAdversarial(child, text); found != nil {
				return found
			}
		}
	}

	return nil
}

func findButtonByTextAppAdversarial(obj fyne.CanvasObject, text string) *widget.Button {
	switch v := obj.(type) {
	case *widget.Button:
		if v.Text == text {
			return v
		}
	case *fyne.Container:
		for _, child := range v.Objects {
			if found := findButtonByTextAppAdversarial(child, text); found != nil {
				return found
			}
		}
	}

	return nil
}
