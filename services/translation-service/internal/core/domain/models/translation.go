package models

// TranslationRequest is the transport-agnostic domain input for a translation.
type TranslationRequest struct {
	Text       string
	TargetLang string // ISO 639-1 code supplied by the caller
	SourceLang string // optional; empty means provider auto-detect
}

// TranslationResult is the transport-agnostic domain output for a translation.
type TranslationResult struct {
	TranslatedText               string
	DetectedLang                 string // ISO 639-1 code reported by the provider
	SourceLang                   string
	TargetLang                   string
	CharacterCount               int // billed character count when available from the provider
	ConfidenceScore              float64
	TranslationProvider          string
	MedicalTermPreservationCheck bool
	AdvisoryMessage              string
}

// SupportedLanguageCodes maps normalized ISO 639-1 inputs to the provider-neutral
// lowercase codes expected by Amazon Translate and accepted by local providers.
var SupportedLanguageCodes = map[string]string{
	"bg": "bg",
	"cs": "cs",
	"da": "da",
	"de": "de",
	"el": "el",
	"en": "en",
	"es": "es",
	"et": "et",
	"fi": "fi",
	"fr": "fr",
	"hu": "hu",
	"id": "id",
	"it": "it",
	"ja": "ja",
	"ko": "ko",
	"lt": "lt",
	"lv": "lv",
	"nb": "no",
	"ne": "ne",
	"nl": "nl",
	"pl": "pl",
	"pt": "pt",
	"ro": "ro",
	"ru": "ru",
	"sk": "sk",
	"sl": "sl",
	"sv": "sv",
	"tr": "tr",
	"uk": "uk",
	"zh": "zh",
}
