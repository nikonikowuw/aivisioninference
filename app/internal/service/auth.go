package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/dto"
	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/errors"
	"github.com/niko-admin/niko-admin/internal/pkg/hash"
	jwtutil "github.com/niko-admin/niko-admin/internal/pkg/jwt"
	"github.com/niko-admin/niko-admin/internal/repository"
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
}

// NewAuthService creates a new AuthService.
func NewAuthService(userRepo *repository.UserRepository, permRepo *repository.PermissionRepository, rdb *redis.Client, jwtManager *jwtutil.Manager) *AuthService {
	return &AuthService{
		userRepo:       userRepo,
		permRepo:       permRepo,
		rdb:            rdb,
		jwtManager:     jwtManager,
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

	accessToken, refreshToken, expiresIn, err := s.jwtManager.GenerateTokenPair(user.ID, roleIDs, user.IsRoot)
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

// ChangePassword verifies old password and updates to new one.
func (s *AuthService) ChangePassword(ctx context.Context, userID, oldPassword, newPassword string) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return errors.New(errors.ErrNotFound, "用户不存在")
	}

	if !hash.Check(oldPassword, user.Password) {
		return errors.New(errors.ErrBadRequest, "旧密码错误")
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
