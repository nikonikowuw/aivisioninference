package i18n

import "testing"

func TestTranslationsCoverDefaultLanguageCodes(t *testing.T) {
	mu.RLock()
	defer mu.RUnlock()

	defaultMessages := storage[DefaultLanguage]
	if len(defaultMessages) == 0 {
		t.Fatal("default language translations are empty")
	}

	for lang, messages := range storage {
		if lang == DefaultLanguage {
			continue
		}
		for code := range defaultMessages {
			if _, ok := messages[code]; !ok {
				t.Fatalf("language %q missing translation for code %d", lang, code)
			}
		}
	}
}
