package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/niko-admin/niko-admin/internal/repository"
	"go.uber.org/zap"
)

const defaultFaceRecognitionAlgorithm = "face_recognition"

// FaceLibrarySyncService 从 Postgres 构建完整人脸库快照并下发给 Engine。
type FaceLibrarySyncService struct {
	embeddingRepo *repository.PersonEmbeddingRepository
	engine        EngineClient
}

func NewFaceLibrarySyncService(embeddingRepo *repository.PersonEmbeddingRepository, engine EngineClient) *FaceLibrarySyncService {
	return &FaceLibrarySyncService{embeddingRepo: embeddingRepo, engine: engine}
}

func (s *FaceLibrarySyncService) Sync(ctx context.Context, algoName string) error {
	if s == nil || s.embeddingRepo == nil || s.engine == nil {
		return nil
	}
	if algoName == "" {
		algoName = defaultFaceRecognitionAlgorithm
	}

	records, err := s.embeddingRepo.ListActiveFaceLibrary(ctx, 0)
	if err != nil {
		return fmt.Errorf("list active face library: %w", err)
	}

	items := make([]faceLibraryItem, 0, len(records))
	for _, record := range records {
		embedding := record.Embedding.Slice()
		if len(embedding) == 0 {
			continue
		}
		items = append(items, faceLibraryItem{
			ID:        record.PersonID,
			Name:      record.PersonName,
			Embedding: embedding,
		})
	}

	snapshot := faceLibrarySnapshot{
		Version: fmt.Sprintf("face_lib_%d", time.Now().UnixNano()),
		Items:   items,
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal face library snapshot: %w", err)
	}

	if err := s.engine.UpdateFaceLibrary(ctx, algoName, payload); err != nil {
		return err
	}
	zap.L().Info("face library synced to engine",
		zap.String("algo_name", algoName),
		zap.String("version", snapshot.Version),
		zap.Int("items", len(items)),
	)
	return nil
}

type faceLibrarySnapshot struct {
	Version string            `json:"version"`
	Items   []faceLibraryItem `json:"items"`
}

type faceLibraryItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	Embedding []float32 `json:"embedding"`
}
