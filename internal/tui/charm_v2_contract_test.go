package tui

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/glieske/recap/internal/ai"
	"github.com/glieske/recap/internal/config"
	"github.com/glieske/recap/internal/storage"
)

func keyRune(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: string(r), Code: r})
}

func appFromModel(t *testing.T, model tea.Model) AppModel {
	t.Helper()
	app, ok := model.(AppModel)
	if !ok {
		t.Fatalf("expected AppModel from Update, got %T", model)
	}
	return app
}

func mustReadSource(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(rel)
	if err == nil {
		return string(b)
	}
	b, err = os.ReadFile(filepath.Join("internal", "tui", rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

func TestCharmV2_ViewMethodsReturnTeaView(t *testing.T) {
	want := reflect.TypeOf(tea.View{})

	views := []tea.View{
		NewAppModel(&config.Config{}, nil, nil, "", false).View(),
		NewListModel(nil, 80, 24).View(),
		NewNewMeetingModel(nil, 80, 24).View(),
		NewEditorModel(nil, nil, 80, 24, "", "").View(),
		NewEmailModel("subject", "body", 80, 24, "en").View(),
		NewHelpModel().View(),
		NewProviderModel("", 80, 24).View(),
		NewPreviewModel("body", "title", 80, 24).View(),
		NewSummaryModel("", "PRJ", "m1", nil, 80, 24).View(),
	}

	for i, got := range views {
		if gotType := reflect.TypeOf(got); gotType != want {
			t.Fatalf("view %d type mismatch: got %v want %v", i, gotType, want)
		}
	}
}

func TestCharmV2_AppViewEnablesAltScreen(t *testing.T) {
	m := NewAppModel(&config.Config{}, nil, nil, "", false)
	v := m.View()

	if v.AltScreen != true {
		t.Fatalf("AppModel.View AltScreen mismatch: got %v want true", v.AltScreen)
	}
}

func TestCharmV2_AppUpdateAcceptsKeyPressMsg(t *testing.T) {
	m := NewAppModel(&config.Config{}, nil, nil, "", false)

	updated, _ := m.Update(keyRune('?'))
	app := appFromModel(t, updated)
	if !app.showHelp {
		t.Fatalf("showHelp mismatch after '?': got false want true")
	}
	if app.screen != ScreenWelcome {
		t.Fatalf("screen mismatch after '?': got %v want %v", app.screen, ScreenWelcome)
	}

	m2 := NewAppModel(&config.Config{}, nil, nil, "", false)
	updated, cmd := m2.Update(keyRune('q'))
	_ = appFromModel(t, updated)
	if cmd == nil {
		t.Fatal("expected non-nil command for q on meeting list")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("q command message mismatch: got %T want tea.QuitMsg", cmd())
	}
}

func TestCharmV2_ViewportAndTextareaResizeViaSetters(t *testing.T) {
	email := NewEmailModel("subject", "body", 20, 10, "en")
	updatedEmailModel, _ := email.Update(tea.WindowSizeMsg{Width: 111, Height: 55})
	updatedEmail, ok := updatedEmailModel.(EmailModel)
	if !ok {
		t.Fatalf("expected EmailModel from Update, got %T", updatedEmailModel)
	}
	if got, want := updatedEmail.viewport.Width(), 111; got != want {
		t.Fatalf("email viewport width mismatch: got %d want %d", got, want)
	}
	if got, want := updatedEmail.viewport.Height(), emailViewportHeight(55); got != want {
		t.Fatalf("email viewport height mismatch: got %d want %d", got, want)
	}

	preview := NewPreviewModel("content", "title", 30, 12)
	updatedPreviewModel, _ := preview.Update(tea.WindowSizeMsg{Width: 77, Height: 19})
	updatedPreview, ok := updatedPreviewModel.(PreviewModel)
	if !ok {
		t.Fatalf("expected PreviewModel from Update, got %T", updatedPreviewModel)
	}
	if got, want := updatedPreview.viewport.Width(), 77; got != want {
		t.Fatalf("preview viewport width mismatch: got %d want %d", got, want)
	}
	if got, want := updatedPreview.viewport.Height(), previewViewportHeight(19); got != want {
		t.Fatalf("preview viewport height mismatch: got %d want %d", got, want)
	}

	editor := NewEditorModel(nil, nil, 80, 24, "", "")
	updatedEditorModel, _ := editor.Update(tea.WindowSizeMsg{Width: 96, Height: 31})
	updatedEditor, ok := updatedEditorModel.(EditorModel)
	if !ok {
		t.Fatalf("expected EditorModel from Update, got %T", updatedEditorModel)
	}
	if got, want := updatedEditor.width, 96; got != want {
		t.Fatalf("editor model width mismatch: got %d want %d", got, want)
	}
	if got := updatedEditor.textarea.Width(); got <= 0 {
		t.Fatalf("editor textarea width must stay positive, got %d", got)
	}
	if got, want := updatedEditor.textarea.Height(), maxEditorHeight(31); got != want {
		t.Fatalf("editor textarea height mismatch: got %d want %d", got, want)
	}

	summary := NewSummaryModel("", "PRJ", "m1", nil, 40, 10)
	updatedSummaryModel, _ := summary.Update(tea.WindowSizeMsg{Width: 120, Height: 44})
	updatedSummary, ok := updatedSummaryModel.(SummaryModel)
	if !ok {
		t.Fatalf("expected SummaryModel from Update, got %T", updatedSummaryModel)
	}
	if got, want := updatedSummary.width, 120; got != want {
		t.Fatalf("summary model width mismatch: got %d want %d", got, want)
	}
	if got := updatedSummary.textarea.Width(); got <= 0 {
		t.Fatalf("summary textarea width must stay positive, got %d", got)
	}
	if got, want := updatedSummary.textarea.Height(), summaryTextareaHeight(44); got != want {
		t.Fatalf("summary textarea height mismatch: got %d want %d", got, want)
	}
}

func TestCharmV2_SourceContracts_NoWithAltScreenAndDraculaThemeFunc(t *testing.T) {
	appSource := mustReadSource(t, "app.go")
	if strings.Contains(appSource, "tea.WithAltScreen") {
		t.Fatal("unexpected tea.WithAltScreen in app.go")
	}

	newMeetingSource := mustReadSource(t, "newmeeting.go")
	if strings.Contains(newMeetingSource, "huh.ThemeDracula()") {
		t.Fatal("unexpected huh.ThemeDracula() call in newmeeting.go")
	}
	if !strings.Contains(newMeetingSource, "huh.ThemeFunc(huh.ThemeDracula)") {
		t.Fatal("missing huh.ThemeFunc(huh.ThemeDracula) in newmeeting.go")
	}
}

func TestCharmV2_CommandsAndSimpleModelMethods(t *testing.T) {
	structureNilProvider := StructureNotesCmd(nil, "raw", ai.MeetingMeta{})()
	if got, ok := structureNilProvider.(AIStructureErrMsg); !ok || got.Err.Error() != "no AI provider configured" {
		t.Fatalf("StructureNotesCmd nil provider mismatch: got %#v", structureNilProvider)
	}

	structureEmptyRaw := StructureNotesCmd(struct{ ai.Provider }{}, "   ", ai.MeetingMeta{})()
	if got, ok := structureEmptyRaw.(AIStructureErrMsg); !ok || got.Err.Error() != "no raw notes to structure" {
		t.Fatalf("StructureNotesCmd empty raw notes mismatch: got %#v", structureEmptyRaw)
	}

	emailNilProvider := GenerateEmailCmd(nil, "md", "en")()
	if got, ok := emailNilProvider.(AIEmailErrMsg); !ok || got.Err.Error() != "no AI provider configured" {
		t.Fatalf("GenerateEmailCmd nil provider mismatch: got %#v", emailNilProvider)
	}

	emailEmptyMD := GenerateEmailCmd(struct{ ai.Provider }{}, "\n\t", "en")()
	if got, ok := emailEmptyMD.(AIEmailErrMsg); !ok || got.Err.Error() != "no structured notes available" {
		t.Fatalf("GenerateEmailCmd empty markdown mismatch: got %#v", emailEmptyMD)
	}

	if got := NewHelpModel().Init(); got != nil {
		t.Fatalf("HelpModel.Init mismatch: got %v want nil", got)
	}

	helpUpdated, helpCmd := NewHelpModel().Update(tea.WindowSizeMsg{Width: 9, Height: 7})
	if helpCmd != nil {
		t.Fatalf("HelpModel.Update cmd mismatch: got %v want nil", helpCmd)
	}
	if got, ok := helpUpdated.(HelpModel); !ok || got.width != 9 || got.height != 7 {
		t.Fatalf("HelpModel.Update state mismatch: got %#v", helpUpdated)
	}

	if got := NewListModel(nil, 80, 24).Init(); got != nil {
		t.Fatalf("ListModel.Init mismatch: got %v want nil", got)
	}

	list := NewListModel(nil, 80, 24)
	listUpdated, listCmd := list.Update(keyRune('n'))
	if listCmd == nil {
		t.Fatal("ListModel.Update expected command for 'n'")
	}
	if _, ok := listUpdated.(ListModel); !ok {
		t.Fatalf("ListModel.Update type mismatch: got %T", listUpdated)
	}
	if msg, ok := listCmd().(NavigateMsg); !ok || msg.Screen != ScreenNewMeeting {
		t.Fatalf("ListModel.Update cmd mismatch: got %#v", listCmd())
	}

	before := len(list.list.Items())
	refreshCmd := (&list).RefreshMeetings()
	if refreshCmd != nil {
		_ = refreshCmd()
	}
	after := len(list.list.Items())
	if gotDelta := after - before; gotDelta != 0 {
		t.Fatalf("ListModel.RefreshMeetings item delta mismatch: got %d want 0", gotDelta)
	}

	newMeeting := NewNewMeetingModel(nil, 80, 24)
	if cmd := newMeeting.Init(); cmd == nil {
		t.Fatal("NewMeetingModel.Init expected non-nil form init command")
	}
	newMeetingUpdated, newMeetingCmd := newMeeting.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if newMeetingCmd == nil {
		t.Fatal("NewMeetingModel.Update expected navigate command on esc")
	}
	if msg, ok := newMeetingCmd().(NavigateMsg); !ok || msg.Screen != ScreenMeetingList {
		t.Fatalf("NewMeetingModel.Update esc cmd mismatch: got %#v", newMeetingCmd())
	}
	if got, ok := newMeetingUpdated.(NewMeetingModel); !ok || !got.cancelled {
		t.Fatalf("NewMeetingModel.Update esc cancelled mismatch: got %#v", newMeetingUpdated)
	}

	provider := NewProviderModel("", 80, 24)
	if got := provider.Init(); got != nil {
		t.Fatalf("ProviderModel.Init mismatch: got %v want nil", got)
	}
	providerUpdated, providerCmd := provider.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if providerCmd == nil {
		t.Fatal("ProviderModel.Update expected command on enter")
	}
	if _, ok := providerUpdated.(ProviderModel); !ok {
		t.Fatalf("ProviderModel.Update type mismatch: got %T", providerUpdated)
	}
	if msg, ok := providerCmd().(ProviderSelectedMsg); !ok || msg.ProviderName == "" {
		t.Fatalf("ProviderModel.Update enter cmd mismatch: got %#v", providerCmd())
	}

	preview := NewPreviewModel("x", "t", 10, 5)
	if got := preview.Init(); got != nil {
		t.Fatalf("PreviewModel.Init mismatch: got %v want nil", got)
	}
	preview.SetContent("")
	if got := preview.content; got != previewPlaceholder {
		t.Fatalf("PreviewModel.SetContent placeholder mismatch: got %q want %q", got, previewPlaceholder)
	}
}

func TestCharmV2_SummaryMethodsAndErrorPaths(t *testing.T) {
	s := NewSummaryModel("base", "PRJ", "m1", nil, 20, 8)
	if got := s.Init(); got == nil {
		t.Fatal("SummaryModel.Init expected non-nil blink command")
	}
	if got := s.Value(); got != "base" {
		t.Fatalf("SummaryModel.Value mismatch: got %q want %q", got, "base")
	}
	if got := s.IsDirty(); got != false {
		t.Fatalf("SummaryModel.IsDirty mismatch: got %v want false", got)
	}
	if got := s.IsOverwritePending(); got != false {
		t.Fatalf("SummaryModel.IsOverwritePending mismatch: got %v want false", got)
	}

	s.SetContent("new")
	if got := s.Value(); got != "new" {
		t.Fatalf("SummaryModel.SetContent value mismatch: got %q want %q", got, "new")
	}

	s.Focus()

	typed, cmd := s.Update(tea.KeyPressMsg(tea.Key{Text: "x", Code: 'x'}))
	if cmd != nil {
		_ = cmd
	}
	s2, ok := typed.(SummaryModel)
	if !ok {
		t.Fatalf("SummaryModel.Update type mismatch: got %T", typed)
	}
	if got := s2.IsDirty(); got != true {
		t.Fatalf("SummaryModel.Update dirty mismatch: got %v want true", got)
	}

	s2.SetContent("incoming")
	if got := s2.IsOverwritePending(); got != true {
		t.Fatalf("SummaryModel overwrite pending mismatch: got %v want true", got)
	}
	s2.RejectOverwrite()
	if got := s2.IsOverwritePending(); got != false {
		t.Fatalf("SummaryModel.RejectOverwrite mismatch: got %v want false", got)
	}

	s2.SetContent("incoming")
	s2.AcceptOverwrite()
	if got := s2.Value(); got != "incoming" {
		t.Fatalf("SummaryModel.AcceptOverwrite value mismatch: got %q want %q", got, "incoming")
	}

	errModel, _ := s2.Update(tea.KeyPressMsg(tea.Key{Text: "s", Code: 's', Mod: tea.ModCtrl}))
	s3, ok := errModel.(SummaryModel)
	if !ok {
		t.Fatalf("SummaryModel ctrl+s type mismatch: got %T", errModel)
	}
	if got := s3.saveCmd()(); reflect.TypeOf(got) != reflect.TypeOf(SummarySaveErrMsg{}) {
		t.Fatalf("SummaryModel.saveCmd error msg type mismatch: got %T", got)
	}

	doneModel, _ := s3.Update(SummarySaveDoneMsg{})
	s4 := doneModel.(SummaryModel)
	if got := s4.IsDirty(); got != false {
		t.Fatalf("SummarySaveDoneMsg dirty mismatch: got %v want false", got)
	}

	targetErr := errors.New("boom")
	errUpdated, _ := s4.Update(SummarySaveErrMsg{Err: targetErr})
	s5 := errUpdated.(SummaryModel)
	if got := strings.Contains(s5.statusMsg, "boom"); got != true {
		t.Fatalf("SummarySaveErrMsg status mismatch: got %q", s5.statusMsg)
	}

	s5.statusMsg = "stale"
	s5.statusExpiry = time.Now().Add(-time.Second)
	cleared, _ := s5.Update(tea.WindowSizeMsg{Width: 50, Height: 20})
	s6 := cleared.(SummaryModel)
	if got := s6.statusMsg; got != "" {
		t.Fatalf("SummaryModel expired status mismatch: got %q want empty", got)
	}
}

func TestCharmV2_AdditionalPublicAPIContracts(t *testing.T) {
	if got := NewAppModel(&config.Config{}, nil, nil, "", false).Init(); got != nil {
		t.Fatalf("AppModel.Init mismatch: got %v want nil", got)
	}

	meeting := storage.Meeting{
		Title:    "Weekly Sync",
		TicketID: "ABC-123",
		Project:  "ABC",
		Date:     time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		Status:   storage.MeetingStatusStructured,
		Tags:     []string{"ops", "urgent"},
	}
	item := MeetingItem{meeting: meeting}
	if got, want := item.FilterValue(), "Weekly Sync ABC-123 ops urgent"; got != want {
		t.Fatalf("MeetingItem.FilterValue mismatch: got %q want %q", got, want)
	}
	if got, want := item.Title(), "[ABC-123] Weekly Sync"; got != want {
		t.Fatalf("MeetingItem.Title mismatch: got %q want %q", got, want)
	}
	if got, want := item.Description(), "2026-01-02 | ABC | ✓ STRUCTURED"; got != want {
		t.Fatalf("MeetingItem.Description mismatch: got %q want %q", got, want)
	}

	pItem := ProviderItem{name: "openrouter", displayName: "OpenRouter", description: "desc", current: true}
	if got, want := pItem.FilterValue(), "OpenRouter"; got != want {
		t.Fatalf("ProviderItem.FilterValue mismatch: got %q want %q", got, want)
	}
	if got, want := pItem.Title(), "✓ OpenRouter"; got != want {
		t.Fatalf("ProviderItem.Title mismatch: got %q want %q", got, want)
	}
	if got, want := pItem.Description(), "desc"; got != want {
		t.Fatalf("ProviderItem.Description mismatch: got %q want %q", got, want)
	}

	listModel := NewListModel(nil, 80, 24)
	if got := strings.Contains(listModel.View().Content, "No meetings yet. Press 'n' to create one."); got != true {
		t.Fatalf("ListModel.View empty state mismatch: got %q", listModel.View().Content)
	}

	providerModel := NewProviderModel("openrouter", 80, 24)
	if got := strings.Contains(providerModel.View().Content, "Select AI Provider"); got != true {
		t.Fatalf("ProviderModel.View content mismatch: got %q", providerModel.View().Content)
	}

	help := NewHelpModel()
	if got := strings.Contains(help.View().Content, "⌨ Keybindings"); got != true {
		t.Fatalf("HelpModel.View content mismatch: got %q", help.View().Content)
	}

	email := NewEmailModel("Sub", "Body", 80, 24, "en")
	if got := email.Init(); got != nil {
		t.Fatalf("EmailModel.Init mismatch: got %v want nil", got)
	}
	emailUpdated, _ := email.Update(EmailContentMsg{Subject: "S2", Body: "B2"})
	emailModel := emailUpdated.(EmailModel)
	emailView := emailModel.View().Content
	if got := strings.Contains(emailView, "Email Summary"); got != true {
		t.Fatalf("EmailModel.View header mismatch: got %q", emailView)
	}
	if got := strings.Contains(emailView, "Subject: S2"); got != true {
		t.Fatalf("EmailModel.View subject mismatch: got %q", emailView)
	}
	if got := strings.Contains(emailView, "B2"); got != true {
		t.Fatalf("EmailModel.View body mismatch: got %q", emailView)
	}
	if got := strings.Contains(emailView, "lang:EN"); got != true {
		t.Fatalf("EmailModel.View footer mismatch: got %q", emailView)
	}

	preview := NewPreviewModel("hello", "title", 80, 24)
	if got := strings.Contains(preview.View().Content, "── Preview: title ──"); got != true {
		t.Fatalf("PreviewModel.View content mismatch: got %q", preview.View().Content)
	}

	editor := NewEditorModel(nil, nil, 80, 24, "GitHub Models", "gpt")
	if got := editor.Init(); got == nil {
		t.Fatal("EditorModel.Init expected non-nil batch command")
	}
	sm := NewSummaryModel("summary", "PRJ", "m1", nil, 40, 10)
	editor.SetSummaryModel(sm)
	if got := editor.IsSplitMode(); got != false {
		t.Fatalf("EditorModel.IsSplitMode mismatch: got %v want false", got)
	}
	if got, want := editor.GetSummaryModel().Value(), "summary"; got != want {
		t.Fatalf("EditorModel.GetSummaryModel mismatch: got %q want %q", got, want)
	}
	if got := strings.Contains(editor.View().Content, "ctrl+s Save"); got != true {
		t.Fatalf("EditorModel.View legend mismatch: got %q", editor.View().Content)
	}

	basicNewMeeting := NewMeetingModel{form: nil}
	if got, want := basicNewMeeting.View().Content, "Unable to render new meeting form"; got != want {
		t.Fatalf("NewMeetingModel.View nil form mismatch: got %q want %q", got, want)
	}

	cycle := "en"
	for i := 0; i < 3; i++ {
		cycle = nextEmailLanguage(cycle)
	}
	if got, want := cycle, "en"; got != want {
		t.Fatalf("nextEmailLanguage 3-step cycle mismatch: got %q want %q", got, want)
	}
}
