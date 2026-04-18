package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/glieske/recap/internal/ai"
)

type mockProvider struct {
	structureResult string
	structureErr    error
	emailResult     string
	emailErr        error
}

func (m *mockProvider) StructureNotes(ctx context.Context, rawNotes string, meta ai.MeetingMeta) (string, error) {
	return m.structureResult, m.structureErr
}

func (m *mockProvider) GenerateEmailSummary(ctx context.Context, structuredMD string, language string) (string, error) {
	return m.emailResult, m.emailErr
}

func TestStructureNotesCmdSuccess(t *testing.T) {
	provider := &mockProvider{structureResult: "# Structured"}

	msg := StructureNotesCmd(provider, "raw notes", ai.MeetingMeta{Title: "Weekly"})()
	doneMsg, ok := msg.(AIStructureDoneMsg)
	if !ok {
		t.Fatalf("expected AIStructureDoneMsg, got %T", msg)
	}

	if doneMsg.StructuredMD != "# Structured" {
		t.Fatalf("expected structured markdown %q, got %q", "# Structured", doneMsg.StructuredMD)
	}
}

func TestStructureNotesCmdError(t *testing.T) {
	provider := &mockProvider{structureErr: errors.New("provider failed")}

	msg := StructureNotesCmd(provider, "raw notes", ai.MeetingMeta{})()
	errMsg, ok := msg.(AIStructureErrMsg)
	if !ok {
		t.Fatalf("expected AIStructureErrMsg, got %T", msg)
	}

	if errMsg.Err == nil || errMsg.Err.Error() != "provider failed" {
		t.Fatalf("expected provider failed error, got %v", errMsg.Err)
	}
}

func TestStructureNotesCmdNilProvider(t *testing.T) {
	msg := StructureNotesCmd(nil, "raw notes", ai.MeetingMeta{})()
	errMsg, ok := msg.(AIStructureErrMsg)
	if !ok {
		t.Fatalf("expected AIStructureErrMsg, got %T", msg)
	}

	if errMsg.Err == nil || errMsg.Err.Error() != "no AI provider configured" {
		t.Fatalf("expected no AI provider configured error, got %v", errMsg.Err)
	}
}

func TestStructureNotesCmdEmptyRawNotes(t *testing.T) {
	provider := &mockProvider{structureResult: "result"}
	cmd := StructureNotesCmd(provider, "   ", ai.MeetingMeta{})
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	msg := cmd()
	errMsg, ok := msg.(AIStructureErrMsg)
	if !ok {
		t.Fatalf("expected AIStructureErrMsg, got %T", msg)
	}
	if errMsg.Err == nil || !strings.Contains(errMsg.Err.Error(), "no raw notes") {
		t.Fatalf("expected 'no raw notes' error, got %v", errMsg.Err)
	}
}

func TestGenerateEmailCmdSuccess(t *testing.T) {
	provider := &mockProvider{emailResult: "Subject: Test\n\nBody text"}

	msg := GenerateEmailCmd(provider, "# Structured", "pl")()
	doneMsg, ok := msg.(AIEmailDoneMsg)
	if !ok {
		t.Fatalf("expected AIEmailDoneMsg, got %T", msg)
	}

	if doneMsg.Subject != "Test" {
		t.Fatalf("expected subject %q, got %q", "Test", doneMsg.Subject)
	}

	if doneMsg.Body != "Body text" {
		t.Fatalf("expected body %q, got %q", "Body text", doneMsg.Body)
	}
}

func TestGenerateEmailCmdError(t *testing.T) {
	provider := &mockProvider{emailErr: errors.New("email failed")}

	msg := GenerateEmailCmd(provider, "# Structured", "pl")()
	errMsg, ok := msg.(AIEmailErrMsg)
	if !ok {
		t.Fatalf("expected AIEmailErrMsg, got %T", msg)
	}

	if errMsg.Err == nil || errMsg.Err.Error() != "email failed" {
		t.Fatalf("expected email failed error, got %v", errMsg.Err)
	}
}

func TestGenerateEmailCmdNilProvider(t *testing.T) {
	msg := GenerateEmailCmd(nil, "# Structured", "pl")()
	errMsg, ok := msg.(AIEmailErrMsg)
	if !ok {
		t.Fatalf("expected AIEmailErrMsg, got %T", msg)
	}

	if errMsg.Err == nil || errMsg.Err.Error() != "no AI provider configured" {
		t.Fatalf("expected no AI provider configured error, got %v", errMsg.Err)
	}
}

func TestGenerateEmailCmdEmptyStructured(t *testing.T) {
	provider := &mockProvider{}

	msg := GenerateEmailCmd(provider, "", "pl")()
	errMsg, ok := msg.(AIEmailErrMsg)
	if !ok {
		t.Fatalf("expected AIEmailErrMsg, got %T", msg)
	}

	if errMsg.Err == nil || errMsg.Err.Error() != "no structured notes available" {
		t.Fatalf("expected no structured notes available error, got %v", errMsg.Err)
	}
}

func TestParseEmailResponseWithSubject(t *testing.T) {
	subject, body := parseEmailResponse("Subject: X\n\nBody")

	if subject != "X" {
		t.Fatalf("expected subject %q, got %q", "X", subject)
	}

	if body != "Body" {
		t.Fatalf("expected body %q, got %q", "Body", body)
	}
}

func TestParseEmailResponseWithoutSubject(t *testing.T) {
	subject, body := parseEmailResponse("Just body text")

	if subject != "Meeting Summary" {
		t.Fatalf("expected subject %q, got %q", "Meeting Summary", subject)
	}

	if body != "Just body text" {
		t.Fatalf("expected body %q, got %q", "Just body text", body)
	}
}

func TestParseEmailResponseMultilineBody(t *testing.T) {
	subject, body := parseEmailResponse("Subject: Sprint Update\n\nLine one\nLine two\nLine three")

	if subject != "Sprint Update" {
		t.Fatalf("expected subject %q, got %q", "Sprint Update", subject)
	}

	if body != "Line one\nLine two\nLine three" {
		t.Fatalf("expected multiline body to be preserved, got %q", body)
	}
}

func TestParseEmailResponseNoBlankLine(t *testing.T) {
	subject, body := parseEmailResponse("Subject: Test\nBody without blank line")
	if subject != "Test" {
		t.Fatalf("expected subject 'Test', got %q", subject)
	}
	if body != "Body without blank line" {
		t.Fatalf("expected body text, got %q", body)
	}
}
