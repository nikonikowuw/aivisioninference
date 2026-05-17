package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/hash"
	jwtutil "github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/repository"
	"github.com/niko-admin/niko-admin/pkg/storage"
)

const menuTreeCacheTTL = 10 * time.Minute

func menuTreeCacheKey(roleIDs []string) string {
	sorted := make([]string, len(roleIDs))
	copy(sorted, roleIDs)
	sort.Strings(sorted)
	h := sha256.Sum256([]byte(strings.Join(sorted, ",")))
	return fmt.Sprintf("perm:menu_tree:%x", h[:8])
}

// AuthService handles authentication business logic.
type AuthService struct {
	userRepo   *repository.UserRepository
	permRepo   *repository.PermissionRepository
	rdb        *redis.Client
	jwtManager *jwtutil.Manager
	storage    storage.Storage
}

func NewAuthService(userRepo *repository.UserRepository, permRepo *repository.PermissionRepository, rdb *redis.Client, jwtManager *jwtutil.Manager, stor storage.Storage) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		permRepo:   permRepo,
		rdb:        rdb,
		jwtManager: jwtManager,
		storage:    stor,
	}
}

// LoginResult holds the data needed by the handler to complete a login response.
type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
	User         dto.UserInfo
}

// Login authenticates a user and returns tokens.
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*LoginResult, error) {
	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		return nil, errors.New(errors.ErrUnauthorized, "用户名或密码错误")
	}

	if !hash.Check(req.Password, user.Password) {
		return nil, errors.New(errors.ErrUnauthorized, "用户名或密码错误")
	}

	if user.Status != 1 {
		return nil, errors.New(errors.ErrForbidden, "用户已被禁用")
	}

	roleInfos, roleIDs := toRoleInfosAndIDs(user.Roles)
	menus := s.getMenuTree(ctx, roleIDs)

	accessToken, refreshToken, expiresIn, err := s.jwtManager.GenerateTokenPair(user.ID, user.Username, roleIDs, user.IsRoot)
	if err != nil {
		zap.L().Error("generate token pair failed", zap.String("user_id", user.ID), zap.Error(err))
		return nil, errors.New(errors.ErrInternal, "")
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    expiresIn,
		User: dto.UserInfo{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			AvatarURL:   user.AvatarURL,
			Email:       user.Email,
			Status:      user.Status,
			Roles:       roleInfos,
			Menus:       menus,
			CreatedAt:   user.CreatedAt.Format(dto.DateTimeFormat),
			UpdatedAt:   user.UpdatedAt.Format(dto.DateTimeFormat),
		},
	}, nil
}

// RefreshTokens rotates refresh tokens and returns a new access token.
func (s *AuthService) RefreshTokens(ctx context.Context, refreshToken string) (accessToken, newRefreshToken string, expiresIn int, err error) {
	return s.jwtManager.RefreshTokens(ctx, refreshToken)
}

// RevokeAccessToken revokes a single access token.
func (s *AuthService) RevokeAccessToken(ctx context.Context, tokenString string) error {
	return s.jwtManager.RevokeAccessToken(ctx, tokenString)
}

// RevokeAllRefreshTokens revokes all refresh tokens for a user.
func (s *AuthService) RevokeAllRefreshTokens(ctx context.Context, userID string) error {
	return s.jwtManager.RevokeAllRefreshTokens(ctx, userID)
}

// GetMe returns user info by ID.
func (s *AuthService) GetMe(ctx context.Context, userID string) (*dto.UserInfo, error) {
	user, err := s.userRepo.FindByIDWithRoles(ctx, userID)
	if err != nil {
		return nil, errors.New(errors.ErrNotFound, "用户不存在")
	}

	roleInfos, roleIDs := toRoleInfosAndIDs(user.Roles)

	menus := s.getMenuTree(ctx, roleIDs)

	return &dto.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		Email:       user.Email,
		Status:      user.Status,
		Roles:       roleInfos,
		Menus:       menus,
		CreatedAt:   user.CreatedAt.Format(dto.DateTimeFormat),
		UpdatedAt:   user.UpdatedAt.Format(dto.DateTimeFormat),
	}, nil
}

// toRoleInfosAndIDs converts model roles to DTO RoleInfo slice and collects role IDs.
func toRoleInfosAndIDs(roles []model.Role) (infos []dto.RoleInfo, ids []string) {
	infos = make([]dto.RoleInfo, 0, len(roles))
	ids = make([]string, 0, len(roles))
	for _, role := range roles {
		ids = append(ids, role.ID)
		infos = append(infos, dto.RoleInfo{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
			Level:       role.Level,
		})
	}
	return infos, ids
}

// getMenuTree returns the menu tree for the given role IDs, using Redis cache.
func (s *AuthService) getMenuTree(ctx context.Context, roleIDs []string) []dto.Menu {
	if len(roleIDs) == 0 {
		return nil
	}

	key := menuTreeCacheKey(roleIDs)

	// Try cache
	if s.rdb != nil {
		cached, err := s.rdb.Get(ctx, key).Bytes()
		if err == nil {
			var menus []dto.Menu
			if json.Unmarshal(cached, &menus) == nil {
				return menus
			}
		}
	}

	// Cache miss — build from DB
	perms, err := s.permRepo.FindMenusByRoleIDs(ctx, roleIDs)
	if err != nil {
		zap.L().Error("find menus by role ids failed", zap.Error(err))
		return nil
	}

	menus := buildMenuTree(perms)

	// Populate cache
	if s.rdb != nil {
		if data, marshalErr := json.Marshal(menus); marshalErr == nil {
			if setErr := s.rdb.Set(ctx, key, data, menuTreeCacheTTL).Err(); setErr != nil {
				zap.L().Warn("set menu tree cache failed", zap.Error(setErr))
			}
		}
	}

	return menus
}

type menuNode struct {
	menu     dto.Menu
	parentID string
}

// buildMenuTree 将权限列表构建为菜单树。
// 通过 DFS 访问状态标记避免循环引用，并对每层子节点按 SortOrder 排序。
func buildMenuTree(perms []model.Permission) []dto.Menu {
	nodes := buildMenuNodes(perms)
	childrenByParent, rootIDs := buildMenuRelations(nodes)
	sortMenuRelations(nodes, childrenByParent, rootIDs)
	return buildMenuRoots(nodes, childrenByParent, rootIDs)
}

func buildMenuNodes(perms []model.Permission) map[string]*menuNode {
	nodes := make(map[string]*menuNode, len(perms))
	for _, perm := range perms {
		if perm.Type != model.PermTypeMenu {
			continue
		}
		if _, exists := nodes[perm.ID]; exists {
			continue
		}
		node := &menuNode{
			menu: dto.Menu{
				ID:        perm.ID,
				Name:      perm.Name,
				Code:      perm.Code,
				Path:      perm.Path,
				Icon:      perm.Icon,
				SortOrder: perm.SortOrder,
			},
		}
		if perm.ParentID != nil {
			node.parentID = *perm.ParentID
		}
		nodes[perm.ID] = node
	}
	return nodes
}

func buildMenuRelations(nodes map[string]*menuNode) (map[string][]string, []string) {
	childrenByParent := make(map[string][]string, len(nodes))
	rootIDs := make([]string, 0, len(nodes))
	for id, node := range nodes {
		if node.parentID == "" {
			rootIDs = append(rootIDs, id)
			continue
		}
		if _, ok := nodes[node.parentID]; !ok {
			rootIDs = append(rootIDs, id)
			continue
		}
		childrenByParent[node.parentID] = append(childrenByParent[node.parentID], id)
	}
	return childrenByParent, rootIDs
}

func sortMenuRelations(nodes map[string]*menuNode, childrenByParent map[string][]string, rootIDs []string) {
	sort.SliceStable(rootIDs, func(i, j int) bool {
		return nodes[rootIDs[i]].menu.SortOrder < nodes[rootIDs[j]].menu.SortOrder
	})
	for parentID := range childrenByParent {
		ids := childrenByParent[parentID]
		sort.SliceStable(ids, func(i, j int) bool {
			return nodes[ids[i]].menu.SortOrder < nodes[ids[j]].menu.SortOrder
		})
		childrenByParent[parentID] = ids
	}
}

func buildMenuRoots(nodes map[string]*menuNode, childrenByParent map[string][]string, rootIDs []string) []dto.Menu {
	const (
		visiting = 1
		visited  = 2
	)
	state := make(map[string]int, len(nodes))

	var walk func(string) *dto.Menu
	walk = func(id string) *dto.Menu {
		node, ok := nodes[id]
		if !ok {
			return nil
		}
		if state[id] == visiting {
			zap.L().Warn("menu cycle detected", zap.String("menu_id", id))
			return nil
		}
		if state[id] == visited {
			m := node.menu
			return &m
		}

		state[id] = visiting
		menu := node.menu
		childIDs := childrenByParent[id]
		menu.Children = make([]dto.Menu, 0, len(childIDs))
		for _, childID := range childIDs {
			child := walk(childID)
			if child == nil {
				continue
			}
			menu.Children = append(menu.Children, *child)
		}
		state[id] = visited
		node.menu = menu
		m := node.menu
		return &m
	}

	roots := make([]dto.Menu, 0, len(rootIDs))
	for _, id := range rootIDs {
		root := walk(id)
		if root == nil {
			continue
		}
		roots = append(roots, *root)
	}
	return roots
}

// UpdateProfile 更新当前用户的个人资料（display_name、email、avatar_url）。
//
// 核心功能：安全地更新用户个人信息，包含邮箱唯一性校验。
// 算法设计：
//  1. 通过 FindByID 获取用户当前信息，作为校验基准。
//  2. 邮箱变更时执行唯一性查询，排除当前用户自身，防止冲突。
//  3. 仅更新请求中非 nil 的字段，实现部分更新语义。
//  4. 更新完成后通过 GetMe 返回最新的完整用户信息（含角色和菜单）。
//
// 参数:
//
//	ctx - 上下文，用于链路追踪与超时控制
//	userID - 当前登录用户的 ID
//	req - 包含可选更新字段的请求 DTO
//
// 返回值:
//
//	*dto.UserInfo - 更新后的用户完整信息
//	error - 业务异常返回 AppError，系统异常记录日志后返回 ErrInternal
func (s *AuthService) UpdateProfile(ctx context.Context, userID string, req *dto.UpdateProfileRequest) (*dto.UserInfo, error) {
	var user *model.User

	if err := s.userRepo.Transaction(ctx, func(txRepo *repository.UserRepository) error {
		var err error
		user, err = txRepo.FindByIDForUpdate(ctx, userID)
		if err != nil {
			return errors.New(errors.ErrNotFound, "")
		}

		if req.Email != nil && *req.Email != user.Email {
			count, err := txRepo.CountByEmail(ctx, *req.Email, userID)
			if err != nil {
				zap.L().Error("count by email failed", zap.Error(err))
				return errors.New(errors.ErrInternal, "")
			}
			if count > 0 {
				return errors.New(errors.ErrEmailTaken, "")
			}
			user.Email = *req.Email
		}

		if req.DisplayName != nil {
			user.DisplayName = *req.DisplayName
		}
		if req.AvatarURL != nil {
			user.AvatarURL = *req.AvatarURL
		}

		if err := txRepo.Update(ctx, user); err != nil {
			zap.L().Error("update user profile failed", zap.String("user_id", userID), zap.Error(err))
			return errors.New(errors.ErrInternal, "")
		}
		return nil
	}); err != nil {
		return nil, err
	}

	info, err := s.GetMe(ctx, userID)
	if err != nil {
		return nil, err
	}
	return info, nil
}

// ChangePassword verifies old password and updates to new one.
func (s *AuthService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return errors.New(errors.ErrNotFound, "")
	}

	if !hash.Check(oldPassword, user.Password) {
		return errors.New(errors.ErrOldPasswordWrong, "")
	}

	hashedPassword, err := hash.Hash(newPassword)
	if err != nil {
		zap.L().Error("hash password failed", zap.Error(err))
		return errors.New(errors.ErrInternal, "")
	}

	if err := s.userRepo.UpdatePassword(ctx, userID, hashedPassword); err != nil {
		zap.L().Error("update password failed", zap.Error(err))
		return errors.New(errors.ErrInternal, "")
	}

	if err := s.jwtManager.RevokeAllRefreshTokens(ctx, userID); err != nil {
		zap.L().Warn("revoke tokens after password change failed",
			zap.String("user_id", userID),
			zap.Error(err),
		)
	}

	return nil
}

const (
	maxAvatarSize    = 2 * 1024 * 1024
	avatarPathPrefix = "avatars"
)

var allowedAvatarTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

func (s *AuthService) UploadAvatar(ctx context.Context, userID string, fileHeader *multipart.FileHeader) (string, error) {
	if fileHeader.Size > maxAvatarSize {
		return "", errors.New(errors.ErrFileTooLarge, "")
	}

	file, err := fileHeader.Open()
	if err != nil {
		zap.L().Error("open uploaded avatar file failed", zap.Error(err))
		return "", errors.New(errors.ErrInternal, "")
	}
	defer file.Close()

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		zap.L().Error("read avatar file header for type detection failed", zap.Error(err))
		return "", errors.New(errors.ErrInternal, "")
	}
	mimeType := http.DetectContentType(buf[:n])
	if !allowedAvatarTypes[mimeType] {
		return "", errors.New(errors.ErrFileInvalidType, "")
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		zap.L().Error("seek avatar file to start failed", zap.Error(err))
		return "", errors.New(errors.ErrInternal, "")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == "" || ext == ".jpeg" {
		switch mimeType {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/gif":
			ext = ".gif"
		case "image/webp":
			ext = ".webp"
		default:
			exts, _ := mime.ExtensionsByType(mimeType)
			if len(exts) > 0 {
				ext = exts[0]
			}
		}
	}
	if ext == ".jpeg" {
		ext = ".jpg"
	}

	storageName := uuid.New().String() + ext
	storagePath := avatarPathPrefix + "/" + storageName

	storedPath, err := s.storage.Save(file, storagePath)
	if err != nil {
		zap.L().Error("save avatar file failed", zap.Error(err))
		return "", errors.New(errors.ErrInternal, "")
	}

	avatarURL := s.storage.GetURL(storedPath)

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return "", errors.New(errors.ErrNotFound, "")
	}

	oldAvatarURL := user.AvatarURL
	user.AvatarURL = avatarURL
	if err := s.userRepo.Update(ctx, user); err != nil {
		zap.L().Error("update user avatar_url failed", zap.String("user_id", userID), zap.Error(err))
		return "", errors.New(errors.ErrInternal, "")
	}

	if oldAvatarURL != "" && strings.HasPrefix(oldAvatarURL, s.storage.GetURL("")) {
		oldPath := strings.TrimPrefix(oldAvatarURL, s.storage.GetURL("")+"/")
		if err := s.storage.Delete(oldPath); err != nil {
			zap.L().Warn("delete old avatar file failed", zap.String("path", oldPath), zap.Error(err))
		}
	}

	return avatarURL, nil
}
