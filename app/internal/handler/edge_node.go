package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
	"github.com/niko-admin/niko-admin/internal/service"
)

type CreateEdgeNodeResponse struct {
	Node  *model.EdgeNode `json:"node"`
	Token string          `json:"token"`
}

// EdgeNodeHandler handles HTTP requests for EdgeNode and EdgeNodeAlgorithm operations.
type EdgeNodeHandler struct {
	svc     *service.EdgeNodeService
	tagSvc  *service.EdgeNodeTagService
}

// NewEdgeNodeHandler creates a new EdgeNodeHandler.
func NewEdgeNodeHandler(svc *service.EdgeNodeService, tagSvc *service.EdgeNodeTagService) *EdgeNodeHandler {
	return &EdgeNodeHandler{svc: svc, tagSvc: tagSvc}
}

// Create creates a new edge node.
//
// @Summary      创建边缘节点
// @Description  创建边缘节点并返回 JWT Token
// @Tags         边缘节点
// @Accept       json
// @Produce      json
// @Param        body  body  dto.CreateEdgeNodeRequest  true  "信息"
// @Success      200   {object}  dto.Response{data=handler.CreateEdgeNodeResponse}
// @Router       /edge-nodes [post]
// @Security     BearerAuth
func (h *EdgeNodeHandler) Create(c *gin.Context) {
	var req dto.CreateEdgeNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	node, token, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.Err(c, err)
		return
	}

	res := CreateEdgeNodeResponse{
		Node:  node,
		Token: token,
	}

	response.OK(c, res)
}

// List returns a paginated list of edge nodes with optional search filters.
//
// @Summary      边缘节点列表
// @Description  分页查询边缘节点列表
// @Tags         边缘节点
// @Produce      json
// @Param        page       query   int     false  "页码"      default(1)
// @Param        page_size  query   int     false  "每页数量"  default(20)
// @Param        keyword    query   string  false  "关键词搜索"
// @Param        status     query   string  false  "状态筛选(online/offline/error/disabled)"
// @Success      200  {object}  dto.Response{data=dto.PageData{list=[]model.EdgeNode}}
// @Router       /edge-nodes [get]
// @Security     BearerAuth
func (h *EdgeNodeHandler) List(c *gin.Context) {
	var req dto.EdgeNodeListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	items, total, err := h.svc.List(c.Request.Context(), req)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.Page(c, items, total, req.GetPage(), req.GetPageSize())
}

// GetByID returns detailed information of an edge node by ID.
//
// @Summary      获取边缘节点详情
// @Description  根据 ID 查询边缘节点详情
// @Tags         边缘节点
// @Produce      json
// @Param        id   path   string  true  "ID"
// @Success      200  {object}  dto.Response{data=model.EdgeNode}
// @Router       /edge-nodes/{id} [get]
// @Security     BearerAuth
func (h *EdgeNodeHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	node, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, node)
}

// Update updates an edge node configuration.
//
// @Summary      更新边缘节点
// @Description  更新边缘节点配置信息
// @Tags         边缘节点
// @Accept       json
// @Produce      json
// @Param        id    path   string                    true  "ID"
// @Param        body  body  dto.UpdateEdgeNodeRequest  true  "信息"
// @Success      200   {object}  dto.Response
// @Router       /edge-nodes/{id} [put]
// @Security     BearerAuth
func (h *EdgeNodeHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateEdgeNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	if err := h.svc.Update(c.Request.Context(), id, req); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}

// Delete soft deletes an edge node if there are no active tasks assigned to it.
//
// @Summary      删除边缘节点
// @Description  删除边缘节点（软删除）
// @Tags         边缘节点
// @Produce      json
// @Param        id  path  string  true  "ID"
// @Success      200  {object}  dto.Response
// @Router       /edge-nodes/{id} [delete]
// @Security     BearerAuth
func (h *EdgeNodeHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}

// DeployAlgorithm triggers an algorithm package deployment to the node.
//
// @Summary      下发算法包到边缘节点
// @Description  下发指定的算法包到指定的边缘节点
// @Tags         边缘节点
// @Accept       json
// @Produce      json
// @Param        id    path   string                        true  "节点 ID"
// @Param        body  body   dto.DeployAlgorithmRequest  true  "信息"
// @Success      200   {object}  dto.Response{data=dto.DeployAlgorithmResponse}
// @Router       /edge-nodes/{id}/deploy-algo [post]
// @Security     BearerAuth
func (h *EdgeNodeHandler) DeployAlgorithm(c *gin.Context) {
	id := c.Param("id")

	var req dto.DeployAlgorithmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	res, err := h.svc.DeployAlgorithm(c.Request.Context(), id, req)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, res)
}

// ListAlgorithms returns all algorithm packages deployed/pending on the node.
//
// @Summary      获取边缘节点已安装算法
// @Description  查询指定边缘节点已部署或待部署的算法列表
// @Tags         边缘节点
// @Produce      json
// @Param        id   path   string  true  "节点 ID"
// @Success      200  {object}  dto.Response{data=[]model.EdgeNodeAlgorithm}
// @Router       /edge-nodes/{id}/algorithms [get]
// @Security     BearerAuth
func (h *EdgeNodeHandler) ListAlgorithms(c *gin.Context) {
	id := c.Param("id")

	items, err := h.svc.ListAlgorithms(c.Request.Context(), id)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, items)
}

// RemoveAlgorithm removes an algorithm package from the node.
//
// @Summary      卸载边缘节点算法包
// @Description  从边缘节点卸载指定的算法包（删除部署记录）
// @Tags         边缘节点
// @Produce      json
// @Param        id        path   string  true  "节点 ID"
// @Param        algo_id   path   string  true  "算法包 ID"
// @Success      200  {object}  dto.Response
// @Router       /edge-nodes/{id}/algorithms/{algo_id} [delete]
// @Security     BearerAuth
func (h *EdgeNodeHandler) RemoveAlgorithm(c *gin.Context) {
	nodeID := c.Param("id")
	algoID := c.Param("algo_id")
	if algoID == "" {
		response.Err(c, apperrors.NewLocalized(apperrors.ErrBadRequest, "缺少算法包 ID"))
		return
	}

	if err := h.svc.RemoveAlgorithm(c.Request.Context(), nodeID, algoID); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}

// Heartbeat handles heartbeats from the inference engine.
//
// @Summary      引擎心跳上报
// @Description  边缘节点上报运行状态、硬件信息与算法列表，并获取待部署算法包
// @Tags         边缘节点
// @Accept       json
// @Produce      json
// @Param        id    path   string                  true  "节点 ID"
// @Param        body  body   dto.HeartbeatRequest  true  "信息"
// @Success      200   {object}  dto.Response{data=dto.HeartbeatResponse}
// @Router       /edge-nodes/{id}/heartbeat [post]
// @Security     BearerAuth
func (h *EdgeNodeHandler) Heartbeat(c *gin.Context) {
	id := c.Param("id")

	var req dto.HeartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	res, err := h.svc.HandleHeartbeat(c.Request.Context(), id, &req)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, res)
}

// @Summary      推荐节点
// @Description  根据算法包 ID 推荐负载最轻的在线节点
// @Tags         边缘节点
// @Produce      json
// @Param        algo_package_id  query  string  true  "算法包 ID"
// @Success      200  {object}  dto.Response{data=model.EdgeNode}
// @Router       /edge-nodes/recommend-node [get]
// @Security     BearerAuth
func (h *EdgeNodeHandler) RecommendNode(c *gin.Context) {
	algoPackageID := c.Query("algo_package_id")
	if algoPackageID == "" {
		response.Err(c, apperrors.NewLocalized(apperrors.ErrBadRequest, "algo_package_id 不能为空"))
		return
	}

	node, err := h.svc.RecommendNode(c.Request.Context(), algoPackageID)
	if err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, gin.H{
		"recommended_node_id": node.ID,
		"node_name":           node.Name,
		"current_load":        node.CurrentLoad,
		"max_load":            node.MaxLoad,
		"load_rate":           float64(node.CurrentLoad) / float64(node.MaxLoad),
	})
}

// UpdateTags updates the tags attached to an edge node.
//
// @Summary      更新节点标签
// @Description  替换边缘节点的标签绑定（全量替换）
// @Tags         边缘节点
// @Accept       json
// @Produce      json
// @Param        id    path   string                          true  "节点 ID"
// @Param        body  body   dto.UpdateEdgeNodeTagsRequest  true  "标签 ID 列表"
// @Success      200   {object}  dto.Response
// @Router       /edge-nodes/{id}/tags [put]
// @Security     BearerAuth
func (h *EdgeNodeHandler) UpdateTags(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateEdgeNodeTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Err(c, badRequestError(c, err))
		return
	}

	if err := h.tagSvc.ReplaceNodeTags(c.Request.Context(), id, req.TagIDs); err != nil {
		response.Err(c, err)
		return
	}

	response.OK(c, nil)
}
