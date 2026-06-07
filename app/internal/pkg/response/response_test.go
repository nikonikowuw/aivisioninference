package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
)

func TestErrLocalizesDefaultMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("lang", "en")

	Err(c, apperrors.New(apperrors.ErrUnauthorized, ""))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
	var body Response
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Message != "Unauthorized" {
		t.Fatalf("expected english default message, got %q", body.Message)
	}
}

func TestErrIgnoresNonLocalizedExplicitMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("lang", "en")

	Err(c, apperrors.New(apperrors.ErrUnauthorized, "缺少认证令牌"))

	var body Response
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Message != "Unauthorized" {
		t.Fatalf("expected translated code message, got %q", body.Message)
	}
}

func TestErrKeepsLocalizedMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("lang", "en")

	Err(c, apperrors.NewLocalized(apperrors.ErrBadRequest, "email is a required field"))

	var body Response
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Message != "email is a required field" {
		t.Fatalf("expected localized message, got %q", body.Message)
	}
}

func TestErrDefaultsToEnglishWhenLanguageMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Err(c, apperrors.New(apperrors.ErrUnauthorized, ""))

	var body Response
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if body.Message != "Unauthorized" {
		t.Fatalf("expected english fallback message, got %q", body.Message)
	}
}
