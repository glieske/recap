package tui

import "testing"

func TestAppSplitWiringMeetingSelectedWiresSummaryModel(t *testing.T) {
	store := newTestStore(t)
	meeting := createProjectAndMeeting(t, store)
	m := NewAppModel(nil, store, nil, "", false, "")

	updated, _ := appUpdate(t, m, MeetingSelectedMsg{Meeting: *meeting})

	if updated.hasEditorModel != true {
		t.Fatalf("expected hasEditorModel to be true, got %v", updated.hasEditorModel)
	}
	if updated.editorModel.hasSummaryModel != true {
		t.Fatalf("expected editorModel.hasSummaryModel to be true, got %v", updated.editorModel.hasSummaryModel)
	}
	if updated.screen != ScreenEditor {
		t.Fatalf("expected screen %v, got %v", ScreenEditor, updated.screen)
	}
}

func TestAppSplitWiringMeetingCreatedWiresSummaryModel(t *testing.T) {
	store := newTestStore(t)
	meeting := createProjectAndMeeting(t, store)
	m := NewAppModel(nil, store, nil, "", false, "")

	updated, _ := appUpdate(t, m, MeetingCreatedMsg{Meeting: meeting})

	if updated.hasEditorModel != true {
		t.Fatalf("expected hasEditorModel to be true, got %v", updated.hasEditorModel)
	}
	if updated.editorModel.hasSummaryModel != true {
		t.Fatalf("expected editorModel.hasSummaryModel to be true, got %v", updated.editorModel.hasSummaryModel)
	}
	if updated.screen != ScreenEditor {
		t.Fatalf("expected screen %v, got %v", ScreenEditor, updated.screen)
	}
}

func TestAppSplitWiringAIStructureDoneUpdatesSummaryContent(t *testing.T) {
	store := newTestStore(t)
	meeting := createProjectAndMeeting(t, store)
	m := NewAppModel(nil, store, nil, "", false, "")

	updated, _ := appUpdate(t, m, MeetingSelectedMsg{Meeting: *meeting})
	if updated.editorModel.hasSummaryModel != true {
		t.Fatalf("expected editorModel.hasSummaryModel to be true before AIStructureDoneMsg")
	}

	const structured = "# Test Summary"
	updated, _ = appUpdate(t, updated, AIStructureDoneMsg{StructuredMD: structured})

	if updated.structuredMD != structured {
		t.Fatalf("expected structuredMD %q, got %q", structured, updated.structuredMD)
	}

	gotSummary := updated.editorModel.GetSummaryModel().Value()
	if gotSummary != structured {
		t.Fatalf("expected summary content %q, got %q", structured, gotSummary)
	}
}

func TestAppSplitWiringTogglePreviewIsNoOpOnEditorScreen(t *testing.T) {
	store := newTestStore(t)
	meeting := createProjectAndMeeting(t, store)
	m := NewAppModel(nil, store, nil, "", false, "")

	updated, _ := appUpdate(t, m, MeetingSelectedMsg{Meeting: *meeting})
	if updated.screen != ScreenEditor {
		t.Fatalf("expected precondition screen %v, got %v", ScreenEditor, updated.screen)
	}

	updated, cmd := appUpdate(t, updated, TogglePreviewMsg{})

	if updated.screen != ScreenEditor {
		t.Fatalf("expected screen to remain %v, got %v", ScreenEditor, updated.screen)
	}
	if cmd != nil {
		t.Fatalf("expected TogglePreviewMsg command to be nil, got non-nil")
	}
}

func TestAppSplitWiringScreenEnumLayoutWithoutPreview(t *testing.T) {
	if ScreenMeetingList != 1 {
		t.Fatalf("expected ScreenMeetingList to equal 1, got %d", ScreenMeetingList)
	}
	if ScreenProviderSelector != 6 {
		t.Fatalf("expected ScreenProviderSelector to equal 6 (no ScreenPreview in enum), got %d", ScreenProviderSelector)
	}
}
