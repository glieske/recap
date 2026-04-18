package tui

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/glieske/recap/internal/storage"
)

type MeetingItem struct {
	meeting storage.Meeting
}

func (m MeetingItem) FilterValue() string {
	return m.meeting.Title + " " + m.meeting.TicketID + " " + strings.Join(m.meeting.Tags, " ")
}

func (m MeetingItem) Title() string {
	return fmt.Sprintf("[%s] %s", m.meeting.TicketID, m.meeting.Title)
}

func (m MeetingItem) Description() string {
	statusBadge := "○ DRAFT"
	if m.meeting.Status == storage.MeetingStatusStructured {
		statusBadge = "✓ STRUCTURED"
	}

	return fmt.Sprintf("%s | %s | %s", m.meeting.Date.Format("2006-01-02"), m.meeting.Project, statusBadge)
}

type MeetingSelectedMsg struct {
	Meeting storage.Meeting
}

type DeleteErrMsg struct {
	Err error
}

type RequestDeleteMsg struct {
	Meeting storage.Meeting
}

type ListModel struct {
	list          list.Model
	store         *storage.Store
	allMeetings   []storage.Meeting
	projectFilter string
	tagFilter     string
	width         int
	height        int
}

func NewListModel(store *storage.Store, width, height int) ListModel {
	meetings := []storage.Meeting{}
	if store != nil {
		loadedMeetings, err := store.ListMeetings("")
		if err == nil {
			meetings = loadedMeetings
		}
	}

	items := meetingsToItems(meetings)
	bubblesList := list.New(items, list.NewDefaultDelegate(), width, height)
	bubblesList.Title = "Meeting Notes"
	bubblesList.SetFilteringEnabled(true)

	return ListModel{
		list:        bubblesList,
		store:       store,
		allMeetings: meetings,
		width:       width,
		height:      height,
	}
}

func (m ListModel) Init() tea.Cmd {
	return nil
}

func (m ListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typedMsg.Width
		m.height = typedMsg.Height
		m.list.SetSize(m.width, m.height)
		return m, nil
	case tea.KeyPressMsg:
		if m.list.FilterState() == list.Filtering {
			updatedList, cmd := m.list.Update(msg)
			m.list = updatedList
			return m, cmd
		}

		switch typedMsg.String() {
		case "n":
			return m, func() tea.Msg {
				return NavigateMsg{Screen: ScreenNewMeeting}
			}
		case "enter":
			selectedItem, ok := m.list.SelectedItem().(MeetingItem)
			if !ok {
				return m, nil
			}

			return m, func() tea.Msg {
				return MeetingSelectedMsg{Meeting: selectedItem.meeting}
			}
		case "d":
			selectedItem, ok := m.list.SelectedItem().(MeetingItem)
			if !ok {
				return m, nil
			}
			return m, func() tea.Msg {
				return RequestDeleteMsg{Meeting: selectedItem.meeting}
			}
		case "f":
			projects := uniqueProjectsFromMeetings(m.allMeetings)
			m.projectFilter = cycleFilterValue(m.projectFilter, projects)
			return m, m.RefreshMeetings()
		case "t":
			tags := uniqueTagsFromMeetings(m.allMeetings)
			m.tagFilter = cycleFilterValue(m.tagFilter, tags)
			return m, m.RefreshMeetings()
		}
	}

	updatedList, cmd := m.list.Update(msg)
	m.list = updatedList
	return m, cmd
}

func (m ListModel) View() tea.View {
	if len(m.list.Items()) == 0 && m.projectFilter == "" && m.tagFilter == "" && m.list.FilterState() == list.Unfiltered {
		emptyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
		return tea.NewView(emptyStyle.Render("No meetings yet. Press 'n' to create one."))
	}

	return tea.NewView(m.list.View())
}

func (m *ListModel) RefreshMeetings() tea.Cmd {
	if m.store == nil {
		m.allMeetings = []storage.Meeting{}
		return m.list.SetItems([]list.Item{})
	}

	allMeetings, err := m.store.ListMeetings("")
	if err != nil {
		m.allMeetings = []storage.Meeting{}
		return m.list.SetItems([]list.Item{})
	}
	m.allMeetings = allMeetings

	meetings := allMeetings
	if m.projectFilter != "" {
		projectMeetings := make([]storage.Meeting, 0, len(allMeetings))
		for _, meeting := range allMeetings {
			if meeting.Project == m.projectFilter {
				projectMeetings = append(projectMeetings, meeting)
			}
		}

		meetings = projectMeetings
	}

	if m.tagFilter != "" {
		filteredMeetings := make([]storage.Meeting, 0, len(meetings))
		for _, meeting := range meetings {
			if meetingHasTag(meeting, m.tagFilter) {
				filteredMeetings = append(filteredMeetings, meeting)
			}
		}

		meetings = filteredMeetings
	}

	return m.list.SetItems(meetingsToItems(meetings))
}

func meetingsToItems(meetings []storage.Meeting) []list.Item {
	items := make([]list.Item, 0, len(meetings))
	for _, meeting := range meetings {
		items = append(items, MeetingItem{meeting: meeting})
	}

	return items
}

func uniqueProjectsFromMeetings(meetings []storage.Meeting) []string {
	projectSet := map[string]struct{}{}
	for _, meeting := range meetings {
		if meeting.Project == "" {
			continue
		}

		projectSet[meeting.Project] = struct{}{}
	}

	projects := make([]string, 0, len(projectSet))
	for project := range projectSet {
		projects = append(projects, project)
	}
	sort.Strings(projects)

	return projects
}

func uniqueTagsFromMeetings(meetings []storage.Meeting) []string {
	tagSet := map[string]struct{}{}
	for _, meeting := range meetings {
		for _, tag := range meeting.Tags {
			trimmedTag := strings.TrimSpace(tag)
			if trimmedTag == "" {
				continue
			}

			tagSet[trimmedTag] = struct{}{}
		}
	}

	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	return tags
}

func cycleFilterValue(currentValue string, values []string) string {
	if len(values) == 0 {
		return ""
	}

	if currentValue == "" {
		return values[0]
	}

	for valueIndex, value := range values {
		if value != currentValue {
			continue
		}

		nextIndex := valueIndex + 1
		if nextIndex >= len(values) {
			return ""
		}

		return values[nextIndex]
	}

	return values[0]
}

func meetingHasTag(meeting storage.Meeting, targetTag string) bool {
	for _, tag := range meeting.Tags {
		if strings.TrimSpace(tag) == targetTag {
			return true
		}
	}

	return false
}
