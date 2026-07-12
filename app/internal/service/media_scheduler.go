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

// MediaResourceVector is a media pipeline's capacity usage or requested delta.
type MediaResourceVector struct {
	DecodeSlots uint32
	EncodeSlots uint32
	EgressBPS   uint64
}

// MediaNodeCandidate contains the static capacity and latest observed usage of a node.
type MediaNodeCandidate struct {
	NodeID       string
	Online       bool
	Enabled      bool
	MetricsValid bool
	MetricsAt    time.Time
	MetricsTTL   time.Duration
	Capacity     MediaResourceVector
	Usage        MediaResourceVector
	Pending      MediaResourceVector
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
func (s *MediaCapacityScheduler) Reserve(ctx context.Context, streamKey string, demand MediaResourceVector, lease time.Duration) (*model.MediaCapacityReservation, MediaScheduleDecision, error) {
	if streamKey == "" || lease <= 0 {
		return nil, MediaScheduleDecision{}, fmt.Errorf("invalid media capacity reservation request")
	}

	var reservation *model.MediaCapacityReservation
	var decision MediaScheduleDecision
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		reservation, decision, err = s.reserveOnce(ctx, streamKey, demand, lease)
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

func (s *MediaCapacityScheduler) reserveOnce(ctx context.Context, streamKey string, demand MediaResourceVector, lease time.Duration) (*model.MediaCapacityReservation, MediaScheduleDecision, error) {
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
		pending := make(map[string]MediaResourceVector)
		for _, item := range reservations {
			usage := pending[item.NodeID]
			usage.DecodeSlots += item.DecodeSlots
			usage.EncodeSlots += item.EncodeSlots
			usage.EgressBPS += item.EgressBPS
			pending[item.NodeID] = usage
		}

		candidates := make([]MediaNodeCandidate, 0, len(nodes))
		for _, node := range nodes {
			ttl := time.Duration(node.MediaMetricsTTLSeconds) * time.Second
			snapshot, valid := s.metrics.GetFreshNodeMediaMetrics(node.ID, now, ttl)
			candidate := MediaNodeCandidate{
				NodeID: node.ID, Online: node.Status == model.NodeStatusOnline, Enabled: node.Enabled,
				MetricsValid: valid, MetricsAt: now, MetricsTTL: ttl,
				Capacity: MediaResourceVector{
					DecodeSlots: nonNegativeUint32(node.MediaDecodeCapacity),
					EncodeSlots: nonNegativeUint32(node.MediaEncodeCapacity),
					EgressBPS:   nonNegativeUint64(node.MediaEgressCapacityBPS),
				},
				Pending: pending[node.ID],
			}
			if valid {
				candidate.Usage = MediaResourceVector{DecodeSlots: snapshot.DecodeSlotsUsed, EncodeSlots: snapshot.EncodeSlotsUsed, EgressBPS: snapshot.EgressBPS}
			}
			candidates = append(candidates, candidate)
		}
		decision = SelectMediaNode(now, candidates, demand)
		if decision.NodeID == "" {
			return nil
		}
		expires := now.Add(lease)
		result = &model.MediaCapacityReservation{
			StreamKey: streamKey, NodeID: decision.NodeID,
			DecodeSlots: demand.DecodeSlots, EncodeSlots: demand.EncodeSlots, EgressBPS: demand.EgressBPS,
			Status: model.MediaReservationPending, LeaseExpires: &expires,
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

func nonNegativeUint32(value int) uint32 {
	if value <= 0 {
		return 0
	}
	return uint32(value)
}

func nonNegativeUint64(value int64) uint64 {
	if value <= 0 {
		return 0
	}
	return uint64(value)
}

// SelectMediaNode applies hard capacity constraints and minimizes the maximum
// post-admission utilization. Node ID is the deterministic tie-breaker.
func SelectMediaNode(now time.Time, candidates []MediaNodeCandidate, demand MediaResourceVector) MediaScheduleDecision {
	type rankedCandidate struct {
		nodeID string
		score  float64
	}

	decision := MediaScheduleDecision{Rejections: make(map[string][]string)}
	ranked := make([]rankedCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		reasons := mediaCandidateRejections(now, candidate, demand)
		if len(reasons) > 0 {
			decision.Rejections[candidate.NodeID] = reasons
			continue
		}

		decodeUsed := candidate.Usage.DecodeSlots + candidate.Pending.DecodeSlots + demand.DecodeSlots
		encodeUsed := candidate.Usage.EncodeSlots + candidate.Pending.EncodeSlots + demand.EncodeSlots
		egressUsed := candidate.Usage.EgressBPS + candidate.Pending.EgressBPS + demand.EgressBPS
		score := maxFloat(
			float64(decodeUsed)/float64(candidate.Capacity.DecodeSlots),
			float64(encodeUsed)/float64(candidate.Capacity.EncodeSlots),
			float64(egressUsed)/float64(candidate.Capacity.EgressBPS),
		)
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

func mediaCandidateRejections(now time.Time, candidate MediaNodeCandidate, demand MediaResourceVector) []string {
	reasons := make([]string, 0, 4)
	if !candidate.Online {
		reasons = append(reasons, "offline")
	}
	if !candidate.Enabled {
		reasons = append(reasons, "disabled")
	}
	if candidate.Capacity.DecodeSlots == 0 || candidate.Capacity.EncodeSlots == 0 || candidate.Capacity.EgressBPS == 0 {
		reasons = append(reasons, "unconfigured")
		return reasons
	}
	if !candidate.MetricsValid || candidate.MetricsTTL <= 0 || candidate.MetricsAt.IsZero() || now.Sub(candidate.MetricsAt) > candidate.MetricsTTL || candidate.MetricsAt.After(now) {
		reasons = append(reasons, "stale_metrics")
	}
	if candidate.Usage.DecodeSlots+candidate.Pending.DecodeSlots+demand.DecodeSlots > candidate.Capacity.DecodeSlots {
		reasons = append(reasons, "decode")
	}
	if candidate.Usage.EncodeSlots+candidate.Pending.EncodeSlots+demand.EncodeSlots > candidate.Capacity.EncodeSlots {
		reasons = append(reasons, "encode")
	}
	if candidate.Usage.EgressBPS+candidate.Pending.EgressBPS+demand.EgressBPS > candidate.Capacity.EgressBPS {
		reasons = append(reasons, "bandwidth")
	}
	return reasons
}

func maxFloat(values ...float64) float64 {
	var result float64
	for _, value := range values {
		if value > result {
			result = value
		}
	}
	return result
}
