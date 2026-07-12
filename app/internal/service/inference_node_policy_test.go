package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupInferenceNodePolicy(t *testing.T) (*InferenceNodePolicy, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:inference_policy_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE edge_nodes (
		id TEXT PRIMARY KEY, name TEXT, endpoint TEXT, auth_token TEXT, status TEXT,
		enabled NUMERIC, current_load INTEGER, max_load INTEGER, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE edge_node_algorithms (
		id TEXT PRIMARY KEY, node_id TEXT, algo_package_id TEXT, status TEXT, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE algorithm_packages (
		id TEXT PRIMARY KEY, algorithm_name TEXT, version TEXT, deleted_at DATETIME
	)`).Error)
	return NewInferenceNodePolicy(repository.NewEdgeNodeRepository(db), repository.NewEdgeNodeAlgorithmRepository(db)), db
}

func TestInferenceNodePolicyValidate(t *testing.T) {
	policy, db := setupInferenceNodePolicy(t)
	ctx := context.Background()
	require.NoError(t, db.Exec(`INSERT INTO algorithm_packages (id, algorithm_name, version)
		VALUES (?, ?, ?)`, "algo-1", "detector", "1.0.0").Error)
	require.NoError(t, db.Exec(`INSERT INTO edge_nodes
		(id, name, endpoint, auth_token, status, enabled, current_load, max_load)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, "node-1", "node-1", "http://node-1", "token", model.NodeStatusOnline, true, 0, 2).Error)
	require.NoError(t, db.Exec(`INSERT INTO edge_node_algorithms
		(id, node_id, algo_package_id, status) VALUES (?, ?, ?, ?)`,
		"deploy-1", "node-1", "algo-1", model.AlgoDeployInstalled).Error)

	_, err := policy.Validate(ctx, "node-1", "algo-1")
	require.NoError(t, err)

	tests := []struct {
		name    string
		updates map[string]interface{}
		want    string
	}{
		{"offline", map[string]interface{}{"status": model.NodeStatusOffline}, apperrors.ErrEdgeNodeOffline},
		{"disabled", map[string]interface{}{"status": model.NodeStatusOnline, "enabled": false}, apperrors.ErrEdgeNodeDisabled},
		{"full", map[string]interface{}{"enabled": true, "current_load": 2}, apperrors.ErrEdgeNodeFull},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, db.Model(&model.EdgeNode{}).Where("id = ?", "node-1").Updates(tt.updates).Error)
			_, err := policy.Validate(ctx, "node-1", "algo-1")
			var appErr *apperrors.AppError
			require.ErrorAs(t, err, &appErr)
			assert.Equal(t, tt.want, appErr.Code)
		})
	}

	require.NoError(t, db.Model(&model.EdgeNode{}).Where("id = ?", "node-1").Updates(map[string]interface{}{
		"status": model.NodeStatusOnline, "enabled": true, "current_load": 0,
	}).Error)
	require.NoError(t, db.Model(&model.EdgeNodeAlgorithm{}).Where("id = ?", "deploy-1").Update("status", model.AlgoDeployFailed).Error)
	_, err = policy.Validate(ctx, "node-1", "algo-1")
	var appErr *apperrors.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperrors.ErrEdgeNodeAlgorithmUnavailable, appErr.Code)
}
