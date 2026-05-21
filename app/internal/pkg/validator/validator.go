// Package validator provides a configured validator instance with
// custom validation rules for request binding.
package validator

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/id"
	"github.com/go-playground/locales/ja"
	"github.com/go-playground/locales/ko"
	"github.com/go-playground/locales/zh"
	zhTWLocale "github.com/go-playground/locales/zh_Hant_TW"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	idTranslations "github.com/go-playground/validator/v10/translations/id"
	jaTranslations "github.com/go-playground/validator/v10/translations/ja"
	koTranslations "github.com/go-playground/validator/v10/translations/ko"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
	zhTWTranslations "github.com/go-playground/validator/v10/translations/zh_tw"
)

var (
	initOnce sync.Once
	initErr  error
	transMap map[string]ut.Translator
)

func registerTagName(v *validator.Validate) {
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
}

// InitGinBindingValidator initializes Gin's default validator and registers
// built-in translations.
func InitGinBindingValidator() error {
	initOnce.Do(func() {
		v, ok := binding.Validator.Engine().(*validator.Validate)
		if !ok {
			initErr = fmt.Errorf("gin binding validator engine type mismatch")
			return
		}

		registerTagName(v)

		uni := ut.New(en.New(), en.New(), zh.New(), zhTWLocale.New(), id.New(), ja.New(), ko.New())
		transMap = make(map[string]ut.Translator, 6)

		enTrans, found := uni.GetTranslator("en")
		if !found {
			initErr = fmt.Errorf("failed to get en translator")
			return
		}
		if err := enTranslations.RegisterDefaultTranslations(v, enTrans); err != nil {
			initErr = err
			return
		}
		transMap["en"] = enTrans

		zhTrans, found := uni.GetTranslator("zh")
		if !found {
			initErr = fmt.Errorf("failed to get zh translator")
			return
		}
		if err := zhTranslations.RegisterDefaultTranslations(v, zhTrans); err != nil {
			initErr = err
			return
		}
		transMap["zh"] = zhTrans

		zhTWTrans, found := uni.GetTranslator("zh_Hant_TW")
		if !found {
			initErr = fmt.Errorf("failed to get zh-tw translator")
			return
		}
		if err := zhTWTranslations.RegisterDefaultTranslations(v, zhTWTrans); err != nil {
			initErr = err
			return
		}
		transMap["zh-tw"] = zhTWTrans

		idTrans, found := uni.GetTranslator("id")
		if !found {
			initErr = fmt.Errorf("failed to get id translator")
			return
		}
		if err := idTranslations.RegisterDefaultTranslations(v, idTrans); err != nil {
			initErr = err
			return
		}
		transMap["id"] = idTrans

		jaTrans, found := uni.GetTranslator("ja")
		if !found {
			initErr = fmt.Errorf("failed to get ja translator")
			return
		}
		if err := jaTranslations.RegisterDefaultTranslations(v, jaTrans); err != nil {
			initErr = err
			return
		}
		transMap["ja"] = jaTrans

		koTrans, found := uni.GetTranslator("ko")
		if !found {
			initErr = fmt.Errorf("failed to get ko translator")
			return
		}
		if err := koTranslations.RegisterDefaultTranslations(v, koTrans); err != nil {
			initErr = err
			return
		}
		transMap["ko"] = koTrans
	})

	return initErr
}

// TranslateValidationError returns localized messages for validator errors.
func TranslateValidationError(err error, lang string) string {
	if err == nil {
		return ""
	}

	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return err.Error()
	}

	normalizedLang := normalizeLang(lang)
	// transMap is initialized at process startup by InitGinBindingValidator().
	// If initialization is skipped, fall back to the original error message.
	if transMap == nil {
		return err.Error()
	}

	trans := transMap[normalizedLang]
	if trans == nil {
		trans = transMap["en"]
	}
	if trans == nil {
		return err.Error()
	}

	msgs := make([]string, 0, len(verrs))
	for _, fe := range verrs {
		msgs = append(msgs, fe.Translate(trans))
	}

	var sep string
	switch normalizedLang {
	case "zh", "zh-tw", "ja", "ko":
		sep = "；"
	default:
		sep = "; "
	}
	return strings.Join(msgs, sep)
}

func normalizeLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	langMap := []struct {
		prefix string
		result string
	}{
		{"zh-tw", "zh-tw"},
		{"zh", "zh"},
		{"en", "en"},
		{"id", "id"},
		{"ja", "ja"},
		{"ko", "ko"},
	}
	for _, item := range langMap {
		if strings.HasPrefix(lang, item.prefix) {
			return item.result
		}
	}
	return "en"
}
