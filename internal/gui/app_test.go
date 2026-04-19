//go:build gui

package gui

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	fyneTest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/glieske/recap/internal/ai"
	"github.com/glieske/recap/internal/config"
	"github.com/glieske/recap/internal/storage"
)

func TestRunExportedWithExpectedSignature(t *testing.T) {
	var runFn func(*config.Config, *storage.Store, ai.Provider, string, string) = Run
	if runFn == nil {
		t.Fatal("expected Run to be exported with expected signature")
	}
}

func TestFyneAppCreation(t *testing.T) {
	headless := fyneTest.NewApp()
	if headless == nil {
		t.Fatal("expected headless fyne test app to be created")
	}
	headless.Quit()

	withID := app.NewWithID("io.recap.test")
	if withID == nil {
		t.Fatal("expected app.NewWithID to create a fyne app")
	}
	withID.Quit()
}

func TestAppTabsStructure(t *testing.T) {
	meetings := container.NewTabItem("Meetings", container.NewCenter(widget.NewLabel("Meetings will appear here")))
	settings := container.NewTabItem("Settings", container.NewCenter(widget.NewLabel("Settings will appear here")))

	tabs := container.NewAppTabs(meetings, settings)

	if got, want := len(tabs.Items), 2; got != want {
		t.Fatalf("expected %d tabs, got %d", want, got)
	}

	if got, want := tabs.Items[0].Text, "Meetings"; got != want {
		t.Fatalf("expected first tab text %q, got %q", want, got)
	}

	if got, want := tabs.Items[1].Text, "Settings"; got != want {
		t.Fatalf("expected second tab text %q, got %q", want, got)
	}
}

func TestAppTabsBoundaryWithNoItems(t *testing.T) {
	tabs := container.NewAppTabs()

	if got, want := len(tabs.Items), 0; got != want {
		t.Fatalf("expected %d tabs, got %d", want, got)
	}
}

func TestAppNavigationFlow_ListEditorBackAndNewMeeting(t *testing.T) {
	headless := fyneTest.NewApp()
	defer headless.Quit()

	store := storage.NewStore(filepath.Join(t.TempDir(), "notes"))
	if _, err := store.CreateProject("Infrastructure", "INFRA"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meetingDate := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	seedMeeting, err := store.CreateMeeting("Weekly sync", meetingDate, []string{"Alice"}, "INFRA", []string{"ops"}, "T-1")
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	win := headless.NewWindow("recap-test")
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

	createdMeeting := storage.Meeting{Title: "Created from dialog"}
	newMeetingDialogCalled := false
	showNewMeetingDialog := func(_ fyne.Window, _ *storage.Store, onCreate func(storage.Meeting)) {
		newMeetingDialogCalled = true
		onCreate(createdMeeting)
	}

	showList = func() {
		listView := NewMeetingsScreen(
			store,
			func(meeting storage.Meeting) {
				showEditor(meeting)
			},
			func() {
				showNewMeetingDialog(win, store, func(meeting storage.Meeting) {
					showEditor(meeting)
				})
			},
		)

		meetingsContainer.Objects = []fyne.CanvasObject{listView}
		meetingsContainer.Refresh()
	}

	showList()

	if got, want := len(meetingsContainer.Objects), 1; got != want {
		t.Fatalf("expected initial meetings container object count %d, got %d", want, got)
	}

	list := findMeetingsList(meetingsContainer)
	if list == nil {
		t.Fatal("expected meetings list to be shown initially")
	}
	if got, want := list.Length(), 1; got != want {
		t.Fatalf("expected initial meetings list length %d, got %d", want, got)
	}

	list.Select(0)

	editorLabel := findLabelByExactText(meetingsContainer, "Editor: "+seedMeeting.Title)
	if editorLabel == nil {
		t.Fatalf("expected editor placeholder label %q after selecting meeting", "Editor: "+seedMeeting.Title)
	}

	backButton := findButtonByText(meetingsContainer, "← Back")
	if backButton == nil {
		t.Fatal("expected back button in editor placeholder")
	}
	if backButton.OnTapped == nil {
		t.Fatal("expected back button tap handler")
	}

	backButton.OnTapped()

	listAfterBack := findMeetingsList(meetingsContainer)
	if listAfterBack == nil {
		t.Fatal("expected meetings list after tapping back")
	}

	newMeetingButton := findButtonByText(meetingsContainer, "+ New Meeting")
	if newMeetingButton == nil {
		t.Fatal("expected + New Meeting button after returning to list")
	}
	if newMeetingButton.OnTapped == nil {
		t.Fatal("expected + New Meeting button tap handler")
	}

	newMeetingButton.OnTapped()

	if got, want := newMeetingDialogCalled, true; got != want {
		t.Fatalf("expected new meeting dialog wiring called %t, got %t", want, got)
	}

	createdEditorLabel := findLabelByExactText(meetingsContainer, "Editor: "+createdMeeting.Title)
	if createdEditorLabel == nil {
		t.Fatalf("expected editor placeholder label %q after new meeting creation", "Editor: "+createdMeeting.Title)
	}
}

func TestVersionWindowTitleContainsProvidedVersion(t *testing.T) {
	headless := fyneTest.NewApp()
	defer headless.Quit()

	version := "1.0.0"
	w := headless.NewWindow(fmt.Sprintf("Recap v%s", version))

	if got, want := w.Title(), "Recap v1.0.0"; got != want {
		t.Fatalf("expected window title %q, got %q", want, got)
	}
}

func TestVersionWindowTitleUsesDevVersion(t *testing.T) {
	headless := fyneTest.NewApp()
	defer headless.Quit()

	version := "dev"
	w := headless.NewWindow(fmt.Sprintf("Recap v%s", version))

	if got, want := w.Title(), "Recap vdev"; got != want {
		t.Fatalf("expected window title %q, got %q", want, got)
	}
}

func TestVersionWindowTitleHandlesEmptyVersionGracefully(t *testing.T) {
	headless := fyneTest.NewApp()
	defer headless.Quit()

	version := ""
	w := headless.NewWindow(fmt.Sprintf("Recap v%s", version))

	if got, want := w.Title(), "Recap v"; got != want {
		t.Fatalf("expected window title %q, got %q", want, got)
	}
}

func findLabelByExactText(obj fyne.CanvasObject, text string) *widget.Label {
	switch v := obj.(type) {
	case *widget.Label:
		if v.Text == text {
			return v
		}
	case *fyne.Container:
		for _, child := range v.Objects {
			if found := findLabelByExactText(child, text); found != nil {
				return found
			}
		}
	}

	return nil
}
