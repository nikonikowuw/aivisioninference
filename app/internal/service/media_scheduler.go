package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/controlproto"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// MediaNodeCandidate contains the static capacity and latest observed usage of a node.
type MediaNodeCandidate struct {
	NodeID       string
	Online       bool
	Enabled      bool
	MetricsValid bool
	MetricsAt    time.Time
	MetricsTTL   time.Duration
	Capacity     uint32
	Usage        uint32
	Pending      uint32
}

// MediaScheduleDecision describes either the selected node or why no node is eligible.
type MediaScheduleDecision struct {
	NodeID     string
	Score      float64
	Rejections map[string][]string
}

type nodeMediaMetricsReader interface {
	GetFreshNodeMediaMetrics(nodeID string, now time.Time, ttl time.Duration) (*controlproto.EngineMetricsSnapshot, bool)
}

// MediaCapacityScheduler atomically creates short-lived capacity reservations.
type MediaCapacityScheduler struct {
	db          *gorm.DB
	nodeRepo    *repository.EdgeNodeRepository
	reserveRepo *repository.MediaCapacityReservationRepository
	metrics     nodeMediaMetricsReader
	now         func() time.Time
}

func NewMediaCapacityScheduler(
	db *gorm.DB,
	nodeRepo *repository.EdgeNodeRepository,
	reserveRepo *repository.MediaCapacityReservationRepository,
	metrics nodeMediaMetricsReader,
) *MediaCapacityScheduler {
	return &MediaCapacityScheduler{db: db, nodeRepo: nodeRepo, reserveRepo: reserveRepo, metrics: metrics, now: time.Now}
}

func (s *MediaCapacityScheduler) Confirm(ctx context.Context, reservationID string) error {
	if reservationID == "" {
		return fmt.Errorf("media reservation id is required")
	}
	return s.reserveRepo.Confirm(ctx, reservationID)
}

func (s *MediaCapacityScheduler) Release(ctx context.Context, reservationID string) error {
	if reservationID == "" {
		return fmt.Errorf("media reservation id is required")
	}
	return s.reserveRepo.Release(ctx, reservationID)
}

// Reserve selects a node and persists a lease in the same database transaction.
func (s *MediaCapacityScheduler) Reserve(ctx context.Context, streamKey string, lease time.Duration) (*model.MediaCapacityReservation, MediaScheduleDecision, error) {
	if streamKey == "" || lease <= 0 {
		return nil, MediaScheduleDecision{}, fmt.Errorf("invalid media capacity reservation request")
	}

	var reservation *model.MediaCapacityReservation
	var decision MediaScheduleDecision
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		reservation, decision, err = s.reserveOnce(ctx, streamKey, lease)
		if err == nil || !isRetryableMediaReservationError(err) {
			return reservation, decision, err
		}
		select {
		case <-ctx.Done():
			return nil, decision, ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 10 * time.Millisecond):
		}
	}
	return nil, decision, err
}

func (s *MediaCapacityScheduler) reserveOnce(ctx context.Context, streamKey string, lease time.Duration) (*model.MediaCapacityReservation, MediaScheduleDecision, error) {
	now := s.now()
	var result *model.MediaCapacityReservation
	var decision MediaScheduleDecision
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		nodeRepo := s.nodeRepo.WithTx(tx)
		reserveRepo := s.reserveRepo.WithTx(tx)
		if err := reserveRepo.DeleteExpired(ctx, now); err != nil {
			return fmt.Errorf("delete expired media reservations: %w", err)
		}
		existing, err := reserveRepo.FindValidByStreamKey(ctx, streamKey, now)
		if err != nil {
			return fmt.Errorf("find media reservation: %w", err)
		}
		if existing != nil {
			result = existing
			decision = MediaScheduleDecision{NodeID: existing.NodeID}
			return nil
		}

		nodes, err := nodeRepo.ListMediaSchedulableForUpdate(ctx)
		if err != nil {
			return fmt.Errorf("lock media nodes: %w", err)
		}
		reservations, err := reserveRepo.ListEffective(ctx, now)
		if err != nil {
			return fmt.Errorf("list media reservations: %w", err)
		}
		pending := make(map[string]uint32)
		for _, item := range reservations {
			pending[item.NodeID] += item.PreviewSlots
		}

		candidates := make([]MediaNodeCandidate, 0, len(nodes))
		for _, node := range nodes {
			ttl := time.Duration(node.MediaMetricsTTLSeconds) * time.Second
			snapshot, valid := s.metrics.GetFreshNodeMediaMetrics(node.ID, now, ttl)
			candidate := MediaNodeCandidate{
				NodeID: node.ID, Online: node.Status == model.NodeStatusOnline, Enabled: node.Enabled,
				MetricsValid: valid, MetricsAt: now, MetricsTTL: ttl,
				Pending: pending[node.ID],
			}
			if valid {
				candidate.Capacity = snapshot.PreviewCapacity
				candidate.Usage = snapshot.PreviewInUse
				candidate.MetricsAt = time.Unix(0, int64(snapshot.TimestampNS))
			}
			candidates = append(candidates, candidate)
		}
		decision = SelectMediaNode(now, candidates)
		if decision.NodeID == "" {
			return nil
		}
		expires := now.Add(lease)
		result = &model.MediaCapacityReservation{
			StreamKey: streamKey, NodeID: decision.NodeID,
			PreviewSlots: 1,
			Status:       model.MediaReservationPending, LeaseExpires: &expires,
		}
		if err := reserveRepo.Create(ctx, result); err != nil {
			return fmt.Errorf("create media reservation: %w", err)
		}
		return nil
	})
	return result, decision, err
}

func isRetryableMediaReservationError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") || strings.Contains(message, "deadlock detected") || strings.Contains(message, "serialization failure")
}

// SelectMediaNode admits one preview slot and minimizes post-admission utilization.
func SelectMediaNode(now time.Time, candidates []MediaNodeCandidate) MediaScheduleDecision {
	type rankedCandidate struct {
		nodeID string
		score  float64
	}

	decision := MediaScheduleDecision{Rejections: make(map[string][]string)}
	ranked := make([]rankedCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		reasons := mediaCandidateRejections(now, candidate)
		if len(reasons) > 0 {
			decision.Rejections[candidate.NodeID] = reasons
			continue
		}

		score := float64(candidate.Usage+candidate.Pending+1) / float64(candidate.Capacity)
		ranked = append(ranked, rankedCandidate{nodeID: candidate.NodeID, score: score})
	}

	if len(ranked) == 0 {
		return decision
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].nodeID < ranked[j].nodeID
		}
		return ranked[i].score < ranked[j].score
	})
	decision.NodeID = ranked[0].nodeID
	decision.Score = ranked[0].score
	return decision
}

func mediaCandidateRejections(now time.Time, candidate MediaNodeCandidate) []string {
	reasons := make([]string, 0, 4)
	if !candidate.Online {
		reasons = append(reasons, "offline")
	}
	if !candidate.Enabled {
		reasons = append(reasons, "disabled")
	}
	if candidate.Capacity == 0 {
		reasons = append(reasons, "unconfigured")
		return reasons
	}
	if !candidate.MetricsValid || candidate.MetricsTTL <= 0 || candidate.MetricsAt.IsZero() || now.Sub(candidate.MetricsAt) > candidate.MetricsTTL || candidate.MetricsAt.After(now) {
		reasons = append(reasons, "stale_metrics")
	}
	if candidate.Usage+candidate.Pending+1 > candidate.Capacity {
		reasons = append(reasons, "preview_full")
	}
	return reasons
}
