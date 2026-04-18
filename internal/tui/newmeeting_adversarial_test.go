package tui

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/glieske/recap/internal/storage"
)

func updateAdversarialNewMeetingModel(t *testing.T, m NewMeetingModel, msg tea.Msg) (NewMeetingModel, tea.Cmd) {
	t.Helper()

	updated, cmd := m.Update(msg)
	updatedModel, ok := updated.(NewMeetingModel)
	if !ok {
		t.Fatalf("expected NewMeetingModel from Update, got %T", updated)
	}

	return updatedModel, cmd
}

func TestAdversarialNewMeetingSplitAndTrimOversizedUnicodeAndInvariants(t *testing.T) {
	oversized := strings.Repeat("A", 12*1024)
	input := oversized + ", \n\t,emoji🙂,\x00null,<script>alert(1)</script>,line1\nline2"

	got := splitAndTrim(input)
	want := []string{oversized, "emoji🙂", "\x00null", "<script>alert(1)</script>", "line1\nline2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitAndTrim mismatch for adversarial payload: got %q want %q", got, want)
	}

	config := &quick.Config{MaxCount: 200}
	propErr := quick.Check(func(s string) bool {
		parts := splitAndTrim(s)
		for _, p := range parts {
			if p == "" {
				return false
			}
			if p != strings.TrimSpace(p) {
				return false
			}
		}
		return true
	}, config)
	if propErr != nil {
		t.Fatalf("splitAndTrim invariant failed: %v", propErr)
	}
}

func TestAdversarialNewMeetingSubmitMeetingCmdNilStoreAndEmptyValues(t *testing.T) {
	t.Run("nil store", func(t *testing.T) {
		m := NewMeetingModel{values: &newMeetingFormValues{}}
		msg := m.submitMeetingCmd()()
		errMsg, ok := msg.(NewMeetingErrMsg)
		if !ok {
			t.Fatalf("expected NewMeetingErrMsg, got %T", msg)
		}
		if errMsg.Err == nil || errMsg.Err.Error() != "store is not configured" {
			t.Fatalf("expected exact store-not-configured error, got %v", errMsg.Err)
		}
	})

	t.Run("empty values invalid date", func(t *testing.T) {
		store := storage.NewStore(t.TempDir())
		m := NewMeetingModel{store: store, values: &newMeetingFormValues{}}

		msg := m.submitMeetingCmd()()
		errMsg, ok := msg.(NewMeetingErrMsg)
		if !ok {
			t.Fatalf("expected NewMeetingErrMsg, got %T", msg)
		}
		if errMsg.Err == nil || !strings.Contains(errMsg.Err.Error(), "invalid date") {
			t.Fatalf("expected invalid date error, got %v", errMsg.Err)
		}
	})
}

func TestAdversarialNewMeetingSubmitMeetingCmdDateBoundaries(t *testing.T) {
	dates := []string{"0001-01-01", "9999-12-31"}

	for _, date := range dates {
		t.Run(date, func(t *testing.T) {
			store := storage.NewStore(t.TempDir())
			if _, err := store.CreateProject("Infra", "INFRA"); err != nil {
				t.Fatalf("CreateProject: %v", err)
			}

			m := NewMeetingModel{
				store: store,
				values: &newMeetingFormValues{
					project: "INFRA",
					title:   "  boundary date payload  ",
					date:    date,
				},
			}

			msg := m.submitMeetingCmd()()
			created, ok := msg.(MeetingCreatedMsg)
			if !ok {
				t.Fatalf("expected MeetingCreatedMsg for %s, got %T", date, msg)
			}
			if created.Meeting == nil {
				t.Fatalf("expected meeting to be non-nil for %s", date)
			}
			if created.Meeting.Date.Format(isoDateLayout) != date {
				t.Fatalf("expected date %s, got %s", date, created.Meeting.Date.Format(isoDateLayout))
			}
		})
	}
}

func TestAdversarialNewMeetingConstructorAndUpdateNilSafety(t *testing.T) {
	t.Run("constructor with nil store", func(t *testing.T) {
		m := NewNewMeetingModel(nil, 91, 27)
		if m.store != nil {
			t.Fatalf("expected nil store in model, got non-nil")
		}
		if m.form == nil {
			t.Fatalf("expected form to be initialized even with nil store")
		}
		if m.creatingProject != true {
			t.Fatalf("expected creatingProject=true, got %v", m.creatingProject)
		}
		if m.values.project != newProjectOptionValue {
			t.Fatalf("expected default project %q, got %q", newProjectOptionValue, m.values.project)
		}
	})

	t.Run("update unknown msg with nil form", func(t *testing.T) {
		m := NewMeetingModel{values: &newMeetingFormValues{}}
		updated, cmd := updateAdversarialNewMeetingModel(t, m, struct{}{})
		if cmd != nil {
			t.Fatalf("expected nil command for unknown message with nil form")
		}
		if updated.form != nil {
			t.Fatalf("expected form to stay nil, got non-nil")
		}
	})

	t.Run("esc with nil form still navigates", func(t *testing.T) {
		m := NewMeetingModel{values: &newMeetingFormValues{}}
		updated, cmd := updateAdversarialNewMeetingModel(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
		if !updated.cancelled {
			t.Fatalf("expected cancelled=true after ESC")
		}
		if cmd == nil {
			t.Fatalf("expected navigate command after ESC")
		}
		nav, ok := cmd().(NavigateMsg)
		if !ok {
			t.Fatalf("expected NavigateMsg, got %T", cmd())
		}
		if nav.Screen != ScreenMeetingList {
			t.Fatalf("expected screen %v, got %v", ScreenMeetingList, nav.Screen)
		}
	})
}

func TestAdversarialNewMeetingBuildFormProjectSliceBoundaries(t *testing.T) {
	model := NewMeetingModel{values: &newMeetingFormValues{project: newProjectOptionValue}}

	emptyForm := buildNewMeetingForm(&model, []storage.Project{})
	if emptyForm == nil {
		t.Fatalf("expected non-nil form for empty projects")
	}
	model.form = emptyForm
	if len(model.View().Content) == 0 {
		t.Fatalf("expected non-empty form view for empty projects")
	}

	projects := make([]storage.Project, 0, 1500)
	for i := 0; i < 1500; i++ {
		prefix := fmt.Sprintf("P%04d", i)
		projects = append(projects, storage.Project{Name: "Proj " + prefix, Prefix: prefix})
	}
	largeForm := buildNewMeetingForm(&model, projects)
	if largeForm == nil {
		t.Fatalf("expected non-nil form for large projects slice")
	}

	model.form = largeForm
	model.values.project = "P0001"
	updated, _ := updateAdversarialNewMeetingModel(t, model, struct{}{})
	if updated.creatingProject {
		t.Fatalf("expected creatingProject=false for existing project selection")
	}
}

func TestAdversarialNewMeetingSubmitMeetingCmdInjectionPayloadBinding(t *testing.T) {
	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("Infrastructure", "INFRA"); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	veryLong := strings.Repeat("Z", 11*1024)
	titlePayload := "   <script>alert(1)</script> ../../etc/passwd ${jndi:ldap://x} " + veryLong + "   "
	participantsPayload := "  ' OR 1=1 -- , <img src=x onerror=alert(1)>, ..\\..\\windows, 😀,\x00bin, ${env:SECRET}, "
	tagsPayload := " prod, <script>x</script>, ../, zero\u200Bwidth, NaN, Infinity "
	externalPayload := "   https://tickets.example/INFRA-999?next=../../admin#${7*7}   "

	m := NewMeetingModel{
		store: store,
		values: &newMeetingFormValues{
			project:        "INFRA",
			title:          titlePayload,
			date:           "2026-04-16",
			participants:   participantsPayload,
			tags:           tagsPayload,
			externalTicket: externalPayload,
		},
	}

	msg := m.submitMeetingCmd()()
	created, ok := msg.(MeetingCreatedMsg)
	if !ok {
		t.Fatalf("expected MeetingCreatedMsg, got %T", msg)
	}
	if created.Meeting == nil {
		t.Fatalf("expected created meeting to be non-nil")
	}

	if created.Meeting.Title != strings.TrimSpace(titlePayload) {
		t.Fatalf("title mismatch: got %q want %q", created.Meeting.Title, strings.TrimSpace(titlePayload))
	}

	wantParticipants := []string{"' OR 1=1 --", "<img src=x onerror=alert(1)>", "..\\..\\windows", "😀", "\x00bin", "${env:SECRET}"}
	if !reflect.DeepEqual(created.Meeting.Participants, wantParticipants) {
		t.Fatalf("participants mismatch: got %q want %q", created.Meeting.Participants, wantParticipants)
	}

	wantTags := []string{"prod", "<script>x</script>", "../", "zero\u200Bwidth", "NaN", "Infinity"}
	if !reflect.DeepEqual(created.Meeting.Tags, wantTags) {
		t.Fatalf("tags mismatch: got %q want %q", created.Meeting.Tags, wantTags)
	}

	if created.Meeting.ExternalTicket != strings.TrimSpace(externalPayload) {
		t.Fatalf("external ticket mismatch: got %q want %q", created.Meeting.ExternalTicket, strings.TrimSpace(externalPayload))
	}
}

func TestAdversarialNewMeetingProjectPrefixValidationBoundaries(t *testing.T) {
	t.Run("invalid prefixes rejected", func(t *testing.T) {
		invalidPrefixes := []string{"A", "ABCDEFGHIJK", "A-1", "A B", "../X", "💥", ""}

		for _, prefix := range invalidPrefixes {
			t.Run(prefix, func(t *testing.T) {
				store := storage.NewStore(t.TempDir())
				m := NewMeetingModel{
					store: store,
					values: &newMeetingFormValues{
						project:        newProjectOptionValue,
						newProjectName: "Attack Surface",
						newProjectPref: prefix,
						title:          "Prefix Boundary",
						date:           "2026-04-16",
					},
				}

				msg := m.submitMeetingCmd()()
				errMsg, ok := msg.(NewMeetingErrMsg)
				if !ok {
					t.Fatalf("expected NewMeetingErrMsg for prefix %q, got %T", prefix, msg)
				}
				if !errors.Is(errMsg.Err, storage.ErrInvalidPrefix) {
					t.Fatalf("expected ErrInvalidPrefix for prefix %q, got %v", prefix, errMsg.Err)
				}
			})
		}
	})

	t.Run("lowercase prefix normalized to uppercase", func(t *testing.T) {
		store := storage.NewStore(t.TempDir())
		m := NewMeetingModel{
			store: store,
			values: &newMeetingFormValues{
				project:        newProjectOptionValue,
				newProjectName: "Lowercase Prefix",
				newProjectPref: "ab",
				title:          "Normalization",
				date:           "2026-04-16",
			},
		}

		msg := m.submitMeetingCmd()()
		created, ok := msg.(MeetingCreatedMsg)
		if !ok {
			t.Fatalf("expected MeetingCreatedMsg, got %T", msg)
		}
		if created.Meeting == nil {
			t.Fatalf("expected created meeting to be non-nil")
		}
		if created.Meeting.Project != "AB" {
			t.Fatalf("expected normalized project prefix AB, got %q", created.Meeting.Project)
		}
	})
}

func TestAdversarialNewMeetingUpdateRepeatedSubmissionGuard(t *testing.T) {
	store := storage.NewStore(t.TempDir())
	if _, err := store.CreateProject("Infrastructure", "INFRA"); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}

	m := NewMeetingModel{
		store: store,
		values: &newMeetingFormValues{
			project: "INFRA",
			title:   "Repeat Submit",
			date:    "2026-04-16",
		},
		form: buildNewMeetingForm(&NewMeetingModel{values: &newMeetingFormValues{project: "INFRA"}}, []storage.Project{{Name: "Infrastructure", Prefix: "INFRA"}}),
	}

	if m.form == nil {
		t.Fatalf("expected initialized form")
	}
	m.form.State = huh.StateCompleted

	updated1, cmd1 := updateAdversarialNewMeetingModel(t, m, struct{}{})
	if !updated1.submitted {
		t.Fatalf("expected submitted=true after first completed update")
	}
	if cmd1 == nil {
		t.Fatalf("expected submit command on first completed update")
	}

	updated2, cmd2 := updateAdversarialNewMeetingModel(t, updated1, struct{}{})
	if !updated2.submitted {
		t.Fatalf("expected submitted to remain true on repeated updates")
	}
	if cmd2 != nil {
		msg := cmd2()
		if _, isCreated := msg.(MeetingCreatedMsg); isCreated {
			t.Fatalf("expected no second MeetingCreatedMsg submission, got %T", msg)
		}
	}
}
