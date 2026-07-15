// Package service 提供业务逻辑层。
package service

import (
	"context"
	"time"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	cachedriver "github.com/niko-admin/niko-admin/internal/pkg/cache"
)

// EdgeNodeRuntimeState 表示边缘节点的最新运行时状态。
type EdgeNodeRuntimeState struct {
	Name          string
	Status        string
	Enabled       bool
	CPUUsage      float64
	MemoryUsage   float64
	CurrentLoad   int
	CPUModel      string
	GPUModel      string
	HALPlatform   string
	EngineVersion string
	TotalMemory   int64
	LastHeartbeat *time.Time
	Uptime        int64
	RuntimeError  string
}

// EdgeNodeRuntimeStateStore 管理边缘节点运行时状态的读取、回填和失效。
type EdgeNodeRuntimeStateStore struct {
	typed *cachedriver.TypedCache[EdgeNodeRuntimeState]
}

// NewEdgeNodeRuntimeStateStore 创建边缘节点运行时状态存储。
func NewEdgeNodeRuntimeStateStore(store cachedriver.Cache) *EdgeNodeRuntimeStateStore {
	return &EdgeNodeRuntimeStateStore{
		typed: cachedriver.NewTypedCache[EdgeNodeRuntimeState](store, 0),
	}
}

// Get 返回节点运行时状态的副本。第二个返回值为 false 表示未找到。
func (c *EdgeNodeRuntimeStateStore) Get(ctx context.Context, id string) (*EdgeNodeRuntimeState, bool) {
	val, err := c.typed.Get(ctx, id)
	if err != nil {
		return nil, false
	}
	return val, true
}

// Set 设置节点运行时状态。
func (c *EdgeNodeRuntimeStateStore) Set(ctx context.Context, id string, state *EdgeNodeRuntimeState) {
	_ = c.typed.Set(ctx, id, state)
}

// Delete 删除节点运行时状态。
func (c *EdgeNodeRuntimeStateStore) Delete(ctx context.Context, id string) {
	_ = c.typed.Del(ctx, id)
}

// UpdateFromHeartbeat 用心跳数据替换节点运行时状态。
// 保留 Name 和 Enabled（配置字段，非心跳上报）。
func (c *EdgeNodeRuntimeStateStore) UpdateFromHeartbeat(ctx context.Context, id string, req *dto.HeartbeatRequest, status string, now time.Time, runtimeError string) {
	existing, _ := c.Get(ctx, id)
	name := ""
	enabled := true
	if existing != nil {
		name = existing.Name
		enabled = existing.Enabled
	}

	hb := now
	state := &EdgeNodeRuntimeState{
		Name:          name,
		Status:        status,
		Enabled:       enabled,
		CPUUsage:      req.CPUUsage,
		MemoryUsage:   req.MemoryUsage,
		CurrentLoad:   req.CurrentLoad,
		CPUModel:      req.HardwareInfo.CPUModel,
		GPUModel:      req.HardwareInfo.GPUModel,
		HALPlatform:   req.HALPlatform,
		EngineVersion: req.EngineVersion,
		TotalMemory:   req.HardwareInfo.TotalMemory,
		LastHeartbeat: &hb,
		Uptime:        req.Uptime,
		RuntimeError:  runtimeError,
	}
	_ = c.typed.Set(ctx, id, state)
}

// LoadFromNode 从 EdgeNode DB 模型回填运行时状态。
func (c *EdgeNodeRuntimeStateStore) LoadFromNode(ctx context.Context, node *model.EdgeNode) {
	state := &EdgeNodeRuntimeState{
		Name:          node.Name,
		Status:        node.Status,
		Enabled:       node.Enabled,
		CPUUsage:      node.CPUUsage,
		MemoryUsage:   node.MemoryUsage,
		CurrentLoad:   node.CurrentLoad,
		CPUModel:      node.CPUModel,
		GPUModel:      node.GPUModel,
		HALPlatform:   node.HALPlatform,
		EngineVersion: node.EngineVersion,
		TotalMemory:   node.TotalMemory,
		Uptime:        node.Uptime,
		RuntimeError:  node.RuntimeError,
	}
	if node.LastHeartbeat != nil {
		t := *node.LastHeartbeat
		state.LastHeartbeat = &t
	}
	_ = c.typed.Set(ctx, node.ID, state)
}

// ApplyToNode 将最新运行时字段写入 EdgeNode 模型。
// 用于 GetByID 返回最新运行时数据而不重新查询 DB。
// 返回 false 表示缓存未命中。
func (c *EdgeNodeRuntimeStateStore) ApplyToNode(ctx context.Context, node *model.EdgeNode) bool {
	state, err := c.typed.Get(ctx, node.ID)
	if err != nil {
		return false
	}
	node.Status = state.Status
	node.CPUUsage = state.CPUUsage
	node.MemoryUsage = state.MemoryUsage
	node.CurrentLoad = state.CurrentLoad
	node.CPUModel = state.CPUModel
	node.GPUModel = state.GPUModel
	node.HALPlatform = state.HALPlatform
	node.EngineVersion = state.EngineVersion
	node.TotalMemory = state.TotalMemory
	if state.LastHeartbeat != nil {
		t := *state.LastHeartbeat
		node.LastHeartbeat = &t
	}
	node.Uptime = state.Uptime
	node.RuntimeError = state.RuntimeError
	return true
}
