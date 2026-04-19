//go:build gui

package gui

import (
	"testing"
	"time"

	fyneTest "fyne.io/fyne/v2/test"

	"github.com/glieske/recap/internal/ai"
	"github.com/glieske/recap/internal/config"
	"github.com/glieske/recap/internal/storage"
)

func testEditor52Meeting() storage.Meeting {
	return storage.Meeting{Title: "T", Date: time.Now(), Project: "P"}
}

func TestEditor52ProviderLabelGitHubModels(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	cfg := &config.Config{AIProvider: "github_models", GitHubModel: "gpt-4o"}
	screen := NewEditorScreen(testEditor52Meeting(), nil, nil, nil, cfg, func() {}, nil)

	if screen.ProviderInfoLabel == nil {
		t.Fatal("expected ProviderInfoLabel to be non-nil")
	}

	if got, want := screen.ProviderInfoLabel.Text, "github_models: gpt-4o"; got != want {
		t.Fatalf("expected provider label %q, got %q", want, got)
	}
}

func TestEditor52ProviderLabelNilConfig(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	screen := NewEditorScreen(testEditor52Meeting(), nil, nil, nil, nil, func() {}, nil)

	if got, want := screen.ProviderInfoLabel.Text, "No AI provider"; got != want {
		t.Fatalf("expected provider label %q, got %q", want, got)
	}
}

func TestEditor52ProviderLabelEmptyProvider(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	cfg := &config.Config{AIProvider: ""}
	screen := NewEditorScreen(testEditor52Meeting(), nil, nil, nil, cfg, func() {}, nil)

	if got, want := screen.ProviderInfoLabel.Text, "No AI provider"; got != want {
		t.Fatalf("expected provider label %q, got %q", want, got)
	}
}

func TestEditor52ProviderLabelUnknownProviderRaw(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	cfg := &config.Config{AIProvider: "custom_provider"}
	screen := NewEditorScreen(testEditor52Meeting(), nil, nil, nil, cfg, func() {}, nil)

	if got, want := screen.ProviderInfoLabel.Text, "custom_provider"; got != want {
		t.Fatalf("expected provider label %q, got %q", want, got)
	}
}

func TestEditor52OnStatusFieldSetAndCapturesMessages(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	var captured []string
	onStatus := func(msg string) {
		captured = append(captured, msg)
	}

	screen := NewEditorScreen(testEditor52Meeting(), nil, nil, nil, nil, func() {}, onStatus)

	if screen.OnStatus == nil {
		t.Fatal("expected OnStatus field to be set")
	}

	screen.OnStatus("Notes auto-saved")

	if got, want := len(captured), 1; got != want {
		t.Fatalf("expected captured message count %d, got %d", want, got)
	}
	if got, want := captured[0], "Notes auto-saved"; got != want {
		t.Fatalf("expected captured message %q, got %q", want, got)
	}
}

func TestEditor52NilOnStatusDoesNotPanic(t *testing.T) {
	app := fyneTest.NewApp()
	defer app.Quit()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected no panic with nil onStatus, got panic: %v", r)
		}
	}()

	_ = NewEditorScreen(testEditor52Meeting(), nil, nil, nil, nil, func() {}, nil)
}

func TestEditor52RunExportedSignature(t *testing.T) {
	var runFn func(*config.Config, *storage.Store, ai.Provider, string, string) = Run
	if runFn == nil {
		t.Fatal("expected Run to be exported with expected signature")
	}
}
