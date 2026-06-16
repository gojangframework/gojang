package i18n

import (
	"net/http"
	"testing"
)

func TestNewTranslatorLoadsSupportedLanguages(t *testing.T) {
	translator, err := NewTranslator()
	if err != nil {
		t.Fatalf("NewTranslator returned error: %v", err)
	}

	langs := translator.GetSupportedLanguages()
	if len(langs) != 7 {
		t.Fatalf("expected 7 supported languages, got %d", len(langs))
	}

	expected := map[string]bool{
		"en": true, "es": true, "ko": true, "ja": true, "zh-hans": true, "zh-hant": true, "th": true,
	}
	for _, lang := range langs {
		if !expected[lang] {
			t.Fatalf("unexpected language %q", lang)
		}
	}
}

func TestTranslate(t *testing.T) {
	translator, err := NewTranslator()
	if err != nil {
		t.Fatalf("NewTranslator returned error: %v", err)
	}

	tests := []struct {
		name string
		lang string
		key  string
		want string
	}{
		{"english", "en", "welcome_to_gojang", "Welcome to Gojang"},
		{"spanish", "es", "welcome_to_gojang", "Bienvenido a Gojang"},
		{"korean", "ko", "welcome_to_gojang", "Gojang에 오신 것을 환영합니다"},
		{"fallback", "fr", "welcome_to_gojang", "Welcome to Gojang"},
		{"unknown key", "en", "missing_key", "missing_key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := translator.Translate(tt.lang, tt.key); got != tt.want {
				t.Fatalf("Translate(%q, %q) = %q, want %q", tt.lang, tt.key, got, tt.want)
			}
		})
	}
}

func TestTranslateWithArguments(t *testing.T) {
	translator, err := NewTranslator()
	if err != nil {
		t.Fatalf("NewTranslator returned error: %v", err)
	}

	got := translator.Translate("en", "welcome_user", "dev@example.com")
	want := "Welcome, dev@example.com!"
	if got != want {
		t.Fatalf("Translate with argument = %q, want %q", got, want)
	}
}

func TestTranslateArray(t *testing.T) {
	translator, err := NewTranslator()
	if err != nil {
		t.Fatalf("NewTranslator returned error: %v", err)
	}

	features := translator.TranslateArray("en", "framework_features")
	if len(features) != 4 {
		t.Fatalf("expected 4 framework features, got %d", len(features))
	}
	if features[0] != "Blazing fast Go performance" {
		t.Fatalf("unexpected first feature %q", features[0])
	}

	missing := translator.TranslateArray("en", "missing_array")
	if len(missing) != 0 {
		t.Fatalf("expected missing array to return empty slice, got %d items", len(missing))
	}
}

func TestDetectLanguage(t *testing.T) {
	translator, err := NewTranslator()
	if err != nil {
		t.Fatalf("NewTranslator returned error: %v", err)
	}

	tests := []struct {
		name       string
		cookieLang string
		urlLang    string
		acceptLang string
		want       string
	}{
		{"cookie wins", "ko", "es", "en-US,en;q=0.9", "ko"},
		{"invalid cookie falls back to URL", "fr", "es", "ko-KR,ko;q=0.9", "es"},
		{"URL wins over header", "", "ja", "es-ES,es;q=0.9", "ja"},
		{"invalid URL falls back to header", "", "fr", "th-TH,th;q=0.9", "th"},
		{"simplified Chinese header", "", "", "zh-CN,zh;q=0.9", "zh-hans"},
		{"traditional Chinese header", "", "", "zh-TW,zh;q=0.9", "zh-hant"},
		{"fallback", "", "", "fr-FR,fr;q=0.9", "en"},
		{"empty fallback", "", "", "", "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url := "/"
			if tt.urlLang != "" {
				url = "/?lang=" + tt.urlLang
			}
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Fatalf("NewRequest returned error: %v", err)
			}
			if tt.cookieLang != "" {
				req.AddCookie(&http.Cookie{Name: "lang", Value: tt.cookieLang})
			}
			if tt.acceptLang != "" {
				req.Header.Set("Accept-Language", tt.acceptLang)
			}

			if got := translator.DetectLanguage(req); got != tt.want {
				t.Fatalf("DetectLanguage() = %q, want %q", got, tt.want)
			}
		})
	}
}
