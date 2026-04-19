//go:build gui

package gui

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"fyne.io/fyne/v2"
	fyneTest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/glieske/recap/internal/storage"
)

func TestDeleteButtonExistsInEditorScreen(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	meeting := storage.Meeting{
		ID:      "delete-exists-id",
		Title:   "Delete Exists",
		Project: "DEL",
		Date:    time.Now(),
	}

	screen := NewEditorScreen(meeting, nil, nil, nil, nil, func() {}, nil)
	if screen.DeleteButton == nil {
		t.Fatal("expected DeleteButton to be non-nil")
	}
	if got, want := screen.DeleteButton.Text, "🗑 Delete"; got != want {
		t.Fatalf("expected DeleteButton text %q, got %q", want, got)
	}
}

func TestDeleteNilStoreIsNoOp(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	win := app.NewWindow("delete-nil-store")
	defer win.Close()

	meeting := storage.Meeting{
		ID:      "delete-nil-store-id",
		Title:   "Nil Store",
		Project: "DEL",
		Date:    time.Now(),
	}

	backCalls := 0
	statusCalls := 0
	screen := NewEditorScreen(meeting, nil, nil, win, nil, func() {
		backCalls++
	}, func(string) {
		statusCalls++
	})
	win.SetContent(screen.Content)

	screen.DeleteButton.OnTapped()
	time.Sleep(50 * time.Millisecond)

	if got, want := backCalls, 0; got != want {
		t.Fatalf("expected onBack calls %d, got %d", want, got)
	}
	if got, want := statusCalls, 0; got != want {
		t.Fatalf("expected onStatus calls %d, got %d", want, got)
	}
	if overlay := win.Canvas().Overlays().Top(); overlay != nil {
		t.Fatal("expected no confirmation dialog overlay when store is nil")
	}
}

func TestDeleteSuccessCallsDeleteMeetingAndOnBack(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	if _, err := store.CreateProject("Delete Success", "DEL"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting(
		"Delete Success Meeting",
		time.Now(),
		nil,
		"DEL",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	meetingDir := store.MeetingDir(meeting.Project, meeting.ID)
	if _, statErr := os.Stat(meetingDir); statErr != nil {
		t.Fatalf("expected meeting directory to exist before delete, got error: %v", statErr)
	}

	win := app.NewWindow("delete-success")
	defer win.Close()

	backCalls := 0
	var statuses []string
	screen := NewEditorScreen(*meeting, store, nil, win, nil, func() {
		backCalls++
	}, func(msg string) {
		statuses = append(statuses, msg)
	})
	win.SetContent(screen.Content)

	fyneTest.Tap(screen.DeleteButton)
	yes := deleteWaitForButtonInOverlay(win, "Yes", 2*time.Second)
	if yes == nil {
		t.Fatal("expected delete confirmation dialog with Yes button")
	}
	fyneTest.Tap(yes)
	time.Sleep(100 * time.Millisecond)

	if _, statErr := os.Stat(meetingDir); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected meeting directory to be deleted, stat error: %v", statErr)
	}
	if got, want := backCalls, 1; got != want {
		t.Fatalf("expected onBack calls %d, got %d", want, got)
	}
	if len(statuses) == 0 || statuses[len(statuses)-1] != "Meeting deleted" {
		t.Fatalf("expected final status %q, got %v", "Meeting deleted", statuses)
	}
}

func TestDeleteErrorShowsDialogAndReportsStatus(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	if _, err := store.CreateProject("Delete Error", "DELERR"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting(
		"Delete Error Meeting",
		time.Now(),
		nil,
		"DELERR",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	projectDir := filepath.Join(tempDir, "DELERR")
	if chmodErr := os.Chmod(projectDir, 0o555); chmodErr != nil {
		t.Fatalf("failed to set project dir read-only: %v", chmodErr)
	}
	defer func() {
		_ = os.Chmod(projectDir, 0o755)
	}()

	win := app.NewWindow("delete-error")
	defer win.Close()

	backCalls := 0
	var statuses []string
	screen := NewEditorScreen(*meeting, store, nil, win, nil, func() {
		backCalls++
	}, func(msg string) {
		statuses = append(statuses, msg)
	})
	win.SetContent(screen.Content)

	fyneTest.Tap(screen.DeleteButton)
	yes := deleteWaitForButtonInOverlay(win, "Yes", 2*time.Second)
	if yes == nil {
		t.Fatal("expected delete confirmation dialog with Yes button")
	}
	fyneTest.Tap(yes)

	ok := deleteWaitForButtonInOverlay(win, "OK", 2*time.Second)
	if ok == nil {
		t.Fatal("expected error dialog with OK button when delete fails")
	}

	if got, want := backCalls, 0; got != want {
		t.Fatalf("expected onBack calls %d on delete error, got %d", want, got)
	}
	if len(statuses) == 0 || statuses[len(statuses)-1] != "Delete failed" {
		t.Fatalf("expected final status %q on delete error, got %v", "Delete failed", statuses)
	}
}

func TestDeleteCancelDoesNothing(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	if _, err := store.CreateProject("Delete Cancel", "DCAN"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting(
		"Delete Cancel Meeting",
		time.Now(),
		nil,
		"DCAN",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	meetingDir := store.MeetingDir(meeting.Project, meeting.ID)

	win := app.NewWindow("delete-cancel")
	defer win.Close()

	backCalls := 0
	var statuses []string
	screen := NewEditorScreen(*meeting, store, nil, win, nil, func() {
		backCalls++
	}, func(msg string) {
		statuses = append(statuses, msg)
	})
	win.SetContent(screen.Content)

	fyneTest.Tap(screen.DeleteButton)
	no := deleteWaitForButtonInOverlay(win, "No", 2*time.Second)
	if no == nil {
		t.Fatal("expected delete confirmation dialog with No button")
	}
	fyneTest.Tap(no)
	time.Sleep(100 * time.Millisecond)

	if _, statErr := os.Stat(meetingDir); statErr != nil {
		t.Fatalf("expected meeting directory to remain after cancel, got error: %v", statErr)
	}
	if got, want := backCalls, 0; got != want {
		t.Fatalf("expected onBack calls %d on cancel, got %d", want, got)
	}
	if got, want := len(statuses), 0; got != want {
		t.Fatalf("expected status callback count %d on cancel, got %d (%v)", want, got, statuses)
	}
}

func TestDeleteErrorPathTraversalProjectShowsErrorAndReportsStatus(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)

	notesFile := filepath.Join(tempDir, "notes-file")
	if err := os.WriteFile(notesFile, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	deleteSetUnexportedStringField(t, store, "notesDir", notesFile)

	meeting := storage.Meeting{
		ID:      "delete-path-traversal",
		Title:   "Traversal",
		Project: "../bad",
		Date:    time.Now(),
	}

	win := app.NewWindow("delete-path-traversal")
	defer win.Close()

	backCalls := 0
	var statuses []string
	screen := NewEditorScreen(meeting, store, nil, win, nil, func() {
		backCalls++
	}, func(msg string) {
		statuses = append(statuses, msg)
	})
	win.SetContent(screen.Content)

	fyneTest.Tap(screen.DeleteButton)
	yes := deleteWaitForButtonInOverlay(win, "Yes", 2*time.Second)
	if yes == nil {
		t.Fatal("expected delete confirmation dialog with Yes button")
	}
	fyneTest.Tap(yes)

	ok := deleteWaitForButtonInOverlay(win, "OK", 2*time.Second)
	if ok == nil {
		t.Fatal("expected error dialog with OK button on traversal-induced delete failure")
	}

	if got, want := backCalls, 0; got != want {
		t.Fatalf("expected onBack calls %d on delete error, got %d", want, got)
	}
	if len(statuses) == 0 || statuses[len(statuses)-1] != "Delete failed" {
		t.Fatalf("expected final status %q on delete error, got %v", "Delete failed", statuses)
	}
}

func deleteFindButtonInOverlay(obj fyne.CanvasObject, text string) *widget.Button {
	if btn, ok := obj.(*widget.Button); ok && btn.Text == text {
		return btn
	}

	if c, ok := obj.(*fyne.Container); ok {
		for _, child := range c.Objects {
			if found := deleteFindButtonInOverlay(child, text); found != nil {
				return found
			}
		}
	}

	if w, ok := obj.(fyne.Widget); ok {
		r := fyneTest.WidgetRenderer(w)
		if r != nil {
			for _, child := range r.Objects() {
				if found := deleteFindButtonInOverlay(child, text); found != nil {
					return found
				}
			}
		}
	}

	return nil
}

func deleteWaitForButtonInOverlay(win fyne.Window, label string, timeout time.Duration) *widget.Button {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		overlay := win.Canvas().Overlays().Top()
		if overlay != nil {
			if button := deleteFindButtonInOverlay(overlay, label); button != nil {
				return button
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

func deleteSetUnexportedStringField(t *testing.T, target any, fieldName string, value string) {
	t.Helper()
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		t.Fatalf("target must be non-nil pointer, got %T", target)
	}

	elem := v.Elem()
	field := elem.FieldByName(fieldName)
	if !field.IsValid() {
		t.Fatalf("field %q not found", fieldName)
	}
	if field.Kind() != reflect.String {
		t.Fatalf("field %q is not string", fieldName)
	}

	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().SetString(value)
}
