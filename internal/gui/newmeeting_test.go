//go:build gui

package gui

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"testing/quick"

	"fyne.io/fyne/v2"
	fyneTest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/glieske/recap/internal/storage"
)

func TestNewMeetingSplitAndTrimCases(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected []string
	}{
		{name: "empty string", input: "", expected: nil},
		{name: "blank string", input: "   \t\n", expected: nil},
		{name: "single value", input: "Alice", expected: []string{"Alice"}},
		{name: "normal csv", input: "Alice, Bob, Charlie", expected: []string{"Alice", "Bob", "Charlie"}},
		{name: "trailing comma", input: "Alice, Bob,", expected: []string{"Alice", "Bob"}},
		{name: "leading and repeated commas", input: ", Alice,,Bob , ,Charlie,", expected: []string{"Alice", "Bob", "Charlie"}},
		{name: "unicode values", input: "😀, zażółć, café", expected: []string{"😀", "zażółć", "café"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitAndTrim(tc.input)
			if !reflect.DeepEqual(got, tc.expected) {
				t.Fatalf("splitAndTrim(%q) = %#v, want %#v", tc.input, got, tc.expected)
			}
		})
	}
}

func TestNewMeetingSplitAndTrimIdempotentProperty(t *testing.T) {
	property := func(input string) bool {
		first := splitAndTrim(input)
		second := splitAndTrim(strings.Join(first, ", "))
		return reflect.DeepEqual(second, first)
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 250}); err != nil {
		t.Fatalf("splitAndTrim idempotency property failed: %v", err)
	}
}

func TestNewMeetingShowDialogNoPanicWithHeadlessWindow(t *testing.T) {
	app := fyneTest.NewApp()
	t.Cleanup(app.Quit)

	win := app.NewWindow("New Meeting")

	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		ShowNewMeetingDialog(win, nil, nil)
	}()

	if panicked {
		t.Fatal("ShowNewMeetingDialog panicked for headless window")
	}
}

func TestNewMeetingShowDialogLoadsProjectOptionsFromStore(t *testing.T) {
	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("Alpha", "ALPHA"); err != nil {
		t.Fatalf("CreateProject ALPHA returned error: %v", err)
	}
	if _, err := store.CreateProject("Beta", "BETA"); err != nil {
		t.Fatalf("CreateProject BETA returned error: %v", err)
	}

	app := fyneTest.NewApp()
	t.Cleanup(app.Quit)

	win := app.NewWindow("New Meeting")

	panicked := false
	func() {
		defer func() {
			if recover() != nil {
				panicked = true
			}
		}()
		ShowNewMeetingDialog(win, store, nil)
	}()
	if panicked {
		t.Fatal("ShowNewMeetingDialog panicked with valid store")
	}

	topOverlay := win.Canvas().Overlays().Top()
	if topOverlay == nil {
		t.Fatal("expected dialog overlay to be present")
	}

	projectSelect := findSelectInObject(topOverlay)
	if projectSelect == nil {
		t.Fatal("expected project select widget in dialog")
	}

	options := append([]string(nil), projectSelect.Options...)
	sort.Strings(options)
	expected := []string{"ALPHA", "BETA"}
	if !reflect.DeepEqual(options, expected) {
		t.Fatalf("project select options = %v, want %v", options, expected)
	}
}

func findSelectInObject(obj fyne.CanvasObject) *widget.Select {
	if obj == nil {
		return nil
	}

	if sel, ok := obj.(*widget.Select); ok {
		return sel
	}

	if c, ok := obj.(*fyne.Container); ok {
		for _, child := range c.Objects {
			if found := findSelectInObject(child); found != nil {
				return found
			}
		}
	}

	if w, ok := obj.(fyne.Widget); ok {
		r := fyneTest.WidgetRenderer(w)
		if r != nil {
			for _, child := range r.Objects() {
				if found := findSelectInObject(child); found != nil {
					return found
				}
			}
		}
	}

	return nil
}
