//go:build gui

package gui

import (
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	fyneTest "fyne.io/fyne/v2/test"

	"github.com/glieske/recap/internal/config"
	"github.com/glieske/recap/internal/storage"
)

func TestEditor52Adversarial(t *testing.T) {
	t.Run("xss-like provider name displays raw text", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		meeting := storage.Meeting{Title: "T", Date: time.Now(), Project: "P"}
		maliciousProvider := "<script>alert('xss')</script>"
		cfg := &config.Config{AIProvider: maliciousProvider}

		screen := NewEditorScreen(meeting, nil, nil, nil, cfg, func() {}, nil)
		if screen == nil {
			t.Fatal("expected non-nil *EditorScreen")
		}
		if screen.ProviderInfoLabel == nil {
			t.Fatal("expected ProviderInfoLabel to be non-nil")
		}

		if got, want := screen.ProviderInfoLabel.Text, maliciousProvider; got != want {
			t.Fatalf("expected provider label %q, got %q", want, got)
		}
	})

	t.Run("very long github model name keeps provider prefix", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		meeting := storage.Meeting{Title: "T", Date: time.Now(), Project: "P"}
		longModel := strings.Repeat("m", 10000)
		cfg := &config.Config{
			AIProvider:  "github_models",
			GitHubModel: longModel,
		}

		screen := NewEditorScreen(meeting, nil, nil, nil, cfg, func() {}, nil)
		if screen == nil {
			t.Fatal("expected non-nil *EditorScreen")
		}

		if got, want := strings.HasPrefix(screen.ProviderInfoLabel.Text, "github_models:"), true; got != want {
			t.Fatalf("expected provider label prefix present=%t, got %t", want, got)
		}
		if got, want := screen.ProviderInfoLabel.Text, "github_models: "+longModel; got != want {
			t.Fatalf("expected provider label length %d, got %d", len(want), len(got))
		}
	})

	t.Run("concurrent onStatus calls are stable", func(t *testing.T) {
		app := fyneTest.NewApp()
		defer app.Quit()

		meeting := storage.Meeting{Title: "T", Date: time.Now(), Project: "P"}

		var (
			mu       sync.Mutex
			captured = make([]string, 0, 128)
		)

		onStatus := func(msg string) {
			mu.Lock()
			captured = append(captured, msg)
			mu.Unlock()
		}

		screen := NewEditorScreen(meeting, nil, nil, nil, nil, func() {}, onStatus)
		if screen == nil {
			t.Fatal("expected non-nil *EditorScreen")
		}
		if screen.OnStatus == nil {
			t.Fatal("expected OnStatus to be set")
		}

		const goroutines = 128
		var wg sync.WaitGroup
		wg.Add(goroutines)

		for i := 0; i < goroutines; i++ {
			i := i
			go func() {
				defer wg.Done()
				screen.OnStatus("status-" + strconv.Itoa(i))
			}()
		}

		wg.Wait()

		mu.Lock()
		defer mu.Unlock()

		if got, want := len(captured), goroutines; got != want {
			t.Fatalf("expected captured message count %d, got %d", want, got)
		}

		seen := make(map[string]struct{}, goroutines)
		for _, msg := range captured {
			seen[msg] = struct{}{}
		}
		if got, want := len(seen), goroutines; got != want {
			t.Fatalf("expected unique captured message count %d, got %d", want, got)
		}
	})
}
