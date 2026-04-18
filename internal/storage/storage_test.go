package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

func TestNewStoreCreatesNotesDir(t *testing.T) {
	baseDir := t.TempDir()
	notesDir := filepath.Join(baseDir, "notes")

	if _, err := os.Stat(notesDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected notes dir to not exist before NewStore, err=%v", err)
	}

	store := NewStore(notesDir)
	if store == nil {
		t.Fatalf("expected non-nil store")
	}

	info, err := os.Stat(notesDir)
	if err != nil {
		t.Fatalf("expected notes dir to exist after NewStore, got error: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected notes dir path to be a directory")
	}
}

func TestCreateMeetingCreatesExpectedFilesAndMeta(t *testing.T) {
	store := NewStore(t.TempDir())
	meetingDate := time.Date(2026, time.April, 16, 12, 30, 0, 0, time.UTC)
	participants := []string{"Alice", "Bob"}

	if _, err := store.CreateProject("Test Project", "TEST"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting("Sprint Planning", meetingDate, participants, "TEST", nil, "")
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	expectedID := "2026-04-16-sprint-planning"
	if meeting.ID != expectedID {
		t.Fatalf("expected meeting ID %q, got %q", expectedID, meeting.ID)
	}
	if meeting.Title != "Sprint Planning" {
		t.Fatalf("expected title %q, got %q", "Sprint Planning", meeting.Title)
	}
	if !meeting.Date.Equal(meetingDate) {
		t.Fatalf("expected meeting date %v, got %v", meetingDate, meeting.Date)
	}
	if meeting.Status != MeetingStatusDraft {
		t.Fatalf("expected status %q, got %q", MeetingStatusDraft, meeting.Status)
	}

	meetingDir := store.MeetingDir("TEST", meeting.ID)
	if _, err := os.Stat(meetingDir); err != nil {
		t.Fatalf("meeting directory missing: %v", err)
	}

	rawPath := filepath.Join(meetingDir, rawNotesFileName)
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		t.Fatalf("raw.txt missing or unreadable: %v", err)
	}
	if string(raw) != "" {
		t.Fatalf("expected empty raw.txt, got %q", string(raw))
	}

	metaPath := filepath.Join(meetingDir, metaFileName)
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("meta.json missing or unreadable: %v", err)
	}

	var meta meetingMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatalf("meta.json is invalid JSON: %v", err)
	}
	if meta.Title != "Sprint Planning" {
		t.Fatalf("expected meta title %q, got %q", "Sprint Planning", meta.Title)
	}
	if meta.Date != "2026-04-16" {
		t.Fatalf("expected meta date %q, got %q", "2026-04-16", meta.Date)
	}
	if len(meta.Participants) != 2 || meta.Participants[0] != "Alice" || meta.Participants[1] != "Bob" {
		t.Fatalf("unexpected participants in meta: %#v", meta.Participants)
	}
}

func TestCreateMeetingDuplicateTitleSameDayGetsNumericSuffix(t *testing.T) {
	store := NewStore(t.TempDir())
	meetingDate := time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC)

	if _, err := store.CreateProject("Test Project", "TEST"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	first, err := store.CreateMeeting("Team Sync", meetingDate, nil, "TEST", nil, "")
	if err != nil {
		t.Fatalf("first CreateMeeting returned error: %v", err)
	}
	second, err := store.CreateMeeting("Team Sync", meetingDate, nil, "TEST", nil, "")
	if err != nil {
		t.Fatalf("second CreateMeeting returned error: %v", err)
	}

	if first.ID != "2026-04-16-team-sync" {
		t.Fatalf("unexpected first ID %q", first.ID)
	}
	if second.ID != "2026-04-16-team-sync-2" {
		t.Fatalf("expected second ID with -2 suffix, got %q", second.ID)
	}
}

func TestListMeetingsEmptyDirReturnsEmptySlice(t *testing.T) {
	store := NewStore(t.TempDir())

	meetings, err := store.ListMeetings("")
	if err != nil {
		t.Fatalf("ListMeetings returned error: %v", err)
	}
	if len(meetings) != 0 {
		t.Fatalf("expected empty meetings slice, got %d items", len(meetings))
	}
}

func TestListMeetingsSortedByDateDescending(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Test Project", "TEST"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	olderDate := time.Date(2026, time.April, 14, 0, 0, 0, 0, time.UTC)
	newerDate := time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC)

	older, err := store.CreateMeeting("Older", olderDate, nil, "TEST", nil, "")
	if err != nil {
		t.Fatalf("CreateMeeting older returned error: %v", err)
	}
	newer, err := store.CreateMeeting("Newer", newerDate, nil, "TEST", nil, "")
	if err != nil {
		t.Fatalf("CreateMeeting newer returned error: %v", err)
	}

	meetings, err := store.ListMeetings("TEST")
	if err != nil {
		t.Fatalf("ListMeetings returned error: %v", err)
	}

	if len(meetings) != 2 {
		t.Fatalf("expected 2 meetings, got %d", len(meetings))
	}
	if meetings[0].ID != newer.ID {
		t.Fatalf("expected first meeting ID %q (newest), got %q", newer.ID, meetings[0].ID)
	}
	if meetings[1].ID != older.ID {
		t.Fatalf("expected second meeting ID %q (oldest), got %q", older.ID, meetings[1].ID)
	}
}

func TestListMeetingsSkipsDirectoriesWithoutMetaJSON(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Test Project", "TEST"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting("Valid", time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC), nil, "TEST", nil, "")
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	orphanDir := store.MeetingDir("TEST", "2026-04-15-no-meta")
	if err := os.MkdirAll(orphanDir, 0o755); err != nil {
		t.Fatalf("failed creating orphan dir: %v", err)
	}

	meetings, err := store.ListMeetings("TEST")
	if err != nil {
		t.Fatalf("ListMeetings returned error: %v", err)
	}

	if len(meetings) != 1 {
		t.Fatalf("expected exactly 1 meeting, got %d", len(meetings))
	}
	if meetings[0].ID != meeting.ID {
		t.Fatalf("expected returned meeting ID %q, got %q", meeting.ID, meetings[0].ID)
	}
}

func TestListMeetingsDetectsStructuredStatusWhenFileExists(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Test Project", "TEST"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting("Structured", time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC), nil, "TEST", nil, "")
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	if err := store.SaveStructuredNotes("TEST", meeting.ID, "# Structured\n"); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	meetings, err := store.ListMeetings("TEST")
	if err != nil {
		t.Fatalf("ListMeetings returned error: %v", err)
	}
	if len(meetings) != 1 {
		t.Fatalf("expected 1 meeting, got %d", len(meetings))
	}
	if meetings[0].Status != MeetingStatusStructured {
		t.Fatalf("expected status %q, got %q", MeetingStatusStructured, meetings[0].Status)
	}
}

func TestSaveAndLoadRawNotesRoundTrip(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Test Project", "TEST"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting("Raw Roundtrip", time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC), nil, "TEST", nil, "")
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	content := "Line one\nLine two\nEmoji: 😀\nUnicode: zażółć"
	if err := store.SaveRawNotes("TEST", meeting.ID, content); err != nil {
		t.Fatalf("SaveRawNotes returned error: %v", err)
	}

	loaded, err := store.LoadRawNotes("TEST", meeting.ID)
	if err != nil {
		t.Fatalf("LoadRawNotes returned error: %v", err)
	}
	if loaded != content {
		t.Fatalf("raw notes mismatch, expected %q, got %q", content, loaded)
	}
}

func TestSaveAndLoadStructuredNotesRoundTrip(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Test Project", "TEST"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting("Structured Roundtrip", time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC), nil, "TEST", nil, "")
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	content := "# Summary\n- action item 1\n- action item 2"
	if err := store.SaveStructuredNotes("TEST", meeting.ID, content); err != nil {
		t.Fatalf("SaveStructuredNotes returned error: %v", err)
	}

	loaded, err := store.LoadStructuredNotes("TEST", meeting.ID)
	if err != nil {
		t.Fatalf("LoadStructuredNotes returned error: %v", err)
	}
	if loaded != content {
		t.Fatalf("structured notes mismatch, expected %q, got %q", content, loaded)
	}
}

func TestLoadStructuredNotesReturnsErrNotExistWhenAbsent(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Test Project", "TEST"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting("No Structured", time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC), nil, "TEST", nil, "")
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	_, err = store.LoadStructuredNotes("TEST", meeting.ID)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}

func TestDeleteMeetingRemovesMeetingDirectory(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Test Project", "TEST"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	meeting, err := store.CreateMeeting("Delete Me", time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC), nil, "TEST", nil, "")
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	meetingDir := store.MeetingDir("TEST", meeting.ID)
	if _, err := os.Stat(meetingDir); err != nil {
		t.Fatalf("expected meeting dir to exist before delete, got %v", err)
	}

	if err := store.DeleteMeeting("TEST", meeting.ID); err != nil {
		t.Fatalf("DeleteMeeting returned error: %v", err)
	}

	if _, err := os.Stat(meetingDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected meeting dir removed, stat err=%v", err)
	}
}

func TestMeetingDirRejectsPathTraversal(t *testing.T) {
	store := NewStore(t.TempDir())

	invalid := filepath.Join(filepath.Clean(store.notesDir), "_invalid_meeting_id")

	testIDs := []string{"../escape", "..", "safe/child", `safe\\child`}
	for _, id := range testIDs {
		t.Run(id, func(t *testing.T) {
			got := store.MeetingDir("TEST", id)
			if got != invalid {
				t.Fatalf("expected invalid path %q for %q, got %q", invalid, id, got)
			}
		})
	}
}

func TestMeetingDirRejectsPathTraversalInProjectParam(t *testing.T) {
	store := NewStore(t.TempDir())

	invalid := filepath.Join(filepath.Clean(store.notesDir), "_invalid_meeting_id")

	testProjects := []string{"../escape", "..", "safe/child", `safe\\child`}
	for _, project := range testProjects {
		t.Run(project, func(t *testing.T) {
			got := store.MeetingDir(project, "2026-04-16-team-sync")
			if got != invalid {
				t.Fatalf("expected invalid path %q for project %q, got %q", invalid, project, got)
			}
		})
	}
}

func TestCreateMeetingRejectsPathTraversalInProject(t *testing.T) {
	store := NewStore(t.TempDir())

	meetingDate := time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC)
	projects := []string{"../SEC", "SEC/DEV", `SEC\\DEV`, ".."}

	for _, project := range projects {
		t.Run(project, func(t *testing.T) {
			_, err := store.CreateMeeting("Traversal Attempt", meetingDate, nil, project, nil, "")
			if err == nil {
				t.Fatalf("expected error for project %q", project)
			}
			if !strings.Contains(err.Error(), "invalid project prefix") {
				t.Fatalf("expected invalid project prefix error, got %v", err)
			}
		})
	}
}

func TestListMeetingsRejectsPathTraversalProjectFilter(t *testing.T) {
	store := NewStore(t.TempDir())

	filters := []string{"../SEC", "SEC/DEV", `SEC\\DEV`, ".."}
	for _, filter := range filters {
		t.Run(filter, func(t *testing.T) {
			meetings, err := store.ListMeetings(filter)
			if err == nil {
				t.Fatalf("expected error for project filter %q", filter)
			}
			if meetings != nil {
				t.Fatalf("expected nil meetings on invalid filter, got %v", meetings)
			}
			if !strings.Contains(err.Error(), "invalid project filter") {
				t.Fatalf("expected invalid project filter error, got %v", err)
			}
		})
	}
}

func TestCreateMeetingFailsWhenProjectMetaJSONMalformed(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Security", "SEC"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	projectMetaPath := filepath.Join(store.notesDir, "SEC", projectMetaFileName)
	if err := os.WriteFile(projectMetaPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("failed to write malformed project.json: %v", err)
	}

	_, err := store.CreateMeeting("Broken Project Meta", time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC), nil, "SEC", nil, "")
	if err == nil {
		t.Fatalf("expected CreateMeeting to fail for malformed project metadata")
	}
	if !strings.Contains(err.Error(), "unmarshal project metadata") {
		t.Fatalf("expected unmarshal project metadata error, got %v", err)
	}
}

func TestListMeetingsSkipsMalformedMeetingMetaJSON(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Security", "SEC"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	validMeeting, err := store.CreateMeeting("Valid Meeting", time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC), nil, "SEC", nil, "")
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	malformedDir := store.MeetingDir("SEC", "2026-04-16-malformed-meta")
	if err := os.MkdirAll(malformedDir, 0o755); err != nil {
		t.Fatalf("failed creating malformed meeting dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(malformedDir, metaFileName), []byte("{"), 0o644); err != nil {
		t.Fatalf("failed writing malformed meta.json: %v", err)
	}

	meetings, err := store.ListMeetings("SEC")
	if err != nil {
		t.Fatalf("ListMeetings returned error: %v", err)
	}
	if len(meetings) != 1 {
		t.Fatalf("expected only valid meeting to be listed, got %d entries", len(meetings))
	}
	if meetings[0].ID != validMeeting.ID {
		t.Fatalf("expected valid meeting ID %q, got %q", validMeeting.ID, meetings[0].ID)
	}
}

func TestCreateMeetingPreservesAdversarialTagInputs(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Tag Security", "TAGS"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	tags := []string{
		"",
		strings.Repeat("A", 11*1024),
		"<script>alert(1)</script>",
		"../path-traversal-probe",
		"zero\u200bwidth",
	}
	externalTicket := "${jndi:ldap://example.invalid/a}"

	meeting, err := store.CreateMeeting(
		"Adversarial Tags",
		time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC),
		nil,
		"TAGS",
		tags,
		externalTicket,
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	metaPath := filepath.Join(store.MeetingDir("TAGS", meeting.ID), metaFileName)
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read meta.json returned error: %v", err)
	}

	var meta meetingMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatalf("unmarshal meta.json returned error: %v", err)
	}

	if len(meta.Tags) != len(tags) {
		t.Fatalf("expected %d tags in meta, got %d", len(tags), len(meta.Tags))
	}
	for i := range tags {
		if meta.Tags[i] != tags[i] {
			t.Fatalf("expected meta tag %d to equal input", i)
		}
	}
	if meta.ExternalTicket != externalTicket {
		t.Fatalf("expected external ticket %q, got %q", externalTicket, meta.ExternalTicket)
	}
}

func TestCreateMeetingWithExtremelyLongTitleUsesBoundedSlug(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Long Title", "LONG"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	title := strings.Repeat("A", 150) + " ### $$$ " + strings.Repeat("b", 150)
	meeting, err := store.CreateMeeting(
		title,
		time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC),
		nil,
		"LONG",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	if meeting.Title != title {
		t.Fatalf("expected title to be preserved exactly")
	}

	const idPrefix = "2026-04-16-"
	if !strings.HasPrefix(meeting.ID, idPrefix) {
		t.Fatalf("expected ID to start with %q, got %q", idPrefix, meeting.ID)
	}
	slug := strings.TrimPrefix(meeting.ID, idPrefix)
	if len(slug) != 50 {
		t.Fatalf("expected slug length 50, got %d", len(slug))
	}
	if strings.Contains(slug, "..") || strings.Contains(slug, "/") || strings.Contains(slug, "\\") {
		t.Fatalf("slug contains traversal pattern: %q", slug)
	}
}

func TestMeetingDirKeepsEncodedTraversalTokensInsideNotesDir(t *testing.T) {
	store := NewStore(t.TempDir())

	project := "%2e%2e"
	meetingID := "%2fetc%2fpasswd"
	meetingDir := store.MeetingDir(project, meetingID)

	cleanRoot := filepath.Clean(store.notesDir)
	rel, err := filepath.Rel(cleanRoot, meetingDir)
	if err != nil {
		t.Fatalf("filepath.Rel returned error: %v", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Fatalf("expected encoded traversal tokens to stay inside notes dir, rel=%q", rel)
	}
}

func TestCreateMeetingAutoGeneratesSequentialTicketIDsPerProject(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Infrastructure", "INFRA"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	first, err := store.CreateMeeting("Weekly Infra Sync", time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC), nil, "INFRA", nil, "")
	if err != nil {
		t.Fatalf("first CreateMeeting returned error: %v", err)
	}

	second, err := store.CreateMeeting("Incident Review", time.Date(2026, time.April, 17, 0, 0, 0, 0, time.UTC), nil, "INFRA", nil, "")
	if err != nil {
		t.Fatalf("second CreateMeeting returned error: %v", err)
	}

	if first.TicketID != "INFRA-001" {
		t.Fatalf("expected first ticket ID %q, got %q", "INFRA-001", first.TicketID)
	}
	if second.TicketID != "INFRA-002" {
		t.Fatalf("expected second ticket ID %q, got %q", "INFRA-002", second.TicketID)
	}

	project, err := store.GetProject("INFRA")
	if err != nil {
		t.Fatalf("GetProject returned error: %v", err)
	}
	if project.NextSequence != 3 {
		t.Fatalf("expected NextSequence %d after two meetings, got %d", 3, project.NextSequence)
	}
}

func TestListMeetingsFiltersByProjectPrefix(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Project Alpha", "ALPHA"); err != nil {
		t.Fatalf("CreateProject ALPHA returned error: %v", err)
	}
	if _, err := store.CreateProject("Project Beta", "BETA"); err != nil {
		t.Fatalf("CreateProject BETA returned error: %v", err)
	}

	alphaMeeting, err := store.CreateMeeting("Alpha Planning", time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC), nil, "ALPHA", nil, "")
	if err != nil {
		t.Fatalf("CreateMeeting ALPHA returned error: %v", err)
	}
	betaMeeting, err := store.CreateMeeting("Beta Planning", time.Date(2026, time.April, 17, 0, 0, 0, 0, time.UTC), nil, "BETA", nil, "")
	if err != nil {
		t.Fatalf("CreateMeeting BETA returned error: %v", err)
	}

	alphaMeetings, err := store.ListMeetings("ALPHA")
	if err != nil {
		t.Fatalf("ListMeetings(ALPHA) returned error: %v", err)
	}
	if len(alphaMeetings) != 1 {
		t.Fatalf("expected 1 ALPHA meeting, got %d", len(alphaMeetings))
	}
	if alphaMeetings[0].ID != alphaMeeting.ID {
		t.Fatalf("expected ALPHA meeting ID %q, got %q", alphaMeeting.ID, alphaMeetings[0].ID)
	}
	if alphaMeetings[0].Project != "ALPHA" {
		t.Fatalf("expected ALPHA project value %q, got %q", "ALPHA", alphaMeetings[0].Project)
	}

	betaMeetings, err := store.ListMeetings("BETA")
	if err != nil {
		t.Fatalf("ListMeetings(BETA) returned error: %v", err)
	}
	if len(betaMeetings) != 1 {
		t.Fatalf("expected 1 BETA meeting, got %d", len(betaMeetings))
	}
	if betaMeetings[0].ID != betaMeeting.ID {
		t.Fatalf("expected BETA meeting ID %q, got %q", betaMeeting.ID, betaMeetings[0].ID)
	}
	if betaMeetings[0].Project != "BETA" {
		t.Fatalf("expected BETA project value %q, got %q", "BETA", betaMeetings[0].Project)
	}
}

func TestTagsAndExternalTicketPersistInMetaAndList(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Operations", "OPS"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	tags := []string{"production", "incident"}
	externalTicket := "EXT-4321"
	meeting, err := store.CreateMeeting(
		"Production Incident Review",
		time.Date(2026, time.April, 18, 0, 0, 0, 0, time.UTC),
		[]string{"Alice", "Bob"},
		"OPS",
		tags,
		externalTicket,
	)
	if err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	metaPath := filepath.Join(store.MeetingDir("OPS", meeting.ID), metaFileName)
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("read meta.json returned error: %v", err)
	}

	var meta meetingMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatalf("meta.json unmarshal returned error: %v", err)
	}
	if len(meta.Tags) != 2 || meta.Tags[0] != "production" || meta.Tags[1] != "incident" {
		t.Fatalf("expected meta tags [production incident], got %#v", meta.Tags)
	}
	if meta.ExternalTicket != externalTicket {
		t.Fatalf("expected meta external ticket %q, got %q", externalTicket, meta.ExternalTicket)
	}

	meetings, err := store.ListMeetings("OPS")
	if err != nil {
		t.Fatalf("ListMeetings returned error: %v", err)
	}
	if len(meetings) != 1 {
		t.Fatalf("expected 1 meeting, got %d", len(meetings))
	}
	if meetings[0].ID != meeting.ID {
		t.Fatalf("expected meeting ID %q, got %q", meeting.ID, meetings[0].ID)
	}
	if len(meetings[0].Tags) != 2 || meetings[0].Tags[0] != "production" || meetings[0].Tags[1] != "incident" {
		t.Fatalf("expected listed tags [production incident], got %#v", meetings[0].Tags)
	}
	if meetings[0].ExternalTicket != externalTicket {
		t.Fatalf("expected listed external ticket %q, got %q", externalTicket, meetings[0].ExternalTicket)
	}
}

func TestListMeetingsIncludesLegacyFlatMeetingDirs(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Core", "CORE"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}
	if _, err := store.CreateMeeting("Current Meeting", time.Date(2026, time.April, 16, 0, 0, 0, 0, time.UTC), nil, "CORE", nil, ""); err != nil {
		t.Fatalf("CreateMeeting returned error: %v", err)
	}

	legacyID := "2026-04-10-legacy-sync"
	legacyDir := filepath.Join(store.notesDir, legacyID)
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatalf("failed creating legacy dir: %v", err)
	}

	legacyMeta := meetingMeta{
		Title:        "Legacy Sync",
		Date:         "2026-04-10",
		Participants: []string{"Legacy User"},
	}
	legacyMetaBytes, err := json.Marshal(legacyMeta)
	if err != nil {
		t.Fatalf("legacy meta marshal returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, metaFileName), legacyMetaBytes, 0o644); err != nil {
		t.Fatalf("write legacy meta.json returned error: %v", err)
	}

	meetings, err := store.ListMeetings("")
	if err != nil {
		t.Fatalf("ListMeetings returned error: %v", err)
	}

	foundLegacy := false
	for _, meeting := range meetings {
		if meeting.ID == legacyID {
			foundLegacy = true
			if meeting.Title != "Legacy Sync" {
				t.Fatalf("expected legacy title %q, got %q", "Legacy Sync", meeting.Title)
			}
			if meeting.Project != "" {
				t.Fatalf("expected empty project for legacy meeting, got %q", meeting.Project)
			}
		}
	}

	if !foundLegacy {
		t.Fatalf("expected legacy meeting ID %q to be listed", legacyID)
	}
}

func TestSlugifyTitleExamples(t *testing.T) {
	t.Run("spaces and lowercase", func(t *testing.T) {
		got := slugifyTitle("  Sprint   Planning Meeting  ")
		expected := "sprint-planning-meeting"
		if got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	})

	t.Run("max length 50", func(t *testing.T) {
		input := strings.Repeat("a", 70)
		got := slugifyTitle(input)
		if len(got) != 50 {
			t.Fatalf("expected slug length 50, got %d", len(got))
		}
		if got != strings.Repeat("a", 50) {
			t.Fatalf("unexpected truncated slug: %q", got)
		}
	})
}

func TestSlugifyTitlePropertyIdempotentAndBounded(t *testing.T) {
	property := func(input string) bool {
		once := slugifyTitle(input)
		twice := slugifyTitle(once)

		if once != twice {
			return false
		}
		if len(once) > 50 {
			return false
		}
		if once != strings.ToLower(once) {
			return false
		}
		if strings.ContainsRune(once, ' ') {
			return false
		}

		return true
	}

	if err := quick.Check(property, &quick.Config{MaxCount: 200}); err != nil {
		t.Fatalf("slugify property check failed: %v", err)
	}
}
