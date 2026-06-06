package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/response"
)

// RegisterAlgorithmPackageReadRoutes 注册算法包只读接口，供 AI 任务表单选择算法。
func RegisterAlgorithmPackageReadRoutes(api *gin.RouterGroup, db *gorm.DB, rbacMiddleware gin.HandlerFunc) {
	group := api.Group("/algorithmpackages", rbacMiddleware)
	{
		group.GET("", func(c *gin.Context) {
			var req struct {
				dto.PageRequest
				Keyword string `form:"keyword"`
				Status  string `form:"status"`
			}
			if err := c.ShouldBindQuery(&req); err != nil {
				response.Err(c, apperrors.New(apperrors.ErrBadRequest, err.Error()))
				return
			}

			query := db.WithContext(c.Request.Context()).Model(&model.AlgorithmPackage{})
			if req.Keyword != "" {
				like := "%" + req.Keyword + "%"
				query = query.Where("algorithm_name ILIKE ? OR algorithm_alias ILIKE ? OR domain ILIKE ?", like, like, like)
			}
			if req.Status != "" {
				query = query.Where("status = ?", req.Status)
			}

			var total int64
			if err := query.Count(&total).Error; err != nil {
				response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
				return
			}

			var items []model.AlgorithmPackage
			page := req.GetPage()
			pageSize := req.GetPageSize()
			if err := query.Order("created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&items).Error; err != nil {
				response.Err(c, apperrors.New(apperrors.ErrInternal, ""))
				return
			}
			response.Page(c, items, total, page, pageSize)
		})

		group.GET("/:id", func(c *gin.Context) {
			id := c.Param("id")
			if id == "" {
				response.Err(c, apperrors.New(apperrors.ErrBadRequest, ""))
				return
			}
			var item model.AlgorithmPackage
			if err := db.WithContext(c.Request.Context()).Where("id = ?", id).First(&item).Error; err != nil {
				response.Err(c, apperrors.New(apperrors.ErrNotFound, ""))
				return
			}
			response.OK(c, item)
		})
	}
}
