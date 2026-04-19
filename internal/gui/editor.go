//go:build gui

package gui

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/glieske/recap/internal/ai"
	"github.com/glieske/recap/internal/config"
	"github.com/glieske/recap/internal/languages"
	"github.com/glieske/recap/internal/storage"
)

type EditorScreen struct {
	Meeting           storage.Meeting
	Store             *storage.Store
	Provider          ai.Provider
	Window            fyne.Window
	Config            *config.Config
	OnStatus          func(string)
	DeleteButton      *widget.Button
	ProviderInfoLabel *widget.Label
	RawEntry          *widget.Entry
	Preview           *widget.RichText
	Content           fyne.CanvasObject
	Cancel            context.CancelFunc
}

func NewEditorScreen(meeting storage.Meeting, store *storage.Store, provider ai.Provider, win fyne.Window, cfg *config.Config, onBack func(), onStatus func(string)) *EditorScreen {
	ctx, cancel := context.WithCancel(context.Background())
	var structuredMD string
	reportStatus := func(msg string) {
		if onStatus != nil {
			onStatus(msg)
		}
	}

	var es *EditorScreen
	backButton := widget.NewButton("← Back", func() {
		if es != nil && es.Cancel != nil {
			es.Cancel()
		}
		onBack()
	})
	deleteButton := widget.NewButton("🗑 Delete", func() {
		if store == nil || win == nil {
			return
		}
		dialog.ShowConfirm(
			"Delete Meeting?",
			fmt.Sprintf("Are you sure you want to delete '%s'? This cannot be undone.", meeting.Title),
			func(confirmed bool) {
				if !confirmed {
					return
				}
				if err := store.DeleteMeeting(meeting.Project, meeting.ID); err != nil {
					dialog.ShowError(fmt.Errorf("delete failed: %w", err), win)
					reportStatus("Delete failed")
					return
				}
				reportStatus("Meeting deleted")
				onBack()
			},
			win,
		)
	})

	titleLabel := widget.NewLabelWithStyle(meeting.Title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	dateLabel := widget.NewLabel(meeting.Date.Format("2006-01-02"))
	projectLabel := widget.NewLabel(meeting.Project)
	statusLabel := widget.NewLabel(string(meeting.Status))
	providerText := "No AI provider"
	if cfg != nil && cfg.AIProvider != "" {
		switch cfg.AIProvider {
		case "github_models":
			providerText = fmt.Sprintf("github_models: %s", cfg.GitHubModel)
		case "openrouter":
			providerText = fmt.Sprintf("openrouter: %s", cfg.OpenRouterModel)
		case "lm_studio":
			providerText = fmt.Sprintf("lm_studio: %s", cfg.LMStudioModel)
		default:
			providerText = cfg.AIProvider
		}
	}
	providerInfoLabel := widget.NewLabel(providerText)

	rawEntry := widget.NewMultiLineEntry()
	rawEntry.SetPlaceHolder("Type raw meeting notes here...")
	rawEntry.Wrapping = fyne.TextWrapWord

	preview := widget.NewRichTextFromMarkdown("")
	preview.Wrapping = fyne.TextWrapWord

	structureButton := widget.NewButton("Structure Notes", nil)
	emailButton := widget.NewButton("Generate Email", nil)
	emailButton.Disable()
	doStructure := func() {
		rawText := rawEntry.Text
		meta := ai.MeetingMeta{
			Title:        meeting.Title,
			Date:         meeting.Date.Format("2006-01-02"),
			Participants: meeting.Participants,
			TicketID:     meeting.TicketID,
		}

		progress := widget.NewProgressBarInfinite()
		progressDialog := dialog.NewCustomWithoutButtons("Structuring notes...", progress, win)
		progressDialog.Show()
		structureButton.Disable()

		go func() {
			result, err := provider.StructureNotes(ctx, rawText, meta)
			fyne.Do(func() {
				progressDialog.Hide()
				structureButton.Enable()
				if err != nil {
					dialog.ShowError(fmt.Errorf("structuring failed: %w", err), win)
					reportStatus("Structuring failed")
					return
				}

				structuredMD = result
				preview.ParseMarkdown(result)
				reportStatus("Notes structured successfully")
				if provider != nil && strings.TrimSpace(result) != "" {
					emailButton.Enable()
				}
				if store != nil {
					if saveErr := store.SaveStructuredNotes(meeting.Project, meeting.ID, result); saveErr != nil {
						fmt.Fprintf(os.Stderr, "save structured notes: %v\n", saveErr)
					}
				}
			})
		}()
	}
	structureButton.OnTapped = func() {
		if strings.TrimSpace(rawEntry.Text) == "" {
			dialog.ShowInformation("No Notes", "Write some notes before structuring.", win)
			return
		}
		if strings.TrimSpace(structuredMD) != "" {
			dialog.ShowConfirm(
				"Re-structure Notes?",
				"Structured notes already exist. Re-structuring will overwrite them.",
				func(confirmed bool) {
					if !confirmed {
						return
					}
					doStructure()
				},
				win,
			)
			return
		}

		doStructure()
	}

	emailButton.OnTapped = func() {
		if strings.TrimSpace(structuredMD) == "" {
			dialog.ShowInformation("No Structured Notes", "Structure your notes first.", win)
			return
		}

		language := "en"
		if cfg != nil && len(cfg.EmailLanguages) > 0 {
			language = cfg.EmailLanguages[0]
		}

		progress := widget.NewProgressBarInfinite()
		progressDialog := dialog.NewCustomWithoutButtons("Generating email...", progress, win)
		progressDialog.Show()
		emailButton.Disable()

		go func() {
			result, err := provider.GenerateEmailSummary(ctx, structuredMD, language)
			fyne.Do(func() {
				progressDialog.Hide()
				emailButton.Enable()
				if err != nil {
					fmt.Fprintf(os.Stderr, "generate email summary: %v\n", err)
					dialog.ShowError(fmt.Errorf("email generation failed: %w", err), win)
					reportStatus("Email generation failed")
					return
				}
				reportStatus("Email generated")

				emailEntry := widget.NewMultiLineEntry()
				emailEntry.SetText(result)
				emailEntry.Wrapping = fyne.TextWrapWord

				var configuredLangs []string
				if cfg != nil {
					configuredLangs = cfg.EmailLanguages
				}
				if len(configuredLangs) == 0 {
					configuredLangs = []string{"en"}
				}
				langNames := make([]string, 0, len(configuredLangs))
				langCodes := make(map[string]string, len(configuredLangs))
				codeToName := make(map[string]string, len(configuredLangs))
				for _, code := range configuredLangs {
					name := languages.DisplayName(code)
					langNames = append(langNames, name)
					langCodes[name] = code
					codeToName[code] = name
				}

				currentResult := result
				langSelect := widget.NewSelect(langNames, nil)
				langSelect.SetSelected(codeToName[language])

				langSelect.OnChanged = func(selected string) {
					newCode := langCodes[selected]
					regenProgress := widget.NewProgressBarInfinite()
					regenDialog := dialog.NewCustomWithoutButtons("Regenerating email...", regenProgress, win)
					regenDialog.Show()
					langSelect.Disable()

					go func() {
						newResult, regenErr := provider.GenerateEmailSummary(ctx, structuredMD, newCode)
						fyne.Do(func() {
							regenDialog.Hide()
							langSelect.Enable()
							if regenErr != nil {
								fmt.Fprintf(os.Stderr, "regenerate email: %v\n", regenErr)
								dialog.ShowError(fmt.Errorf("email regeneration failed: %w", regenErr), win)
								return
							}

							currentResult = newResult
							emailEntry.SetText(newResult)
						})
					}()
				}

				emailDialog := dialog.NewCustom(
					"Email Summary",
					"Close",
					container.NewBorder(
						langSelect,
						widget.NewButton("Copy to Clipboard", func() {
							win.Clipboard().SetContent(currentResult)
						}),
						nil,
						nil,
						emailEntry,
					),
					win,
				)
				emailDialog.Resize(fyne.NewSize(600, 400))
				emailDialog.Show()
			})
		}()
	}

	if provider == nil {
		structureButton.Disable()
		emailButton.Disable()
	}

	toolbar := container.NewHBox(
		backButton,
		deleteButton,
		titleLabel,
		layout.NewSpacer(),
		providerInfoLabel,
		structureButton,
		emailButton,
		dateLabel,
		projectLabel,
		statusLabel,
	)

	if store != nil {
		rawContent, err := store.LoadRawNotes(meeting.Project, meeting.ID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load raw notes: %v\n", err)
		} else {
			rawEntry.SetText(rawContent)
		}
	}

	if store != nil {
		structuredContent, err := store.LoadStructuredNotes(meeting.Project, meeting.ID)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				preview.ParseMarkdown("*No structured notes yet.*")
			} else {
				fmt.Fprintf(os.Stderr, "load structured notes: %v\n", err)
			}
		} else {
			structuredMD = structuredContent
			preview.ParseMarkdown(structuredContent)
			if provider != nil && strings.TrimSpace(structuredContent) != "" {
				emailButton.Enable()
			}
		}
	}

	if store != nil {
		var saveTimer *time.Timer
		rawEntry.OnChanged = func(content string) {
			if saveTimer != nil {
				saveTimer.Stop()
			}

			saveTimer = time.AfterFunc(2*time.Second, func() {
				if saveErr := store.SaveRawNotes(meeting.Project, meeting.ID, content); saveErr != nil {
					fmt.Fprintf(os.Stderr, "auto-save: %v\n", saveErr)
					fyne.Do(func() { reportStatus("Auto-save failed") })
					return
				}
				fyne.Do(func() { reportStatus("Notes auto-saved") })
			})
		}
	}

	if win != nil {
		doManualSave := func(_ fyne.Shortcut) {
			if store == nil {
				return
			}
			if saveErr := store.SaveRawNotes(meeting.Project, meeting.ID, rawEntry.Text); saveErr != nil {
				fmt.Fprintf(os.Stderr, "manual save: %v\n", saveErr)
				reportStatus("Manual save failed")
				return
			}
			reportStatus("Notes saved")
		}
		win.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierControl}, doManualSave)
		win.Canvas().AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierSuper}, doManualSave)
	}

	editorMinSize := canvas.NewRectangle(color.Transparent)
	editorMinSize.SetMinSize(fyne.NewSize(360, 0))
	editorPane := container.NewStack(editorMinSize, rawEntry)

	split := container.NewHSplit(editorPane, preview)
	split.SetOffset(0.5)

	content := container.NewBorder(toolbar, nil, nil, nil, split)

	es = &EditorScreen{
		Meeting:           meeting,
		Store:             store,
		Provider:          provider,
		Window:            win,
		Config:            cfg,
		OnStatus:          onStatus,
		DeleteButton:      deleteButton,
		ProviderInfoLabel: providerInfoLabel,
		RawEntry:          rawEntry,
		Preview:           preview,
		Content:           content,
		Cancel:            cancel,
	}

	return es
}
