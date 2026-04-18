package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const projectMetaFileName = "project.json"

var (
	projectPrefixPattern = regexp.MustCompile("^[A-Z0-9]{2,10}$")

	ErrInvalidPrefix   = errors.New("project prefix must be 2-10 uppercase alphanumeric characters")
	ErrEmptyName       = errors.New("project name must not be empty")
	ErrPrefixExists    = errors.New("project with this prefix already exists")
	ErrProjectNotFound = errors.New("project not found")
)

// Project contains project-level metadata for ticket generation.
type Project struct {
	Name         string
	Prefix       string
	NextSequence int
}

type projectMeta struct {
	Name         string `json:"name"`
	Prefix       string `json:"prefix"`
	NextSequence int    `json:"next_sequence"`
}

// CreateProject creates notesDir/PREFIX/project.json with NextSequence initialized to 1.
func (s *Store) CreateProject(name, prefix string) (*Project, error) {
	if !projectPrefixPattern.MatchString(prefix) {
		return nil, ErrInvalidPrefix
	}

	if strings.TrimSpace(name) == "" {
		return nil, ErrEmptyName
	}

	projectDir := filepath.Join(s.notesDir, prefix)
	targetPath := filepath.Join(projectDir, projectMetaFileName)

	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return nil, fmt.Errorf("create project directory: %w", err)
	}

	meta := projectMeta{
		Name:         name,
		Prefix:       prefix,
		NextSequence: 1,
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal project metadata: %w", err)
	}

	metaFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, ErrPrefixExists
		}

		return nil, fmt.Errorf("create project metadata: %w", err)
	}

	if _, err := metaFile.Write(metaBytes); err != nil {
		if closeErr := metaFile.Close(); closeErr != nil {
			return nil, fmt.Errorf("write project metadata: %w (close error: %v)", err, closeErr)
		}

		return nil, fmt.Errorf("write project metadata: %w", err)
	}

	if err := metaFile.Close(); err != nil {
		return nil, fmt.Errorf("close project metadata file: %w", err)
	}

	return &Project{
		Name:         name,
		Prefix:       prefix,
		NextSequence: 1,
	}, nil
}

// ListProjects returns all projects sorted by prefix ascending.
func (s *Store) ListProjects() ([]Project, error) {
	entries, err := os.ReadDir(s.notesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Project{}, nil
		}

		return nil, fmt.Errorf("read notes directory: %w", err)
	}

	projects := make([]Project, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metaPath := filepath.Join(s.notesDir, entry.Name(), projectMetaFileName)
		metaBytes, readErr := os.ReadFile(metaPath)
		if readErr != nil {
			if errors.Is(readErr, os.ErrNotExist) {
				continue
			}

			return nil, fmt.Errorf("read project metadata for %q: %w", entry.Name(), readErr)
		}

		var meta projectMeta
		if unmarshalErr := json.Unmarshal(metaBytes, &meta); unmarshalErr != nil {
			return nil, fmt.Errorf("unmarshal project metadata for %q: %w", entry.Name(), unmarshalErr)
		}

		projects = append(projects, Project{
			Name:         meta.Name,
			Prefix:       meta.Prefix,
			NextSequence: meta.NextSequence,
		})
	}

	sort.Slice(projects, func(leftIndex, rightIndex int) bool {
		return projects[leftIndex].Prefix < projects[rightIndex].Prefix
	})

	return projects, nil
}

// GetProject loads notesDir/PREFIX/project.json.
func (s *Store) GetProject(prefix string) (*Project, error) {
	if !projectPrefixPattern.MatchString(prefix) {
		return nil, ErrInvalidPrefix
	}

	metaPath := filepath.Join(s.notesDir, prefix, projectMetaFileName)
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrProjectNotFound
		}

		return nil, fmt.Errorf("read project metadata: %w", err)
	}

	var meta projectMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return nil, fmt.Errorf("unmarshal project metadata: %w", err)
	}

	return &Project{
		Name:         meta.Name,
		Prefix:       meta.Prefix,
		NextSequence: meta.NextSequence,
	}, nil
}
