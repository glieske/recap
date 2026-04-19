package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/ai"
	"github.com/glieske/recap/internal/config"
	"github.com/glieske/recap/internal/storage"
)

type providerFactoryFunc func(*config.Config) (ai.Provider, error)

type Screen int

const (
	ScreenWelcome Screen = iota
	ScreenMeetingList
	ScreenNewMeeting
	ScreenEditor
	ScreenEmail
	ScreenHelp
	ScreenProviderSelector
)

type NavigateMsg struct {
	Screen Screen
}

type AppModel struct {
	screen          Screen
	previousScreen  Screen
	width           int
	height          int
	store           *storage.Store
	provider        ai.Provider
	providerFactory providerFactoryFunc
	cfg             *config.Config
	configPath      string
	autoNewMeeting  bool
	version         string
	err             error

	listModel       ListModel
	welcomeModel    WelcomeModel
	newMeetingModel NewMeetingModel
	editorModel     EditorModel
	emailModel      EmailModel
	helpModel       HelpModel
	showHelp        bool
	helpModal       ModalModel
	showNewMeeting  bool
	newMeetingModal ModalModel
	showProvider    bool
	showSettings    bool
	providerModal   ModalModel
	providerModel   ProviderModel
	settingsModel   SettingsModel
	settingsModal   ModalModel

	hasNewMeetingModel bool
	hasEditorModel     bool
	hasEmailModel      bool
	hasProviderModel   bool
	hasSettingsModel   bool

	showConfirm          bool
	confirmModal         ModalModel
	confirmModel         ConfirmModel
	hasConfirmModel      bool
	pendingDeleteMeeting *storage.Meeting

	currentMeeting *storage.Meeting
	structuredMD   string
	aiRunning      bool
	statusMsg      string
}

func NewAppModel(cfg *config.Config, store *storage.Store, provider ai.Provider, configPath string, autoNewMeeting bool, version string) AppModel {
	return AppModel{
		screen:          ScreenWelcome,
		previousScreen:  ScreenWelcome,
		width:           80,
		height:          24,
		store:           store,
		provider:        provider,
		providerFactory: ai.NewProvider,
		cfg:             cfg,
		configPath:      configPath,
		autoNewMeeting:  autoNewMeeting,
		version:         version,
		listModel:       NewListModel(store, 80, 24),
		welcomeModel:    NewWelcomeModel(80, 24, version),
		helpModel:       NewHelpModel(),
	}
}

func (m AppModel) Init() tea.Cmd {
	if m.autoNewMeeting {
		return tea.Batch(m.listModel.Init(), func() tea.Msg {
			return NavigateMsg{Screen: ScreenNewMeeting}
		})
	}

	return m.listModel.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typedMsg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typedMsg.Width
		m.height = typedMsg.Height
		m.welcomeModel.width = typedMsg.Width
		m.welcomeModel.height = typedMsg.Height

		cmds := make([]tea.Cmd, 0, 6)

		if updatedModel, cmd := m.listModel.Update(typedMsg); true {
			if updatedList, ok := updatedModel.(ListModel); ok {
				m.listModel = updatedList
			}
			cmds = append(cmds, cmd)
		}

		if m.hasNewMeetingModel && !m.showNewMeeting {
			updatedModel, cmd := m.newMeetingModel.Update(typedMsg)
			if updatedNewMeeting, ok := updatedModel.(NewMeetingModel); ok {
				m.newMeetingModel = updatedNewMeeting
			}
			cmds = append(cmds, cmd)
		}

		if m.hasEditorModel {
			updatedModel, cmd := m.editorModel.Update(typedMsg)
			if updatedEditor, ok := updatedModel.(EditorModel); ok {
				m.editorModel = updatedEditor
			}
			cmds = append(cmds, cmd)
		}

		if m.hasEmailModel {
			updatedModel, cmd := m.emailModel.Update(typedMsg)
			if updatedEmail, ok := updatedModel.(EmailModel); ok {
				m.emailModel = updatedEmail
			}
			cmds = append(cmds, cmd)
		}

		if updatedModel, cmd := m.helpModel.Update(typedMsg); true {
			if updatedHelp, ok := updatedModel.(HelpModel); ok {
				m.helpModel = updatedHelp
			}
			cmds = append(cmds, cmd)
		}

		if m.showHelp {
			updatedModel, cmd := m.helpModal.Update(typedMsg)
			if updatedModal, ok := updatedModel.(ModalModel); ok {
				m.helpModal = updatedModal
			}
			cmds = append(cmds, cmd)
		}

		if m.showNewMeeting {
			updatedModel, cmd := m.newMeetingModal.Update(typedMsg)
			if updatedModal, ok := updatedModel.(ModalModel); ok {
				m.newMeetingModal = updatedModal
			}
			cmds = append(cmds, cmd)
		}

		if m.showProvider {
			updatedModel, cmd := m.providerModal.Update(typedMsg)
			if updatedModal, ok := updatedModel.(ModalModel); ok {
				m.providerModal = updatedModal
			}
			cmds = append(cmds, cmd)
		}

		if m.showSettings {
			updatedModel, cmd := m.settingsModal.Update(typedMsg)
			if updatedModal, ok := updatedModel.(ModalModel); ok {
				m.settingsModal = updatedModal
			}
			cmds = append(cmds, cmd)
		}

		return m, tea.Batch(cmds...)
	case NavigateMsg:
		if typedMsg.Screen == ScreenNewMeeting {
			m.newMeetingModel = NewNewMeetingModel(m.store, m.width, m.height)
			m.hasNewMeetingModel = true
			m.showNewMeeting = true
			m.newMeetingModal = NewModalModel("New Meeting", &m.newMeetingModel, m.width, m.height)
			return m, m.newMeetingModal.Init()
		}

		m.screen = typedMsg.Screen
		switch typedMsg.Screen {
		case ScreenMeetingList:
			m.showNewMeeting = false
			return m, m.listModel.RefreshMeetings()
		default:
			return m, nil
		}
	case MeetingSelectedMsg:
		meeting := typedMsg.Meeting
		m.currentMeeting = &meeting
		provName, provModel := m.providerDisplayInfo()
		m.editorModel = NewEditorModel(m.currentMeeting, m.store, m.width, m.height, provName, provModel, m.version)
		m.hasEditorModel = true

		m.structuredMD = ""
		if m.store != nil {
			structuredMD, loadErr := m.store.LoadStructuredNotes(meeting.Project, meeting.ID)
			if loadErr == nil {
				m.structuredMD = structuredMD
			}
		}

		sm := NewSummaryModel(m.structuredMD, meeting.Project, meeting.ID, m.store, m.width/2, maxEditorHeight(m.height))
		m.editorModel.SetSummaryModel(sm)
		if m.structuredMD != "" {
			m.editorModel.ActivateSplitMode()
		}

		m.screen = ScreenEditor
		return m, m.editorModel.Init()
	case MeetingCreatedMsg:
		m.showNewMeeting = false
		if typedMsg.Meeting == nil {
			return m, nil
		}

		meeting := *typedMsg.Meeting
		m.currentMeeting = &meeting
		provName, provModel := m.providerDisplayInfo()
		m.editorModel = NewEditorModel(m.currentMeeting, m.store, m.width, m.height, provName, provModel, m.version)
		m.hasEditorModel = true

		m.structuredMD = ""
		if m.store != nil {
			structuredMD, loadErr := m.store.LoadStructuredNotes(meeting.Project, meeting.ID)
			if loadErr == nil {
				m.structuredMD = structuredMD
			}
		}

		sm := NewSummaryModel(m.structuredMD, meeting.Project, meeting.ID, m.store, m.width/2, maxEditorHeight(m.height))
		m.editorModel.SetSummaryModel(sm)
		if m.structuredMD != "" {
			m.editorModel.ActivateSplitMode()
		}

		m.screen = ScreenEditor
		return m, m.editorModel.Init()
	case RequestDeleteMsg:
		m.pendingDeleteMeeting = &typedMsg.Meeting
		m.confirmModel = NewConfirmModel(
			"Delete meeting ["+typedMsg.Meeting.Title+"]?",
			"delete-meeting",
		)
		m.hasConfirmModel = true
		m.showConfirm = true
		m.confirmModal = NewModalModel("Confirm", &m.confirmModel, m.width, m.height)
		return m, m.confirmModal.Init()
	case TriggerAIMsg:
		if m.currentMeeting == nil || m.provider == nil || !m.hasEditorModel {
			return m, nil
		}

		if m.structuredMD != "" {
			m.confirmModel = NewConfirmModel(
				"Re-generate AI summary? Current summary will be overwritten.",
				"regenerate-ai",
			)
			m.hasConfirmModel = true
			m.showConfirm = true
			m.confirmModal = NewModalModel("Confirm", &m.confirmModel, m.width, m.height)
			return m, m.confirmModal.Init()
		}

		rawNotes := m.editorModel.textarea.Value()
		meta := ai.MeetingMeta{
			Title:        m.currentMeeting.Title,
			Date:         m.currentMeeting.Date.Format("2006-01-02"),
			Participants: m.currentMeeting.Participants,
			TicketID:     m.currentMeeting.TicketID,
		}

		m.aiRunning = true
		m.statusMsg = "Structuring notes..."
		return m, StructureNotesCmd(m.provider, rawNotes, meta)
	case TriggerEmailMsg:
		if m.structuredMD == "" || m.provider == nil {
			return m, nil
		}

		m.aiRunning = true
		m.statusMsg = "Generating email..."
		return m, GenerateEmailCmd(m.provider, m.structuredMD, m.emailLanguage())
	case AIStructureDoneMsg:
		m.aiRunning = false
		m.structuredMD = typedMsg.StructuredMD
		m.statusMsg = "Structured notes updated"

		if m.currentMeeting != nil && m.store != nil {
			saveErr := m.store.SaveStructuredNotes(m.currentMeeting.Project, m.currentMeeting.ID, typedMsg.StructuredMD)
			if saveErr != nil {
				m.err = saveErr
			}
		}

		if m.hasEditorModel {
			summaryModel := m.editorModel.GetSummaryModel()
			summaryModel.SetContent(typedMsg.StructuredMD)
			m.editorModel.SetSummaryModel(summaryModel)
			m.editorModel.ActivateSplitMode()
		}

		return m, nil
	case AIStructureErrMsg:
		m.aiRunning = false
		m.err = typedMsg.Err
		m.statusMsg = ""
		return m, nil
	case AIEmailDoneMsg:
		m.aiRunning = false
		m.statusMsg = "Email generated"
		m.emailModel = NewEmailModel(typedMsg.Subject, typedMsg.Body, m.width, m.height, m.emailLanguage())
		m.hasEmailModel = true
		m.screen = ScreenEmail
		return m, nil
	case AIEmailErrMsg:
		m.aiRunning = false
		m.err = typedMsg.Err
		m.statusMsg = ""
		return m, nil
	case TogglePreviewMsg:
		return m, nil
	case LanguageChangedMsg:
		if m.cfg != nil {
			m.cfg.EmailLanguage = typedMsg.Language
		}
		if m.screen == ScreenEmail && m.structuredMD != "" && m.provider != nil {
			m.aiRunning = true
			m.statusMsg = "Regenerating email..."
			return m, GenerateEmailCmd(m.provider, m.structuredMD, typedMsg.Language)
		}
		return m, nil
	case RegenerateEmailMsg:
		if m.structuredMD == "" || m.provider == nil {
			return m, nil
		}

		m.aiRunning = true
		m.statusMsg = "Generating email..."
		return m, GenerateEmailCmd(m.provider, m.structuredMD, m.emailLanguage())
	case ProviderSelectedMsg:
		oldProvider := m.cfg.AIProvider
		m.cfg.AIProvider = typedMsg.ProviderName
		newProvider, provErr := m.providerFactory(m.cfg)
		if provErr != nil {
			m.cfg.AIProvider = oldProvider
			m.err = provErr
			m.statusMsg = "Failed to switch provider: " + provErr.Error()
		} else {
			m.provider = newProvider
			m.statusMsg = "Switched to " + typedMsg.ProviderName
			m.err = nil
		}
		if m.hasEditorModel {
			provName, provModel := m.providerDisplayInfo()
			m.editorModel.providerName = provName
			m.editorModel.providerModel = provModel
		}
		m.screen = ScreenEditor
		m.showProvider = false
		return m, nil
	case SettingsUpdatedMsg:
		m.showSettings = false
		if typedMsg.Config != nil {
			newProvider, provErr := m.providerFactory(typedMsg.Config)
			if provErr != nil {
				m.err = provErr
				m.statusMsg = "Settings saved (provider error: " + provErr.Error() + ")"
			} else {
				m.provider = newProvider
				m.statusMsg = "Settings saved"
				m.err = nil
			}
			if m.hasEditorModel {
				provName, provModel := m.providerDisplayInfo()
				m.editorModel.providerName = provName
				m.editorModel.providerModel = provModel
			}
		}
		return m, nil
	case SaveDoneMsg:
		if !m.hasEditorModel {
			return m, nil
		}

		updatedModel, cmd := m.editorModel.Update(typedMsg)
		if updatedEditor, ok := updatedModel.(EditorModel); ok {
			m.editorModel = updatedEditor
		}

		return m, cmd
	case SaveErrMsg:
		if !m.hasEditorModel {
			return m, nil
		}

		updatedModel, cmd := m.editorModel.Update(typedMsg)
		if updatedEditor, ok := updatedModel.(EditorModel); ok {
			m.editorModel = updatedEditor
		}

		return m, cmd
	case ConfirmResultMsg:
		m.showConfirm = false
		if !typedMsg.Confirmed {
			return m, nil
		}
		switch typedMsg.Action {
		case "delete-meeting":
			if m.pendingDeleteMeeting != nil && m.store != nil {
				meeting := m.pendingDeleteMeeting
				m.pendingDeleteMeeting = nil
				if err := m.store.DeleteMeeting(meeting.Project, meeting.ID); err != nil {
					return m, func() tea.Msg { return DeleteErrMsg{Err: err} }
				}
				return m, m.listModel.RefreshMeetings()
			}
			m.pendingDeleteMeeting = nil
			return m, nil
		case "regenerate-ai":
			rawNotes := m.editorModel.textarea.Value()
			meta := ai.MeetingMeta{
				Title:        m.currentMeeting.Title,
				Date:         m.currentMeeting.Date.Format("2006-01-02"),
				Participants: m.currentMeeting.Participants,
				TicketID:     m.currentMeeting.TicketID,
			}
			m.aiRunning = true
			m.statusMsg = "Structuring notes..."
			return m, StructureNotesCmd(m.provider, rawNotes, meta)
		case "switch-provider":
			currentProv := ""
			if m.cfg != nil {
				currentProv = m.cfg.AIProvider
			}
			m.providerModel = NewProviderModel(currentProv, m.width, m.height)
			m.hasProviderModel = true
			m.showProvider = true
			m.providerModal = NewModalModel("AI Provider", &m.providerModel, m.width, m.height)
			return m, m.providerModal.Init()
		}
		return m, nil
	case DismissModalMsg:
		if m.showConfirm {
			m.showConfirm = false
		} else if m.showProvider {
			m.showProvider = false
		} else if m.showSettings {
			m.showSettings = false
		} else if m.showNewMeeting {
			m.showNewMeeting = false
		} else if m.showHelp {
			m.showHelp = false
		}
		return m, nil
	case WelcomeSelectMsg:
		switch typedMsg.Choice {
		case "notes":
			m.screen = ScreenMeetingList
			return m, m.listModel.RefreshMeetings()
		case "new":
			nm := NewNewMeetingModel(m.store, m.width, m.height)
			m.newMeetingModel = nm
			m.showNewMeeting = true
			m.newMeetingModal = NewModalModel("New Meeting", &m.newMeetingModel, m.width, m.height)
			return m, m.newMeetingModal.Init()
		case "settings":
			sm := NewSettingsModel(m.cfg, m.configPath, m.width, m.height)
			m.settingsModel = sm
			m.hasSettingsModel = true
			m.showSettings = true
			m.settingsModal = NewModalModel("Settings", &m.settingsModel, m.width, m.height)
			return m, m.settingsModal.Init()
		case "quit":
			return m, tea.Quit
		}
		return m, nil
	case tea.KeyPressMsg:
		if typedMsg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		if m.showConfirm {
			updatedModel, cmd := m.confirmModal.Update(msg)
			if updatedModal, ok := updatedModel.(ModalModel); ok {
				m.confirmModal = updatedModal
			}
			return m, cmd
		}

		if m.showProvider {
			updatedModel, cmd := m.providerModal.Update(msg)
			if updatedModal, ok := updatedModel.(ModalModel); ok {
				m.providerModal = updatedModal
			}
			return m, cmd
		}

		if m.showSettings {
			updatedModel, cmd := m.settingsModal.Update(msg)
			if updatedModal, ok := updatedModel.(ModalModel); ok {
				m.settingsModal = updatedModal
			}
			return m, cmd
		}

		if m.showNewMeeting {
			updatedModel, cmd := m.newMeetingModal.Update(msg)
			if updatedModal, ok := updatedModel.(ModalModel); ok {
				m.newMeetingModal = updatedModal
			}
			return m, cmd
		}

		if m.showHelp {
			if typedMsg.String() == "?" {
				m.showHelp = false
				return m, nil
			}

			updatedModel, cmd := m.helpModal.Update(msg)
			if updatedModal, ok := updatedModel.(ModalModel); ok {
				m.helpModal = updatedModal
			}
			return m, cmd
		}

		switch typedMsg.String() {
		case "q":
			if m.screen == ScreenMeetingList || m.screen == ScreenWelcome {
				return m, tea.Quit
			}

			return m.delegateActiveModel(typedMsg)
		case "?", "ctrl+/":
			if typedMsg.String() == "?" && m.screen == ScreenEditor {
				return m.delegateActiveModel(typedMsg)
			}
			m.showHelp = !m.showHelp
			if m.showHelp {
				m.helpModel.screen = m.screen
				m.helpModal = NewModalModel("Help", m.helpModel, m.width, m.height)
			}
			return m, nil
		case "esc":
			switch m.screen {
			case ScreenEditor:
				m.screen = ScreenMeetingList
				if m.hasEditorModel {
					return m, tea.Sequence(m.editorModel.saveCmd(), m.listModel.RefreshMeetings())
				}

				return m, m.listModel.RefreshMeetings()
			case ScreenEmail:
				m.screen = ScreenMeetingList
				return m, m.listModel.RefreshMeetings()
			case ScreenMeetingList:
				m.screen = ScreenWelcome
				return m, nil
			case ScreenWelcome:
				return m, nil
			}

			return m, nil
		case "ctrl+,":
			sm := NewSettingsModel(m.cfg, m.configPath, m.width, m.height)
			m.settingsModel = sm
			m.hasSettingsModel = true
			m.showSettings = true
			m.settingsModal = NewModalModel("Settings", &m.settingsModel, m.width, m.height)
			return m, m.settingsModal.Init()
		case "ctrl+o":
			if m.screen == ScreenEditor && m.aiRunning {
				m.confirmModel = NewConfirmModel(
					"AI is currently running. Switch provider anyway?",
					"switch-provider",
				)
				m.hasConfirmModel = true
				m.showConfirm = true
				m.confirmModal = NewModalModel("Confirm", &m.confirmModel, m.width, m.height)
				return m, m.confirmModal.Init()
			}
			currentProv := ""
			if m.cfg != nil {
				currentProv = m.cfg.AIProvider
			}
			m.providerModel = NewProviderModel(currentProv, m.width, m.height)
			m.hasProviderModel = true
			m.showProvider = true
			m.providerModal = NewModalModel("AI Provider", &m.providerModel, m.width, m.height)
			return m, m.providerModal.Init()
		default:
			return m.delegateActiveModel(typedMsg)
		}
	}

	return m.delegateActiveModel(msg)
}

func safeViewContent(fn func() tea.View) (content string) {
	defer func() {
		if recover() != nil {
			content = ""
		}
	}()

	return fn().Content
}

func (m AppModel) View() tea.View {
	view := "Unknown screen"
	switch m.screen {
	case ScreenWelcome:
		view = safeViewContent(func() tea.View { return m.welcomeModel.View() })
	case ScreenMeetingList:
		view = safeViewContent(func() tea.View { return m.listModel.View() })
	case ScreenEditor:
		view = safeViewContent(func() tea.View { return m.editorModel.View() })
	case ScreenEmail:
		view = safeViewContent(func() tea.View { return m.emailModel.View() })
	}

	statusLines := make([]string, 0, 3)
	if m.statusMsg != "" {
		statusLines = append(statusLines, m.statusMsg)
	}

	if m.aiRunning {
		statusLines = append(statusLines, "⏳ AI processing...")
	}

	if m.err != nil {
		statusLines = append(statusLines, "Error: "+m.err.Error())
	}

	if len(statusLines) > 0 {
		view += "\n" + strings.Join(statusLines, "\n")
	}

	if m.showNewMeeting {
		modalView := safeViewContent(func() tea.View { return m.newMeetingModal.View() })
		view = RenderOverlay(view, modalView, m.width, m.height)
	}

	if m.showHelp {
		modalView := safeViewContent(func() tea.View { return m.helpModal.View() })
		view = RenderOverlay(view, modalView, m.width, m.height)
	}

	if m.showSettings {
		modalView := safeViewContent(func() tea.View { return m.settingsModal.View() })
		view = RenderOverlay(view, modalView, m.width, m.height)
	}

	if m.showProvider {
		modalView := safeViewContent(func() tea.View { return m.providerModal.View() })
		view = RenderOverlay(view, modalView, m.width, m.height)
	}

	if m.showConfirm {
		modalView := safeViewContent(func() tea.View { return m.confirmModal.View() })
		view = RenderOverlay(view, modalView, m.width, m.height)
	}

	v := tea.NewView(view)
	v.AltScreen = true
	return v
}

func (m AppModel) providerDisplayInfo() (name string, model string) {
	if m.cfg == nil {
		return "", ""
	}

	switch m.cfg.AIProvider {
	case "github_models":
		return "GitHub Models", m.cfg.GitHubModel
	case "openrouter":
		return "OpenRouter", m.cfg.OpenRouterModel
	case "lm_studio":
		return "LM Studio", m.cfg.LMStudioModel
	default:
		return m.cfg.AIProvider, ""
	}
}

func (m AppModel) emailLanguage() string {
	if m.cfg != nil && m.cfg.EmailLanguage != "" {
		return m.cfg.EmailLanguage
	}

	return "pl"
}

func (m AppModel) delegateActiveModel(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.showConfirm {
		updated, cmd := m.confirmModal.Update(msg)
		if modal, ok := updated.(ModalModel); ok {
			m.confirmModal = modal
		}
		return m, cmd
	}

	if m.showProvider {
		updated, cmd := m.providerModal.Update(msg)
		if modal, ok := updated.(ModalModel); ok {
			m.providerModal = modal
		}
		return m, cmd
	}

	if m.showSettings {
		updated, cmd := m.settingsModal.Update(msg)
		if modal, ok := updated.(ModalModel); ok {
			m.settingsModal = modal
		}
		return m, cmd
	}

	if m.showNewMeeting {
		updated, cmd := m.newMeetingModal.Update(msg)
		if modal, ok := updated.(ModalModel); ok {
			m.newMeetingModal = modal
		}
		return m, cmd
	}

	if m.showHelp {
		updated, cmd := m.helpModal.Update(msg)
		if modal, ok := updated.(ModalModel); ok {
			m.helpModal = modal
		}
		return m, cmd
	}

	switch m.screen {
	case ScreenWelcome:
		updatedModel, cmd := m.welcomeModel.Update(msg)
		if updatedWelcome, ok := updatedModel.(WelcomeModel); ok {
			m.welcomeModel = updatedWelcome
		}
		return m, cmd
	case ScreenMeetingList:
		updatedModel, cmd := m.listModel.Update(msg)
		if updatedList, ok := updatedModel.(ListModel); ok {
			m.listModel = updatedList
		}
		return m, cmd
	case ScreenEditor:
		if !m.hasEditorModel {
			return m, nil
		}

		updatedModel, cmd := m.editorModel.Update(msg)
		if updatedEditor, ok := updatedModel.(EditorModel); ok {
			m.editorModel = updatedEditor
		}
		return m, cmd
	case ScreenEmail:
		if !m.hasEmailModel {
			return m, nil
		}

		updatedModel, cmd := m.emailModel.Update(msg)
		if updatedEmail, ok := updatedModel.(EmailModel); ok {
			m.emailModel = updatedEmail
		}
		return m, cmd
	default:
		return m, nil
	}
}
