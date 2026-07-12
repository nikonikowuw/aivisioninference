// Package service 提供业务逻辑层实现，包含认证鉴权、资源管理和系统配置等核心业务流程。
package service

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
)

const (
	maxBatchIDs       = 100
	maxCSVExportRows  = 10000
	maxCSVImportRows  = 10000
	maxExportSelected = 10000
)

// newBatchResult 创建一个批量操作的初始结果，Items 按 ids 顺序预分配。
func newBatchResult(ids []string) dto.BatchResult {
	items := make([]dto.BatchItemResult, len(ids))
	for i, id := range ids {
		items[i] = dto.BatchItemResult{ID: id}
	}
	return dto.BatchResult{Total: len(ids), Items: items}
}

func runBatch(ids []string, lang string, fn func(id string) error) dto.BatchResult {
	if len(ids) > maxBatchIDs {
		return dto.BatchResult{
			Total:  1,
			Failed: 1,
			Items: []dto.BatchItemResult{{
				ID:      "batch",
				Code:    apperrors.ErrBadRequest,
				Message: apperrors.DefaultMessage(apperrors.ErrBadRequest, lang),
			}},
		}
	}

	result := newBatchResult(ids)
	for i, id := range ids {
		if err := fn(id); err != nil {
			code, message := batchErrorMessage(err, lang)
			result.Items[i].Code = code
			result.Items[i].Message = message
			result.Failed++
			continue
		}
		result.Items[i].Success = true
		result.Success++
	}
	return result
}

func batchErrorMessage(err error, lang string) (string, string) {
	if appErr, ok := err.(*apperrors.AppError); ok {
		if appErr.IsLocalizedMessage() {
			return appErr.Code, appErr.Message
		}
		return appErr.Code, apperrors.DefaultMessage(appErr.Code, lang)
	}
	zap.L().Warn("batch item failed with non-AppError", zap.Error(err))
	return apperrors.ErrInternal, apperrors.DefaultMessage(apperrors.ErrInternal, lang)
}


func csvRowErrorMessage(rowNo int, message string, lang string) string {
	switch lang {
	case "zh", "zh-tw":
		return fmt.Sprintf("第%d行：%s", rowNo, message)
	case "ja":
		return fmt.Sprintf("%d行目: %s", rowNo, message)
	case "ko":
		return fmt.Sprintf("%d행: %s", rowNo, message)
	case "id":
		return fmt.Sprintf("baris %d: %s", rowNo, message)
	default:
		return fmt.Sprintf("row %d: %s", rowNo, message)
	}
}
