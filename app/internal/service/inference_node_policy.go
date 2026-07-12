package service

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"
	"gorm.io/gorm"
)

// InferenceNodePolicy is the single source of truth for AI task node admission.
type InferenceNodePolicy struct {
	nodeRepo     *repository.EdgeNodeRepository
	nodeAlgoRepo *repository.EdgeNodeAlgorithmRepository
}

func NewInferenceNodePolicy(nodeRepo *repository.EdgeNodeRepository, nodeAlgoRepo *repository.EdgeNodeAlgorithmRepository) *InferenceNodePolicy {
	return &InferenceNodePolicy{nodeRepo: nodeRepo, nodeAlgoRepo: nodeAlgoRepo}
}

func (p *InferenceNodePolicy) Validate(ctx context.Context, nodeID, algoPackageID string) (*model.EdgeNode, error) {
	node, err := p.nodeRepo.FindByID(ctx, nodeID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.New(apperrors.ErrEdgeNodeNotFound, "")
	}
	if err != nil {
		return nil, fmt.Errorf("find inference node: %w", err)
	}
	if !node.Enabled || node.Status == model.NodeStatusDisabled {
		return nil, apperrors.New(apperrors.ErrEdgeNodeDisabled, "")
	}
	if node.Status != model.NodeStatusOnline {
		return nil, apperrors.New(apperrors.ErrEdgeNodeOffline, "")
	}
	if node.MaxLoad <= 0 || node.CurrentLoad >= node.MaxLoad {
		return nil, apperrors.New(apperrors.ErrEdgeNodeFull, "")
	}
	deployment, err := p.nodeAlgoRepo.FindByNodeAndAlgo(ctx, nodeID, algoPackageID)
	if errors.Is(err, gorm.ErrRecordNotFound) || (err == nil && deployment.Status != model.AlgoDeployInstalled) {
		return nil, apperrors.New(apperrors.ErrEdgeNodeAlgorithmUnavailable, "")
	}
	if err != nil {
		return nil, fmt.Errorf("find node algorithm deployment: %w", err)
	}
	return node, nil
}

func (p *InferenceNodePolicy) Candidates(ctx context.Context, algoPackageID string) ([]model.EdgeNode, error) {
	nodes, err := p.nodeRepo.FindOnlineNodesWithAlgorithm(ctx, algoPackageID)
	if err != nil {
		return nil, err
	}
	eligible := nodes[:0]
	for i := range nodes {
		if nodes[i].MaxLoad > 0 && nodes[i].CurrentLoad < nodes[i].MaxLoad {
			eligible = append(eligible, nodes[i])
		}
	}
	sort.Slice(eligible, func(i, j int) bool {
		left, right := loadRate(eligible[i]), loadRate(eligible[j])
		if left != right {
			return left < right
		}
		return eligible[i].ID < eligible[j].ID
	})
	return eligible, nil
}
