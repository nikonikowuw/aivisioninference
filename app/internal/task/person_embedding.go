// Package task 提供人员特征提取的异步任务处理器。
package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/hibiken/asynq"
	"github.com/pgvector/pgvector-go"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/internal/service"
	"github.com/niko-admin/niko-admin/pkg/storage"
)

const (
	TypePersonEmbedding        = "person:embedding"
	TypePersonEmbeddingRebuild = "person:embedding:rebuild"
	TypePersonImport           = "person:import"
)

// PersonEmbeddingPayload 特征提取任务载荷。
type PersonEmbeddingPayload struct {
	PersonID string `json:"person_id"`
}

// PersonEmbeddingRebuildPayload 全量重提取任务载荷。
type PersonEmbeddingRebuildPayload struct {
	AlgoName    string `json:"algo_name"`
	AlgoVersion string `json:"algo_version"`
}

// PersonEmbeddingHandler 处理人员特征提取任务。
type PersonEmbeddingHandler struct {
	personRepo           *repository.PersonRepository
	embeddingRepo        *repository.PersonEmbeddingRepository
	algorithmPackageRepo *repository.AlgorithmPackageRepository
	faceSync             *service.FaceLibrarySyncService
	scheduler            *service.FaceEmbeddingScheduler
	engine               service.EngineClient
	storage              storage.Storage
}

// NewPersonEmbeddingHandler 创建特征提取任务处理器。
func NewPersonEmbeddingHandler(personRepo *repository.PersonRepository, embeddingRepo *repository.PersonEmbeddingRepository, algorithmPackageRepo *repository.AlgorithmPackageRepository, faceSync *service.FaceLibrarySyncService, scheduler *service.FaceEmbeddingScheduler, engine service.EngineClient, storage storage.Storage) *PersonEmbeddingHandler {
	return &PersonEmbeddingHandler{
		personRepo:           personRepo,
		embeddingRepo:        embeddingRepo,
		algorithmPackageRepo: algorithmPackageRepo,
		faceSync:             faceSync,
		scheduler:            scheduler,
		engine:               engine,
		storage:              storage,
	}
}

// RegisterHandlers 注册处理器。
func (h *PersonEmbeddingHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypePersonEmbedding, h.handleEmbedding)
	mux.HandleFunc(TypePersonEmbeddingRebuild, h.handleEmbeddingRebuild)
}

// handleEmbedding 处理特征提取任务。
func (h *PersonEmbeddingHandler) handleEmbedding(ctx context.Context, t *asynq.Task) error {
	var payload PersonEmbeddingPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("unmarshal payload: %w", err)
	}
	if payload.PersonID == "" {
		return fmt.Errorf("person_id is required")
	}

	person, err := h.personRepo.FindByID(ctx, payload.PersonID)
	if err != nil {
		return fmt.Errorf("find person: %w", err)
	}

	// 更新状态为提取中
	if err := h.personRepo.UpdateEmbeddingStatus(ctx, person.ID, model.EmbeddingStatusExtracting, "", "", false); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	vec, err := h.extractEmbedding(ctx, person)
	if err != nil {
		zap.L().Error("embedding extraction failed", zap.String("person_id", person.ID), zap.Error(err))
		_ = h.personRepo.UpdateEmbeddingStatus(ctx, person.ID, model.EmbeddingStatusFailed, "EXTRACTION_FAILED", "person.error.embeddingExtract", true)
		return fmt.Errorf("extract embedding: %w", err)
	}

	// 删除旧 Embedding
	_ = h.embeddingRepo.DeleteByPersonRecordID(ctx, person.ID)

	// 写入新 Embedding
	embedding := &model.PersonEmbedding{
		PersonRecordID: person.ID,
		Embedding:      pgvector.NewVector(vec),
		Version:        1,
	}
	if err := h.embeddingRepo.Create(ctx, embedding); err != nil {
		zap.L().Error("write embedding failed", zap.String("person_id", person.ID), zap.Error(err))
		_ = h.personRepo.UpdateEmbeddingStatus(ctx, person.ID, model.EmbeddingStatusFailed, "DB_WRITE_ERROR", "person.error.embeddingWrite", true)
		return fmt.Errorf("create embedding: %w", err)
	}

	// 更新状态为 active
	if err := h.personRepo.UpdateEmbeddingStatus(ctx, person.ID, model.EmbeddingStatusActive, "", "", false); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	zap.L().Info("person embedding extracted", zap.String("person_id", person.ID), zap.String("person_name", person.PersonName))
	return nil
}

func (h *PersonEmbeddingHandler) handleEmbeddingRebuild(ctx context.Context, t *asynq.Task) error {
	var payload PersonEmbeddingRebuildPayload
	if len(t.Payload()) > 0 {
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return fmt.Errorf("unmarshal rebuild payload: %w", err)
		}
	}
	if payload.AlgoName == "" {
		payload.AlgoName = "face_recognition"
	}

	persons, err := h.personRepo.ListEnabledForEmbedding(ctx, 0)
	if err != nil {
		return fmt.Errorf("list enabled persons: %w", err)
	}

	var failed int
	for i := range persons {
		if err := h.rebuildPersonEmbedding(ctx, &persons[i]); err != nil {
			failed++
			zap.L().Error("person embedding rebuild failed",
				zap.String("person_id", persons[i].ID),
				zap.String("algo_name", payload.AlgoName),
				zap.String("algo_version", payload.AlgoVersion),
				zap.Error(err),
			)
		}
	}

	if h.faceSync != nil {
		if err := h.faceSync.Sync(ctx, payload.AlgoName); err != nil {
			return fmt.Errorf("sync face library: %w", err)
		}
	}

	zap.L().Info("person embedding rebuild completed",
		zap.String("algo_name", payload.AlgoName),
		zap.String("algo_version", payload.AlgoVersion),
		zap.Int("total", len(persons)),
		zap.Int("failed", failed),
	)
	if failed > 0 {
		return fmt.Errorf("person embedding rebuild partially failed: %d/%d", failed, len(persons))
	}
	return nil
}

func (h *PersonEmbeddingHandler) rebuildPersonEmbedding(ctx context.Context, person *model.Person) error {
	if err := h.personRepo.UpdateEmbeddingStatus(ctx, person.ID, model.EmbeddingStatusExtracting, "", "", false); err != nil {
		return fmt.Errorf("update status extracting: %w", err)
	}

	vec, err := h.extractEmbedding(ctx, person)
	if err != nil {
		_ = h.personRepo.UpdateEmbeddingStatus(ctx, person.ID, model.EmbeddingStatusFailed, "EXTRACTION_FAILED", "person.error.embeddingExtract", true)
		return fmt.Errorf("extract embedding: %w", err)
	}

	_ = h.embeddingRepo.DeleteByPersonRecordID(ctx, person.ID)
	embedding := &model.PersonEmbedding{
		PersonRecordID: person.ID,
		Embedding:      pgvector.NewVector(vec),
		Version:        1,
	}
	if err := h.embeddingRepo.Create(ctx, embedding); err != nil {
		_ = h.personRepo.UpdateEmbeddingStatus(ctx, person.ID, model.EmbeddingStatusFailed, "DB_WRITE_ERROR", "person.error.embeddingWrite", true)
		return fmt.Errorf("create embedding: %w", err)
	}
	if err := h.personRepo.UpdateEmbeddingStatus(ctx, person.ID, model.EmbeddingStatusActive, "", "", false); err != nil {
		return fmt.Errorf("update status active: %w", err)
	}
	return nil
}

func (h *PersonEmbeddingHandler) extractEmbedding(ctx context.Context, person *model.Person) ([]float32, error) {
	if h.engine == nil || h.scheduler == nil {
		return nil, fmt.Errorf("engine client is not configured")
	}
	imageBytes, err := h.readPersonImage(person.ImageURL)
	if err != nil {
		return nil, err
	}

	if h.algorithmPackageRepo == nil {
		return nil, fmt.Errorf("algorithm package repository is not configured")
	}
	algoPackage, err := h.algorithmPackageRepo.FindLatestPassedByAlgorithm(ctx, "face_recognition")
	if err != nil {
		return nil, fmt.Errorf("find face_recognition algorithm package: %w", err)
	}

	if _, err := h.scheduler.EnsureReadyNode(ctx, "face_recognition", algoPackage.Version, algoPackage.ID, h.engine); err != nil {
		return nil, err
	}

	excludedNodeIDs := make([]string, 0, service.MaxEmbeddingAlternateRetries()+1)
	var lastErr error
	for attempt := 1; attempt <= service.MaxEmbeddingAlternateRetries()+1; attempt++ {
		reservation, err := h.scheduler.AcquireExcept(ctx, algoPackage.ID, excludedNodeIDs)
		if err != nil {
			lastErr = err
			break
		}

		result, extractErr := h.engine.ExtractFaceEmbedding(ctx, reservation.NodeID, "face_recognition", algoPackage.Version, imageBytes)
		if releaseErr := h.scheduler.Release(ctx, reservation); releaseErr != nil {
			zap.L().Warn("release embedding reservation failed",
				zap.String("person_id", person.ID),
				zap.String("node_id", reservation.NodeID),
				zap.String("reservation_id", reservation.ReservationID),
				zap.Error(releaseErr),
			)
		}

		if extractErr == nil {
			if len(result.Embedding) != 512 {
				return nil, fmt.Errorf("unexpected embedding dimension: %d", len(result.Embedding))
			}
			return result.Embedding, nil
		}

		lastErr = extractErr
		if !isRetryableEmbeddingError(extractErr, result.ErrorCode) {
			break
		}
		excludedNodeIDs = append(excludedNodeIDs, reservation.NodeID)
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("embedding extraction failed without result")
}

func (h *PersonEmbeddingHandler) readPersonImage(imageURL string) ([]byte, error) {
	if h.storage == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	storagePath := h.personImageStoragePath(imageURL)
	rc, err := h.storage.Get(storagePath)
	if err != nil {
		fallback := strings.TrimPrefix(imageURL, "/")
		if fallback != storagePath {
			rc, err = h.storage.Get(fallback)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("read person image %q: %w", imageURL, err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read person image bytes: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("person image is empty")
	}
	return data, nil
}

func (h *PersonEmbeddingHandler) personImageStoragePath(imageURL string) string {
	baseURL := ""
	if h.storage != nil {
		baseURL = h.storage.GetURL("")
	}
	return service.ExtractPersonImageStoragePath(imageURL, baseURL)
}

func isRetryableEmbeddingError(err error, errorCode string) bool {
	if err == nil {
		return false
	}
	switch errorCode {
	case "ENGINE_TIMEOUT", "NODE_OFFLINE", "ALGO_NOT_READY", "ALGO_WARMUP_TIMEOUT":
		return true
	case "INVALID_IMAGE", "INVALID_IMAGE_BASE64", "MISSING_IMAGE":
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "timeout") || strings.Contains(msg, "ALGO_NOT_READY")
}
