//go:build gui

package gui

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fyneTest "fyne.io/fyne/v2/test"

	"github.com/glieske/recap/internal/ai"
	"github.com/glieske/recap/internal/config"
	"github.com/glieske/recap/internal/storage"
)

func TestEditorAdversarial(t *testing.T) {
	t.Run("nil store does not panic during construction", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		meeting := storage.Meeting{Title: "attack-nil-store"}

		screen := mustConstructEditorWithoutPanic(t, meeting, nil, nil, nil, nil, func() {})
		if screen.Store != nil {
			t.Fatalf("expected Store to be nil, got %v", screen.Store)
		}
		if got, want := screen.Meeting.Title, "attack-nil-store"; got != want {
			t.Fatalf("expected Meeting.Title %q, got %q", want, got)
		}
	})

	t.Run("nil onBack does not panic during construction", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		meeting := storage.Meeting{Title: "attack-nil-onback"}

		screen := mustConstructEditorWithoutPanic(t, meeting, nil, nil, nil, nil, nil)
		if screen == nil {
			t.Fatal("expected non-nil screen")
		}
		if got, want := screen.Meeting.Title, "attack-nil-onback"; got != want {
			t.Fatalf("expected Meeting.Title %q, got %q", want, got)
		}
		if back := findButtonByText(screen.Content, "← Back"); back == nil {
			t.Fatal("expected back button to be present")
		}
	})

	t.Run("zero-value meeting renders deterministic zero-state labels", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		meeting := storage.Meeting{}

		screen := mustConstructEditorWithoutPanic(t, meeting, nil, nil, nil, nil, func() {})
		if got, want := screen.Meeting.Date.Format("2006-01-02"), "0001-01-01"; got != want {
			t.Fatalf("expected zero date label %q, got %q", want, got)
		}
		if dateLabel := findLabelByText(screen.Content, "0001-01-01"); dateLabel == nil {
			t.Fatal("expected zero date label in toolbar")
		}
		if projectLabel := findLabelByText(screen.Content, ""); projectLabel == nil {
			t.Fatal("expected at least one empty label for zero-value project/status")
		}
	})

	t.Run("oversized title 10000 chars does not panic", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		longTitle := strings.Repeat("A", 10000)
		meeting := storage.Meeting{Title: longTitle, Project: "INFRA", Status: storage.MeetingStatusDraft}

		screen := mustConstructEditorWithoutPanic(t, meeting, nil, nil, nil, nil, func() {})
		if got, want := len(screen.Meeting.Title), 10000; got != want {
			t.Fatalf("expected title length %d, got %d", want, got)
		}
		if label := findLabelByText(screen.Content, longTitle); label == nil {
			t.Fatal("expected long title label to exist")
		}
	})

	t.Run("unicode and emoji in title and project does not panic", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		title := "🧪 研究会 — sprint 🚀"
		project := "プロジェクト✨"
		meeting := storage.Meeting{Title: title, Project: project, Status: storage.MeetingStatusDraft}

		screen := mustConstructEditorWithoutPanic(t, meeting, nil, nil, nil, nil, func() {})
		if got, want := screen.Meeting.Title, title; got != want {
			t.Fatalf("expected unicode title %q, got %q", want, got)
		}
		if got, want := screen.Meeting.Project, project; got != want {
			t.Fatalf("expected unicode project %q, got %q", want, got)
		}
		if label := findLabelByText(screen.Content, title); label == nil {
			t.Fatal("expected unicode title label to exist")
		}
		if label := findLabelByText(screen.Content, project); label == nil {
			t.Fatal("expected unicode project label to exist")
		}
	})

	t.Run("nil provider and nil window does not panic and structure button is disabled", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		meeting := storage.Meeting{Title: "nil-provider-nil-window"}

		screen := mustConstructEditorWithoutPanic(t, meeting, nil, nil, nil, nil, func() {})
		if got, want := screen.Window, fyne.Window(nil); got != want {
			t.Fatalf("expected Window to be nil, got %v", got)
		}

		structureButton := findButtonByText(screen.Content, "Structure Notes")
		if structureButton == nil {
			t.Fatal("expected Structure Notes button to exist")
		}
		if got, want := structureButton.Disabled(), true; got != want {
			t.Fatalf("expected Structure Notes disabled=%t, got %t", want, got)
		}
	})

	t.Run("provider returning error re-enables structure button", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		win := app.NewWindow("adversarial-error-provider")
		meeting := storage.Meeting{Title: "error-provider"}
		provider := &errorProvider{}

		screen := mustConstructEditorWithoutPanic(t, meeting, nil, provider, win, nil, func() {})
		structureButton := findButtonByText(screen.Content, "Structure Notes")
		if structureButton == nil {
			t.Fatal("expected Structure Notes button to exist")
		}

		screen.RawEntry.SetText("raw notes for structuring")
		fyneTest.Tap(structureButton)

		deadline := time.Now().Add(2 * time.Second)
		for structureButton.Disabled() && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}

		if got, want := structureButton.Disabled(), false; got != want {
			t.Fatalf("expected Structure Notes disabled=%t after provider error, got %t", want, got)
		}
	})

	t.Run("cancel during in-flight structuring handles cancelled context gracefully", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		win := app.NewWindow("adversarial-cancel")
		meeting := storage.Meeting{Title: "cancel-in-flight"}
		provider := newBlockingCancelAwareProvider()

		screen := mustConstructEditorWithoutPanic(t, meeting, nil, provider, win, nil, func() {})
		structureButton := findButtonByText(screen.Content, "Structure Notes")
		if structureButton == nil {
			t.Fatal("expected Structure Notes button to exist")
		}

		screen.RawEntry.SetText("long running structuring request")
		fyneTest.Tap(structureButton)

		select {
		case <-provider.started:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for provider.StructureNotes to start")
		}

		screen.Cancel()

		select {
		case <-provider.done:
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for provider.StructureNotes to finish after cancel")
		}

		deadline := time.Now().Add(2 * time.Second)
		for structureButton.Disabled() && time.Now().Before(deadline) {
			time.Sleep(10 * time.Millisecond)
		}

		if got, want := provider.cancelObserved.Load(), int32(1); got != want {
			t.Fatalf("expected cancelObserved=%d, got %d", want, got)
		}
		if got, want := structureButton.Disabled(), false; got != want {
			t.Fatalf("expected Structure Notes disabled=%t after cancellation, got %t", want, got)
		}
	})

	t.Run("empty and whitespace-only notes show info dialog and do not call provider", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		win := app.NewWindow("adversarial-whitespace")
		meeting := storage.Meeting{Title: "whitespace-only"}
		provider := &countingProvider{}

		screen := mustConstructEditorWithoutPanic(t, meeting, nil, provider, win, nil, func() {})
		structureButton := findButtonByText(screen.Content, "Structure Notes")
		if structureButton == nil {
			t.Fatal("expected Structure Notes button to exist")
		}

		screen.RawEntry.SetText("  \t\n  \n\t")
		fyneTest.Tap(structureButton)

		if got, want := provider.calls.Load(), int32(0); got != want {
			t.Fatalf("expected StructureNotes call count %d, got %d", want, got)
		}
		if got, want := structureButton.Disabled(), false; got != want {
			t.Fatalf("expected Structure Notes disabled=%t for whitespace-only input, got %t", want, got)
		}
		if overlay := win.Canvas().Overlays().Top(); overlay == nil {
			t.Fatal("expected info dialog overlay for whitespace-only notes")
		}
	})
}

type errorProvider struct{}

func (m *errorProvider) StructureNotes(ctx context.Context, rawNotes string, meta ai.MeetingMeta) (string, error) {
	return "", errors.New("API timeout")
}

func (m *errorProvider) GenerateEmailSummary(ctx context.Context, structuredMD string, language string) (string, error) {
	return "", errors.New("not implemented")
}

type blockingCancelAwareProvider struct {
	started         chan struct{}
	done            chan struct{}
	startedSignaled atomic.Bool
	cancelObserved  atomic.Int32
}

func newBlockingCancelAwareProvider() *blockingCancelAwareProvider {
	return &blockingCancelAwareProvider{
		started: make(chan struct{}),
		done:    make(chan struct{}),
	}
}

func (m *blockingCancelAwareProvider) StructureNotes(ctx context.Context, rawNotes string, meta ai.MeetingMeta) (string, error) {
	if m.startedSignaled.CompareAndSwap(false, true) {
		close(m.started)
	}
	defer close(m.done)

	<-ctx.Done()
	if errors.Is(ctx.Err(), context.Canceled) {
		m.cancelObserved.Store(1)
	}

	return "", ctx.Err()
}

func (m *blockingCancelAwareProvider) GenerateEmailSummary(ctx context.Context, structuredMD string, language string) (string, error) {
	return "", errors.New("not implemented")
}

type countingProvider struct {
	calls atomic.Int32
}

func (m *countingProvider) StructureNotes(ctx context.Context, rawNotes string, meta ai.MeetingMeta) (string, error) {
	m.calls.Add(1)
	return "structured", nil
}

func (m *countingProvider) GenerateEmailSummary(ctx context.Context, structuredMD string, language string) (string, error) {
	return "", errors.New("not implemented")
}

func mustConstructEditorWithoutPanic(t *testing.T, meeting storage.Meeting, store *storage.Store, provider ai.Provider, win fyne.Window, cfg *config.Config, onBack func()) *EditorScreen {
	t.Helper()

	var screen *EditorScreen
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected construction without panic, got panic: %v", r)
			}
		}()
		screen = NewEditorScreen(meeting, store, provider, win, cfg, onBack, nil)
	}()

	if screen == nil {
		t.Fatal("expected non-nil *EditorScreen")
	}

	return screen
}
