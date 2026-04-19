package languages

// Language represents a supported language code and its display name.
type Language struct {
	Code string
	Name string
}

var AllLanguages = []Language{
	{Code: "en", Name: "English"},
	{Code: "pl", Name: "Polish"},
	{Code: "de", Name: "German"},
	{Code: "no", Name: "Norwegian"},
	{Code: "zh", Name: "Chinese"},
	{Code: "hi", Name: "Hindi"},
	{Code: "es", Name: "Spanish"},
	{Code: "fr", Name: "French"},
	{Code: "ar", Name: "Arabic"},
	{Code: "bn", Name: "Bengali"},
	{Code: "pt", Name: "Portuguese"},
	{Code: "ru", Name: "Russian"},
	{Code: "ja", Name: "Japanese"},
}

var DefaultEnabledCodes = []string{"en", "pl", "de", "no"}

const (
	MaxSelected = 5
	MinSelected = 1
)

var languageNamesByCode map[string]string

func init() {
	languageNamesByCode = make(map[string]string, len(AllLanguages))
	for _, language := range AllLanguages {
		languageNamesByCode[language.Code] = language.Name
	}
}

// ValidCode reports whether a language code exists in AllLanguages.
func ValidCode(code string) bool {
	_, exists := languageNamesByCode[code]
	return exists
}

// DisplayName returns a language display name for a code, defaulting to English.
func DisplayName(code string) string {
	if name, exists := languageNamesByCode[code]; exists {
		return name
	}

	return "English"
}
