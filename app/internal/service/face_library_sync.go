package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/niko-admin/niko-admin/internal/repository"
	"go.uber.org/zap"
)

const defaultFaceRecognitionAlgorithm = "face_recognition"

// FaceLibrarySyncService 从 Postgres 构建完整人脸库快照并下发给 Engine。
type FaceLibrarySyncService struct {
	embeddingRepo        *repository.PersonEmbeddingRepository
	algorithmPackageRepo *repository.AlgorithmPackageRepository
	scheduler            *FaceEmbeddingScheduler
	engine               EngineClient
}

func NewFaceLibrarySyncService(embeddingRepo *repository.PersonEmbeddingRepository, algorithmPackageRepo *repository.AlgorithmPackageRepository, scheduler *FaceEmbeddingScheduler, engine EngineClient) *FaceLibrarySyncService {
	return &FaceLibrarySyncService{embeddingRepo: embeddingRepo, algorithmPackageRepo: algorithmPackageRepo, scheduler: scheduler, engine: engine}
}

func (s *FaceLibrarySyncService) Sync(ctx context.Context, algoName string) error {
	if s == nil || s.embeddingRepo == nil || s.engine == nil || s.algorithmPackageRepo == nil || s.scheduler == nil {
		return nil
	}
	if algoName == "" {
		algoName = defaultFaceRecognitionAlgorithm
	}
	algoPackage, err := s.algorithmPackageRepo.FindLatestPassedByAlgorithm(ctx, algoName)
	if err != nil {
		return fmt.Errorf("find face library algorithm package: %w", err)
	}
	nodeIDs, err := s.scheduler.ListReadyFaceLibraryNodeIDs(ctx, algoPackage.ID)
	if err != nil {
		return fmt.Errorf("list face library target nodes: %w", err)
	}
	if len(nodeIDs) == 0 {
		return nil
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

	var failed []string
	for _, nodeID := range nodeIDs {
		if err := s.engine.UpdateFaceLibrary(ctx, nodeID, algoName, payload); err != nil {
			failed = append(failed, fmt.Sprintf("%s:%v", nodeID, err))
		}
	}
	zap.L().Info("face library synced to engine",
		zap.String("algo_name", algoName),
		zap.String("version", snapshot.Version),
		zap.Int("target_nodes", len(nodeIDs)),
		zap.Int("items", len(items)),
	)
	if len(failed) > 0 {
		return fmt.Errorf("face library sync partially failed: %s", strings.Join(failed, "; "))
	}
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
