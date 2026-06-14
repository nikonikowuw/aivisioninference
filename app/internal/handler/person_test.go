package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestPersonHandler_ViewImage_PathTraversal 测试 ViewImage 的路径遍历防护和文件扩展名校验。
// 这些验证在调用 svc.GetImage 之前执行，因此 nil service 即可测试。
func TestPersonHandler_ViewImage_PathTraversal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &PersonHandler{svc: nil}

	tests := []struct {
		name       string
		filename   string
		wantErrors int
	}{
		{
			name:       "path traversal with double dots",
			filename:   "../../etc/passwd",
			wantErrors: 1,
		},
		{
			name:       "path traversal with forward slash",
			filename:   "dir/image.jpg",
			wantErrors: 1,
		},
		{
			name:       "path traversal with backslash",
			filename:   "..\\..\\file.jpg",
			wantErrors: 1,
		},
		{
			name:       "unsupported extension exe",
			filename:   "virus.exe",
			wantErrors: 1,
		},
		{
			name:       "unsupported extension svg",
			filename:   "image.svg",
			wantErrors: 1,
		},
		{
			name:       "no extension",
			filename:   "file",
			wantErrors: 1,
		},
		{
			name:       "empty filename",
			filename:   "",
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			c.Params = []gin.Param{{Key: "filename", Value: tt.filename}}

			h.ViewImage(c)
			assert.Len(t, c.Errors, tt.wantErrors)
		})
	}
}

// TestPersonHandler_BatchToggle_Validation 测试批量启用/禁用请求的参数校验。
// 注意：合法请求会进一步调用 svc.BatchToggle，本测试仅覆盖验证拦截路径。
func TestPersonHandler_BatchToggle_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &PersonHandler{svc: nil}

	tests := []struct {
		name       string
		body       interface{}
		wantErrors int
	}{
		{
			name:       "missing ids field",
			body:       map[string]interface{}{"enabled": true},
			wantErrors: 1,
		},
		{
			name:       "empty ids array",
			body:       map[string]interface{}{"ids": []string{}, "enabled": true},
			wantErrors: 1,
		},
		{
			name:       "ids contains empty string",
			body:       map[string]interface{}{"ids": []string{""}, "enabled": true},
			wantErrors: 1,
		},
		{
			name:       "ids exceeds max limit",
			body: func() interface{} {
				ids := make([]string, 101)
				for i := range ids {
					ids[i] = "id"
				}
				return map[string]interface{}{"ids": ids, "enabled": true}
			}(),
			wantErrors: 1,
		},
		{
			name:       "invalid json",
			body:       "{invalid json}",
			wantErrors: 1,
		},
		{
			name:       "empty body",
			body:       "",
			wantErrors: 1,
		},
		{
			name:       "ids is not an array",
			body:       map[string]interface{}{"ids": "not-an-array", "enabled": true},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			var bodyBytes []byte
			switch b := tt.body.(type) {
			case string:
				bodyBytes = []byte(b)
			default:
				bodyBytes, _ = json.Marshal(b)
			}

			c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			h.BatchToggle(c)
			assert.Len(t, c.Errors, tt.wantErrors)
		})
	}
}

// TestPersonHandler_BatchDelete_Validation 测试批量删除请求的参数校验。
func TestPersonHandler_BatchDelete_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &PersonHandler{svc: nil}

	tests := []struct {
		name       string
		body       interface{}
		wantErrors int
	}{
		{
			name:       "missing ids field",
			body:       map[string]interface{}{},
			wantErrors: 1,
		},
		{
			name:       "empty ids array",
			body:       map[string]interface{}{"ids": []string{}},
			wantErrors: 1,
		},
		{
			name:       "ids with empty element",
			body:       map[string]interface{}{"ids": []string{""}},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			bodyBytes, _ := json.Marshal(tt.body)
			c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			h.BatchDelete(c)
			assert.Len(t, c.Errors, tt.wantErrors)
		})
	}
}

// TestPersonHandler_BatchRetryEmbedding_Validation 测试批量重提特征请求的参数校验。
func TestPersonHandler_BatchRetryEmbedding_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &PersonHandler{svc: nil}

	tests := []struct {
		name       string
		body       interface{}
		wantErrors int
	}{
		{
			name:       "missing ids field",
			body:       map[string]interface{}{},
			wantErrors: 1,
		},
		{
			name:       "empty ids array",
			body:       map[string]interface{}{"ids": []string{}},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			bodyBytes, _ := json.Marshal(tt.body)
			c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			h.BatchRetryEmbedding(c)
			assert.Len(t, c.Errors, tt.wantErrors)
		})
	}
}

// TestPersonHandler_SearchByFace_MissingImage 测试以图搜人接口缺少图片文件时的处理。
func TestPersonHandler_SearchByFace_MissingImage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &PersonHandler{svc: nil}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/search-by-face", nil)
	c.Request.Header.Set("Content-Type", "multipart/form-data")

	h.SearchByFace(c)
	assert.Len(t, c.Errors, 1)
}

// TestPersonHandler_CreateGroup_Validation 测试创建分组请求的参数校验。
func TestPersonHandler_CreateGroup_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &PersonHandler{svc: nil}

	tests := []struct {
		name       string
		body       interface{}
		wantErrors int
	}{
		{
			name:       "missing group_name",
			body:       map[string]interface{}{"description": "test"},
			wantErrors: 1,
		},
		{
			name:       "empty group_name",
			body:       map[string]interface{}{"group_name": ""},
			wantErrors: 1,
		},
		{
			name:       "empty body",
			body:       "",
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			var bodyBytes []byte
			switch b := tt.body.(type) {
			case string:
				bodyBytes = []byte(b)
			default:
				bodyBytes, _ = json.Marshal(b)
			}

			c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			h.CreateGroup(c)
			assert.Len(t, c.Errors, tt.wantErrors)
		})
	}
}

// TestPersonHandler_UpdateGroup_Validation 测试更新分组请求的参数校验。
func TestPersonHandler_UpdateGroup_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &PersonHandler{svc: nil}

	tests := []struct {
		name       string
		body       interface{}
		wantErrors int
	}{
		{
			name:       "empty group_name",
			body:       map[string]interface{}{"group_name": ""},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			bodyBytes, _ := json.Marshal(tt.body)
			c.Request = httptest.NewRequest(http.MethodPut, "/", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Params = []gin.Param{{Key: "id", Value: "some-id"}}

			h.UpdateGroup(c)
			assert.Len(t, c.Errors, tt.wantErrors)
		})
	}
}

// TestPersonHandler_CreateTag_Validation 测试创建标签请求的参数校验。
func TestPersonHandler_CreateTag_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &PersonHandler{svc: nil}

	tests := []struct {
		name       string
		body       interface{}
		wantErrors int
	}{
		{
			name:       "missing tag_name",
			body:       map[string]interface{}{"color": "blue"},
			wantErrors: 1,
		},
		{
			name:       "empty tag_name",
			body:       map[string]interface{}{"tag_name": ""},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			bodyBytes, _ := json.Marshal(tt.body)
			c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			h.CreateTag(c)
			assert.Len(t, c.Errors, tt.wantErrors)
		})
	}
}

// TestPersonHandler_ImportByURL_Validation 测试通过URL导入请求的参数校验。
func TestPersonHandler_ImportByURL_Validation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &PersonHandler{svc: nil}

	tests := []struct {
		name       string
		body       interface{}
		wantErrors int
	}{
		{
			name:       "missing file_url",
			body:       map[string]interface{}{"overwrite_on_duplicate": true},
			wantErrors: 1,
		},
		{
			name:       "empty file_url",
			body:       map[string]interface{}{"file_url": ""},
			wantErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			bodyBytes, _ := json.Marshal(tt.body)
			c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer(bodyBytes))
			c.Request.Header.Set("Content-Type", "application/json")

			h.ImportByURL(c)
			assert.Len(t, c.Errors, tt.wantErrors)
		})
	}
}
