//go:build gui

package gui

import (
	"errors"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/glieske/recap/internal/storage"
)

func ShowNewMeetingDialog(win fyne.Window, store *storage.Store, onCreate func(storage.Meeting)) {
	if win == nil {
		return
	}

	titleEntry := widget.NewEntry()
	titleEntry.SetPlaceHolder("Meeting title")

	dateEntry := widget.NewEntry()
	dateEntry.SetPlaceHolder("YYYY-MM-DD")
	dateEntry.SetText(time.Now().Format("2006-01-02"))

	participantsEntry := widget.NewEntry()
	participantsEntry.SetPlaceHolder("Alice, Bob, Charlie")

	projectOptions := []string{}
	if store != nil {
		projects, err := store.ListProjects()
		if err == nil {
			projectOptions = make([]string, 0, len(projects))
			for _, project := range projects {
				projectOptions = append(projectOptions, project.Prefix)
			}
		}
	}

	projectSelect := widget.NewSelect(projectOptions, nil)

	tagsEntry := widget.NewEntry()
	tagsEntry.SetPlaceHolder("tag1, tag2")

	externalTicketEntry := widget.NewEntry()
	externalTicketEntry.SetPlaceHolder("JIRA-123")

	form := widget.NewForm(
		widget.NewFormItem("Title", titleEntry),
		widget.NewFormItem("Date", dateEntry),
		widget.NewFormItem("Participants", participantsEntry),
		widget.NewFormItem("Project", projectSelect),
		widget.NewFormItem("Tags", tagsEntry),
		widget.NewFormItem("External Ticket", externalTicketEntry),
	)

	dialog.ShowCustomConfirm("New Meeting", "Create", "Cancel", form, func(confirmed bool) {
		if !confirmed {
			return
		}

		title := strings.TrimSpace(titleEntry.Text)
		if title == "" {
			dialog.ShowError(errors.New("title is required"), win)
			return
		}

		date, err := time.Parse("2006-01-02", strings.TrimSpace(dateEntry.Text))
		if err != nil {
			dialog.ShowError(errors.New("invalid date format, use YYYY-MM-DD"), win)
			return
		}

		project := strings.TrimSpace(projectSelect.Selected)
		if project == "" {
			dialog.ShowError(errors.New("project is required"), win)
			return
		}

		participants := splitAndTrim(participantsEntry.Text)
		tags := splitAndTrim(tagsEntry.Text)
		externalTicket := strings.TrimSpace(externalTicketEntry.Text)

		if store == nil {
			dialog.ShowError(errors.New("store is required"), win)
			return
		}

		meeting, createErr := store.CreateMeeting(title, date, participants, project, tags, externalTicket)
		if createErr != nil {
			dialog.ShowError(createErr, win)
			return
		}

		if onCreate != nil && meeting != nil {
			onCreate(*meeting)
		}
	}, win)
}

func splitAndTrim(input string) []string {
	if strings.TrimSpace(input) == "" {
		return nil
	}

	parts := strings.Split(input, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
