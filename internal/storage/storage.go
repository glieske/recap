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
	"time"
)

const (
	dateLayout         = "2006-01-02"
	rawNotesFileName   = "raw.txt"
	structuredFileName = "structured.md"
	metaFileName       = "meta.json"
	maxSlugLength      = 50
)

var (
	whitespacePattern = regexp.MustCompile(`\s+`)
	nonSlugPattern    = regexp.MustCompile(`[^a-z0-9-]`)
	hyphenPattern     = regexp.MustCompile(`-+`)
)

// MeetingStatus defines processing stage of meeting notes.
type MeetingStatus string

const (
	MeetingStatusDraft      MeetingStatus = "draft"
	MeetingStatusStructured MeetingStatus = "structured"
)

// Meeting represents meeting metadata and current notes status.
type Meeting struct {
	ID             string
	Title          string
	Date           time.Time
	Participants   []string
	Status         MeetingStatus
	TicketID       string
	Project        string
	Tags           []string
	ExternalTicket string
}

// Store persists meetings in notesDir using one directory per meeting.
type Store struct {
	notesDir string
}

type meetingMeta struct {
	Title          string   `json:"title"`
	Date           string   `json:"date"`
	Participants   []string `json:"participants"`
	TicketID       string   `json:"ticket_id,omitempty"`
	Project        string   `json:"project,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	ExternalTicket string   `json:"external_ticket,omitempty"`
}

// NewStore creates a Store and ensures notesDir exists.
func NewStore(notesDir string) *Store {
	_ = os.MkdirAll(notesDir, 0o755)

	return &Store{notesDir: notesDir}
}

// CreateMeeting creates meeting directory, raw.txt, and meta.json.
func (s *Store) CreateMeeting(
	title string,
	date time.Time,
	participants []string,
	project string,
	tags []string,
	externalTicket string,
) (*Meeting, error) {
	if err := os.MkdirAll(s.notesDir, 0o755); err != nil {
		return nil, fmt.Errorf("ensure notes dir: %w", err)
	}

	ticketID, err := s.incrementProjectSequence(project)
	if err != nil {
		return nil, fmt.Errorf("increment project sequence: %w", err)
	}

	slug := slugifyTitle(title)
	if slug == "" {
		slug = "meeting"
	}

	baseMeetingID := fmt.Sprintf("%s-%s", date.Format(dateLayout), slug)
	meetingID := baseMeetingID
	meetingDir := ""
	meetingSuffix := 1

	for {
		meetingDir = s.MeetingDir(project, meetingID)
		mkdirErr := os.Mkdir(meetingDir, 0o755)
		if mkdirErr == nil {
			break
		}

		if !errors.Is(mkdirErr, os.ErrExist) {
			return nil, fmt.Errorf("create meeting directory: %w", mkdirErr)
		}

		meetingSuffix++
		meetingID = fmt.Sprintf("%s-%d", baseMeetingID, meetingSuffix)
	}

	if err := os.WriteFile(filepath.Join(meetingDir, rawNotesFileName), []byte{}, 0o644); err != nil {
		return nil, fmt.Errorf("create raw notes file: %w", err)
	}

	meta := meetingMeta{
		Title:          title,
		Date:           date.Format(dateLayout),
		Participants:   append([]string(nil), participants...),
		TicketID:       ticketID,
		Project:        project,
		Tags:           append([]string(nil), tags...),
		ExternalTicket: externalTicket,
	}

	metaBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, fmt.Errorf("marshal meeting metadata: %w", err)
	}

	if err := os.WriteFile(filepath.Join(meetingDir, metaFileName), metaBytes, 0o644); err != nil {
		return nil, fmt.Errorf("write meeting metadata: %w", err)
	}

	return &Meeting{
		ID:             meetingID,
		Title:          meta.Title,
		Date:           date,
		Participants:   append([]string(nil), participants...),
		Status:         MeetingStatusDraft,
		TicketID:       meta.TicketID,
		Project:        meta.Project,
		Tags:           append([]string(nil), meta.Tags...),
		ExternalTicket: meta.ExternalTicket,
	}, nil
}

// ListMeetings returns all meetings sorted by date descending.
func (s *Store) ListMeetings(projectFilter string) ([]Meeting, error) {
	if projectFilter != "" && hasPathTraversal(projectFilter) {
		return nil, fmt.Errorf("invalid project filter")
	}

	entries, err := os.ReadDir(s.notesDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Meeting{}, nil
		}

		return nil, fmt.Errorf("read notes directory: %w", err)
	}

	projects := make([]string, 0, len(entries))
	meetings := make([]Meeting, 0)
	if projectFilter != "" {
		projectMetaPath := filepath.Join(s.notesDir, projectFilter, projectMetaFileName)
		if _, statErr := os.Stat(projectMetaPath); statErr != nil {
			if errors.Is(statErr, os.ErrNotExist) {
				return []Meeting{}, nil
			}

			return nil, fmt.Errorf("check project metadata for %q: %w", projectFilter, statErr)
		}

		projects = append(projects, projectFilter)
	} else {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			projectPrefix := entry.Name()
			projectMetaPath := filepath.Join(s.notesDir, projectPrefix, projectMetaFileName)
			if _, statErr := os.Stat(projectMetaPath); statErr != nil {
				if errors.Is(statErr, os.ErrNotExist) {
					metaPath := filepath.Join(s.notesDir, projectPrefix, metaFileName)
					metaBytes, readErr := os.ReadFile(metaPath)
					if readErr != nil {
						if errors.Is(readErr, os.ErrNotExist) {
							continue
						}

						return nil, fmt.Errorf("read meeting metadata for %q: %w", projectPrefix, readErr)
					}

					var meta meetingMeta
					if unmarshalErr := json.Unmarshal(metaBytes, &meta); unmarshalErr != nil {
						continue
					}

					meetingDate, parseErr := time.Parse(dateLayout, meta.Date)
					if parseErr != nil {
						continue
					}

					meetingID := projectPrefix
					meetingPath := filepath.Join(s.notesDir, meetingID)
					status := MeetingStatusDraft
					if _, structuredStatErr := os.Stat(filepath.Join(meetingPath, structuredFileName)); structuredStatErr == nil {
						status = MeetingStatusStructured
					} else if !errors.Is(structuredStatErr, os.ErrNotExist) {
						return nil, fmt.Errorf("check structured notes for %q: %w", meetingID, structuredStatErr)
					}

					meetings = append(meetings, Meeting{
						ID:             meetingID,
						Title:          meta.Title,
						Date:           meetingDate,
						Participants:   append([]string(nil), meta.Participants...),
						Status:         status,
						TicketID:       meta.TicketID,
						Project:        meta.Project,
						Tags:           append([]string(nil), meta.Tags...),
						ExternalTicket: meta.ExternalTicket,
					})

					continue
				}

				return nil, fmt.Errorf("check project metadata for %q: %w", projectPrefix, statErr)
			}

			projects = append(projects, projectPrefix)
		}
	}

	for _, projectPrefix := range projects {
		projectPath := filepath.Join(s.notesDir, projectPrefix)
		projectEntries, readProjectErr := os.ReadDir(projectPath)
		if readProjectErr != nil {
			return nil, fmt.Errorf("read project directory for %q: %w", projectPrefix, readProjectErr)
		}

		for _, entry := range projectEntries {
			if !entry.IsDir() {
				continue
			}

			meetingID := entry.Name()
			meetingPath := s.MeetingDir(projectPrefix, meetingID)
			metaPath := filepath.Join(meetingPath, metaFileName)

			metaBytes, readErr := os.ReadFile(metaPath)
			if readErr != nil {
				if errors.Is(readErr, os.ErrNotExist) {
					continue
				}

				return nil, fmt.Errorf("read meeting metadata for %q: %w", meetingID, readErr)
			}

			var meta meetingMeta
			if unmarshalErr := json.Unmarshal(metaBytes, &meta); unmarshalErr != nil {
				continue
			}

			meetingDate, parseErr := time.Parse(dateLayout, meta.Date)
			if parseErr != nil {
				continue
			}

			status := MeetingStatusDraft
			if _, statErr := os.Stat(filepath.Join(meetingPath, structuredFileName)); statErr == nil {
				status = MeetingStatusStructured
			} else if !errors.Is(statErr, os.ErrNotExist) {
				return nil, fmt.Errorf("check structured notes for %q: %w", meetingID, statErr)
			}

			resolvedProject := meta.Project
			if resolvedProject == "" {
				resolvedProject = projectPrefix
			}

			meetings = append(meetings, Meeting{
				ID:             meetingID,
				Title:          meta.Title,
				Date:           meetingDate,
				Participants:   append([]string(nil), meta.Participants...),
				Status:         status,
				TicketID:       meta.TicketID,
				Project:        resolvedProject,
				Tags:           append([]string(nil), meta.Tags...),
				ExternalTicket: meta.ExternalTicket,
			})
		}
	}

	sort.Slice(meetings, func(leftIndex, rightIndex int) bool {
		leftDate := meetings[leftIndex].Date
		rightDate := meetings[rightIndex].Date
		if leftDate.Equal(rightDate) {
			return meetings[leftIndex].ID > meetings[rightIndex].ID
		}

		return leftDate.After(rightDate)
	})

	return meetings, nil
}

// LoadRawNotes returns raw.txt contents for a meeting.
func (s *Store) LoadRawNotes(project, meetingID string) (string, error) {
	rawPath := filepath.Join(s.MeetingDir(project, meetingID), rawNotesFileName)
	contentBytes, err := os.ReadFile(rawPath)
	if err != nil {
		return "", fmt.Errorf("read raw notes: %w", err)
	}

	return string(contentBytes), nil
}

// SaveRawNotes atomically writes raw.txt contents.
func (s *Store) SaveRawNotes(project, meetingID, content string) error {
	meetingPath := s.MeetingDir(project, meetingID)
	targetPath := filepath.Join(meetingPath, rawNotesFileName)
	tempPath := filepath.Join(meetingPath, rawNotesFileName+".tmp")

	if err := os.WriteFile(tempPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write temporary raw notes: %w", err)
	}

	if err := os.Rename(tempPath, targetPath); err != nil {
		if removeErr := os.Remove(tempPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return fmt.Errorf("rename temporary raw notes: %w (cleanup error: %v)", err, removeErr)
		}

		return fmt.Errorf("rename temporary raw notes: %w", err)
	}

	return nil
}

// LoadStructuredNotes returns structured.md contents, or os.ErrNotExist when absent.
func (s *Store) LoadStructuredNotes(project, meetingID string) (string, error) {
	structuredPath := filepath.Join(s.MeetingDir(project, meetingID), structuredFileName)
	contentBytes, err := os.ReadFile(structuredPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", os.ErrNotExist
		}

		return "", fmt.Errorf("read structured notes: %w", err)
	}

	return string(contentBytes), nil
}

// SaveStructuredNotes writes structured.md contents.
func (s *Store) SaveStructuredNotes(project, meetingID, content string) error {
	meetingPath := s.MeetingDir(project, meetingID)
	targetPath := filepath.Join(meetingPath, structuredFileName)
	tempPath := filepath.Join(meetingPath, structuredFileName+".tmp")

	if err := os.WriteFile(tempPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write temporary structured notes: %w", err)
	}

	if err := os.Rename(tempPath, targetPath); err != nil {
		if removeErr := os.Remove(tempPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return fmt.Errorf("rename temporary structured notes: %w (cleanup error: %v)", err, removeErr)
		}

		return fmt.Errorf("rename temporary structured notes: %w", err)
	}

	return nil
}

// DeleteMeeting removes a meeting directory and all files.
func (s *Store) DeleteMeeting(project, meetingID string) error {
	if err := os.RemoveAll(s.MeetingDir(project, meetingID)); err != nil {
		return fmt.Errorf("delete meeting directory: %w", err)
	}

	return nil
}

// MeetingDir returns absolute meeting directory path.
func (s *Store) MeetingDir(project, meetingID string) string {
	cleanNotesDir := filepath.Clean(s.notesDir)
	invalidPath := filepath.Join(cleanNotesDir, "_invalid_meeting_id")

	if hasPathTraversal(meetingID) || (project != "" && hasPathTraversal(project)) {
		return invalidPath
	}

	candidatePath := ""
	if project != "" {
		candidatePath = filepath.Clean(filepath.Join(cleanNotesDir, project, meetingID))
	} else {
		candidatePath = filepath.Clean(filepath.Join(cleanNotesDir, meetingID))
	}

	relativePath, err := filepath.Rel(cleanNotesDir, candidatePath)
	if err != nil {
		return invalidPath
	}

	if relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return invalidPath
	}

	return candidatePath
}

// incrementProjectSequence atomically reads, increments, and writes back project.json NextSequence.
// Returns the generated ticket ID (e.g. "INFRA-003").
func (s *Store) incrementProjectSequence(project string) (string, error) {
	if strings.TrimSpace(project) == "" {
		return "", fmt.Errorf("project prefix is required")
	}

	if hasPathTraversal(project) {
		return "", fmt.Errorf("invalid project prefix")
	}

	projectMetaPath := filepath.Join(s.notesDir, project, projectMetaFileName)
	metaBytes, err := os.ReadFile(projectMetaPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("read project metadata: %w", ErrProjectNotFound)
		}

		return "", fmt.Errorf("read project metadata: %w", err)
	}

	var meta projectMeta
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return "", fmt.Errorf("unmarshal project metadata: %w", err)
	}

	if meta.Prefix == "" {
		meta.Prefix = project
	}

	if meta.NextSequence < 1 {
		meta.NextSequence = 1
	}

	ticketID := fmt.Sprintf("%s-%03d", meta.Prefix, meta.NextSequence)
	meta.NextSequence++

	updatedMetaBytes, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("marshal project metadata: %w", err)
	}

	tempPath := filepath.Join(s.notesDir, project, projectMetaFileName+".tmp")
	if err := os.WriteFile(tempPath, updatedMetaBytes, 0o644); err != nil {
		return "", fmt.Errorf("write temporary project metadata: %w", err)
	}

	if err := os.Rename(tempPath, projectMetaPath); err != nil {
		if removeErr := os.Remove(tempPath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			return "", fmt.Errorf("rename temporary project metadata: %w (cleanup error: %v)", err, removeErr)
		}

		return "", fmt.Errorf("rename temporary project metadata: %w", err)
	}

	return ticketID, nil
}

func hasPathTraversal(value string) bool {
	return strings.Contains(value, "..") || strings.Contains(value, "/") || strings.Contains(value, "\\")
}

func slugifyTitle(title string) string {
	lowerTitle := strings.ToLower(title)
	withHyphens := whitespacePattern.ReplaceAllString(lowerTitle, "-")
	filtered := nonSlugPattern.ReplaceAllString(withHyphens, "")
	collapsed := hyphenPattern.ReplaceAllString(filtered, "-")
	trimmed := strings.Trim(collapsed, "-")

	if len(trimmed) > maxSlugLength {
		trimmed = trimmed[:maxSlugLength]
		trimmed = strings.Trim(trimmed, "-")
	}

	return trimmed
}
