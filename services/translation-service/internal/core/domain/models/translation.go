package models

// TranslationRequest is the transport-agnostic domain input for a translation.
type TranslationRequest struct {
	Text       string
	TargetLang string // ISO 639-1 code supplied by the caller
	SourceLang string // optional; empty means provider auto-detect
}

// TranslationResult is the transport-agnostic domain output for a translation.
type TranslationResult struct {
	TranslatedText string
	DetectedLang   string // ISO 639-1 code reported by the provider
	CharacterCount int    // billed character count when available from the provider
}

// SupportedLanguageCodes maps normalized ISO 639-1 input codes to the
// provider-specific code sent to DeepL.
var SupportedLanguageCodes = map[string]string{
	"bg": "BG",
	"cs": "CS",
	"da": "DA",
	"de": "DE",
	"el": "EL",
	"en": "EN",
	"es": "ES",
	"et": "ET",
	"fi": "FI",
	"fr": "FR",
	"hu": "HU",
	"id": "ID",
	"it": "IT",
	"ja": "JA",
	"ko": "KO",
	"lt": "LT",
	"lv": "LV",
	"nb": "NB",
	"ne": "NE",
	"nl": "NL",
	"pl": "PL",
	"pt": "PT",
	"ro": "RO",
	"ru": "RU",
	"sk": "SK",
	"sl": "SL",
	"sv": "SV",
	"tr": "TR",
	"uk": "UK",
	"zh": "ZH",
}
