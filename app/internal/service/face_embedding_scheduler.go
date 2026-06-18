package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/repository"
)

const (
	faceEmbeddingLeaseTTL        = 45 * time.Second
	faceEmbeddingWarmupLockTTL   = 35 * time.Second
	faceEmbeddingWarmupWaitTTL   = 30 * time.Second
	faceEmbeddingWarmupPollEvery = 2 * time.Second
	maxEmbeddingAlternateRetries = 2
)

type embeddingLeaseStore interface {
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
	Get(ctx context.Context, key string) *redis.StringCmd
}

type FaceEmbeddingReservation struct {
	NodeID        string
	ReservationID string
	AlgoPackageID string
	slotKeys      []string
}

type FaceEmbeddingScheduler struct {
	nodeRepo *repository.EdgeNodeRepository
	rdb      embeddingLeaseStore
}

func NewFaceEmbeddingScheduler(nodeRepo *repository.EdgeNodeRepository, rdb *redis.Client) *FaceEmbeddingScheduler {
	return &FaceEmbeddingScheduler{nodeRepo: nodeRepo, rdb: rdb}
}

func NewFaceEmbeddingSchedulerWithStore(nodeRepo *repository.EdgeNodeRepository, store embeddingLeaseStore) *FaceEmbeddingScheduler {
	return &FaceEmbeddingScheduler{nodeRepo: nodeRepo, rdb: store}
}

func (s *FaceEmbeddingScheduler) EnsureReadyNode(ctx context.Context, algoName, algoVersion, algoPackageID string, engine EngineClient) (string, error) {
	readyNodes, err := s.nodeRepo.FindReadyEmbeddingNodesByAlgorithm(ctx, algoPackageID)
	if err != nil {
		return "", fmt.Errorf("find ready embedding nodes: %w", err)
	}
	if len(readyNodes) > 0 {
		return readyNodes[0].ID, nil
	}

	if engine == nil {
		return "", fmt.Errorf("no ready embedding nodes available")
	}

	installedNodes, err := s.nodeRepo.FindInstalledEmbeddingNodesByAlgorithm(ctx, algoPackageID)
	if err != nil {
		return "", fmt.Errorf("find installed embedding nodes: %w", err)
	}
	if len(installedNodes) == 0 {
		return "", fmt.Errorf("no installed embedding nodes available")
	}

	sort.SliceStable(installedNodes, func(i, j int) bool {
		if installedNodes[i].CurrentLoad != installedNodes[j].CurrentLoad {
			return installedNodes[i].CurrentLoad < installedNodes[j].CurrentLoad
		}
		return installedNodes[i].UpdatedAt.After(installedNodes[j].UpdatedAt)
	})

	lockKey := faceEmbeddingWarmupLockKey(algoName, algoVersion)
	requestID, err := randomReservationID()
	if err != nil {
		return "", fmt.Errorf("generate warmup request id: %w", err)
	}

	var lockAcquired bool
	for _, node := range installedNodes {
		ok, lockErr := s.rdb.SetNX(ctx, lockKey, node.ID+":"+requestID, faceEmbeddingWarmupLockTTL).Result()
		if lockErr != nil {
			return "", fmt.Errorf("acquire warmup lock: %w", lockErr)
		}
		if !ok {
			value, getErr := s.rdb.Get(ctx, lockKey).Result()
			if getErr == nil && value != "" {
				if warmNodeID, waitErr := s.waitForAnyReadyNode(ctx, algoPackageID); waitErr == nil {
					return warmNodeID, nil
				}
			}
			continue
		}

		lockAcquired = true
		if err := engine.WarmupAlgorithm(ctx, node.ID, algoName, algoVersion); err != nil {
			_ = releaseRedisValue(ctx, s.rdb, lockKey, node.ID+":"+requestID)
			continue
		}
		warmNodeID, waitErr := s.waitForNodeReady(ctx, node.ID, algoPackageID)
		_ = releaseRedisValue(ctx, s.rdb, lockKey, node.ID+":"+requestID)
		if waitErr != nil {
			continue
		}
		return warmNodeID, nil
	}

	if lockAcquired {
		return "", fmt.Errorf("warmup finished but no node became ready")
	}
	return "", fmt.Errorf("no installed embedding nodes could be warmed up")
}

func (s *FaceEmbeddingScheduler) Acquire(ctx context.Context, algoPackageID string) (*FaceEmbeddingReservation, error) {
	return s.acquire(ctx, algoPackageID, nil)
}

func (s *FaceEmbeddingScheduler) AcquireExcept(ctx context.Context, algoPackageID string, excludeNodeIDs []string) (*FaceEmbeddingReservation, error) {
	return s.acquire(ctx, algoPackageID, excludeNodeIDs)
}

func (s *FaceEmbeddingScheduler) acquire(ctx context.Context, algoPackageID string, excludeNodeIDs []string) (*FaceEmbeddingReservation, error) {
	nodes, err := s.nodeRepo.FindReadyEmbeddingNodesByAlgorithm(ctx, algoPackageID)
	if err != nil {
		return nil, fmt.Errorf("find ready embedding nodes: %w", err)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no ready embedding nodes available")
	}

	excluded := make(map[string]struct{}, len(excludeNodeIDs))
	for _, id := range excludeNodeIDs {
		excluded[id] = struct{}{}
	}

	sort.SliceStable(nodes, func(i, j int) bool {
		return nodeEmbeddingCapacity(nodes[i]) > nodeEmbeddingCapacity(nodes[j])
	})

	for _, node := range nodes {
		if _, skip := excluded[node.ID]; skip {
			continue
		}
		capacity := nodeEmbeddingCapacity(node)
		if capacity <= 0 {
			continue
		}
		reservationID, err := randomReservationID()
		if err != nil {
			return nil, fmt.Errorf("generate reservation id: %w", err)
		}
		for slot := 0; slot < capacity; slot++ {
			key := faceEmbeddingSlotKey(node.ID, slot)
			ok, err := s.rdb.SetNX(ctx, key, reservationID, faceEmbeddingLeaseTTL).Result()
			if err != nil {
				return nil, fmt.Errorf("acquire embedding slot: %w", err)
			}
			if ok {
				return &FaceEmbeddingReservation{
					NodeID:        node.ID,
					ReservationID: reservationID,
					AlgoPackageID: algoPackageID,
					slotKeys:      []string{key},
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("no embedding slots available")
}

func (s *FaceEmbeddingScheduler) Release(ctx context.Context, reservation *FaceEmbeddingReservation) error {
	if reservation == nil {
		return nil
	}
	for _, key := range reservation.slotKeys {
		if err := releaseRedisValue(ctx, s.rdb, key, reservation.ReservationID); err != nil {
			return err
		}
	}
	return nil
}

func (s *FaceEmbeddingScheduler) ListReadyFaceLibraryNodeIDs(ctx context.Context, algoPackageID string) ([]string, error) {
	nodes, err := s.nodeRepo.FindReadyFaceLibraryNodesByAlgorithm(ctx, algoPackageID)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(nodes))
	for _, node := range nodes {
		result = append(result, node.ID)
	}
	return result, nil
}

func (s *FaceEmbeddingScheduler) waitForNodeReady(ctx context.Context, nodeID, algoPackageID string) (string, error) {
	deadline := time.Now().Add(faceEmbeddingWarmupWaitTTL)
	for time.Now().Before(deadline) {
		nodes, err := s.nodeRepo.FindReadyEmbeddingNodesByAlgorithm(ctx, algoPackageID)
		if err != nil {
			return "", err
		}
		for _, node := range nodes {
			if node.ID == nodeID {
				return node.ID, nil
			}
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(faceEmbeddingWarmupPollEvery):
		}
	}
	return "", fmt.Errorf("warmup wait timeout")
}

func (s *FaceEmbeddingScheduler) waitForAnyReadyNode(ctx context.Context, algoPackageID string) (string, error) {
	deadline := time.Now().Add(faceEmbeddingWarmupWaitTTL)
	for time.Now().Before(deadline) {
		nodes, err := s.nodeRepo.FindReadyEmbeddingNodesByAlgorithm(ctx, algoPackageID)
		if err != nil {
			return "", err
		}
		if len(nodes) > 0 {
			return nodes[0].ID, nil
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(faceEmbeddingWarmupPollEvery):
		}
	}
	return "", fmt.Errorf("warmup wait timeout")
}

func MaxEmbeddingAlternateRetries() int {
	return maxEmbeddingAlternateRetries
}

func nodeEmbeddingCapacity(node model.EdgeNode) int {
	if node.EmbeddingCapacity > 0 {
		return node.EmbeddingCapacity
	}
	if node.MaxLoad > 0 {
		return node.MaxLoad
	}
	return 1
}

func faceEmbeddingSlotKey(nodeID string, slot int) string {
	return fmt.Sprintf("face_embedding:node:%s:slot:%d", nodeID, slot)
}

func faceEmbeddingWarmupLockKey(algoName, algoVersion string) string {
	return fmt.Sprintf("face_runtime:warmup:%s:%s", algoName, algoVersion)
}

func releaseRedisValue(ctx context.Context, store embeddingLeaseStore, key, expected string) error {
	script := `
if redis.call("GET", KEYS[1]) == ARGV[1] then
	return redis.call("DEL", KEYS[1])
end
return 0
`
	return store.Eval(ctx, script, []string{key}, expected).Err()
}

func randomReservationID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
