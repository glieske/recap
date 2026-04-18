package tui

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/glieske/recap/internal/storage"
)

const (
	newProjectOptionValue = "__new_project__"
	isoDateLayout         = "2006-01-02"
)

var projectPrefixInputPattern = regexp.MustCompile(`^[A-Z0-9]{2,10}$`)

type MeetingCreatedMsg struct {
	Meeting *storage.Meeting
}

type NewMeetingErrMsg struct {
	Err error
}

type newMeetingFormValues struct {
	project        string
	title          string
	date           string
	participants   string
	tags           string
	externalTicket string
	newProjectName string
	newProjectPref string
}

type NewMeetingModel struct {
	form             *huh.Form
	store            *storage.Store
	submitted        bool
	cancelled        bool
	creatingProject  bool
	newProjectName   string
	newProjectPrefix string
	width            int
	height           int

	values *newMeetingFormValues
	err    error
}

func NewNewMeetingModel(store *storage.Store, width, height int) NewMeetingModel {
	projects := make([]storage.Project, 0)
	if store != nil {
		listedProjects, err := store.ListProjects()
		if err == nil {
			projects = listedProjects
		}
	}

	defaultProjectValue := newProjectOptionValue
	if len(projects) > 0 {
		defaultProjectValue = projects[0].Prefix
	}

	model := NewMeetingModel{
		store:            store,
		creatingProject:  len(projects) == 0,
		width:            width,
		height:           height,
		newProjectName:   "",
		newProjectPrefix: "",
		values: &newMeetingFormValues{
			project:        defaultProjectValue,
			date:           time.Now().Format(isoDateLayout),
			newProjectPref: "",
		},
	}

	model.form = buildNewMeetingForm(&model, projects)

	return model
}

func (m NewMeetingModel) Init() tea.Cmd {
	if m.form == nil {
		return nil
	}

	return m.form.Init()
}

func (m NewMeetingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typedMsg.Width
		m.height = typedMsg.Height
		return m, nil
	case tea.KeyPressMsg:
		switch typedMsg.String() {
		case "esc", "ctrl+c":
			m.cancelled = true
			return m, func() tea.Msg {
				return NavigateMsg{Screen: ScreenMeetingList}
			}
		}
	case NewMeetingErrMsg:
		m.err = typedMsg.Err
		return m, nil
	}

	if m.form == nil {
		return m, nil
	}

	updatedForm, formCmd := m.form.Update(msg)
	if castForm, ok := updatedForm.(*huh.Form); ok {
		m.form = castForm
	}

	m.creatingProject = m.values.project == newProjectOptionValue
	m.newProjectName = strings.TrimSpace(m.values.newProjectName)
	m.newProjectPrefix = strings.ToUpper(strings.TrimSpace(m.values.newProjectPref))

	if m.form.State == huh.StateCompleted && !m.submitted {
		m.submitted = true
		return m, m.submitMeetingCmd()
	}

	if m.form.State == huh.StateAborted && !m.cancelled {
		m.cancelled = true
		return m, func() tea.Msg {
			return NavigateMsg{Screen: ScreenMeetingList}
		}
	}

	return m, formCmd
}

func (m NewMeetingModel) View() tea.View {
	if m.form == nil {
		return tea.NewView("Unable to render new meeting form")
	}

	view := m.form.View()
	if m.err != nil {
		view += "\n\nError: " + m.err.Error()
	}

	if m.cancelled {
		view += "\n\nCancelled"
	}

	return tea.NewView(view)
}

func (m NewMeetingModel) submitMeetingCmd() tea.Cmd {
	return func() tea.Msg {
		if m.store == nil {
			return NewMeetingErrMsg{Err: errors.New("store is not configured")}
		}

		selectedProject := strings.TrimSpace(m.values.project)
		if selectedProject == newProjectOptionValue {
			projectName := strings.TrimSpace(m.values.newProjectName)
			projectPrefix := strings.ToUpper(strings.TrimSpace(m.values.newProjectPref))

			_, createProjectErr := m.store.CreateProject(projectName, projectPrefix)
			if createProjectErr != nil {
				return NewMeetingErrMsg{Err: createProjectErr}
			}

			selectedProject = projectPrefix
		}

		meetingDate, parseErr := time.Parse(isoDateLayout, strings.TrimSpace(m.values.date))
		if parseErr != nil {
			return NewMeetingErrMsg{Err: fmt.Errorf("invalid date: %w", parseErr)}
		}

		participants := splitAndTrim(m.values.participants)
		tags := splitAndTrim(m.values.tags)
		externalTicket := strings.TrimSpace(m.values.externalTicket)

		meeting, createMeetingErr := m.store.CreateMeeting(
			strings.TrimSpace(m.values.title),
			meetingDate,
			participants,
			selectedProject,
			tags,
			externalTicket,
		)
		if createMeetingErr != nil {
			return NewMeetingErrMsg{Err: createMeetingErr}
		}

		return MeetingCreatedMsg{Meeting: meeting}
	}
}

func buildNewMeetingForm(model *NewMeetingModel, projects []storage.Project) *huh.Form {
	projectOptions := make([]huh.Option[string], 0, len(projects)+1)
	for _, project := range projects {
		optionLabel := fmt.Sprintf("%s (%s)", project.Name, project.Prefix)
		projectOptions = append(projectOptions, huh.NewOption(optionLabel, project.Prefix))
	}

	projectOptions = append(projectOptions, huh.NewOption("[+ New Project]", newProjectOptionValue))

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Project").
				Description("Select an existing project or create a new one").
				Options(projectOptions...).
				Value(&model.values.project),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("New Project Name").
				Description("Required only when creating a new project").
				Value(&model.values.newProjectName).
				Validate(func(value string) error {
					if model.values.project != newProjectOptionValue {
						return nil
					}

					if strings.TrimSpace(value) == "" {
						return errors.New("project name is required")
					}

					return nil
				}),
			huh.NewInput().
				Title("New Project Prefix").
				Description("2-10 uppercase alphanumeric characters, e.g. INFRA").
				Value(&model.values.newProjectPref).
				Validate(func(value string) error {
					if model.values.project != newProjectOptionValue {
						return nil
					}

					normalizedValue := strings.ToUpper(strings.TrimSpace(value))
					if !projectPrefixInputPattern.MatchString(normalizedValue) {
						return errors.New("prefix must be 2-10 uppercase alphanumeric characters")
					}

					return nil
				}),
		).WithHideFunc(func() bool {
			return model.values.project != newProjectOptionValue
		}),
		huh.NewGroup(
			huh.NewInput().
				Title("Title").
				Value(&model.values.title).
				Validate(func(value string) error {
					if strings.TrimSpace(value) == "" {
						return errors.New("title is required")
					}

					return nil
				}),
			huh.NewInput().
				Title("Date").
				Description("YYYY-MM-DD").
				Value(&model.values.date).
				Validate(func(value string) error {
					trimmedValue := strings.TrimSpace(value)
					if trimmedValue == "" {
						return errors.New("date is required")
					}

					if _, err := time.Parse(isoDateLayout, trimmedValue); err != nil {
						return errors.New("date must be in YYYY-MM-DD format")
					}

					return nil
				}),
			huh.NewInput().
				Title("Participants").
				Description("Comma-separated names (optional)").
				Value(&model.values.participants),
			huh.NewInput().
				Title("Tags").
				Description("Comma-separated tags (optional)").
				Value(&model.values.tags),
			huh.NewInput().
				Title("External Ticket").
				Description("Optional reference, e.g. Jira URL").
				Value(&model.values.externalTicket),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeDracula))
}

func splitAndTrim(value string) []string {
	chunks := strings.Split(value, ",")
	cleanedValues := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		trimmedChunk := strings.TrimSpace(chunk)
		if trimmedChunk == "" {
			continue
		}

		cleanedValues = append(cleanedValues, trimmedChunk)
	}

	return cleanedValues
}
