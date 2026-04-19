//go:build gui

package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/glieske/recap/internal/ai"
	"github.com/glieske/recap/internal/config"
	"github.com/glieske/recap/internal/storage"
)

func Run(cfg *config.Config, store *storage.Store, provider ai.Provider, configPath string, version string) {
	a := app.New()
	w := a.NewWindow(fmt.Sprintf("Recap v%s", version))
	w.Resize(fyne.NewSize(900, 700))

	meetingsContainer := container.NewStack()

	var showList func()
	var showEditor func(storage.Meeting)
	var setStatus func(string)

	showEditor = func(meeting storage.Meeting) {
		editor := NewEditorScreen(meeting, store, provider, w, cfg, func() {
			showList()
		}, setStatus)

		meetingsContainer.Objects = []fyne.CanvasObject{editor.Content}
		meetingsContainer.Refresh()
	}

	showList = func() {
		listView := NewMeetingsScreen(
			store,
			func(meeting storage.Meeting) {
				showEditor(meeting)
			},
			func() {
				ShowNewMeetingDialog(w, store, func(meeting storage.Meeting) {
					showEditor(meeting)
				})
			},
		)

		meetingsContainer.Objects = []fyne.CanvasObject{listView}
		meetingsContainer.Refresh()
	}

	statusBar := widget.NewLabel("Ready")
	setStatus = func(msg string) {
		statusBar.SetText(msg)
	}

	showList()

	tabs := container.NewAppTabs(
		container.NewTabItem("Meetings", meetingsContainer),
		container.NewTabItem("Settings", NewSettingsScreen(cfg, configPath, w)),
	)

	w.SetContent(container.NewBorder(nil, statusBar, nil, nil, tabs))
	w.ShowAndRun()
}
