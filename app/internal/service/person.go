// Package service 提供人员管理模块的业务逻辑。
package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"image"
	_ "image/gif" // registers gif decoder with image.Decode
	_ "image/jpeg" // registers jpeg decoder with image.Decode
	"image/png"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
	"github.com/xuri/excelize/v2"
	"go.uber.org/zap"
	_ "golang.org/x/image/webp" // registers webp decoder with image.Decode
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/pkg/storage"
)

const (
	personEmbeddingTaskType        = "person:embedding"
	personEmbeddingRebuildTaskType = "person:embedding:rebuild"
	personImportTaskType           = "person:import"

	personImageMaxSize = 10 << 20 // 10MB
	// personImportMaxSize 限制导入压缩包最大体积，防止内存溢出。
	personImportMaxSize = 100 << 20 // 100MB

	// exportBatchSize 限制单次导出的最大记录数，防止 OOM 与过长的 HTTP 响应。
	// 如需全量导出，请改用异步导出任务。
	exportBatchSize = 10000
)

// personRepo 人员记录持久化接口。
type personRepo interface {
	Create(ctx context.Context, item *model.Person) error
	FindByID(ctx context.Context, id string) (*model.Person, error)
	FindByIDs(ctx context.Context, ids []string) ([]model.Person, error)
	List(ctx context.Context, req dto.PersonListRequest) ([]model.Person, int64, error)
	ListForExport(ctx context.Context, req dto.PersonListRequest, limit int) ([]model.Person, error)
	Update(ctx context.Context, item *model.Person) error
	SoftDelete(ctx context.Context, id string) error
	BatchSoftDelete(ctx context.Context, ids []string) error
	BatchToggle(ctx context.Context, ids []string, enabled bool) error
	UpdateEmbeddingStatus(ctx context.Context, id, status, code, messageKey string, retryable bool) error
	ExistsByPersonCode(ctx context.Context, code, excludeID string) (bool, error)
	ExistsByImageMD5(ctx context.Context, md5, excludeID string) (bool, error)
	ReplaceGroups(ctx context.Context, personID string, groupIDs []string) error
	DB() *gorm.DB
}

// personGroupRepo 分组持久化接口。
type personGroupRepo interface {
	Create(ctx context.Context, item *model.PersonGroup) error
	FindByID(ctx context.Context, id string) (*model.PersonGroup, error)
	Update(ctx context.Context, item *model.PersonGroup) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]model.PersonGroup, error)
	CountPersons(ctx context.Context, id string) (int64, error)
	BatchCountPersons(ctx context.Context) (map[string]int64, error)
}

// importTaskRepo 导入任务持久化接口。
type importTaskRepo interface {
	Create(ctx context.Context, item *model.ImportTask) error
	FindByID(ctx context.Context, id string) (*model.ImportTask, error)
	Update(ctx context.Context, item *model.ImportTask) error
	List(ctx context.Context, req dto.PageRequest) ([]model.ImportTask, int64, error)
}

// personTagRepo 标签持久化接口。
type personTagRepo interface {
	Create(ctx context.Context, item *model.PersonTag) error
	FindByID(ctx context.Context, id string) (*model.PersonTag, error)
	Update(ctx context.Context, item *model.PersonTag) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]model.PersonTag, error)
	BatchCountPersons(ctx context.Context) (map[string]int64, error)
	CountPersons(ctx context.Context, id string) (int64, error)
}

// personTagRelationRepo 标签关联持久化接口。
type personTagRelationRepo interface {
	BatchCreate(ctx context.Context, personID string, tagIDs []string) error
	DeleteByPersonRecordID(ctx context.Context, personID string) error
	ListTagIDsByPerson(ctx context.Context, personID string) ([]string, error)
}

// 安全图片扩展名集合
var imageExtSet = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
}

// PersonService 处理人员管理的业务逻辑。
type PersonService struct {
	personRepo          personRepo
	groupRepo           personGroupRepo
	tagRepo             personTagRepo
	tagRelationRepo     personTagRelationRepo
	importTaskRepo      importTaskRepo
	storage             storage.Storage
	taskClient          taskClient
	embeddingRepo       *repository.PersonEmbeddingRepository
	algorithmPackageRepo *repository.AlgorithmPackageRepository
	engine              EngineClient
}

// NewPersonService 创建人员 Service。
func NewPersonService(personRepo personRepo, groupRepo personGroupRepo, tagRepo personTagRepo, tagRelationRepo personTagRelationRepo, importTaskRepo importTaskRepo, storage storage.Storage, taskClient taskClient, embeddingRepo *repository.PersonEmbeddingRepository, algorithmPackageRepo *repository.AlgorithmPackageRepository, engine EngineClient) *PersonService {
	return &PersonService{personRepo: personRepo, groupRepo: groupRepo, tagRepo: tagRepo, tagRelationRepo: tagRelationRepo, importTaskRepo: importTaskRepo, storage: storage, taskClient: taskClient, embeddingRepo: embeddingRepo, algorithmPackageRepo: algorithmPackageRepo, engine: engine}
}

// List 查询人员列表。
func (s *PersonService) List(ctx context.Context, req dto.PersonListRequest) ([]dto.PersonResponse, int64, error) {
	items, total, err := s.personRepo.List(ctx, req)
	if err != nil {
		zap.L().Error("person list failed", zap.Error(err))
		return nil, 0, apperrors.New(apperrors.ErrInternal, "")
	}
	result := make([]dto.PersonResponse, len(items))
	for i := range items {
		result[i] = toPersonResponse(&items[i])
	}
	return result, total, nil
}

// GetByID 查询人员详情。
func (s *PersonService) GetByID(ctx context.Context, id string) (*dto.PersonResponse, error) {
	item, err := s.personRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "")
	}
	resp := toPersonResponse(item)
	return &resp, nil
}

// ExtractPersonImageStoragePath 从图片 URL 提取人员存储相对路径（无前导斜杠）。
// 此函数供 service 和 task 共享使用。
func ExtractPersonImageStoragePath(imageURL string, storageBaseURL string) string {
	if imageURL == "" {
		return ""
	}
	// 1. 如果包含 API 前缀，取文件名
	if marker := "/api/v1/persons/image/"; strings.Contains(imageURL, marker) {
		return "persons/" + filepath.Base(imageURL)
	}
	// 2. 如果包含存储基准 URL，去除 baseURL 部分并清理前导斜杠
	if storageBaseURL != "" && strings.HasPrefix(imageURL, storageBaseURL) {
		path := strings.TrimPrefix(imageURL, storageBaseURL)
		return strings.TrimPrefix(path, "/")
	}
	// 3. 处理 /uploads 前缀：确保返回相对于 uploadDir 的路径
	path := strings.TrimPrefix(imageURL, "/")
	for strings.HasPrefix(path, "uploads/") {
		path = strings.TrimPrefix(path, "uploads/")
	}
	if parts := strings.Split(path, "/uploads/"); len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return strings.TrimPrefix(path, "/")
}

// getStoragePath 从图片 URL 提取存储相对路径
func (s *PersonService) getStoragePath(imageURL string) string {
	baseURL := ""
	if s.storage != nil {
		baseURL = s.storage.GetURL("")
	}
	return ExtractPersonImageStoragePath(imageURL, baseURL)
}

// processImage 处理上传的图片或已有的 URL，返回 URL 和 MD5
func (s *PersonService) processImage(ctx context.Context, imageURL string, fileHeader *multipart.FileHeader, excludeID string) (string, string, error) {
	var (
		url string
		md5Hash string
	)

	if imageURL != "" {
		url = imageURL
		storagePath := s.getStoragePath(url)
		rc, err := s.storage.Get(storagePath)
		if err != nil {
			rc, err = s.storage.Get(strings.TrimPrefix(url, "/"))
		}
		if err == nil {
			defer rc.Close()
			hash := md5.New()
			if _, err := io.Copy(hash, rc); err == nil {
				md5Hash = fmt.Sprintf("%x", hash.Sum(nil))
			}
		} else {
			zap.L().Error("get image from storage failed", zap.String("url", url), zap.Error(err))
			return "", "", apperrors.New(apperrors.ErrInternal, "无法读取图片文件计算特征")
		}
	} else if fileHeader != nil {
		if fileHeader.Size > personImageMaxSize {
			return "", "", apperrors.New(apperrors.ErrFileTooLarge, "")
		}
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if !imageExtSet[ext] {
			return "", "", apperrors.New(apperrors.ErrFileInvalidType, "")
		}
		file, err := fileHeader.Open()
		if err != nil {
			return "", "", apperrors.New(apperrors.ErrBadRequest, "")
		}
		defer file.Close()

		hash := md5.New()
		data, err := io.ReadAll(io.TeeReader(file, hash))
		if err != nil {
			return "", "", apperrors.New(apperrors.ErrInternal, "")
		}
		md5Hash = fmt.Sprintf("%x", hash.Sum(nil))

		fileName := "persons/" + uuid.New().String() + ext
		if _, err := s.storage.Save(bytes.NewReader(data), fileName); err != nil {
			zap.L().Error("save person image failed", zap.Error(err))
			return "", "", apperrors.New(apperrors.ErrInternal, "")
		}
		url = fmt.Sprintf("/api/v1/persons/image/%s", filepath.Base(fileName))
	} else {
		return "", "", apperrors.New(apperrors.ErrPersonImageRequired, "")
	}

	if md5Hash != "" {
		if exists, _ := s.personRepo.ExistsByImageMD5(ctx, md5Hash, excludeID); exists {
			return "", "", apperrors.New(apperrors.ErrPersonImageDuplicate, "")
		}
	}

	return url, md5Hash, nil
}

// Create 创建人员。
func (s *PersonService) Create(ctx context.Context, req dto.PersonCreateRequest, fileHeader *multipart.FileHeader) (*dto.PersonResponse, error) {
	imageURL, imageMD5, err := s.processImage(ctx, req.ImageURL, fileHeader, "")
	if err != nil {
		return nil, err
	}

	if req.PersonCode != "" {
		if exists, _ := s.personRepo.ExistsByPersonCode(ctx, req.PersonCode, ""); exists {
			return nil, apperrors.New(apperrors.ErrPersonCodeDuplicate, "")
		}
	}

	personCode := req.PersonCode
	if personCode == "" {
		personCode = "P" + time.Now().Format("20060102150405") + uuid.New().String()[:6]
	}

	gender := req.Gender
	if gender == "" {
		gender = model.GenderUnknown
	}
	enabled := req.Enabled == nil || *req.Enabled

	person := &model.Person{
		PersonCode:      personCode,
		PersonName:      req.PersonName,
		Gender:          gender,
		Phone:           req.Phone,
		IDNumber:        req.IDNumber,
		ImageURL:        imageURL,
		ImageMD5:        imageMD5,
		EmbeddingStatus: model.EmbeddingStatusPending,
		Enabled:         enabled,
		Remark:          req.Remark,
	}

	if err := s.personRepo.Create(ctx, person); err != nil {
		zap.L().Error("create person failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	if len(req.GroupIDs) > 0 {
		_ = s.personRepo.ReplaceGroups(ctx, person.ID, req.GroupIDs)
	}

	if len(req.TagIDs) > 0 && s.tagRelationRepo != nil {
		_ = s.tagRelationRepo.BatchCreate(ctx, person.ID, req.TagIDs)
	}

	_ = s.taskClient.EnqueueWithID(ctx, personEmbeddingTaskType, map[string]string{"person_id": person.ID}, personEmbeddingTaskType+":"+person.ID)

	person, _ = s.personRepo.FindByID(ctx, person.ID)
	resp := toPersonResponse(person)
	return &resp, nil
}

// Update 更新人员。
func (s *PersonService) Update(ctx context.Context, id string, req dto.PersonUpdateRequest, fileHeader *multipart.FileHeader) (*dto.PersonResponse, error) {
	person, err := s.personRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "")
	}

	if req.PersonName != "" {
		person.PersonName = req.PersonName
	}
	if req.Gender != "" {
		person.Gender = req.Gender
	}
	person.Phone = req.Phone
	person.IDNumber = req.IDNumber
	person.Remark = req.Remark
	if req.Enabled != nil {
		person.Enabled = *req.Enabled
	}
	if req.PersonCode != "" && req.PersonCode != person.PersonCode {
		if exists, _ := s.personRepo.ExistsByPersonCode(ctx, req.PersonCode, id); exists {
			return nil, apperrors.New(apperrors.ErrPersonCodeDuplicate, "")
		}
		person.PersonCode = req.PersonCode
	}

	// 更新图片
	if req.ImageURL != "" || fileHeader != nil {
		imageURL, imageMD5, err := s.processImage(ctx, req.ImageURL, fileHeader, id)
		if err != nil {
			return nil, err
		}
		person.ImageURL = imageURL
		person.ImageMD5 = imageMD5
		person.EmbeddingStatus = model.EmbeddingStatusPending
		person.EmbeddingErrorCode = ""
		person.EmbeddingErrorMessageKey = ""
		person.EmbeddingRetryable = false
		
		_ = s.taskClient.EnqueueWithID(ctx, personEmbeddingTaskType, map[string]string{"person_id": person.ID}, personEmbeddingTaskType+":"+person.ID)
	}

	if err := s.personRepo.Update(ctx, person); err != nil {
		zap.L().Error("update person failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	if req.GroupIDs != nil {
		_ = s.personRepo.ReplaceGroups(ctx, person.ID, req.GroupIDs)
	}

	if req.TagIDs != nil && s.tagRelationRepo != nil {
		_ = s.tagRelationRepo.DeleteByPersonRecordID(ctx, person.ID)
		if len(req.TagIDs) > 0 {
			_ = s.tagRelationRepo.BatchCreate(ctx, person.ID, req.TagIDs)
		}
	}

	person, _ = s.personRepo.FindByID(ctx, person.ID)
	resp := toPersonResponse(person)
	return &resp, nil
}

// Delete 删除人员。
func (s *PersonService) Delete(ctx context.Context, id string) error {
	if _, err := s.personRepo.FindByID(ctx, id); err != nil {
		return apperrors.New(apperrors.ErrNotFound, "")
	}
	// 先清理 Redis 中该 person 的 pending embedding 任务
	_ = s.taskClient.RemovePending(ctx, personEmbeddingTaskType, id)
	return s.personRepo.SoftDelete(ctx, id)
}

// BatchDelete 批量删除人员。
func (s *PersonService) BatchDelete(ctx context.Context, ids []string) error {
	for _, id := range ids {
		_ = s.taskClient.RemovePending(ctx, personEmbeddingTaskType, id)
	}
	return s.personRepo.BatchSoftDelete(ctx, ids)
}

// BatchToggle 批量启用/禁用。
func (s *PersonService) BatchToggle(ctx context.Context, ids []string, enabled bool) error {
	return s.personRepo.BatchToggle(ctx, ids, enabled)
}

// RetryEmbedding 重提特征。
func (s *PersonService) RetryEmbedding(ctx context.Context, id string) error {
	person, err := s.personRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrNotFound, "")
	}
	// 仅阻止正在提取中的任务，允许 active 状态重新提取（算法升级场景）
	if person.EmbeddingStatus == model.EmbeddingStatusExtracting {
		return apperrors.New(apperrors.ErrPersonStatusNoRetry, "")
	}
	if err := s.personRepo.UpdateEmbeddingStatus(ctx, id, model.EmbeddingStatusPending, "", "", false); err != nil {
		return apperrors.New(apperrors.ErrInternal, "")
	}
	return s.taskClient.EnqueueWithID(ctx, personEmbeddingTaskType, map[string]string{"person_id": id}, personEmbeddingTaskType+":"+id)
}

// BatchRetryResult 批量重提特征结果。
type BatchRetryResult struct {
	Success int
	Failed  int
	Errors  map[string]string // personID -> 错误信息
}

// BatchRetryEmbedding 批量重提特征。
// 返回成功数量、失败数量以及按 personID 分类的失败原因，便于前端精准展示。
func (s *PersonService) BatchRetryEmbedding(ctx context.Context, ids []string) (BatchRetryResult, error) {
	// 先批量查询，仅对需要重提的人员执行操作
	persons, err := s.personRepo.FindByIDs(ctx, ids)
	if err != nil {
		return BatchRetryResult{}, apperrors.New(apperrors.ErrInternal, "")
	}

	result := BatchRetryResult{Errors: make(map[string]string)}
	var pendingIDs []string
	for _, p := range persons {
		// 仅阻止正在提取中的任务，允许 active 状态重新提取（算法升级场景）
		if p.EmbeddingStatus == model.EmbeddingStatusExtracting {
			result.Failed++
			result.Errors[p.ID] = apperrors.New(apperrors.ErrPersonStatusNoRetry, "").Error()
			continue
		}
		pendingIDs = append(pendingIDs, p.ID)
	}

	// 批量更新状态为 pending
	if len(pendingIDs) > 0 {
		for _, id := range pendingIDs {
			if err := s.personRepo.UpdateEmbeddingStatus(ctx, id, model.EmbeddingStatusPending, "", "", false); err != nil {
				result.Failed++
				result.Errors[id] = apperrors.New(apperrors.ErrInternal, "").Error()
				continue
			}
			// 逐条投递任务（Asynq Enqueue 本身开销很小）
			if err := s.taskClient.EnqueueWithID(ctx, personEmbeddingTaskType, map[string]string{"person_id": id}, personEmbeddingTaskType+":"+id); err != nil {
				result.Failed++
				result.Errors[id] = err.Error()
				continue
			}
			result.Success++
		}
	}
	return result, nil
}

// ExportExcel 导出人员 Excel 并内嵌人脸图。
// 限制单次导出的最大记录数，防止 OOM。
func (s *PersonService) ExportExcel(ctx context.Context, req dto.PersonListRequest, writer io.Writer) error {
	items, err := s.personRepo.ListForExport(ctx, req, exportBatchSize)
	if err != nil {
		return apperrors.New(apperrors.ErrInternal, "")
	}

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Persons"
	f.SetSheetName("Sheet1", sheetName)

	// 设置表头
	headers := []string{"图片", "人员姓名", "人员编号", "性别", "手机号", "特征状态", "启用状态", "创建时间"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
	}

	// 设置行高（为了显示图片清晰一些）
	for i := range items {
		_ = f.SetRowHeight(sheetName, i+2, 60)
	}
	_ = f.SetColWidth(sheetName, "A", "A", 12)

	for i, item := range items {
		rowIdx := i + 2
		// 写入文字数据
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", rowIdx), item.PersonName)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", rowIdx), item.PersonCode)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", rowIdx), item.Gender)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", rowIdx), item.Phone)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", rowIdx), item.EmbeddingStatus)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("%v", item.Enabled))
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", rowIdx), item.CreatedAt.Format(time.RFC3339))

		// 解析 ImageURL 为存储路径并读取图片（带 fallback 机制）
		storagePath := s.getStoragePath(item.ImageURL)
		var imgData []byte
		if s.storage != nil && item.ImageURL != "" {
			rc, err := s.storage.Get(storagePath)
			if err != nil {
				// fallback: 尝试直接去掉前导斜杠的路径
				fallbackPath := strings.TrimPrefix(item.ImageURL, "/")
				rc, err = s.storage.Get(fallbackPath)
			}
			if err == nil {
				imgData, _ = io.ReadAll(rc)
				rc.Close()
			}
			if len(imgData) == 0 {
				zap.L().Warn("人员图片加载失败，导出仅包含文本",
					zap.String("person_id", item.ID),
					zap.String("image_url", item.ImageURL),
					zap.String("storage_path", storagePath),
					zap.Error(err),
				)
			}
		}
		if len(imgData) > 0 {
			hintExt := filepath.Ext(storagePath)
			if hintExt == "" {
				hintExt = ".jpg"
			}
			preparedData, ext, err := prepareExportImage(imgData, hintExt)
			if err != nil {
				zap.L().Warn("人员图片解码/转换失败，跳过该图片",
					zap.String("person_id", item.ID),
					zap.String("image_url", item.ImageURL),
					zap.String("storage_path", storagePath),
					zap.Error(err),
				)
			} else {
				if err := f.AddPictureFromBytes(sheetName, fmt.Sprintf("A%d", rowIdx), &excelize.Picture{
					Extension: ext,
					File:      preparedData,
					Format: &excelize.GraphicOptions{
						AutoFit:         true,
						LockAspectRatio: true,
						OffsetX:         5,
						OffsetY:         5,
					},
				}); err != nil {
					zap.L().Warn("导出内嵌图片失败，跳过该图片",
						zap.String("person_id", item.ID),
						zap.String("image_url", item.ImageURL),
						zap.String("extension", ext),
						zap.Error(err),
					)
				}
			}
		}
	}

	if err := f.Write(writer); err != nil {
		zap.L().Error("write excel failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}
	return nil
}

// GetImage 获取图片文件流。
func (s *PersonService) GetImage(ctx context.Context, path string) (io.ReadCloser, error) {
	return s.storage.Get(path)
}

// ListGroups 查询分组列表。
func (s *PersonService) ListGroups(ctx context.Context) ([]dto.PersonGroupResponse, error) {
	groups, err := s.groupRepo.List(ctx)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	// 批量查询所有分组的人数，避免 N+1 查询
	counts, err := s.groupRepo.BatchCountPersons(ctx)
	if err != nil {
		// 人数查询失败不阻塞主流程，降级为逐个查询
		counts = make(map[string]int64)
		for _, g := range groups {
			c, _ := s.groupRepo.CountPersons(ctx, g.ID)
			counts[g.ID] = c
		}
	}
	result := make([]dto.PersonGroupResponse, len(groups))
	for i, g := range groups {
		result[i] = toGroupResponse(&g, counts[g.ID])
	}
	return result, nil
}

// CreateGroup 创建分组。
func (s *PersonService) CreateGroup(ctx context.Context, req dto.PersonGroupRequest) (*dto.PersonGroupResponse, error) {
	group := &model.PersonGroup{GroupName: req.GroupName, Description: req.Description, ParentID: req.ParentID, SortOrder: req.SortOrder}
	if err := s.groupRepo.Create(ctx, group); err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	resp := toGroupResponse(group, 0)
	return &resp, nil
}

// UpdateGroup 更新分组。
func (s *PersonService) UpdateGroup(ctx context.Context, id string, req dto.PersonGroupRequest) (*dto.PersonGroupResponse, error) {
	group, err := s.groupRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "")
	}
	group.GroupName = req.GroupName
	group.Description = req.Description
	group.ParentID = req.ParentID
	group.SortOrder = req.SortOrder
	if err := s.groupRepo.Update(ctx, group); err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	resp := toGroupResponse(group, 0)
	return &resp, nil
}

// DeleteGroup 删除分组。
func (s *PersonService) DeleteGroup(ctx context.Context, id string) error {
	if _, err := s.groupRepo.FindByID(ctx, id); err != nil {
		return apperrors.New(apperrors.ErrNotFound, "")
	}
	return s.groupRepo.Delete(ctx, id)
}

// ListTags 查询标签列表。
func (s *PersonService) ListTags(ctx context.Context) ([]dto.PersonTagResponse, error) {
	tags, err := s.tagRepo.List(ctx)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	counts, err := s.tagRepo.BatchCountPersons(ctx)
	if err != nil {
		counts = make(map[string]int64)
		for _, t := range tags {
			c, _ := s.tagRepo.CountPersons(ctx, t.ID)
			counts[t.ID] = c
		}
	}
	result := make([]dto.PersonTagResponse, len(tags))
	for i, t := range tags {
		result[i] = toTagResponse(&t, counts[t.ID])
	}
	return result, nil
}

// CreateTag 创建标签。
func (s *PersonService) CreateTag(ctx context.Context, req dto.PersonTagCreateRequest) (*dto.PersonTagResponse, error) {
	tag := &model.PersonTag{TagName: req.TagName, Color: req.Color, SortOrder: req.SortOrder}
	if tag.Color == "" {
		tag.Color = "#1890ff"
	}
	if err := s.tagRepo.Create(ctx, tag); err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	resp := toTagResponse(tag, 0)
	return &resp, nil
}

// UpdateTag 更新标签。
func (s *PersonService) UpdateTag(ctx context.Context, id string, req dto.PersonTagUpdateRequest) (*dto.PersonTagResponse, error) {
	tag, err := s.tagRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "")
	}
	if req.TagName != "" {
		tag.TagName = req.TagName
	}
	if req.Color != "" {
		tag.Color = req.Color
	}
	tag.SortOrder = req.SortOrder
	if err := s.tagRepo.Update(ctx, tag); err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	resp := toTagResponse(tag, 0)
	return &resp, nil
}

// DeleteTag 删除标签。
func (s *PersonService) DeleteTag(ctx context.Context, id string) error {
	if _, err := s.tagRepo.FindByID(ctx, id); err != nil {
		return apperrors.New(apperrors.ErrNotFound, "")
	}
	return s.tagRepo.Delete(ctx, id)
}

// ListImportTasks 查询导入任务列表。
func (s *PersonService) ListImportTasks(ctx context.Context, req dto.PageRequest) ([]dto.PersonImportTaskResponse, int64, error) {
	items, total, err := s.importTaskRepo.List(ctx, req)
	if err != nil {
		return nil, 0, apperrors.New(apperrors.ErrInternal, "")
	}
	result := make([]dto.PersonImportTaskResponse, len(items))
	for i := range items {
		result[i] = toImportTaskResponse(&items[i])
	}
	return result, total, nil
}

// GetImportTask 查询导入任务详情。
func (s *PersonService) GetImportTask(ctx context.Context, id string) (*dto.PersonImportTaskResponse, error) {
	item, err := s.importTaskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrNotFound, "")
	}
	resp := toImportTaskResponse(item)
	return &resp, nil
}

// isSupportedArchive 检查文件名是否为支持的压缩包格式。
func isSupportedArchive(name string) bool {
	name = strings.ToLower(name)
	return strings.HasSuffix(name, ".zip") ||
		strings.HasSuffix(name, ".tar.gz") ||
		strings.HasSuffix(name, ".tgz") ||
		strings.HasSuffix(name, ".tar.bz2") ||
		strings.HasSuffix(name, ".tbz2")
}

// enqueueAndRespond 创建导入任务记录并投递异步任务，返回统一的响应 DTO。
// 投递失败时自动标记任务状态为失败。
func (s *PersonService) enqueueAndRespond(ctx context.Context, task *model.ImportTask, payload map[string]interface{}) (*dto.PersonImportTaskResponse, error) {
	if err := s.importTaskRepo.Create(ctx, task); err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	if err := s.taskClient.Enqueue(ctx, personImportTaskType, payload); err != nil {
		zap.L().Error("enqueue import task failed", zap.String("task_id", task.ID), zap.Error(err))
		task.Status = model.ImportTaskStatusFailed
		_ = s.importTaskRepo.Update(ctx, task)
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	resp := toImportTaskResponse(task)
	return &resp, nil
}

// CreateImportTask 创建导入任务。
func (s *PersonService) CreateImportTask(ctx context.Context, fileHeader *multipart.FileHeader, overwrite bool) (*dto.PersonImportTaskResponse, error) {
	if fileHeader == nil {
		return nil, apperrors.New(apperrors.ErrArchiveRequired, "")
	}
	if fileHeader.Size > personImportMaxSize {
		return nil, apperrors.New(apperrors.ErrFileTooLarge, "")
	}
	if !isSupportedArchive(fileHeader.Filename) {
		return nil, apperrors.New(apperrors.ErrArchiveUnsupported, "")
	}

	// 保存压缩包到 Storage
	file, err := fileHeader.Open()
	if err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "")
	}
	defer file.Close()

	storagePath := "imports/persons/" + uuid.New().String() + filepath.Ext(fileHeader.Filename)
	if _, err = s.storage.Save(file, storagePath); err != nil {
		zap.L().Error("save import archive failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	task := &model.ImportTask{
		TaskType: model.ImportTaskTypePerson,
		FileName: fileHeader.Filename,
		FileURL:  storagePath,
		Status:   model.ImportTaskStatusPending,
	}
	payload := map[string]interface{}{"task_id": task.ID, "file_url": storagePath, "overwrite": overwrite}
	return s.enqueueAndRespond(ctx, task, payload)
}

// CreateImportTaskByURL 通过已上传的文件 URL 创建导入任务（分片上传后调用）。
func (s *PersonService) CreateImportTaskByURL(ctx context.Context, fileURL string, overwrite bool) (*dto.PersonImportTaskResponse, error) {
	if fileURL == "" {
		return nil, apperrors.New(apperrors.ErrArchiveRequired, "")
	}
	if !isSupportedArchive(fileURL) {
		return nil, apperrors.New(apperrors.ErrArchiveUnsupported, "")
	}

	task := &model.ImportTask{
		TaskType: model.ImportTaskTypePerson,
		FileName: filepath.Base(fileURL),
		FileURL:  fileURL,
		Status:   model.ImportTaskStatusPending,
	}
	payload := map[string]interface{}{"task_id": task.ID, "file_url": fileURL, "overwrite": overwrite}
	return s.enqueueAndRespond(ctx, task, payload)
}

// SearchByFace 以图搜人：上传人脸图片实时提取 embedding 并在人员库中 1:N 搜索。
func (s *PersonService) SearchByFace(ctx context.Context, fileHeader *multipart.FileHeader, topK int, threshold float64) ([]dto.PersonSearchByFaceResponse, error) {
	if fileHeader == nil {
		return nil, apperrors.New(apperrors.ErrPersonImageRequired, "")
	}
	if fileHeader.Size > personImageMaxSize {
		return nil, apperrors.New(apperrors.ErrFileTooLarge, "")
	}
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !imageExtSet[ext] {
		return nil, apperrors.New(apperrors.ErrFileInvalidType, "")
	}

	// 读取图片 bytes
	file, err := fileHeader.Open()
	if err != nil {
		return nil, apperrors.New(apperrors.ErrBadRequest, "")
	}
	defer file.Close()

	imageBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	if len(imageBytes) == 0 {
		return nil, apperrors.New(apperrors.ErrPersonImageRequired, "")
	}

	// 解析最新 face_recognition 算法包运行时
	if s.algorithmPackageRepo == nil || s.engine == nil {
		zap.L().Error("face recognition engine or algorithm package repo not configured")
		return nil, apperrors.New(apperrors.ErrEngineNotReady, "")
	}
	algoVersion, soPath, algoParamsJSON, err := s.resolveFaceRecognitionRuntime(ctx)
	if err != nil {
		zap.L().Error("resolve face recognition runtime failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrFaceExtractFailed, "")
	}

	// 调用引擎提取特征
	result, err := s.engine.ExtractFaceEmbedding(ctx, "face_recognition", algoVersion, soPath, algoParamsJSON, imageBytes)
	if err != nil {
		zap.L().Error("extract face embedding failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrFaceExtractFailed, "")
	}
	if len(result.Embedding) != 512 {
		zap.L().Error("unexpected embedding dimension", zap.Int("dim", len(result.Embedding)))
		return nil, apperrors.New(apperrors.ErrFaceExtractFailed, "")
	}

	// 转为 pgvector 执行相似搜索
	queryVector := pgvector.NewVector(result.Embedding)
	records, err := s.embeddingRepo.SearchByFace(ctx, queryVector, topK, threshold)
	if err != nil {
		zap.L().Error("face search by embedding failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	if len(records) == 0 {
		return []dto.PersonSearchByFaceResponse{}, nil
	}

	// 拼装响应
	resp := make([]dto.PersonSearchByFaceResponse, len(records))
	for i, rec := range records {
		// 余弦相似度缩放到 [0, 1] 范围: (cos + 1) / 2
		similarity := (2.0 - rec.Distance) / 2.0
		if similarity < 0 {
			similarity = 0
		}
		if similarity > 1 {
			similarity = 1
		}
		resp[i] = dto.PersonSearchByFaceResponse{
			Person:     toPersonResponse(&rec.Person),
			Similarity: similarity,
			Distance:   rec.Distance,
		}
	}
	return resp, nil
}

// resolveFaceRecognitionRuntime 查找最新通过的 face_recognition 算法包，返回 version、soPath、algoParamsJSON。
func (s *PersonService) resolveFaceRecognitionRuntime(ctx context.Context) (string, string, string, error) {
	algoPackage, err := s.algorithmPackageRepo.FindLatestPassedByAlgorithm(ctx, "face_recognition")
	if err != nil {
		return "", "", "", fmt.Errorf("find face_recognition algorithm package: %w", err)
	}
	soPath, err := ResolveRuntimeSoPath(algoPackage)
	if err != nil {
		return "", "", "", fmt.Errorf("resolve face_recognition runtime so: %w", err)
	}
	return algoPackage.Version, soPath, "", nil
}

// Helper functions

func toPersonResponse(p *model.Person) dto.PersonResponse {
	resp := dto.PersonResponse{
		ID:                       p.ID,
		PersonCode:               p.PersonCode,
		PersonName:               p.PersonName,
		Gender:                   p.Gender,
		Phone:                    p.Phone,
		IDNumber:                 p.IDNumber,
		ImageURL:                 p.ImageURL,
		ImageMD5:                 p.ImageMD5,
		FaceQualityScore:         p.FaceQualityScore,
		EmbeddingStatus:          p.EmbeddingStatus,
		EmbeddingErrorCode:       p.EmbeddingErrorCode,
		EmbeddingErrorMessageKey: p.EmbeddingErrorMessageKey,
		EmbeddingRetryable:       p.EmbeddingRetryable,
		Enabled:                  p.Enabled,
		Remark:                   p.Remark,
		CreatedAt:                p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:                p.UpdatedAt.Format(time.RFC3339),
	}
	for _, g := range p.Groups {
		resp.Groups = append(resp.Groups, dto.PersonGroupResponse{ID: g.ID, GroupName: g.GroupName, Description: g.Description, ParentID: safeStr(g.ParentID), SortOrder: g.SortOrder})
	}
	return resp
}

// toGroupResponse 将分组模型转换为响应 DTO。
func toGroupResponse(g *model.PersonGroup, personCount int64) dto.PersonGroupResponse {
	return dto.PersonGroupResponse{
		ID: g.ID, GroupName: g.GroupName, Description: g.Description,
		ParentID: safeStr(g.ParentID), SortOrder: g.SortOrder, PersonCount: personCount,
		CreatedAt: g.CreatedAt.Format(time.RFC3339), UpdatedAt: g.UpdatedAt.Format(time.RFC3339),
	}
}

// toTagResponse 将标签模型转换为响应 DTO。
func toTagResponse(t *model.PersonTag, personCount int64) dto.PersonTagResponse {
	return dto.PersonTagResponse{
		ID: t.ID, TagName: t.TagName, Color: t.Color,
		SortOrder: t.SortOrder, PersonCount: personCount,
		CreatedAt: t.CreatedAt.Format(time.RFC3339), UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
	}
}

// toImportTaskResponse 将导入任务模型转换为响应 DTO。
func toImportTaskResponse(t *model.ImportTask) dto.PersonImportTaskResponse {
	return dto.PersonImportTaskResponse{
		ID: t.ID, TaskType: t.TaskType, FileName: t.FileName, FileURL: t.FileURL,
		TotalRows: t.TotalRows, SuccessRows: t.SuccessRows, FailedRows: t.FailedRows,
		FailDetailURL: t.FailDetailURL, Status: t.Status,
		CreatedAt: t.CreatedAt.Format(time.RFC3339), UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
	}
}

func safeStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// prepareExportImage 解码图片字节并根据实际格式确保扩展名是 excelize 支持的（jpg/png/gif）。
// 对于 webp 等非支持格式，自动重新编码为 PNG。
// 返回嵌入安全的图片字节和对应的扩展名。
func prepareExportImage(data []byte, hintExt string) ([]byte, string, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", fmt.Errorf("image decode: %w", err)
	}

	// 根据解码格式映射到 excelize 支持的扩展名
	var safeExt string
	switch format {
	case "png":
		safeExt = ".png"
	case "jpeg":
		safeExt = ".jpg"
	case "gif":
		safeExt = ".gif"
	default:
		// webp 等不被 excelize 支持的格式 -> 转 PNG
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", fmt.Errorf("re-encode %s as png: %w", format, err)
		}
		return buf.Bytes(), ".png", nil
	}

	// 已是兼容格式，直接返回原始字节和标准扩展名
	return data, safeExt, nil
}
