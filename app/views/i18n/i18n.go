package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/text/language"
)

//go:embed *.json
var translationFiles embed.FS

var supportedLanguages = []string{"en", "es", "ko", "ja", "zh-hans", "zh-hant", "th"}

// Translator handles simple JSON-backed translations for templates.
type Translator struct {
	translations map[string]map[string]interface{}
	fallback     string
}

// NewTranslator creates a translator from embedded translation files.
func NewTranslator() (*Translator, error) {
	t := &Translator{
		translations: make(map[string]map[string]interface{}),
		fallback:     "en",
	}

	for _, lang := range supportedLanguages {
		if err := t.loadLanguage(lang); err != nil {
			return nil, fmt.Errorf("loading language %s: %w", lang, err)
		}
	}

	return t, nil
}

func (t *Translator) loadLanguage(lang string) error {
	filename := fmt.Sprintf("%s.json", lang)
	data, err := os.ReadFile(filepath.Join("gojang", "views", "i18n", filename))
	if err != nil {
		data, err = os.ReadFile(filepath.Join("app", "views", "i18n", filename))
	}
	if err != nil {
		data, err = translationFiles.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("reading translation file %s: %w", filename, err)
		}
	}

	translations := make(map[string]interface{})
	if err := json.Unmarshal(data, &translations); err != nil {
		return fmt.Errorf("parsing translation file %s: %w", filename, err)
	}

	t.translations[lang] = translations
	return nil
}

// Translate returns the translated string for a key and language.
func (t *Translator) Translate(lang, key string, args ...interface{}) string {
	lang = t.normalizeLang(lang)

	if translations, ok := t.translations[lang]; ok {
		if value, ok := translations[key]; ok {
			return formatValue(value, args...)
		}
	}

	if translations, ok := t.translations[t.fallback]; ok {
		if value, ok := translations[key]; ok {
			return formatValue(value, args...)
		}
	}

	return key
}

// TranslateArray returns a translated string array for a key and language.
func (t *Translator) TranslateArray(lang, key string) []string {
	lang = t.normalizeLang(lang)

	for _, candidate := range []string{lang, t.fallback} {
		translations, ok := t.translations[candidate]
		if !ok {
			continue
		}
		value, ok := translations[key]
		if !ok {
			continue
		}
		arr, ok := value.([]interface{})
		if !ok {
			continue
		}
		result := make([]string, len(arr))
		for i, item := range arr {
			result[i] = fmt.Sprint(item)
		}
		return result
	}

	return []string{}
}

// GetSupportedLanguages returns the configured language codes.
func (t *Translator) GetSupportedLanguages() []string {
	langs := make([]string, 0, len(t.translations))
	for lang := range t.translations {
		langs = append(langs, lang)
	}
	return langs
}

// IsSupportedLanguage reports whether the language is available.
func (t *Translator) IsSupportedLanguage(lang string) bool {
	_, ok := t.translations[t.normalizeLang(lang)]
	return ok
}

// NormalizeLanguage returns a supported language code or the fallback.
func (t *Translator) NormalizeLanguage(lang string) string {
	normalized := t.normalizeLang(lang)
	if _, ok := t.translations[normalized]; ok {
		return normalized
	}
	return t.fallback
}

// DetectLanguage detects the request language.
// Priority: Cookie > URL parameter > Accept-Language header > fallback.
func (t *Translator) DetectLanguage(r *http.Request) string {
	if r == nil {
		return t.fallback
	}

	resolveDirectLang := func(raw string) (string, bool) {
		lang := strings.ToLower(strings.TrimSpace(raw))
		if lang == "" {
			return "", false
		}
		lang = t.normalizeLang(lang)
		if _, ok := t.translations[lang]; ok {
			return lang, true
		}
		return "", false
	}

	if cookie, err := r.Cookie("lang"); err == nil {
		if lang, ok := resolveDirectLang(cookie.Value); ok {
			return lang
		}
	}

	if r.URL != nil {
		if lang, ok := resolveDirectLang(r.URL.Query().Get("lang")); ok {
			return lang
		}
	}

	acceptLang := r.Header.Get("Accept-Language")
	if acceptLang == "" {
		return t.fallback
	}

	supportedTags := make([]language.Tag, 0, len(t.translations))
	keys := make([]string, 0, len(t.translations))
	for _, key := range supportedLanguages {
		if _, ok := t.translations[key]; !ok {
			continue
		}
		tag, err := language.Parse(key)
		if err != nil {
			continue
		}
		supportedTags = append(supportedTags, tag)
		keys = append(keys, key)
	}
	matcher := language.NewMatcher(supportedTags)

	for _, part := range strings.Split(acceptLang, ",") {
		part = strings.TrimSpace(strings.Split(part, ";")[0])
		if part == "" {
			continue
		}
		tag := language.Make(part)
		_, idx, conf := matcher.Match(tag)
		if conf != language.No && idx >= 0 && idx < len(keys) {
			return keys[idx]
		}
	}

	return t.fallback
}

func (t *Translator) normalizeLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if _, ok := t.translations[lang]; ok {
		return lang
	}

	baseLang := strings.Split(lang, "-")[0]
	if _, ok := t.translations[baseLang]; ok {
		return baseLang
	}

	return lang
}

func formatValue(value interface{}, args ...interface{}) string {
	text, ok := value.(string)
	if !ok {
		return fmt.Sprint(value)
	}

	result := text
	for i, arg := range args {
		result = strings.ReplaceAll(result, fmt.Sprintf("{%d}", i), fmt.Sprint(arg))
	}
	return result
}
