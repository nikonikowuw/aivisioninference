// Package task 提供人员特征提取的异步任务处理器。
package task

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"

	"github.com/hibiken/asynq"
	"github.com/pgvector/pgvector-go"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/repository"
)

const (
	TypePersonEmbedding = "person:embedding"
	TypePersonImport    = "person:import"
)

// PersonEmbeddingPayload 特征提取任务载荷。
type PersonEmbeddingPayload struct {
	PersonID string `json:"person_id"`
}

// PersonEmbeddingHandler 处理人员特征提取任务。
type PersonEmbeddingHandler struct {
	personRepo    *repository.PersonRepository
	embeddingRepo *repository.PersonEmbeddingRepository
}

// NewPersonEmbeddingHandler 创建特征提取任务处理器。
func NewPersonEmbeddingHandler(personRepo *repository.PersonRepository, embeddingRepo *repository.PersonEmbeddingRepository) *PersonEmbeddingHandler {
	return &PersonEmbeddingHandler{personRepo: personRepo, embeddingRepo: embeddingRepo}
}

// RegisterHandlers 注册处理器。
func (h *PersonEmbeddingHandler) RegisterHandlers(mux *asynq.ServeMux) {
	mux.HandleFunc(TypePersonEmbedding, h.handleEmbedding)
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

	// 提取特征向量
	// TODO: Replace this placeholder SHA256 logic with real InsightFace model call
	// Currently using deterministic placeholder vectors for development.
	vec, err := h.extractEmbedding(person)
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

// extractEmbedding 提取 512 维特征向量。
// WARNING: This is a deterministic placeholder implementation using SHA256.
// It MUST be replaced with real InsightFace model call before production deployment.
// Do NOT rely on this for actual face recognition / similarity search.
func (h *PersonEmbeddingHandler) extractEmbedding(person *model.Person) ([]float32, error) {
	hash := sha256.Sum256([]byte(person.ImageURL + person.PersonName))
	vec := make([]float32, 512)
	for i := 0; i < 512; i++ {
		start := (i * 4) % 28
		bits := binary.LittleEndian.Uint32(hash[start : start+4])
		// 生成 [-1, 1] 范围的浮点数
		vec[i] = (float32(bits)/float32(math.MaxUint32))*2 - 1
	}
	// 归一化
	var norm float32
	for _, v := range vec {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))
	if norm > 0 {
		for i := range vec {
			vec[i] /= norm
		}
	}
	return vec, nil
}
