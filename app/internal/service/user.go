// Package service 提供业务逻辑层实现，包含认证鉴权、资源管理和系统配置等核心业务流程。
package service

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	stdmail "net/mail"
	"strconv"
	"strings"
	"unicode/utf8"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/csvx"
	apperrors "github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/hash"
	"github.com/niko-admin/niko-admin/internal/repository"
)

// UserService 处理用户操作的业务逻辑
type UserService struct {
	userRepo *repository.UserRepository
}

// NewUserService 创建新的 UserService
func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// List 返回分页的用户列表，支持可选过滤条件
func (s *UserService) List(ctx context.Context, req dto.UserListRequest) ([]model.User, int64, error) {
	return s.userRepo.List(ctx, req)
}

// Create 创建新用户，包含密码哈希和可选的角色关联
func (s *UserService) Create(ctx context.Context, req dto.CreateUserRequest) (*model.User, error) {
	count, err := s.userRepo.CountByUsername(ctx, req.Username, "")
	if err != nil {
		zap.L().Error("check username uniqueness failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	if count > 0 {
		return nil, apperrors.New(apperrors.ErrUsernameTaken, "")
	}
	if req.Email != "" {
		count, err := s.userRepo.CountByEmail(ctx, req.Email, "")
		if err != nil {
			zap.L().Error("check email uniqueness failed", zap.Error(err))
			return nil, apperrors.New(apperrors.ErrInternal, "")
		}
		if count > 0 {
			return nil, apperrors.New(apperrors.ErrEmailTaken, "")
		}
	}

	hashedPassword, err := hash.Hash(req.Password)
	if err != nil {
		zap.L().Error("hash password failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	user := model.User{
		Username:    req.Username,
		Password:    hashedPassword,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		AvatarURL:   req.AvatarURL,
		Status:      req.Status,
	}

	// 在事务中创建用户和角色关联，确保一致性。
	if err := s.userRepo.CreateWithRoles(ctx, &user, req.RoleIDs); err != nil {
		zap.L().Error("create user failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	// 重新查询以返回完整的用户数据（含角色关联），而非直接返回创建后的 user 对象。
	return s.userRepo.FindByIDWithRoles(ctx, user.ID)
}

// GetByID 根据用户ID返回用户
func (s *UserService) GetByID(ctx context.Context, id string) (*model.User, error) {
	user, err := s.userRepo.FindByIDWithRoles(ctx, id)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrUserNotFound, "")
	}
	return user, nil
}

// Update 更新现有用户
func (s *UserService) Update(ctx context.Context, id string, req dto.UpdateUserRequest, currentUserID string, isRoot bool) error {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return apperrors.New(apperrors.ErrUserNotFound, "")
	}

	// write=true 表示当前操作需要写入能力，检查层级确保当前用户有足够权限修改目标用户。
	if err := checkUserHierarchy(ctx, s.userRepo, currentUserID, id, isRoot, true); err != nil {
		return err
	}

	// 用户名变更时检查唯一性，排除自身以避免与当前用户名冲突。
	if req.Username != "" && req.Username != user.Username {
		count, err := s.userRepo.CountByUsername(ctx, req.Username, id)
		if err != nil {
			zap.L().Error("check username uniqueness failed", zap.Error(err))
			return apperrors.New(apperrors.ErrInternal, "")
		}
		if count > 0 {
			return apperrors.New(apperrors.ErrUsernameTaken, "")
		}
	}

	// 仅更新请求中非空的字段，实现 PUT 的部分更新语义（类比 PATCH）。
	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.DisplayName != "" {
		user.DisplayName = req.DisplayName
	}
	if req.AvatarURL != "" {
		user.AvatarURL = req.AvatarURL
	}
	// Status 使用指针类型以便区分「不更新」和「更新为 0」两种状态。
	if req.Status != nil {
		user.Status = *req.Status
	}

	if err := s.userRepo.UpdateWithRoles(ctx, user, req.RoleIDs); err != nil {
		zap.L().Error("update user failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	return nil
}

// Delete 根据用户ID软删除用户
func (s *UserService) Delete(ctx context.Context, id, currentUserID string, isRoot bool) error {
	if id == currentUserID {
		return apperrors.New(apperrors.ErrCannotDeleteSelf, "")
	}

	if err := checkUserHierarchy(ctx, s.userRepo, currentUserID, id, isRoot, false); err != nil {
		return err
	}

	if err := s.userRepo.Delete(ctx, id); err != nil {
		zap.L().Error("delete user failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}
	return nil
}

// BatchDelete 批量软删除用户，并逐条返回处理结果。
func (s *UserService) BatchDelete(ctx context.Context, ids []string, currentUserID string, isRoot bool, lang string) dto.BatchResult {
	return runBatch(ids, lang, func(id string) error {
		return s.Delete(ctx, id, currentUserID, isRoot)
	})
}

// BatchUpdateStatus 批量更新用户状态，并保留单条更新的权限校验和自禁用保护。
func (s *UserService) BatchUpdateStatus(ctx context.Context, ids []string, status int, currentUserID string, isRoot bool, lang string) dto.BatchResult {
	return runBatch(ids, lang, func(id string) error {
		if id == currentUserID && status == 0 {
			return apperrors.New(apperrors.ErrCannotDisableSelf, "")
		}
		return s.Update(ctx, id, dto.UpdateUserRequest{Status: &status}, currentUserID, isRoot)
	})
}

// ExportCSV 导出当前筛选条件下的用户列表 CSV。
func (s *UserService) ExportCSV(ctx context.Context, req dto.UserListRequest) ([]byte, error) {
	items, err := s.userRepo.ListForExport(ctx, req, maxCSVExportRows)
	if err != nil {
		zap.L().Error("export users failed", zap.Error(err))
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}

	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.ID,
			item.Username,
			item.DisplayName,
			item.Email,
			strconv.Itoa(item.Status),
			userRoleNames(item),
			item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	data, err := csvx.Build([]string{"ID", "Username", "DisplayName", "Email", "Status", "Roles", "CreatedAt"}, rows)
	if err != nil {
		return nil, apperrors.New(apperrors.ErrInternal, "")
	}
	return data, nil
}

// ImportCSV 从 CSV 文件批量导入用户，列顺序为 username,email,display_name,password,status。
func (s *UserService) ImportCSV(ctx context.Context, reader io.Reader, lang string) (dto.BatchResult, error) {
	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1
	header, err := csvReader.Read()
	if err != nil {
		return dto.BatchResult{}, apperrors.New(apperrors.ErrCSVInvalidContent, apperrors.DefaultMessage(apperrors.ErrCSVInvalidContent, lang))
	}
	if !validUserImportHeader(header) {
		return dto.BatchResult{}, apperrors.New(apperrors.ErrCSVHeaderInvalid, apperrors.DefaultMessage(apperrors.ErrCSVHeaderInvalid, lang))
	}

	seenUsernames := make(map[string]int)
	result := dto.BatchResult{Items: make([]dto.BatchItemResult, 0)}
	for rowNo := 2; ; rowNo++ {
		row, err := csvReader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return dto.BatchResult{}, apperrors.New(apperrors.ErrCSVInvalidContent, apperrors.DefaultMessage(apperrors.ErrCSVInvalidContent, lang))
		}
		if isEmptyCSVRow(row) {
			continue
		}
		if result.Total >= maxCSVImportRows {
			return dto.BatchResult{}, apperrors.New(apperrors.ErrCSVRowLimitExceeded, apperrors.DefaultMessage(apperrors.ErrCSVRowLimitExceeded, lang))
		}

		item := dto.BatchItemResult{ID: strconv.Itoa(rowNo)}
		result.Total++
		if len(row) < 4 {
			appendImportFailure(&result, item, rowNo, apperrors.ErrCSVColumnRequired, lang)
			continue
		}

		username := strings.TrimSpace(row[0])
		if firstRow, ok := seenUsernames[strings.ToLower(username)]; ok {
			appendImportFailureMessage(&result, item, rowNo, apperrors.ErrCSVDuplicateUsername, fmt.Sprintf("%s (%d)", apperrors.DefaultMessage(apperrors.ErrCSVDuplicateUsername, lang), firstRow), lang)
			continue
		}
		seenUsernames[strings.ToLower(username)] = rowNo

		status, err := importUserStatus(row)
		if err != nil {
			appendImportFailure(&result, item, rowNo, apperrors.ErrCSVStatusInvalid, lang)
			continue
		}
		req := dto.CreateUserRequest{
			Username:    username,
			Email:       strings.TrimSpace(row[1]),
			DisplayName: strings.TrimSpace(row[2]),
			Password:    strings.TrimSpace(row[3]),
			Status:      status,
		}
		if code := validateImportUserRequest(req); code != "" {
			appendImportFailure(&result, item, rowNo, code, lang)
			continue
		}
		if _, err := s.Create(ctx, req); err != nil {
			code, message := batchErrorMessage(err, lang)
			appendImportFailureMessage(&result, item, rowNo, code, message, lang)
			continue
		}
		item.Success = true
		result.Success++
		result.Items = append(result.Items, item)
	}

	if result.Total == 0 {
		return dto.BatchResult{}, apperrors.New(apperrors.ErrCSVInvalidContent, apperrors.DefaultMessage(apperrors.ErrCSVInvalidContent, lang))
	}
	return result, nil
}

func validUserImportHeader(header []string) bool {
	expected := []string{"username", "email", "display_name", "password"}
	if len(header) < len(expected) {
		return false
	}
	for i, name := range expected {
		if normalizeCSVHeader(header[i]) != name {
			return false
		}
	}
	if len(header) >= 5 && normalizeCSVHeader(header[4]) != "status" {
		return false
	}
	return true
}

func normalizeCSVHeader(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(value, "\ufeff")))
}

func isEmptyCSVRow(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func importUserStatus(row []string) (int, error) {
	if len(row) <= 4 || strings.TrimSpace(row[4]) == "" {
		return 1, nil
	}
	status, err := strconv.Atoi(strings.TrimSpace(row[4]))
	if err != nil || (status != 0 && status != 1) {
		return 0, errors.New("invalid status")
	}
	return status, nil
}

func validateImportUserRequest(req dto.CreateUserRequest) string {
	if utf8.RuneCountInString(req.Username) < 2 || utf8.RuneCountInString(req.Username) > 32 {
		return apperrors.ErrBadRequest
	}
	if req.Email != "" {
		if _, err := stdmail.ParseAddress(req.Email); err != nil || utf8.RuneCountInString(req.Email) > 255 {
			return apperrors.ErrCSVInvalidEmail
		}
	}
	if utf8.RuneCountInString(req.DisplayName) > 64 {
		return apperrors.ErrBadRequest
	}
	if len(req.Password) < 6 || len(req.Password) > 72 {
		return apperrors.ErrCSVWeakPassword
	}
	return ""
}

func appendImportFailure(result *dto.BatchResult, item dto.BatchItemResult, rowNo int, code string, lang string) {
	appendImportFailureMessage(result, item, rowNo, code, apperrors.DefaultMessage(code, lang), lang)
}

func appendImportFailureMessage(result *dto.BatchResult, item dto.BatchItemResult, rowNo int, code string, message string, lang string) {
	item.Code = code
	item.Message = csvRowErrorMessage(rowNo, message, lang)
	result.Failed++
	result.Items = append(result.Items, item)
}

func userRoleNames(user model.User) string {
	names := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		names = append(names, role.Name)
	}
	return strings.Join(names, ",")
}

// ResetPassword 允许管理员直接重置指定用户的密码（不需要旧密码），执行层级安全检查。
func (s *UserService) ResetPassword(ctx context.Context, targetUserID, password, currentUserID string, isRoot bool) error {
	// 禁止管理员通过此接口重置自己的密码，防止误操作导致自己无法登录。
	// 自己改密码应走 ChangePassword 流程（需验证旧密码）。
	if targetUserID == currentUserID {
		return apperrors.New(apperrors.ErrCannotResetSelf, "")
	}

	// 检查目标用户存在
	_, err := s.userRepo.FindByID(ctx, targetUserID)
	if err != nil {
		return apperrors.New(apperrors.ErrUserNotFound, "")
	}

	// 层级权限校验：上级才能重置下级的密码，防止越权操作。
	if err := checkUserHierarchy(ctx, s.userRepo, currentUserID, targetUserID, isRoot, true); err != nil {
		return err
	}

	// 哈希密码
	hashedPassword, err := hash.Hash(password)
	if err != nil {
		zap.L().Error("hash password failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	// 更新密码
	if err := s.userRepo.UpdatePassword(ctx, targetUserID, hashedPassword); err != nil {
		zap.L().Error("reset password failed", zap.Error(err))
		return apperrors.New(apperrors.ErrInternal, "")
	}

	return nil
}
