//go:build gui

package gui

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/glieske/recap/internal/storage"
)

type meetingsScreen struct {
	store    *storage.Store
	meetings []storage.Meeting
	list     *widget.List
	filter   *widget.Select
	onSelect func(storage.Meeting)
}

func NewMeetingsScreen(store *storage.Store, onSelect func(storage.Meeting), onNew func()) fyne.CanvasObject {
	screen := &meetingsScreen{
		store:    store,
		meetings: []storage.Meeting{},
		onSelect: onSelect,
	}

	screen.filter = widget.NewSelect([]string{"All"}, nil)
	screen.filter.SetSelected("All")
	screen.filter.OnChanged = func(string) {
		screen.refresh()
	}

	screen.list = widget.NewList(
		func() int {
			return len(screen.meetings)
		},
		func() fyne.CanvasObject {
			titleLabel := widget.NewLabel("")
			titleLabel.Wrapping = fyne.TextTruncate
			titleColumn := container.NewGridWrap(fyne.NewSize(280, titleLabel.MinSize().Height), titleLabel)

			dateLabel := widget.NewLabel("")
			dateColumn := container.NewGridWrap(fyne.NewSize(110, dateLabel.MinSize().Height), dateLabel)

			projectLabel := widget.NewLabel("")
			projectColumn := container.NewGridWrap(fyne.NewSize(100, projectLabel.MinSize().Height), projectLabel)

			statusLabel := widget.NewLabel("")
			statusColumn := container.NewGridWrap(fyne.NewSize(100, statusLabel.MinSize().Height), statusLabel)

			return container.NewHBox(titleColumn, dateColumn, projectColumn, statusColumn)
		},
		func(id widget.ListItemID, item fyne.CanvasObject) {
			if id < 0 || id >= len(screen.meetings) {
				return
			}

			meeting := screen.meetings[id]
			columns, ok := item.(*fyne.Container)
			if !ok || len(columns.Objects) != 4 {
				return
			}

			titleContainer, ok := columns.Objects[0].(*fyne.Container)
			if !ok || len(titleContainer.Objects) != 1 {
				return
			}
			titleLabel, ok := titleContainer.Objects[0].(*widget.Label)
			if !ok {
				return
			}

			dateContainer, ok := columns.Objects[1].(*fyne.Container)
			if !ok || len(dateContainer.Objects) != 1 {
				return
			}
			dateLabel, ok := dateContainer.Objects[0].(*widget.Label)
			if !ok {
				return
			}

			projectContainer, ok := columns.Objects[2].(*fyne.Container)
			if !ok || len(projectContainer.Objects) != 1 {
				return
			}
			projectLabel, ok := projectContainer.Objects[0].(*widget.Label)
			if !ok {
				return
			}

			statusContainer, ok := columns.Objects[3].(*fyne.Container)
			if !ok || len(statusContainer.Objects) != 1 {
				return
			}
			statusLabel, ok := statusContainer.Objects[0].(*widget.Label)
			if !ok {
				return
			}

			titleLabel.SetText(meeting.Title)
			dateLabel.SetText(meeting.Date.Format("2006-01-02"))
			projectLabel.SetText(meeting.Project)
			statusLabel.SetText(string(meeting.Status))
		},
	)

	screen.list.OnSelected = func(id widget.ListItemID) {
		if id < 0 || id >= len(screen.meetings) {
			screen.list.UnselectAll()
			return
		}

		if screen.onSelect != nil {
			screen.onSelect(screen.meetings[id])
		}

		screen.list.UnselectAll()
	}

	refreshButton := widget.NewButton("Refresh", func() {
		screen.refresh()
	})

	newMeetingButton := widget.NewButton("+ New Meeting", func() {
		if onNew != nil {
			onNew()
		}
	})

	toolbar := container.NewHBox(screen.filter, refreshButton, layout.NewSpacer(), newMeetingButton)
	content := container.NewBorder(toolbar, nil, nil, nil, screen.list)

	screen.refresh()

	return content
}

func (m *meetingsScreen) refresh() {
	projectFilter := ""
	selectedProject := m.filter.Selected
	if selectedProject != "" && selectedProject != "All" {
		projectFilter = selectedProject
	}

	if m.store == nil {
		fmt.Fprintf(os.Stderr, "refresh meetings: store is nil\n")
		m.meetings = []storage.Meeting{}
		m.list.Refresh()
		return
	}

	projects, projectsErr := m.store.ListProjects()
	if projectsErr != nil {
		fmt.Fprintf(os.Stderr, "refresh projects: %v\n", projectsErr)
	} else {
		options := make([]string, 0, len(projects)+1)
		options = append(options, "All")
		for _, project := range projects {
			options = append(options, project.Prefix)
		}

		m.filter.Options = options

		selectionExists := false
		for _, option := range options {
			if option == selectedProject {
				selectionExists = true
				break
			}
		}

		if selectedProject == "" || !selectionExists {
			m.filter.SetSelected("All")
		} else {
			m.filter.Refresh()
		}
	}

	meetings, meetingsErr := m.store.ListMeetings(projectFilter)
	if meetingsErr != nil {
		fmt.Fprintf(os.Stderr, "refresh meetings: %v\n", meetingsErr)
		m.meetings = []storage.Meeting{}
		m.list.Refresh()
		return
	}

	m.meetings = meetings
	m.list.Refresh()
}
