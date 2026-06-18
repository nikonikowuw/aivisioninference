package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/niko-admin/niko-admin/internal/repository"
	"go.uber.org/zap"
)

const (
	defaultFaceRecognitionAlgorithm = "face_recognition"
	faceLibrarySnapshotPageSize     = 1000
	maxFaceLibrarySyncConcurrency   = 16
)

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

	version := fmt.Sprintf("face_lib_%d", time.Now().UnixNano())
	payload, itemCount, err := s.buildFaceLibrarySnapshotPayload(ctx, version)
	if err != nil {
		return err
	}

	failed := s.updateFaceLibraryOnNodes(ctx, nodeIDs, algoName, payload)
	zap.L().Info("face library synced to engine",
		zap.String("algo_name", algoName),
		zap.String("version", version),
		zap.Int("target_nodes", len(nodeIDs)),
		zap.Int("items", itemCount),
		zap.Int("concurrency", faceLibrarySyncConcurrency(len(nodeIDs))),
		zap.Int("failed_nodes", len(failed)),
	)
	if len(failed) > 0 {
		return fmt.Errorf("face library sync partially failed: %s", strings.Join(failed, "; "))
	}
	return nil
}

func (s *FaceLibrarySyncService) buildFaceLibrarySnapshotPayload(ctx context.Context, version string) ([]byte, int, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)

	buf.WriteString(`{"version":`)
	if err := encoder.Encode(version); err != nil {
		return nil, 0, fmt.Errorf("marshal face library version: %w", err)
	}
	buf.WriteString(`,"items":[`)

	cursor := repository.FaceLibraryCursor{}
	itemCount := 0
	first := true
	for {
		if err := ctx.Err(); err != nil {
			return nil, 0, fmt.Errorf("build face library snapshot canceled: %w", err)
		}
		records, nextCursor, err := s.embeddingRepo.ListActiveFaceLibraryPage(ctx, cursor, faceLibrarySnapshotPageSize)
		if err != nil {
			return nil, 0, fmt.Errorf("list active face library page: %w", err)
		}
		if len(records) == 0 {
			break
		}

		for _, record := range records {
			embedding := record.Embedding.Slice()
			if len(embedding) == 0 {
				continue
			}
			if !first {
				buf.WriteByte(',')
			}
			first = false
			if err := encoder.Encode(faceLibraryItem{
				ID:        record.PersonID,
				Name:      record.PersonName,
				Embedding: embedding,
			}); err != nil {
				return nil, 0, fmt.Errorf("marshal face library item: %w", err)
			}
			itemCount++
		}

		cursor = nextCursor
	}

	buf.WriteString(`]}`)
	return buf.Bytes(), itemCount, nil
}

type faceLibrarySyncJob struct {
	index  int
	nodeID string
}

type faceLibrarySyncResult struct {
	index  int
	nodeID string
	err    error
}

func (s *FaceLibrarySyncService) updateFaceLibraryOnNodes(ctx context.Context, nodeIDs []string, algoName string, payload []byte) []string {
	if len(nodeIDs) == 0 {
		return nil
	}

	workerCount := faceLibrarySyncConcurrency(len(nodeIDs))
	jobs := make(chan faceLibrarySyncJob)
	results := make(chan faceLibrarySyncResult, len(nodeIDs))

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				err := s.engine.UpdateFaceLibrary(ctx, job.nodeID, algoName, payload)
				results <- faceLibrarySyncResult{
					index:  job.index,
					nodeID: job.nodeID,
					err:    err,
				}
			}
		}()
	}

	for i, nodeID := range nodeIDs {
		jobs <- faceLibrarySyncJob{index: i, nodeID: nodeID}
	}
	close(jobs)
	wg.Wait()
	close(results)

	failedByIndex := make([]string, len(nodeIDs))
	for result := range results {
		if result.err != nil {
			failedByIndex[result.index] = fmt.Sprintf("%s:%v", result.nodeID, result.err)
		}
	}

	failed := make([]string, 0)
	for _, failure := range failedByIndex {
		if failure != "" {
			failed = append(failed, failure)
		}
	}
	return failed
}

func faceLibrarySyncConcurrency(nodeCount int) int {
	if nodeCount < maxFaceLibrarySyncConcurrency {
		return nodeCount
	}
	return maxFaceLibrarySyncConcurrency
}

type faceLibraryItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	Embedding []float32 `json:"embedding"`
}
