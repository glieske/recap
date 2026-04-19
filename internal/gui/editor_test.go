//go:build gui

package gui

import (
	"context"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	fyneTest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/glieske/recap/internal/ai"
	"github.com/glieske/recap/internal/config"
	"github.com/glieske/recap/internal/languages"
	"github.com/glieske/recap/internal/storage"
)

type mockProvider struct{}

func (m *mockProvider) StructureNotes(ctx context.Context, rawNotes string, meta ai.MeetingMeta) (string, error) {
	return "structured", nil
}

func (m *mockProvider) GenerateEmailSummary(ctx context.Context, structuredMD string, language string) (string, error) {
	return "email", nil
}

func TestEditorNewEditorScreenPopulatesStructAndLayout(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	meeting := storage.Meeting{
		Title:   "Weekly Sync",
		Date:    time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		Project: "INFRA",
		Status:  storage.MeetingStatusDraft,
	}

	screen := NewEditorScreen(meeting, nil, nil, nil, nil, func() {}, nil)
	if screen == nil {
		t.Fatal("expected NewEditorScreen to return non-nil *EditorScreen")
	}

	if got, want := screen.Meeting.Title, meeting.Title; got != want {
		t.Fatalf("expected Meeting.Title %q, got %q", want, got)
	}
	if got, want := screen.Meeting.Date, meeting.Date; !got.Equal(want) {
		t.Fatalf("expected Meeting.Date %v, got %v", want, got)
	}
	if got, want := screen.Meeting.Project, meeting.Project; got != want {
		t.Fatalf("expected Meeting.Project %q, got %q", want, got)
	}

	if screen.RawEntry == nil {
		t.Fatal("expected RawEntry to be non-nil")
	}
	if got, want := screen.RawEntry.PlaceHolder, "Type raw meeting notes here..."; got != want {
		t.Fatalf("expected RawEntry placeholder %q, got %q", want, got)
	}
	if got, want := screen.RawEntry.Wrapping, fyne.TextWrapWord; got != want {
		t.Fatalf("expected RawEntry wrapping %v, got %v", want, got)
	}

	if screen.Preview == nil {
		t.Fatal("expected Preview to be non-nil")
	}

	if screen.Content == nil {
		t.Fatal("expected Content to be non-nil")
	}

	backButton := findButtonByText(screen.Content, "← Back")
	if backButton == nil {
		t.Fatal("expected toolbar to include back button")
	}

	titleLabel := findLabelByText(screen.Content, meeting.Title)
	if titleLabel == nil {
		t.Fatalf("expected toolbar to include title label %q", meeting.Title)
	}
	if got, want := titleLabel.TextStyle.Bold, true; got != want {
		t.Fatalf("expected title label bold=%t, got %t", want, got)
	}

	dateLabel := findLabelByText(screen.Content, meeting.Date.Format("2006-01-02"))
	if dateLabel == nil {
		t.Fatalf("expected toolbar to include date label %q", meeting.Date.Format("2006-01-02"))
	}

	projectLabel := findLabelByText(screen.Content, meeting.Project)
	if projectLabel == nil {
		t.Fatalf("expected toolbar to include project label %q", meeting.Project)
	}

	statusLabel := findLabelByText(screen.Content, string(meeting.Status))
	if statusLabel == nil {
		t.Fatalf("expected toolbar to include status label %q", string(meeting.Status))
	}

	split := findSplit(screen.Content)
	if split == nil {
		t.Fatal("expected Content to include HSplit")
	}
	if got, want := split.Offset, 0.5; got != want {
		t.Fatalf("expected HSplit offset %v, got %v", want, got)
	}
	// Leading pane is a Stack wrapper containing RawEntry (for min-width enforcement)
	leadingStack, ok := split.Leading.(*fyne.Container)
	if !ok {
		t.Fatal("expected HSplit leading pane to be a container (Stack wrapper)")
	}
	foundRawEntry := false
	for _, obj := range leadingStack.Objects {
		if obj == screen.RawEntry {
			foundRawEntry = true
			break
		}
	}
	if !foundRawEntry {
		t.Fatal("expected HSplit leading Stack to contain RawEntry")
	}
	if got, want := split.Trailing, fyne.CanvasObject(screen.Preview); got != want {
		t.Fatalf("expected HSplit trailing pane to be Preview")
	}
}

func TestEditorStructureButtonExistsInToolbar(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	meeting := storage.Meeting{Title: "Structure Button Test"}
	screen := NewEditorScreen(meeting, nil, nil, nil, nil, func() {}, nil)

	structureButton := findButtonByText(screen.Content, "Structure Notes")
	if structureButton == nil {
		t.Fatal("expected toolbar to include Structure Notes button")
	}
}

func TestEditorStructureButtonDisabledWhenProviderNil(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	meeting := storage.Meeting{Title: "Nil Provider Test"}
	screen := NewEditorScreen(meeting, nil, nil, nil, nil, func() {}, nil)

	structureButton := findButtonByText(screen.Content, "Structure Notes")
	if structureButton == nil {
		t.Fatal("expected toolbar to include Structure Notes button")
	}

	if got, want := structureButton.Disabled(), true; got != want {
		t.Fatalf("expected Structure Notes button disabled=%t, got %t", want, got)
	}
}

func TestEditorStructureButtonEnabledWhenProviderPresent(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	meeting := storage.Meeting{Title: "Provider Present Test"}
	provider := &mockProvider{}
	screen := NewEditorScreen(meeting, nil, provider, nil, nil, func() {}, nil)

	structureButton := findButtonByText(screen.Content, "Structure Notes")
	if structureButton == nil {
		t.Fatal("expected toolbar to include Structure Notes button")
	}

	if got, want := structureButton.Disabled(), false; got != want {
		t.Fatalf("expected Structure Notes button disabled=%t, got %t", want, got)
	}
}

func TestEditorNewEditorScreenSetsCancelFunc(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	meeting := storage.Meeting{Title: "Cancel Func Test"}
	screen := NewEditorScreen(meeting, nil, nil, nil, nil, func() {}, nil)

	if screen.Cancel == nil {
		t.Fatal("expected Cancel to be non-nil")
	}
}

func TestEditorNewEditorScreenStoresProvider(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	meeting := storage.Meeting{Title: "Provider Field Test"}
	provider := &mockProvider{}
	screen := NewEditorScreen(meeting, nil, provider, nil, nil, func() {}, nil)

	if got, want := screen.Provider, ai.Provider(provider); got != want {
		t.Fatalf("expected Provider field to store provided instance")
	}
}

func findSplit(obj fyne.CanvasObject) *container.Split {
	switch v := obj.(type) {
	case *container.Split:
		return v
	case *fyne.Container:
		for _, child := range v.Objects {
			if found := findSplit(child); found != nil {
				return found
			}
		}
	}

	return nil
}

func findLabelByText(obj fyne.CanvasObject, text string) *widget.Label {
	switch v := obj.(type) {
	case *widget.Label:
		if v.Text == text {
			return v
		}
	case *fyne.Container:
		for _, child := range v.Objects {
			if found := findLabelByText(child, text); found != nil {
				return found
			}
		}
	}

	return nil
}

type recordingEmailProvider struct {
	mu             sync.Mutex
	structureValue string
	structureErr   error
	emailValue     string
	emailErr       error
	emailCalls     int
	lastStructured string
	lastLanguage   string
}

func (m *recordingEmailProvider) StructureNotes(ctx context.Context, rawNotes string, meta ai.MeetingMeta) (string, error) {
	if m.structureErr != nil {
		return "", m.structureErr
	}

	return m.structureValue, nil
}

func (m *recordingEmailProvider) GenerateEmailSummary(ctx context.Context, structuredMD string, language string) (string, error) {
	m.mu.Lock()
	m.emailCalls++
	m.lastStructured = structuredMD
	m.lastLanguage = language
	m.mu.Unlock()

	if m.emailErr != nil {
		return "", m.emailErr
	}

	return m.emailValue, nil
}

func (m *recordingEmailProvider) snapshot() (calls int, structured string, language string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.emailCalls, m.lastStructured, m.lastLanguage
}

func TestEditorEmailButtonExistsInToolbar(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	win := app.NewWindow("email-button")
	meeting := storage.Meeting{Title: "Email Button"}
	screen := NewEditorScreen(meeting, nil, &mockProvider{}, win, nil, func() {}, nil)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected toolbar to include Generate Email button")
	}
}

func TestEditorEmailDisabledByDefaultWithoutStructuredNotes(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	win := app.NewWindow("email-disabled-default")
	meeting := storage.Meeting{Title: "Email Default Disabled"}
	screen := NewEditorScreen(meeting, nil, &mockProvider{}, win, nil, func() {}, nil)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	if got, want := emailButton.Disabled(), true; got != want {
		t.Fatalf("expected Generate Email disabled=%t, got %t", want, got)
	}
}

func TestEditorEmailDisabledWhenProviderNilEvenWithStructuredNotes(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("TestProject", "PROJ"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	meeting, err := store.CreateMeeting(
		"Provider Nil",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"PROJ",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	structuredMD := "# Structured\n\n- item"
	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, structuredMD); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	win := app.NewWindow("provider-nil")
	screen := NewEditorScreen(*meeting, store, nil, win, nil, func() {}, nil)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	if got, want := emailButton.Disabled(), true; got != want {
		t.Fatalf("expected Generate Email disabled=%t when provider=nil, got %t", want, got)
	}
}

func TestEditorEmailEnabledAfterStructureNotesSuccess(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	provider := &recordingEmailProvider{structureValue: "# Structured"}
	win := app.NewWindow("email-enable-after-structure")
	meeting := storage.Meeting{Title: "Enable After Structure"}
	screen := NewEditorScreen(meeting, nil, provider, win, nil, func() {}, nil)
	win.SetContent(screen.Content)

	structureButton := findButtonByText(screen.Content, "Structure Notes")
	if structureButton == nil {
		t.Fatal("expected Structure Notes button")
	}

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	if got, want := emailButton.Disabled(), true; got != want {
		t.Fatalf("expected Generate Email initially disabled=%t, got %t", want, got)
	}

	screen.RawEntry.SetText("raw note content")
	fyneTest.Tap(structureButton)

	deadline := time.Now().Add(2 * time.Second)
	for emailButton.Disabled() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	if got, want := emailButton.Disabled(), false; got != want {
		t.Fatalf("expected Generate Email disabled=%t after structuring, got %t", want, got)
	}
}

func TestEditorEmailEnabledWhenStructuredNotesLoadedOnInit(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("TestProject", "PROJ"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	meeting, err := store.CreateMeeting(
		"Load Structured",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"PROJ",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	structuredMD := "# Already Structured"
	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, structuredMD); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	win := app.NewWindow("load-structured")
	screen := NewEditorScreen(*meeting, store, &mockProvider{}, win, nil, func() {}, nil)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	if got, want := emailButton.Disabled(), false; got != want {
		t.Fatalf("expected Generate Email disabled=%t when structured notes loaded, got %t", want, got)
	}
}

func TestEditorEmailGenerateCallsProviderAndSupportsClipboard(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("TestProject", "PROJ"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	meeting, err := store.CreateMeeting(
		"Generate Email",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"PROJ",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	structuredMD := "# Structured\n\n- point"
	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, structuredMD); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	provider := &recordingEmailProvider{emailValue: "Subject: Test\n\nBody"}
	win := app.NewWindow("email-generate")
	screen := NewEditorScreen(*meeting, store, provider, win, &config.Config{EmailLanguages: []string{"en", "pl", "no"}}, func() {}, nil)
	win.SetContent(screen.Content)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	fyneTest.Tap(emailButton)

	deadline := time.Now().Add(2 * time.Second)
	for {
		calls, gotStructured, gotLanguage := provider.snapshot()
		if calls == 1 {
			if gotStructured != structuredMD {
				t.Fatalf("expected structuredMD %q, got %q", structuredMD, gotStructured)
			}
			if gotLanguage != "en" {
				t.Fatalf("expected default language %q, got %q", "en", gotLanguage)
			}
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("expected GenerateEmailSummary to be called once, got %d", calls)
		}

		time.Sleep(10 * time.Millisecond)
	}

	// Verify email dialog appears with correct content in the overlay
	copyButton := waitForButtonInOverlay(win, "Copy to Clipboard", 2*time.Second)
	if copyButton == nil {
		t.Fatal("expected email preview dialog with Copy to Clipboard button")
	}

	// Find the MultiLineEntry in the overlay to verify email content
	overlay := win.Canvas().Overlays().Top()
	emailEntry := findMultiLineEntryInOverlay(overlay)
	if emailEntry == nil {
		t.Fatal("expected MultiLineEntry in email dialog")
	}

	if got, want := emailEntry.Text, "Subject: Test\n\nBody"; got != want {
		t.Fatalf("expected email entry text %q, got %q", want, got)
	}
}

func TestEditorEmailUsesConfigEmailLanguage(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("TestProject", "PROJ"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	meeting, err := store.CreateMeeting(
		"Email Language",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"PROJ",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	structuredMD := "# Structured for language"
	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, structuredMD); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	provider := &recordingEmailProvider{emailValue: "Email body"}
	win := app.NewWindow("email-language")
	cfg := &config.Config{EmailLanguages: []string{"en"}}
	screen := NewEditorScreen(*meeting, store, provider, win, cfg, func() {}, nil)
	win.SetContent(screen.Content)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	fyneTest.Tap(emailButton)

	deadline := time.Now().Add(2 * time.Second)
	for {
		calls, _, gotLanguage := provider.snapshot()
		if calls == 1 {
			if gotLanguage != "en" {
				t.Fatalf("expected config language %q, got %q", "en", gotLanguage)
			}
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("expected GenerateEmailSummary to be called once, got %d", calls)
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func findButtonInOverlay(obj fyne.CanvasObject, text string) *widget.Button {
	if btn, ok := obj.(*widget.Button); ok && btn.Text == text {
		return btn
	}

	if c, ok := obj.(*fyne.Container); ok {
		for _, child := range c.Objects {
			if found := findButtonInOverlay(child, text); found != nil {
				return found
			}
		}
	}

	if w, ok := obj.(fyne.Widget); ok {
		r := fyneTest.WidgetRenderer(w)
		if r != nil {
			for _, child := range r.Objects() {
				if found := findButtonInOverlay(child, text); found != nil {
					return found
				}
			}
		}
	}

	return nil
}

func findMultiLineEntryInOverlay(obj fyne.CanvasObject) *widget.Entry {
	if e, ok := obj.(*widget.Entry); ok && e.MultiLine {
		return e
	}

	if c, ok := obj.(*fyne.Container); ok {
		for _, child := range c.Objects {
			if found := findMultiLineEntryInOverlay(child); found != nil {
				return found
			}
		}
	}

	if w, ok := obj.(fyne.Widget); ok {
		r := fyneTest.WidgetRenderer(w)
		if r != nil {
			for _, child := range r.Objects() {
				if found := findMultiLineEntryInOverlay(child); found != nil {
					return found
				}
			}
		}
	}

	return nil
}

func waitForButtonInOverlay(win fyne.Window, label string, timeout time.Duration) *widget.Button {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		overlay := win.Canvas().Overlays().Top()
		if overlay != nil {
			if button := findButtonInOverlay(overlay, label); button != nil {
				return button
			}
		}

		time.Sleep(10 * time.Millisecond)
	}

	return nil
}

type blockingEmailProvider struct {
	mu           sync.Mutex
	startedCh    chan struct{}
	finishedCh   chan struct{}
	finishedOnce sync.Once
	emailCalls   int
}

func (m *blockingEmailProvider) StructureNotes(ctx context.Context, rawNotes string, meta ai.MeetingMeta) (string, error) {
	return "# Structured", nil
}

func (m *blockingEmailProvider) GenerateEmailSummary(ctx context.Context, structuredMD string, language string) (string, error) {
	m.mu.Lock()
	m.emailCalls++
	m.mu.Unlock()

	select {
	case m.startedCh <- struct{}{}:
	default:
	}

	<-ctx.Done()
	m.finishedOnce.Do(func() { close(m.finishedCh) })
	return "", ctx.Err()
}

func (m *blockingEmailProvider) snapshotCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.emailCalls
}

func TestEditorEmailAdversarialProviderErrorReEnablesButtonAndShowsErrorDialog(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("TestProject", "PROJ"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	meeting, err := store.CreateMeeting(
		"Email Provider Error",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"PROJ",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	structuredMD := "# Structured\n\n- point"
	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, structuredMD); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	provider := &recordingEmailProvider{emailErr: context.DeadlineExceeded}
	win := app.NewWindow("email-provider-error")
	screen := NewEditorScreen(*meeting, store, provider, win, &config.Config{EmailLanguages: []string{"en", "pl", "no"}}, func() {}, nil)
	win.SetContent(screen.Content)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	fyneTest.Tap(emailButton)

	callDeadline := time.Now().Add(2 * time.Second)
	for {
		calls, _, _ := provider.snapshot()
		if calls == 1 {
			break
		}
		if time.Now().After(callDeadline) {
			t.Fatalf("expected GenerateEmailSummary to be called once, got %d", calls)
		}
		time.Sleep(10 * time.Millisecond)
	}

	enableDeadline := time.Now().Add(2 * time.Second)
	for emailButton.Disabled() && time.Now().Before(enableDeadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got, want := emailButton.Disabled(), false; got != want {
		t.Fatalf("expected Generate Email disabled=%t after provider error, got %t", want, got)
	}

	okButton := waitForButtonInOverlay(win, "OK", 2*time.Second)
	if okButton == nil {
		t.Fatal("expected error dialog with OK button")
	}
}

func TestEditorEmailAdversarialNilConfigDefaultsLanguageToEnglish(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("TestProject", "PROJ"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	meeting, err := store.CreateMeeting(
		"Email Nil Config",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"PROJ",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	structuredMD := "# Structured"
	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, structuredMD); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	provider := &recordingEmailProvider{emailValue: "Subject: S\n\nBody"}
	win := app.NewWindow("email-nil-config")
	screen := NewEditorScreen(*meeting, store, provider, win, nil, func() {}, nil)
	win.SetContent(screen.Content)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	fyneTest.Tap(emailButton)

	deadline := time.Now().Add(2 * time.Second)
	for {
		calls, _, language := provider.snapshot()
		if calls == 1 {
			if got, want := language, "en"; got != want {
				t.Fatalf("expected default language %q for nil config, got %q", want, got)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected GenerateEmailSummary to be called once, got %d", calls)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestEditorEmailAdversarialEmptyConfigLanguageDefaultsLanguageToEnglish(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("TestProject", "PROJ"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	meeting, err := store.CreateMeeting(
		"Email Empty Language",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"PROJ",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, "# Structured"); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	provider := &recordingEmailProvider{emailValue: "Body"}
	win := app.NewWindow("email-empty-language")
	cfg := &config.Config{EmailLanguages: []string{}}
	screen := NewEditorScreen(*meeting, store, provider, win, cfg, func() {}, nil)
	win.SetContent(screen.Content)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	fyneTest.Tap(emailButton)

	deadline := time.Now().Add(2 * time.Second)
	for {
		calls, _, language := provider.snapshot()
		if calls == 1 {
			if got, want := language, "en"; got != want {
				t.Fatalf("expected default language %q when config language is empty, got %q", want, got)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected GenerateEmailSummary to be called once, got %d", calls)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestEditorEmailAdversarialEmptyStructuredNotesKeepsButtonDisabled(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("TestProject", "PROJ"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	meeting, err := store.CreateMeeting(
		"Empty Structured",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"PROJ",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, ""); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	win := app.NewWindow("email-empty-structured")
	screen := NewEditorScreen(*meeting, store, &mockProvider{}, win, nil, func() {}, nil)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	if got, want := emailButton.Disabled(), true; got != want {
		t.Fatalf("expected Generate Email disabled=%t when structured notes are empty, got %t", want, got)
	}
}

func TestEditorEmailAdversarialContextCancellationDuringGenerationNoPanic(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("TestProject", "PROJ"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	meeting, err := store.CreateMeeting(
		"Cancel Email Generation",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"PROJ",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, "# Structured"); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	provider := &blockingEmailProvider{
		startedCh:  make(chan struct{}, 1),
		finishedCh: make(chan struct{}),
	}

	backCalls := 0
	win := app.NewWindow("email-cancel")
	screen := NewEditorScreen(*meeting, store, provider, win, nil, func() { backCalls++ }, nil)
	win.SetContent(screen.Content)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}
	backButton := findButtonByText(screen.Content, "← Back")
	if backButton == nil {
		t.Fatal("expected back button")
	}

	fyneTest.Tap(emailButton)

	select {
	case <-provider.startedCh:
	case <-time.After(2 * time.Second):
		t.Fatal("expected GenerateEmailSummary call to start")
	}

	fyneTest.Tap(backButton)

	select {
	case <-provider.finishedCh:
	case <-time.After(2 * time.Second):
		t.Fatal("expected in-flight GenerateEmailSummary to finish after cancellation")
	}

	if got, want := provider.snapshotCalls(), 1; got != want {
		t.Fatalf("expected email generation call count %d, got %d", want, got)
	}
	if got, want := backCalls, 1; got != want {
		t.Fatalf("expected back callback calls=%d, got %d", want, got)
	}
}

func TestEditor43StructureFirstTimeSkipsConfirmationAndStructuresDirectly(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("TestProject", "PROJ"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	meeting, err := store.CreateMeeting(
		"First Structure",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"PROJ",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	provider := &recordingEmailProvider{structureValue: "# Structured"}
	win := app.NewWindow("editor43-first-structure")
	screen := NewEditorScreen(*meeting, store, provider, win, nil, func() {}, nil)
	win.SetContent(screen.Content)

	screen.RawEntry.SetText("raw notes to structure")
	structureButton := findButtonByText(screen.Content, "Structure Notes")
	if structureButton == nil {
		t.Fatal("expected Structure Notes button")
	}

	fyneTest.Tap(structureButton)
	time.Sleep(100 * time.Millisecond)

	if yes := waitForButtonInOverlay(win, "Yes", 150*time.Millisecond); yes != nil {
		t.Fatal("expected no re-structure confirmation dialog for first-time structuring")
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		structured, loadErr := store.LoadStructuredNotes(meeting.Project, meeting.ID)
		if loadErr == nil {
			if got, want := structured, "# Structured"; got != want {
				t.Fatalf("expected structured notes %q, got %q", want, got)
			}
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("expected structured notes to be saved, last error: %v", loadErr)
		}

		time.Sleep(10 * time.Millisecond)
	}
}

func TestEditor43StructureWithExistingShowsConfirmationDialog(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("TestProject", "PROJ"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	meeting, err := store.CreateMeeting(
		"Second Structure",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"PROJ",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, "# Existing Structured"); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	provider := &recordingEmailProvider{structureValue: "# Re-Structured"}
	win := app.NewWindow("editor43-confirm")
	screen := NewEditorScreen(*meeting, store, provider, win, nil, func() {}, nil)
	win.SetContent(screen.Content)

	screen.RawEntry.SetText("new raw notes")
	structureButton := findButtonByText(screen.Content, "Structure Notes")
	if structureButton == nil {
		t.Fatal("expected Structure Notes button")
	}

	fyneTest.Tap(structureButton)
	time.Sleep(100 * time.Millisecond)

	yes := waitForButtonInOverlay(win, "Yes", 1*time.Second)
	if yes == nil {
		t.Fatal("expected re-structure confirmation dialog with Yes button")
	}
	no := waitForButtonInOverlay(win, "No", 1*time.Second)
	if no == nil {
		t.Fatal("expected re-structure confirmation dialog with No button")
	}

	if got, wantErr := store.LoadStructuredNotes(meeting.Project, meeting.ID); wantErr == nil {
		if got != "# Existing Structured" {
			t.Fatalf("expected existing structured notes to remain unchanged before confirmation, got %q", got)
		}
	} else {
		t.Fatalf("LoadStructuredNotes returned error: %v", wantErr)
	}
}

func TestEditor43EmailDialogIncludesLanguageSelectWithExpectedOptions(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("TestProject", "PROJ"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	meeting, err := store.CreateMeeting(
		"Email Language Options",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"PROJ",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, "# Structured\n\n- point"); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	provider := &recordingEmailProvider{emailValue: "Subject: Demo\n\nBody"}
	win := app.NewWindow("editor43-email-options")
	screen := NewEditorScreen(*meeting, store, provider, win, &config.Config{EmailLanguages: []string{"en", "pl", "no"}}, func() {}, nil)
	win.SetContent(screen.Content)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	fyneTest.Tap(emailButton)
	time.Sleep(100 * time.Millisecond)

	copyButton := waitForButtonInOverlay(win, "Copy to Clipboard", 2*time.Second)
	if copyButton == nil {
		t.Fatal("expected email dialog to be shown")
	}

	overlay := win.Canvas().Overlays().Top()
	if overlay == nil {
		t.Fatal("expected overlay for email dialog")
	}

	langSelect := findSelectInOverlay(overlay)
	if langSelect == nil {
		t.Fatal("expected language select in email dialog")
	}

	if got, want := langSelect.Options, []string{"English", "Polish", "Norwegian"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected language options %v, got %v", want, got)
	}
}

func TestEditor43EmailDialogInitialLanguageSelectionMatchesConfig(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("TestProject", "PROJ"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	meeting, err := store.CreateMeeting(
		"Email Language Selected",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"PROJ",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, "# Structured\n\n- point"); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	provider := &recordingEmailProvider{emailValue: "Subject: Demo\n\nBody"}
	cfg := &config.Config{EmailLanguages: []string{"en"}}
	win := app.NewWindow("editor43-email-selected")
	screen := NewEditorScreen(*meeting, store, provider, win, cfg, func() {}, nil)
	win.SetContent(screen.Content)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	fyneTest.Tap(emailButton)
	time.Sleep(100 * time.Millisecond)

	copyButton := waitForButtonInOverlay(win, "Copy to Clipboard", 2*time.Second)
	if copyButton == nil {
		t.Fatal("expected email dialog to be shown")
	}

	overlay := win.Canvas().Overlays().Top()
	if overlay == nil {
		t.Fatal("expected overlay for email dialog")
	}

	langSelect := findSelectInOverlay(overlay)
	if langSelect == nil {
		t.Fatal("expected language select in email dialog")
	}

	if got, want := langSelect.Selected, "English"; got != want {
		t.Fatalf("expected selected language %q, got %q", want, got)
	}
}

func findSelectInOverlay(obj fyne.CanvasObject) *widget.Select {
	if sel, ok := obj.(*widget.Select); ok {
		return sel
	}

	if c, ok := obj.(*fyne.Container); ok {
		for _, child := range c.Objects {
			if found := findSelectInOverlay(child); found != nil {
				return found
			}
		}
	}

	if w, ok := obj.(fyne.Widget); ok {
		r := fyneTest.WidgetRenderer(w)
		if r != nil {
			for _, child := range r.Objects() {
				if found := findSelectInOverlay(child); found != nil {
					return found
				}
			}
		}
	}

	return nil
}

type structureCountingProvider struct {
	mu             sync.Mutex
	structureCalls int
}

func (m *structureCountingProvider) StructureNotes(ctx context.Context, rawNotes string, meta ai.MeetingMeta) (string, error) {
	m.mu.Lock()
	m.structureCalls++
	m.mu.Unlock()
	return "# Structured Once", nil
}

func (m *structureCountingProvider) GenerateEmailSummary(ctx context.Context, structuredMD string, language string) (string, error) {
	return "", nil
}

func (m *structureCountingProvider) snapshotStructureCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.structureCalls
}

type blockingRegenProvider struct {
	mu            sync.Mutex
	emailCalls    int
	languages     []string
	regenStarted  chan struct{}
	releaseRegen  chan struct{}
	regenFinished chan struct{}
}

func (m *blockingRegenProvider) StructureNotes(ctx context.Context, rawNotes string, meta ai.MeetingMeta) (string, error) {
	return "# Structured", nil
}

func (m *blockingRegenProvider) GenerateEmailSummary(ctx context.Context, structuredMD string, language string) (string, error) {
	m.mu.Lock()
	m.emailCalls++
	call := m.emailCalls
	m.languages = append(m.languages, language)
	m.mu.Unlock()

	if call == 1 {
		return "Subject: Initial\n\nBody", nil
	}

	select {
	case m.regenStarted <- struct{}{}:
	default:
	}

	select {
	case <-m.releaseRegen:
	case <-ctx.Done():
	}

	select {
	case m.regenFinished <- struct{}{}:
	default:
	}

	return "Subject: Regen\n\nBody", nil
}

func (m *blockingRegenProvider) snapshotEmailCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.emailCalls
}

type regenErrorProvider struct {
	mu         sync.Mutex
	emailCalls int
	languages  []string
}

func (m *regenErrorProvider) StructureNotes(ctx context.Context, rawNotes string, meta ai.MeetingMeta) (string, error) {
	return "# Structured", nil
}

func (m *regenErrorProvider) GenerateEmailSummary(ctx context.Context, structuredMD string, language string) (string, error) {
	m.mu.Lock()
	m.emailCalls++
	call := m.emailCalls
	m.languages = append(m.languages, language)
	m.mu.Unlock()

	if call == 1 {
		return "Subject: Previous\n\nBody", nil
	}

	return "", context.DeadlineExceeded
}

func TestEditor43AdversarialRapidDoubleTapStructureShowsSingleConfirmationAndSingleStructuringCall(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("TestProject", "PROJ"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	meeting, err := store.CreateMeeting(
		"Double Tap Re-Structure",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"PROJ",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, "# Existing"); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	provider := &structureCountingProvider{}
	win := app.NewWindow("editor43-adversarial-double-tap")
	screen := NewEditorScreen(*meeting, store, provider, win, nil, func() {}, nil)
	win.SetContent(screen.Content)
	screen.RawEntry.SetText("new raw notes")

	structureButton := findButtonByText(screen.Content, "Structure Notes")
	if structureButton == nil {
		t.Fatal("expected Structure Notes button")
	}

	fyneTest.Tap(structureButton)
	time.Sleep(100 * time.Millisecond)

	yes := waitForButtonInOverlay(win, "Yes", 1*time.Second)
	if yes == nil {
		t.Fatal("expected re-structure confirmation dialog")
	}

	fyneTest.Tap(yes)
	time.Sleep(200 * time.Millisecond)

	deadline := time.Now().Add(2 * time.Second)
	for {
		calls := provider.snapshotStructureCalls()
		if calls >= 1 {
			if got, want := calls, 1; got != want {
				t.Fatalf("expected StructureNotes call count %d, got %d", want, got)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected StructureNotes to be called once, got %d", calls)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestEditor43AdversarialLanguageSelectDisabledDuringInFlightRegeneration(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("TestProject", "PROJ"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	meeting, err := store.CreateMeeting(
		"In-Flight Regen",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"PROJ",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}
	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, "# Structured"); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	provider := &blockingRegenProvider{
		regenStarted:  make(chan struct{}, 1),
		releaseRegen:  make(chan struct{}),
		regenFinished: make(chan struct{}, 1),
	}
	win := app.NewWindow("editor43-adversarial-regen-disabled")
	screen := NewEditorScreen(*meeting, store, provider, win, nil, func() {}, nil)
	win.SetContent(screen.Content)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	fyneTest.Tap(emailButton)
	time.Sleep(100 * time.Millisecond)

	copyButton := waitForButtonInOverlay(win, "Copy to Clipboard", 2*time.Second)
	if copyButton == nil {
		t.Fatal("expected email dialog to be shown")
	}

	overlay := win.Canvas().Overlays().Top()
	if overlay == nil {
		t.Fatal("expected email dialog overlay")
	}
	langSelect := findSelectInOverlay(overlay)
	if langSelect == nil {
		t.Fatal("expected language select in email dialog")
	}

	langSelect.SetSelected("English")

	select {
	case <-provider.regenStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("expected regeneration call to start")
	}

	if got, want := langSelect.Disabled(), true; got != want {
		t.Fatalf("expected language select disabled=%t during regeneration, got %t", want, got)
	}

	for i := 0; i < 5; i++ {
		fyneTest.Tap(langSelect)
	}
	time.Sleep(100 * time.Millisecond)

	if got, want := provider.snapshotEmailCalls(), 2; got != want {
		t.Fatalf("expected GenerateEmailSummary call count %d (initial + one regen), got %d", want, got)
	}

	close(provider.releaseRegen)
	select {
	case <-provider.regenFinished:
	case <-time.After(2 * time.Second):
		t.Fatal("expected regeneration to finish")
	}
}

func TestEditor43AdversarialEmailRegenErrorReEnablesSelectShowsErrorAndKeepsPreviousEmail(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("TestProject", "PROJ"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	meeting, err := store.CreateMeeting(
		"Regen Error",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"PROJ",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}
	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, "# Structured"); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	provider := &regenErrorProvider{}
	win := app.NewWindow("editor43-adversarial-regen-error")
	screen := NewEditorScreen(*meeting, store, provider, win, nil, func() {}, nil)
	win.SetContent(screen.Content)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	fyneTest.Tap(emailButton)
	time.Sleep(100 * time.Millisecond)

	copyButton := waitForButtonInOverlay(win, "Copy to Clipboard", 2*time.Second)
	if copyButton == nil {
		t.Fatal("expected email dialog to be shown")
	}

	overlay := win.Canvas().Overlays().Top()
	if overlay == nil {
		t.Fatal("expected email dialog overlay")
	}

	emailEntry := findMultiLineEntryInOverlay(overlay)
	if emailEntry == nil {
		t.Fatal("expected email text entry in dialog")
	}
	if got, want := emailEntry.Text, "Subject: Previous\n\nBody"; got != want {
		t.Fatalf("expected initial email entry text %q, got %q", want, got)
	}

	langSelect := findSelectInOverlay(overlay)
	if langSelect == nil {
		t.Fatal("expected language select in email dialog")
	}

	langSelect.SetSelected("English")
	time.Sleep(100 * time.Millisecond)

	okButton := waitForButtonInOverlay(win, "OK", 2*time.Second)
	if okButton == nil {
		t.Fatal("expected error dialog with OK button after regeneration failure")
	}

	reenableDeadline := time.Now().Add(2 * time.Second)
	for langSelect.Disabled() && time.Now().Before(reenableDeadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got, want := langSelect.Disabled(), false; got != want {
		t.Fatalf("expected language select disabled=%t after regeneration error, got %t", want, got)
	}

	if got, want := emailEntry.Text, "Subject: Previous\n\nBody"; got != want {
		t.Fatalf("expected email entry text to remain %q after regen error, got %q", want, got)
	}

}

func TestCtrlS_SavesNotesImmediately(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("Test Project", "CTRL"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting(
		"Shortcut Save",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"CTRL",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	var statuses []string
	win := app.NewWindow("ctrl-s-save")
	screen := NewEditorScreen(*meeting, store, nil, win, nil, func() {}, func(msg string) {
		statuses = append(statuses, msg)
	})
	win.SetContent(screen.Content)

	expected := "manual save via ctrl+s"
	screen.RawEntry.SetText(expected)

	shortcut := &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl}
	shortcutable, ok := win.Canvas().(fyne.Shortcutable)
	if !ok {
		t.Fatal("window canvas does not implement fyne.Shortcutable")
	}
	shortcutable.TypedShortcut(shortcut)

	saved, err := store.LoadRawNotes(meeting.Project, meeting.ID)
	if err != nil {
		t.Fatalf("LoadRawNotes returned error: %v", err)
	}
	if got, want := saved, expected; got != want {
		t.Fatalf("expected raw notes %q after Ctrl+S, got %q", want, got)
	}

	if len(statuses) == 0 || statuses[len(statuses)-1] != "Notes saved" {
		t.Fatalf("expected final status %q, got %v", "Notes saved", statuses)
	}
}

func TestCtrlS_CmdS_SavesNotesImmediately(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("Test Project", "CMDS"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting(
		"Shortcut Save Cmd",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"CMDS",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	var statuses []string
	win := app.NewWindow("cmd-s-save")
	screen := NewEditorScreen(*meeting, store, nil, win, nil, func() {}, func(msg string) {
		statuses = append(statuses, msg)
	})
	win.SetContent(screen.Content)

	expected := "manual save via cmd+s"
	screen.RawEntry.SetText(expected)

	shortcut := &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierSuper}
	shortcutable, ok := win.Canvas().(fyne.Shortcutable)
	if !ok {
		t.Fatal("window canvas does not implement fyne.Shortcutable")
	}
	shortcutable.TypedShortcut(shortcut)

	saved, err := store.LoadRawNotes(meeting.Project, meeting.ID)
	if err != nil {
		t.Fatalf("LoadRawNotes returned error: %v", err)
	}
	if got, want := saved, expected; got != want {
		t.Fatalf("expected raw notes %q after Cmd+S, got %q", want, got)
	}

	if len(statuses) == 0 || statuses[len(statuses)-1] != "Notes saved" {
		t.Fatalf("expected final status %q, got %v", "Notes saved", statuses)
	}
}

func TestCtrlS_ReportsManualSaveFailedOnError(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("Test Project", "FAIL"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting(
		"Shortcut Save Error",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"FAIL",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	meetingDir := store.MeetingDir(meeting.Project, meeting.ID)
	if err := os.RemoveAll(meetingDir); err != nil {
		t.Fatalf("failed to prepare save failure scenario: %v", err)
	}

	var statuses []string
	win := app.NewWindow("ctrl-s-fail")
	screen := NewEditorScreen(*meeting, store, nil, win, nil, func() {}, func(msg string) {
		statuses = append(statuses, msg)
	})
	win.SetContent(screen.Content)

	screen.RawEntry.SetText("content that cannot be saved")

	shortcut := &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl}
	shortcutable, ok := win.Canvas().(fyne.Shortcutable)
	if !ok {
		t.Fatal("window canvas does not implement fyne.Shortcutable")
	}
	shortcutable.TypedShortcut(shortcut)

	if len(statuses) == 0 || statuses[len(statuses)-1] != "Manual save failed" {
		t.Fatalf("expected final status %q, got %v", "Manual save failed", statuses)
	}
}

func TestCtrlS_NilStoreIsNoOp(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	meeting := storage.Meeting{ID: "m1", Project: "CTRL", Title: "No Store"}
	var statusCalls int

	win := app.NewWindow("ctrl-s-nil-store")
	screen := NewEditorScreen(meeting, nil, nil, win, nil, func() {}, func(string) {
		statusCalls++
	})
	win.SetContent(screen.Content)
	screen.RawEntry.SetText("this should not be saved")

	shortcut := &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl}
	shortcutable, ok := win.Canvas().(fyne.Shortcutable)
	if !ok {
		t.Fatal("window canvas does not implement fyne.Shortcutable")
	}
	shortcutable.TypedShortcut(shortcut)

	if got, want := statusCalls, 0; got != want {
		t.Fatalf("expected onStatus calls %d when store=nil, got %d", want, got)
	}
}

func TestCtrlS_NilWindowDoesNotRegisterShortcut(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("Test Project", "NWIN"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting(
		"Nil Window",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"NWIN",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	var statuses []string
	screen := NewEditorScreen(*meeting, store, nil, nil, nil, func() {}, func(msg string) {
		statuses = append(statuses, msg)
	})

	if screen == nil {
		t.Fatal("expected NewEditorScreen to return non-nil screen when window is nil")
	}

	screen.RawEntry.SetText("changed without shortcut")
	saved, err := store.LoadRawNotes(meeting.Project, meeting.ID)
	if err != nil {
		t.Fatalf("LoadRawNotes returned error: %v", err)
	}
	if got, want := saved, ""; got != want {
		t.Fatalf("expected raw notes %q before autosave fires, got %q", want, got)
	}

	if len(statuses) != 0 {
		t.Fatalf("expected no status callbacks when window=nil and no shortcut can fire, got %v", statuses)
	}
}

func TestEditorDynamicLanguage_DefaultUsesFirstConfiguredLanguage(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("Test Project", "DLANG1"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting(
		"Dynamic Language Default",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"DLANG1",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, "# Structured"); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	provider := &recordingEmailProvider{emailValue: "Email body"}
	cfg := &config.Config{EmailLanguages: []string{"pl", "de"}}
	win := app.NewWindow("dynamic-language-default")
	screen := NewEditorScreen(*meeting, store, provider, win, cfg, func() {}, nil)
	win.SetContent(screen.Content)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	fyneTest.Tap(emailButton)

	deadline := time.Now().Add(2 * time.Second)
	for {
		calls, _, language := provider.snapshot()
		if calls == 1 {
			if got, want := language, "pl"; got != want {
				t.Fatalf("expected first configured language %q, got %q", want, got)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected GenerateEmailSummary to be called once, got %d", calls)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestEditorDynamicLanguage_SingleConfiguredLanguageDropdownOnlyEnglish(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("Test Project", "DLANG2"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting(
		"Single Configured Language",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"DLANG2",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, "# Structured"); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	provider := &recordingEmailProvider{emailValue: "Email body"}
	cfg := &config.Config{EmailLanguages: []string{"en"}}
	win := app.NewWindow("dynamic-language-single")
	screen := NewEditorScreen(*meeting, store, provider, win, cfg, func() {}, nil)
	win.SetContent(screen.Content)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	fyneTest.Tap(emailButton)

	copyButton := waitForButtonInOverlay(win, "Copy to Clipboard", 2*time.Second)
	if copyButton == nil {
		t.Fatal("expected email dialog to be shown")
	}

	overlay := win.Canvas().Overlays().Top()
	if overlay == nil {
		t.Fatal("expected overlay for email dialog")
	}

	langSelect := findSelectInOverlay(overlay)
	if langSelect == nil {
		t.Fatal("expected language select in email dialog")
	}

	if got, want := langSelect.Options, []string{"English"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected language options %v, got %v", want, got)
	}

	if got, want := langSelect.Selected, "English"; got != want {
		t.Fatalf("expected selected language %q, got %q", want, got)
	}
}

func TestEditorDynamicLanguage_NilConfigFallsBackToEnglishCode(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("Test Project", "DLANG3"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting(
		"Nil Config Fallback",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"DLANG3",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, "# Structured"); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	provider := &recordingEmailProvider{emailValue: "Email body"}
	win := app.NewWindow("dynamic-language-nil-config")
	screen := NewEditorScreen(*meeting, store, provider, win, nil, func() {}, nil)
	win.SetContent(screen.Content)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	fyneTest.Tap(emailButton)

	deadline := time.Now().Add(2 * time.Second)
	for {
		calls, _, language := provider.snapshot()
		if calls == 1 {
			if got, want := language, "en"; got != want {
				t.Fatalf("expected fallback language %q for nil cfg, got %q", want, got)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected GenerateEmailSummary to be called once, got %d", calls)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestEditorDynamicLanguage_EmptyEmailLanguagesFallsBackToEnglishCode(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("Test Project", "DLANG4"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting(
		"Empty Config Fallback",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"DLANG4",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, "# Structured"); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	provider := &recordingEmailProvider{emailValue: "Email body"}
	cfg := &config.Config{EmailLanguages: []string{}}
	win := app.NewWindow("dynamic-language-empty-config")
	screen := NewEditorScreen(*meeting, store, provider, win, cfg, func() {}, nil)
	win.SetContent(screen.Content)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	fyneTest.Tap(emailButton)

	deadline := time.Now().Add(2 * time.Second)
	for {
		calls, _, language := provider.snapshot()
		if calls == 1 {
			if got, want := language, "en"; got != want {
				t.Fatalf("expected fallback language %q for empty EmailLanguages, got %q", want, got)
			}
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("expected GenerateEmailSummary to be called once, got %d", calls)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestEditorDynamicLanguage_MaxConfiguredLanguagesAppearInDropdown(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("Test Project", "DLANG5"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting(
		"Max Configured Languages",
		time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		nil,
		"DLANG5",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	if err := store.SaveStructuredNotes(meeting.Project, meeting.ID, "# Structured"); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	codes := []string{"ja", "zh", "es", "fr", "ar"}
	provider := &recordingEmailProvider{emailValue: "Email body"}
	cfg := &config.Config{EmailLanguages: codes}
	win := app.NewWindow("dynamic-language-max-config")
	screen := NewEditorScreen(*meeting, store, provider, win, cfg, func() {}, nil)
	win.SetContent(screen.Content)

	emailButton := findButtonByText(screen.Content, "Generate Email")
	if emailButton == nil {
		t.Fatal("expected Generate Email button")
	}

	fyneTest.Tap(emailButton)

	copyButton := waitForButtonInOverlay(win, "Copy to Clipboard", 2*time.Second)
	if copyButton == nil {
		t.Fatal("expected email dialog to be shown")
	}

	overlay := win.Canvas().Overlays().Top()
	if overlay == nil {
		t.Fatal("expected overlay for email dialog")
	}

	langSelect := findSelectInOverlay(overlay)
	if langSelect == nil {
		t.Fatal("expected language select in email dialog")
	}

	wantNames := []string{
		languages.DisplayName("ja"),
		languages.DisplayName("zh"),
		languages.DisplayName("es"),
		languages.DisplayName("fr"),
		languages.DisplayName("ar"),
	}
	if got := len(langSelect.Options); got != 5 {
		t.Fatalf("expected 5 language options, got %d (%v)", got, langSelect.Options)
	}
	if got := langSelect.Options; !reflect.DeepEqual(got, wantNames) {
		t.Fatalf("expected language options %v, got %v", wantNames, got)
	}
}
