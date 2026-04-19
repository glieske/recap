package tui

import (
	"errors"
	"testing"
	"time"
)

func newEditorStatusTestApp(t *testing.T) AppModel {
	t.Helper()

	store := newTestStore(t)
	meeting := createProjectAndMeeting(t, store)

	m := NewAppModel(nil, store, appTestProvider{}, "", false, "")
	m.screen = ScreenEditor
	m.currentMeeting = meeting
	m.editorModel = NewEditorModel(meeting, store, 80, 24, "TestProvider", "TestModel", "")
	sm := NewSummaryModel("", meeting.Project, meeting.ID, store, 80, 24)
	m.editorModel.SetSummaryModel(sm)
	m.hasEditorModel = true

	return m
}

func assertExpiryRange(t *testing.T, expiry time.Time, min, max time.Duration) {
	t.Helper()

	remaining := time.Until(expiry)
	if remaining < min || remaining > max {
		t.Fatalf("expected status expiry in range [%v, %v], got %v", min, max, remaining)
	}
}

func TestEditorStatusMsgTriggerAISetsEditorStatusAndExpiry(t *testing.T) {
	m := newEditorStatusTestApp(t)

	updated, _ := appUpdate(t, m, TriggerAIMsg{})

	if updated.editorModel.statusMsg != "⏳ Structuring notes..." {
		t.Fatalf("expected editor status %q, got %q", "⏳ Structuring notes...", updated.editorModel.statusMsg)
	}
	assertExpiryRange(t, updated.editorModel.statusExpiry, 9*time.Minute+55*time.Second, 10*time.Minute+5*time.Second)
}

func TestEditorStatusMsgTriggerEmailSetsEditorStatusAndExpiry(t *testing.T) {
	m := newEditorStatusTestApp(t)
	m.structuredMD = "# Structured"

	updated, _ := appUpdate(t, m, TriggerEmailMsg{})

	if updated.editorModel.statusMsg != "⏳ Generating email..." {
		t.Fatalf("expected editor status %q, got %q", "⏳ Generating email...", updated.editorModel.statusMsg)
	}
	assertExpiryRange(t, updated.editorModel.statusExpiry, 9*time.Minute+55*time.Second, 10*time.Minute+5*time.Second)
}

func TestEditorStatusMsgAIStructureDoneSetsSuccessStatusAndShortExpiry(t *testing.T) {
	m := newEditorStatusTestApp(t)

	updated, _ := appUpdate(t, m, AIStructureDoneMsg{StructuredMD: "# Done"})

	if updated.editorModel.statusMsg != "✓ Structured notes updated" {
		t.Fatalf("expected editor status %q, got %q", "✓ Structured notes updated", updated.editorModel.statusMsg)
	}
	assertExpiryRange(t, updated.editorModel.statusExpiry, 4*time.Second, 6*time.Second)
}

func TestEditorStatusMsgAIEmailDoneSetsSuccessStatusAndShortExpiry(t *testing.T) {
	m := newEditorStatusTestApp(t)

	updated, _ := appUpdate(t, m, AIEmailDoneMsg{Subject: "Subject", Body: "Body"})

	if updated.editorModel.statusMsg != "✓ Email generated" {
		t.Fatalf("expected editor status %q, got %q", "✓ Email generated", updated.editorModel.statusMsg)
	}
	assertExpiryRange(t, updated.editorModel.statusExpiry, 4*time.Second, 6*time.Second)
}

func TestEditorStatusMsgAIStructureErrClearsEditorStatus(t *testing.T) {
	m := newEditorStatusTestApp(t)
	m.editorModel.statusMsg = "existing"

	updated, _ := appUpdate(t, m, AIStructureErrMsg{Err: errors.New("boom")})

	if updated.editorModel.statusMsg != "" {
		t.Fatalf("expected editor status to be cleared, got %q", updated.editorModel.statusMsg)
	}
}

func TestEditorStatusMsgAIEmailErrClearsEditorStatus(t *testing.T) {
	m := newEditorStatusTestApp(t)
	m.editorModel.statusMsg = "existing"

	updated, _ := appUpdate(t, m, AIEmailErrMsg{Err: errors.New("boom")})

	if updated.editorModel.statusMsg != "" {
		t.Fatalf("expected editor status to be cleared, got %q", updated.editorModel.statusMsg)
	}
}
