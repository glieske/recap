package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestProjectCreateProjectValidCreatesDirectoryAndMetadata(t *testing.T) {
	store := NewStore(t.TempDir())

	project, err := store.CreateProject("Infrastructure", "INFRA")
	if err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	if project.Name != "Infrastructure" {
		t.Fatalf("expected Name %q, got %q", "Infrastructure", project.Name)
	}
	if project.Prefix != "INFRA" {
		t.Fatalf("expected Prefix %q, got %q", "INFRA", project.Prefix)
	}
	if project.NextSequence != 1 {
		t.Fatalf("expected NextSequence %d, got %d", 1, project.NextSequence)
	}

	projectDir := filepath.Join(store.notesDir, "INFRA")
	if info, statErr := os.Stat(projectDir); statErr != nil {
		t.Fatalf("expected project directory to exist, got error: %v", statErr)
	} else if !info.IsDir() {
		t.Fatalf("expected %q to be a directory", projectDir)
	}

	metaPath := filepath.Join(projectDir, projectMetaFileName)
	metaBytes, readErr := os.ReadFile(metaPath)
	if readErr != nil {
		t.Fatalf("expected project metadata file, got error: %v", readErr)
	}

	var meta projectMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatalf("project metadata invalid JSON: %v", err)
	}
	if meta.Name != "Infrastructure" {
		t.Fatalf("expected metadata Name %q, got %q", "Infrastructure", meta.Name)
	}
	if meta.Prefix != "INFRA" {
		t.Fatalf("expected metadata Prefix %q, got %q", "INFRA", meta.Prefix)
	}
	if meta.NextSequence != 1 {
		t.Fatalf("expected metadata NextSequence %d, got %d", 1, meta.NextSequence)
	}
}

func TestProjectCreateProjectInvalidPrefixReturnsErrInvalidPrefix(t *testing.T) {
	store := NewStore(t.TempDir())

	testCases := []string{
		"ab",          // lowercase
		"A",           // too short
		"ABCDEFGHIJK", // too long
		"AB-1",        // special char
	}

	for _, prefix := range testCases {
		t.Run(prefix, func(t *testing.T) {
			project, err := store.CreateProject("Name", prefix)
			if !errors.Is(err, ErrInvalidPrefix) {
				t.Fatalf("expected ErrInvalidPrefix for %q, got project=%v err=%v", prefix, project, err)
			}
			if project != nil {
				t.Fatalf("expected nil project for invalid prefix %q, got %#v", prefix, project)
			}
		})
	}
}

func TestProjectCreateProjectEmptyNameReturnsErrEmptyName(t *testing.T) {
	store := NewStore(t.TempDir())

	testCases := []string{"", "   ", "\n\t"}
	for _, name := range testCases {
		t.Run("name="+name, func(t *testing.T) {
			project, err := store.CreateProject(name, "OPS1")
			if !errors.Is(err, ErrEmptyName) {
				t.Fatalf("expected ErrEmptyName for name %q, got project=%v err=%v", name, project, err)
			}
			if project != nil {
				t.Fatalf("expected nil project for empty name, got %#v", project)
			}
		})
	}
}

func TestProjectCreateProjectDuplicatePrefixReturnsErrPrefixExists(t *testing.T) {
	store := NewStore(t.TempDir())

	first, err := store.CreateProject("Infra", "INFRA")
	if err != nil {
		t.Fatalf("first CreateProject returned error: %v", err)
	}
	if first.Prefix != "INFRA" {
		t.Fatalf("expected first project prefix %q, got %q", "INFRA", first.Prefix)
	}

	second, err := store.CreateProject("Duplicate", "INFRA")
	if !errors.Is(err, ErrPrefixExists) {
		t.Fatalf("expected ErrPrefixExists, got project=%v err=%v", second, err)
	}
	if second != nil {
		t.Fatalf("expected nil project on duplicate create, got %#v", second)
	}
}

func TestProjectListProjectsEmptyDirectoryReturnsEmptySlice(t *testing.T) {
	store := NewStore(t.TempDir())

	projects, err := store.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects returned error: %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("expected empty projects, got %d items", len(projects))
	}
}

func TestProjectListProjectsReturnsSortedByPrefixAscending(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Ops", "OPS"); err != nil {
		t.Fatalf("CreateProject OPS returned error: %v", err)
	}
	if _, err := store.CreateProject("Data", "DATA"); err != nil {
		t.Fatalf("CreateProject DATA returned error: %v", err)
	}
	if _, err := store.CreateProject("App", "APP"); err != nil {
		t.Fatalf("CreateProject APP returned error: %v", err)
	}

	projects, err := store.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects returned error: %v", err)
	}

	if len(projects) != 3 {
		t.Fatalf("expected 3 projects, got %d", len(projects))
	}

	expectedPrefixes := []string{"APP", "DATA", "OPS"}
	for i := range expectedPrefixes {
		if projects[i].Prefix != expectedPrefixes[i] {
			t.Fatalf("expected prefix at index %d to be %q, got %q", i, expectedPrefixes[i], projects[i].Prefix)
		}
	}
}

func TestProjectListProjectsSkipsDirectoriesWithoutProjectMetadata(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Infra", "INFRA"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	legacyDir := filepath.Join(store.notesDir, "2026-04-16-legacy-meeting")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatalf("failed to create legacy directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "raw.txt"), []byte("legacy"), 0o644); err != nil {
		t.Fatalf("failed to create legacy raw file: %v", err)
	}

	projects, err := store.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects returned error: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].Prefix != "INFRA" {
		t.Fatalf("expected only project prefix %q, got %q", "INFRA", projects[0].Prefix)
	}
}

func TestProjectListProjectsOnMissingNotesDirReturnsEmptySlice(t *testing.T) {
	base := t.TempDir()
	missingNotesDir := filepath.Join(base, "missing-notes")

	store := &Store{notesDir: missingNotesDir}
	projects, err := store.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects returned error: %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("expected empty projects for missing notes dir, got %d", len(projects))
	}
}

func TestProjectGetProjectExistingReturnsMetadata(t *testing.T) {
	store := NewStore(t.TempDir())

	if _, err := store.CreateProject("Infrastructure", "INFRA"); err != nil {
		t.Fatalf("CreateProject returned error: %v", err)
	}

	project, err := store.GetProject("INFRA")
	if err != nil {
		t.Fatalf("GetProject returned error: %v", err)
	}
	if project.Name != "Infrastructure" {
		t.Fatalf("expected Name %q, got %q", "Infrastructure", project.Name)
	}
	if project.Prefix != "INFRA" {
		t.Fatalf("expected Prefix %q, got %q", "INFRA", project.Prefix)
	}
	if project.NextSequence != 1 {
		t.Fatalf("expected NextSequence %d, got %d", 1, project.NextSequence)
	}
}

func TestProjectGetProjectNonExistentReturnsErrProjectNotFound(t *testing.T) {
	store := NewStore(t.TempDir())

	project, err := store.GetProject("INFRA")
	if !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("expected ErrProjectNotFound, got project=%v err=%v", project, err)
	}
	if project != nil {
		t.Fatalf("expected nil project when not found, got %#v", project)
	}
}

func TestProjectGetProjectInvalidPrefixReturnsErrInvalidPrefix(t *testing.T) {
	store := NewStore(t.TempDir())

	testCases := []string{"../../etc", "a", "AB-1"}
	for _, prefix := range testCases {
		t.Run(prefix, func(t *testing.T) {
			project, err := store.GetProject(prefix)
			if !errors.Is(err, ErrInvalidPrefix) {
				t.Fatalf("expected ErrInvalidPrefix for %q, got project=%v err=%v", prefix, project, err)
			}
			if project != nil {
				t.Fatalf("expected nil project for invalid prefix, got %#v", project)
			}
		})
	}
}

func TestProjectCreateProjectConcurrentSamePrefixOnlyOneSucceeds(t *testing.T) {
	store := NewStore(t.TempDir())

	const workers = 2
	start := make(chan struct{})
	results := make(chan error, workers)

	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := store.CreateProject("Concurrent", "RACE")
			results <- err
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	successCount := 0
	prefixExistsCount := 0
	for err := range results {
		if err == nil {
			successCount++
			continue
		}
		if errors.Is(err, ErrPrefixExists) {
			prefixExistsCount++
			continue
		}

		t.Fatalf("expected only nil or ErrPrefixExists errors, got %v", err)
	}

	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful CreateProject call, got %d", successCount)
	}
	if prefixExistsCount != 1 {
		t.Fatalf("expected exactly 1 ErrPrefixExists result, got %d", prefixExistsCount)
	}

	projects, err := store.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects returned error: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected exactly one persisted project, got %d", len(projects))
	}
	if projects[0].Prefix != "RACE" {
		t.Fatalf("expected persisted project prefix %q, got %q", "RACE", projects[0].Prefix)
	}
}
