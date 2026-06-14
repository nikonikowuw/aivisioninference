// Package service 提供人员管理模块的业务逻辑测试。
package service

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
)

// ---------------------------------------------------------------------------
// MockPersonRepo — testify mock for the unexported personRepo interface.
// ---------------------------------------------------------------------------

type MockPersonRepo struct {
	mock.Mock
}

func (m *MockPersonRepo) Create(ctx context.Context, item *model.Person) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockPersonRepo) FindByID(ctx context.Context, id string) (*model.Person, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Person), args.Error(1)
}

func (m *MockPersonRepo) FindByIDs(ctx context.Context, ids []string) ([]model.Person, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]model.Person), args.Error(1)
}

func (m *MockPersonRepo) List(ctx context.Context, req dto.PersonListRequest) ([]model.Person, int64, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]model.Person), args.Get(1).(int64), args.Error(2)
}

func (m *MockPersonRepo) ListForExport(ctx context.Context, req dto.PersonListRequest, limit int) ([]model.Person, error) {
	args := m.Called(ctx, req, limit)
	return args.Get(0).([]model.Person), args.Error(1)
}

func (m *MockPersonRepo) Update(ctx context.Context, item *model.Person) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockPersonRepo) SoftDelete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPersonRepo) BatchSoftDelete(ctx context.Context, ids []string) error {
	args := m.Called(ctx, ids)
	return args.Error(0)
}

func (m *MockPersonRepo) BatchToggle(ctx context.Context, ids []string, enabled bool) error {
	args := m.Called(ctx, ids, enabled)
	return args.Error(0)
}

func (m *MockPersonRepo) UpdateEmbeddingStatus(ctx context.Context, id, status, code, messageKey string, retryable bool) error {
	args := m.Called(ctx, id, status, code, messageKey, retryable)
	return args.Error(0)
}

func (m *MockPersonRepo) ExistsByPersonCode(ctx context.Context, code, excludeID string) (bool, error) {
	args := m.Called(ctx, code, excludeID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPersonRepo) ExistsByImageMD5(ctx context.Context, md5, excludeID string) (bool, error) {
	args := m.Called(ctx, md5, excludeID)
	return args.Bool(0), args.Error(1)
}

func (m *MockPersonRepo) ReplaceGroups(ctx context.Context, personID string, groupIDs []string) error {
	args := m.Called(ctx, personID, groupIDs)
	return args.Error(0)
}

func (m *MockPersonRepo) DB() *gorm.DB {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*gorm.DB)
}

// ---------------------------------------------------------------------------
// MockStorage — testify mock for storage.Storage interface.
// ---------------------------------------------------------------------------

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Save(reader io.Reader, path string) (string, error) {
	args := m.Called(reader, path)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) Get(path string) (io.ReadCloser, error) {
	args := m.Called(path)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (m *MockStorage) Delete(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockStorage) GetURL(path string) string {
	args := m.Called(path)
	return args.String(0)
}

func (m *MockStorage) ReadAt(path string, p []byte, off int64) (int, error) {
	args := m.Called(path, p, off)
	return args.Int(0), args.Error(1)
}

func (m *MockStorage) Size(path string) (int64, error) {
	args := m.Called(path)
	return args.Get(0).(int64), args.Error(1)
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// generateSmallPNG 生成一个 1x1 红色像素的合法 PNG 字节切片，用于测试图片内嵌。
func generateSmallPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

// generateSmallJPEG 生成一个 1x1 蓝色像素的合法 JPEG 字节切片，用于测试图片内嵌。
func generateSmallJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 0, G: 0, B: 255, A: 255})
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	return buf.Bytes()
}

// reopenXLSX 将 xlsx 字节重新读入 excelize.File，方便后续断言。
func reopenXLSX(t *testing.T, data []byte) *excelize.File {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f
}

// ---------------------------------------------------------------------------
// Tests: ExtractPersonImageStoragePath
// ---------------------------------------------------------------------------

func TestExtractPersonImageStoragePath(t *testing.T) {
	tests := []struct {
		name          string
		imageURL      string
		storageBaseURL string
		want          string
	}{
		{
			name:          "empty url",
			imageURL:      "",
			storageBaseURL: "",
			want:          "",
		},
		{
			name:          "api marker url",
			imageURL:      "/api/v1/persons/image/abc123.jpg",
			storageBaseURL: "",
			want:          "persons/abc123.jpg",
		},
		{
			name:          "api marker with full path",
			imageURL:      "/api/v1/persons/image/subdir/abc123.jpg",
			storageBaseURL: "",
			want:          "persons/abc123.jpg",
		},
		{
			name:          "storage base url match — no trailing slash in base",
			imageURL:      "http://minio:9000/niko/persons/abc.jpg",
			storageBaseURL: "http://minio:9000/niko",
			want:          "persons/abc.jpg",
		},
		{
			name:          "storage base url match — base url has trailing slash",
			imageURL:      "http://minio:9000/niko/persons/abc.jpg",
			storageBaseURL: "http://minio:9000/niko/",
			want:          "persons/abc.jpg",
		},
		{
			name:          "uploads prefix absolute",
			imageURL:      "/uploads/persons/abc.jpg",
			storageBaseURL: "",
			want:          "persons/abc.jpg",
		},
		{
			name:          "uploads prefix relative",
			imageURL:      "uploads/persons/abc.jpg",
			storageBaseURL: "",
			want:          "persons/abc.jpg",
		},
		{
			name:          "uploads in middle of path",
			imageURL:      "/foo/uploads/persons/abc.jpg",
			storageBaseURL: "",
			want:          "persons/abc.jpg",
		},
		{
			name:          "plain relative path",
			imageURL:      "persons/abc.jpg",
			storageBaseURL: "",
			want:          "persons/abc.jpg",
		},
		{
			name:          "plain absolute path",
			imageURL:      "/persons/abc.jpg",
			storageBaseURL: "",
			want:          "persons/abc.jpg",
		},
		{
			name:          "storage base url match — no leading slash after trim",
			imageURL:      "/data/niko/persons/abc.jpg",
			storageBaseURL: "/data/niko",
			want:          "persons/abc.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractPersonImageStoragePath(tt.imageURL, tt.storageBaseURL)
			if got != tt.want {
				t.Errorf("ExtractPersonImageStoragePath(%q, %q) = %q; want %q",
					tt.imageURL, tt.storageBaseURL, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Tests: PersonService.ExportExcel
// ---------------------------------------------------------------------------

// TestPersonServiceExportExcel_Fallback 验证主路径读取失败时 fallback 成功、
// 图片被正确内嵌到 Excel 单元格 A2。
func TestPersonServiceExportExcel_Fallback(t *testing.T) {
	imgBytes := generateSmallPNG(t)

	item := model.Person{
		BaseModel: model.BaseModel{
			ID:        "test-person-id",
			CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		PersonCode:      "P001",
		PersonName:      "测试人员",
		Gender:          model.GenderMale,
		Phone:           "13800138000",
		ImageURL:        "/api/v1/persons/image/test-face.png",
		EmbeddingStatus: model.EmbeddingStatusActive,
		Enabled:         true,
	}

	repo := new(MockPersonRepo)
	repo.On("ListForExport", mock.Anything, mock.Anything, mock.Anything).
		Return([]model.Person{item}, nil)

	stor := new(MockStorage)
	stor.On("GetURL", "").Return("")
	// 主路径失败（getStoragePath 返回 "persons/test-face.png"）
	stor.On("Get", "persons/test-face.png").Return(nil, errors.New("not found"))
	// fallback 路径成功（ImageURL 去掉前导斜杠）
	stor.On("Get", "api/v1/persons/image/test-face.png").
		Return(io.NopCloser(bytes.NewReader(imgBytes)), nil)

	svc := &PersonService{
		personRepo: repo,
		storage:    stor,
	}

	var buf bytes.Buffer
	err := svc.ExportExcel(context.Background(), dto.PersonListRequest{}, &buf)
	require.NoError(t, err)
	require.True(t, buf.Len() > 0, "exported xlsx must not be empty")

	f := reopenXLSX(t, buf.Bytes())

	// 验证文字内容正常
	val, err := f.GetCellValue("Persons", "B2")
	require.NoError(t, err)
	assert.Equal(t, "测试人员", val)

	// 验证图片已内嵌到 A2
	pics, err := f.GetPictures("Persons", "A2")
	require.NoError(t, err)
	require.Len(t, pics, 1, "A2 must have 1 embedded picture")
	assert.Equal(t, ".png", pics[0].Extension)

	repo.AssertExpectations(t)
	stor.AssertExpectations(t)
}

// TestPersonServiceExportExcel_NilStorage 验证 storage = nil 时不同时 panic，
// 仍然可以导出纯文本 Excel，但 A2 格不含图片。
func TestPersonServiceExportExcel_NilStorage(t *testing.T) {
	item := model.Person{
		BaseModel: model.BaseModel{
			ID:        "test-person-id-2",
			CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		PersonCode:      "P002",
		PersonName:      "测试人员2",
		Gender:          model.GenderFemale,
		Phone:           "13900139000",
		ImageURL:        "/api/v1/persons/image/test-face2.png",
		EmbeddingStatus: model.EmbeddingStatusPending,
		Enabled:         true,
	}

	repo := new(MockPersonRepo)
	repo.On("ListForExport", mock.Anything, mock.Anything, mock.Anything).
		Return([]model.Person{item}, nil)

	svc := &PersonService{
		personRepo: repo,
		storage:    nil,
	}

	var buf bytes.Buffer
	err := svc.ExportExcel(context.Background(), dto.PersonListRequest{}, &buf)
	require.NoError(t, err)
	require.True(t, buf.Len() > 0, "exported xlsx must not be empty")

	f := reopenXLSX(t, buf.Bytes())

	// 文字内容正常
	val, err := f.GetCellValue("Persons", "B2")
	require.NoError(t, err)
	assert.Equal(t, "测试人员2", val)

	// A2 格不应有图片
	pics, err := f.GetPictures("Persons", "A2")
	require.NoError(t, err)
	assert.Len(t, pics, 0, "storage is nil, A2 must have no picture")

	repo.AssertExpectations(t)
}

// TestPersonServiceExportExcel_WebpExtension 验证 ImageURL 含 .webp 路径时，
// 图片自动转换为 PNG 格式嵌入 Excel。
func TestPersonServiceExportExcel_WebpExtension(t *testing.T) {
	imgBytes := generateSmallPNG(t)

	item := model.Person{
		BaseModel: model.BaseModel{
			ID:        "test-webp-person",
			CreatedAt: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		},
		PersonCode:      "P-WEBP",
		PersonName:      "WebP人员",
		Gender:          model.GenderFemale,
		Phone:           "13700137000",
		ImageURL:        "/uploads/persons/face.webp",
		EmbeddingStatus: model.EmbeddingStatusActive,
		Enabled:         true,
	}

	repo := new(MockPersonRepo)
	repo.On("ListForExport", mock.Anything, mock.Anything, mock.Anything).
		Return([]model.Person{item}, nil)

	stor := new(MockStorage)
	// getStoragePath: "/uploads/persons/face.webp" → ExtractPersonImageStoragePath → "persons/face.webp"
	stor.On("GetURL", "").Return("")
	stor.On("Get", "persons/face.webp").Return(io.NopCloser(bytes.NewReader(imgBytes)), nil)

	svc := &PersonService{
		personRepo: repo,
		storage:    stor,
	}

	var buf bytes.Buffer
	err := svc.ExportExcel(context.Background(), dto.PersonListRequest{}, &buf)
	require.NoError(t, err)
	require.True(t, buf.Len() > 0, "exported xlsx must not be empty")

	f := reopenXLSX(t, buf.Bytes())

	// 文字内容正常
	val, err := f.GetCellValue("Persons", "B2")
	require.NoError(t, err)
	assert.Equal(t, "WebP人员", val)

	// 图片已内嵌到 A2，扩展名必须为 .png（而不是 .webp）
	pics, err := f.GetPictures("Persons", "A2")
	require.NoError(t, err)
	require.Len(t, pics, 1, "A2 must have 1 embedded picture")
	assert.Equal(t, ".png", pics[0].Extension, "webp image should be converted to png")

	repo.AssertExpectations(t)
	stor.AssertExpectations(t)
}

// TestPersonServiceExportExcel_InvalidImage 验证图片数据损坏时，导出不中断，
// 仅跳过该图片记录日志。
func TestPersonServiceExportExcel_InvalidImage(t *testing.T) {
	item := model.Person{
		BaseModel: model.BaseModel{
			ID:        "test-invalid-img",
			CreatedAt: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		},
		PersonCode:      "P-INVALID",
		PersonName:      "损坏图片人员",
		Gender:          model.GenderMale,
		Phone:           "13600136000",
		ImageURL:        "/uploads/persons/corrupted.jpg",
		EmbeddingStatus: model.EmbeddingStatusFailed,
		Enabled:         true,
	}

	repo := new(MockPersonRepo)
	repo.On("ListForExport", mock.Anything, mock.Anything, mock.Anything).
		Return([]model.Person{item}, nil)

	stor := new(MockStorage)
	stor.On("GetURL", "").Return("")
	// 存储返回无效图片数据
	stor.On("Get", "persons/corrupted.jpg").Return(io.NopCloser(bytes.NewReader([]byte("not-an-image"))), nil)

	svc := &PersonService{
		personRepo: repo,
		storage:    stor,
	}

	var buf bytes.Buffer
	err := svc.ExportExcel(context.Background(), dto.PersonListRequest{}, &buf)
	require.NoError(t, err)
	require.True(t, buf.Len() > 0, "export must not be empty even with invalid image")

	f := reopenXLSX(t, buf.Bytes())

	// 文字内容正常
	val, err := f.GetCellValue("Persons", "B2")
	require.NoError(t, err)
	assert.Equal(t, "损坏图片人员", val)

	// A2 不应有图片（解码失败）
	pics, err := f.GetPictures("Persons", "A2")
	require.NoError(t, err)
	assert.Len(t, pics, 0, "invalid image must not produce a picture in A2")

	repo.AssertExpectations(t)
	stor.AssertExpectations(t)
}

// ---------------------------------------------------------------------------
// Tests: prepareExportImage
// ---------------------------------------------------------------------------

func TestPreparePersonExportImage(t *testing.T) {
	t.Run("png_bytes_png_ext", func(t *testing.T) {
		pngBytes := generateSmallPNG(t)
		out, ext, err := prepareExportImage(pngBytes, ".png")
		require.NoError(t, err)
		assert.Equal(t, ".png", ext)
		assert.Equal(t, pngBytes, out, "png bytes with .png ext must passthrough")
	})

	t.Run("png_bytes_jpg_ext", func(t *testing.T) {
		pngBytes := generateSmallPNG(t)
		out, ext, err := prepareExportImage(pngBytes, ".jpg")
		require.NoError(t, err)
		assert.Equal(t, ".png", ext, "actual format is png, must return .png regardless of hint")
		// PNG identifiable even with .jpg hint
		assert.Equal(t, pngBytes, out, "png bytes passthrough when decoded as png")
	})

	t.Run("png_bytes_webp_ext", func(t *testing.T) {
		pngBytes := generateSmallPNG(t)
		out, ext, err := prepareExportImage(pngBytes, ".webp")
		require.NoError(t, err)
		assert.Equal(t, ".png", ext, "actual format is png, must return .png even with .webp hint")
		// PNG is an excelize-supported format, passthrough
		assert.Equal(t, pngBytes, out)
	})

	t.Run("jpeg_bytes_jpg_ext", func(t *testing.T) {
		jpegBytes := generateSmallJPEG(t)
		out, ext, err := prepareExportImage(jpegBytes, ".jpg")
		require.NoError(t, err)
		assert.Equal(t, ".jpg", ext, "actual format is jpeg, must return .jpg")
		assert.Equal(t, jpegBytes, out, "jpeg bytes with .jpg ext must passthrough")
	})

	t.Run("jpeg_bytes_png_ext", func(t *testing.T) {
		jpegBytes := generateSmallJPEG(t)
		out, ext, err := prepareExportImage(jpegBytes, ".png")
		require.NoError(t, err)
		assert.Equal(t, ".jpg", ext, "actual format is jpeg, must return .jpg even with .png hint")
		assert.Equal(t, jpegBytes, out, "jpeg bytes passthrough when decoded as jpeg")
	})

	t.Run("invalid_bytes", func(t *testing.T) {
		_, _, err := prepareExportImage([]byte("not-an-image"), ".jpg")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "image decode")
	})

	t.Run("empty_bytes", func(t *testing.T) {
		_, _, err := prepareExportImage([]byte{}, ".jpg")
		require.Error(t, err)
	})
}
