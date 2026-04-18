package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/glieske/recap/internal/ai"
)

func TestCommandsAdversarial(t *testing.T) {
	t.Run("very large AI response 100KB parses without panic", func(t *testing.T) {
		largeBody := strings.Repeat("A", 100*1024)
		subject, body := parseEmailResponse("Subject: Large\n\n" + largeBody)

		if subject != "Large" {
			t.Fatalf("expected subject %q, got %q", "Large", subject)
		}
		if len(body) != len(largeBody) {
			t.Fatalf("expected body length %d, got %d", len(largeBody), len(body))
		}
		if body != largeBody {
			t.Fatalf("expected body to match full payload")
		}
	})

	t.Run("response with only Subject colon space does not panic", func(t *testing.T) {
		subject, body := parseEmailResponse("Subject: ")

		if subject != "Meeting Summary" {
			t.Fatalf("expected default subject %q, got %q", "Meeting Summary", subject)
		}
		if body != "Subject:" {
			t.Fatalf("expected body %q, got %q", "Subject:", body)
		}
	})

	t.Run("response with many blank lines finds body", func(t *testing.T) {
		response := "Subject: Status\n\n\n\n\nBody starts here"
		subject, body := parseEmailResponse(response)

		if subject != "Status" {
			t.Fatalf("expected subject %q, got %q", "Status", subject)
		}
		if body != "Body starts here" {
			t.Fatalf("expected body %q, got %q", "Body starts here", body)
		}
	})

	t.Run("unicode and emoji in subject parses correctly", func(t *testing.T) {
		response := "Subject: 🚀 Sprint 日本語 ✅\n\nBody"
		subject, body := parseEmailResponse(response)

		if subject != "🚀 Sprint 日本語 ✅" {
			t.Fatalf("expected unicode subject, got %q", subject)
		}
		if body != "Body" {
			t.Fatalf("expected body %q, got %q", "Body", body)
		}
	})

	t.Run("whitespace-only response returns defaults", func(t *testing.T) {
		subject, body := parseEmailResponse("   \n\t   \n")

		if subject != "Meeting Summary" {
			t.Fatalf("expected default subject %q, got %q", "Meeting Summary", subject)
		}
		if body != "" {
			t.Fatalf("expected empty body, got %q", body)
		}
	})

	t.Run("nil provider error field produces done message without nil dereference", func(t *testing.T) {
		provider := &mockProvider{emailResult: "   ", emailErr: nil}

		msg := GenerateEmailCmd(provider, "# Structured", "pl")()
		doneMsg, ok := msg.(AIEmailDoneMsg)
		if !ok {
			t.Fatalf("expected AIEmailDoneMsg, got %T", msg)
		}
		if doneMsg.Subject != "Meeting Summary" {
			t.Fatalf("expected default subject %q, got %q", "Meeting Summary", doneMsg.Subject)
		}
		if doneMsg.Body != "" {
			t.Fatalf("expected empty body, got %q", doneMsg.Body)
		}
	})

	t.Run("context cancellation race StructureNotesCmd with background is safe", func(t *testing.T) {
		provider := &raceProvider{}
		const workers = 50

		var wg sync.WaitGroup
		errCh := make(chan error, workers)

		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				_ = ctx

				msg := StructureNotesCmd(provider, "raw notes", ai.MeetingMeta{Title: fmt.Sprintf("T-%d", idx)})()
				if _, ok := msg.(AIStructureDoneMsg); !ok {
					errCh <- fmt.Errorf("expected AIStructureDoneMsg, got %T", msg)
					return
				}
			}(i)
		}

		wg.Wait()
		close(errCh)

		for err := range errCh {
			if err != nil {
				t.Fatal(err)
			}
		}
	})
}

type raceProvider struct{}

func (r *raceProvider) StructureNotes(ctx context.Context, rawNotes string, meta ai.MeetingMeta) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("nil context")
	}
	return "# Structured", nil
}

func (r *raceProvider) GenerateEmailSummary(ctx context.Context, structuredMD string, language string) (string, error) {
	return "Subject: X\n\nY", nil
}
