// Package jwt provides JWT token management including access tokens,
// refresh tokens (UUID stored in Redis), blacklisting, and rotation.
package jwt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Claims extends jwt.RegisteredClaims with application-specific fields.
type Claims struct {
	jwt.RegisteredClaims
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	RoleIDs  []string `json:"role_ids"`
	IsRoot   bool     `json:"is_root"`
}

// refreshTokenData is the JSON payload stored in Redis for refresh tokens.
type refreshTokenData struct {
	UserID    string   `json:"user_id"`
	Username  string   `json:"username"`
	RoleIDs   []string `json:"role_ids"`
	ExpiresAt int64    `json:"expires_at"`
	IsRoot    bool     `json:"is_root"`
}

// Manager handles JWT token operations.
type Manager struct {
	secret           []byte
	issuer           string
	audience         string
	accessExpireSec  int
	refreshExpireSec int
	redis            *redis.Client
}

// NewManager creates a new JWT Manager.
func NewManager(secret, issuer, audience string, accessExpireSec, refreshExpireSec int, rdb *redis.Client) *Manager {
	return &Manager{
		secret:           []byte(secret),
		issuer:           issuer,
		audience:         audience,
		accessExpireSec:  accessExpireSec,
		refreshExpireSec: refreshExpireSec,
		redis:            rdb,
	}
}

// GenerateTokenPair creates an access_token (signed JWT) and a refresh_token
// (random UUID stored in Redis).
func (m *Manager) GenerateTokenPair(userID string, username string, roleIDs []string, isRoot bool) (accessToken string, refreshToken string, expiresIn int, err error) {
	expiresIn = m.accessExpireSec
	now := time.Now()

	// Build access token claims
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RoleIDs:  roleIDs,
		IsRoot:   isRoot,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expiresIn) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
		},
	}

	// Sign access token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err = token.SignedString(m.secret)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token (UUID)
	refreshToken = uuid.New().String()

	// Store refresh token in Redis
	ctx := context.Background()
	refreshKey := fmt.Sprintf("refresh:%s:%s", userID, refreshToken)
	data := refreshTokenData{
		UserID:    userID,
		Username:  username,
		RoleIDs:   roleIDs,
		ExpiresAt: now.Add(time.Duration(m.refreshExpireSec) * time.Second).Unix(),
		IsRoot:    isRoot,
	}
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to marshal refresh token data: %w", err)
	}

	if err := m.redis.Set(ctx, refreshKey, dataBytes, time.Duration(m.refreshExpireSec)*time.Second).Err(); err != nil {
		return "", "", 0, fmt.Errorf("failed to store refresh token in redis: %w", err)
	}

	zap.L().Debug("token pair generated",
		zap.String("user_id", userID),
		zap.Int("access_expire_sec", expiresIn),
	)

	return accessToken, refreshToken, expiresIn, nil
}

// ValidateAccessToken validates a JWT access token and checks the Redis blacklist.
func (m *Manager) ValidateAccessToken(tokenString string) (*Claims, error) {
	// Parse and validate JWT
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	// Check blacklist
	ctx := context.Background()
	hash := sha256.Sum256([]byte(tokenString))
	blacklistKey := fmt.Sprintf("blacklist:access:%s", hex.EncodeToString(hash[:]))

	exists, err := m.redis.Exists(ctx, blacklistKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to check token blacklist: %w", err)
	}
	if exists > 0 {
		return nil, ErrTokenRevoked
	}

	return claims, nil
}

// RefreshTokens rotates the refresh token and issues a new token pair.
// If the provided refresh token has been reused (already deleted), it revokes
// all refresh tokens for that user as a security measure.
func (m *Manager) RefreshTokens(ctx context.Context, refreshToken string) (accessToken string, newRefreshToken string, expiresIn int, err error) {

	// 使用 SCAN（而非 KEYS）遍历所有用户的 refresh token，避免在大量 key 时阻塞 Redis。
	// key 格式为 refresh:{user_id}:{token_id}，因此通过 refresh:*:{token} 模式匹配。
	pattern := fmt.Sprintf("refresh:*:%s", refreshToken)
	var keys []string
	var cursor uint64
	for {
		var scannedKeys []string
		var nextCursor uint64
		var scanErr error
		scannedKeys, nextCursor, scanErr = m.redis.Scan(ctx, cursor, pattern, 100).Result()
		if scanErr != nil {
			return "", "", 0, fmt.Errorf("failed to scan refresh tokens: %w", scanErr)
		}
		keys = append(keys, scannedKeys...)
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}

	if len(keys) == 0 {
		// SCAN 未命中 → 检查是否属于 reuse 攻击：如果 refresh:used 标记存在，
		// 说明该 token 已被消耗（之前成功轮换过），当前请求是重放攻击。
		reusedBy, getErr := m.redis.Get(ctx, fmt.Sprintf("refresh:used:%s", refreshToken)).Result()
		if getErr == nil && reusedBy != "" {
			if revokeErr := m.RevokeAllRefreshTokens(ctx, reusedBy); revokeErr != nil {
				return "", "", 0, fmt.Errorf("failed to revoke reused refresh tokens: %w", revokeErr)
			}
			return "", "", 0, ErrRefreshTokenReuse
		}
		if getErr != nil && getErr != redis.Nil {
			return "", "", 0, fmt.Errorf("failed to check refresh token reuse marker: %w", getErr)
		}
		return "", "", 0, ErrRefreshTokenExpired
	}

	var (
		oldKey    string
		dataBytes []byte
	)
	// 用 GetDel 原子地消费 token：如果并发请求同时命中同一个 key，只有一个能获取到值，
	// 其他会拿到 redis.Nil 从而进入后续的 reuse 检测流程。
	for _, key := range keys {
		b, getErr := m.redis.GetDel(ctx, key).Bytes()
		if getErr == redis.Nil {
			continue
		}
		if getErr != nil {
			return "", "", 0, fmt.Errorf("failed to consume refresh token data: %w", getErr)
		}
		if len(b) == 0 {
			continue
		}
		oldKey = key
		dataBytes = b
		break
	}
	if oldKey == "" {
		// GetDel 全部为 Nil → 另一个并发请求已经消费了该 token。
		// 此时判断为 reuse，吊销该用户的所有 session。
		reusedBy, getErr := m.redis.Get(ctx, fmt.Sprintf("refresh:used:%s", refreshToken)).Result()
		if getErr == nil && reusedBy != "" {
			if revokeErr := m.RevokeAllRefreshTokens(ctx, reusedBy); revokeErr != nil {
				return "", "", 0, fmt.Errorf("failed to revoke reused refresh tokens: %w", revokeErr)
			}
			return "", "", 0, ErrRefreshTokenReuse
		}
		if getErr != nil && getErr != redis.Nil {
			return "", "", 0, fmt.Errorf("failed to check refresh token reuse marker: %w", getErr)
		}
		return "", "", 0, ErrRefreshTokenExpired
	}

	var data refreshTokenData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return "", "", 0, fmt.Errorf("failed to unmarshal refresh token data: %w", err)
	}

	userID := data.UserID
	// 设置 refresh:used 标记，保存 userID 以便后续检测到此 token 被重用时吊销该用户所有令牌。
	if err := m.redis.Set(ctx, fmt.Sprintf("refresh:used:%s", refreshToken), userID, time.Duration(m.refreshExpireSec)*time.Second).Err(); err != nil {
		return "", "", 0, fmt.Errorf("failed to set refresh token reuse marker: %w", err)
	}

	// 旧版 refresh token 不包含 Username 字段，轮换时降级为空值，
	// 下次登录后自动补全。审计日志会以 user_id 兜底展示。
	if data.Username == "" {
		zap.L().Warn("refresh token missing username, will be empty until next login",
			zap.String("user_id", userID),
		)
	}

	// 生成新的令牌对，旧的已被删除 + 标记为 used，无法再次使用。
	accessToken, newRefreshToken, expiresIn, err = m.GenerateTokenPair(userID, data.Username, data.RoleIDs, data.IsRoot)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to generate new token pair: %w", err)
	}

	zap.L().Debug("tokens refreshed",
		zap.String("user_id", userID),
		zap.String("old_token", refreshToken[:8]+"..."),
	)

	return accessToken, newRefreshToken, expiresIn, nil
}

// RevokeAccessToken adds the token to the Redis blacklist with a TTL equal
// to the remaining token expiry time.
func (m *Manager) RevokeAccessToken(ctx context.Context, tokenString string) error {
	// 使用 WithoutClaimsValidation 即使 token 已过期也能解析出 claims，
	// 确保登出时能正确设置黑名单 TTL（避免 token 已过期但未到黑名单删除时间的场景）。
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return m.secret, nil
	}, jwt.WithoutClaimsValidation())
	if err != nil {
		return fmt.Errorf("failed to parse token for revocation: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return ErrTokenInvalid
	}

	// Calculate remaining TTL
	var ttl time.Duration
	if claims.ExpiresAt != nil {
		ttl = time.Until(claims.ExpiresAt.Time)
		if ttl <= 0 {
			// Token already expired, no need to blacklist
			return nil
		}
	} else {
		ttl = time.Duration(m.accessExpireSec) * time.Second
	}

	hash := sha256.Sum256([]byte(tokenString))
	blacklistKey := fmt.Sprintf("blacklist:access:%s", hex.EncodeToString(hash[:]))

	if err := m.redis.Set(ctx, blacklistKey, "1", ttl).Err(); err != nil {
		return fmt.Errorf("failed to blacklist access token: %w", err)
	}

	zap.L().Debug("access token revoked",
		zap.String("user_id", claims.UserID),
		zap.Duration("ttl", ttl),
	)

	return nil
}

// RevokeAllRefreshTokens deletes all refresh tokens for a user.
// Used for security: when a refresh token reuse is detected, invalidate everything.
func (m *Manager) RevokeAllRefreshTokens(ctx context.Context, userID string) error {
	// 使用 SCAN 游标遍历该用户的所有 refresh token，分页删除避免阻塞 Redis。
	// 相比 KEYS，SCAN 在大量 key 时不会阻塞单线程。
	pattern := fmt.Sprintf("refresh:%s:*", userID)

	var keys []string
	var cursor uint64
	for {
		var scannedKeys []string
		var nextCursor uint64
		var err error
		scannedKeys, nextCursor, err = m.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("failed to scan refresh tokens for revocation: %w", err)
		}
		keys = append(keys, scannedKeys...)
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}

	if len(keys) == 0 {
		return nil
	}

	// 批量删除所有扫描到的 key，而非逐个删除，减少网络往返。
	if err := m.redis.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("failed to delete refresh tokens: %w", err)
	}

	zap.L().Info("all refresh tokens revoked",
		zap.String("user_id", userID),
		zap.Int("count", len(keys)),
	)

	return nil
}

// RevokeRefreshToken deletes a specific refresh token for a user.
func (m *Manager) RevokeRefreshToken(userID, tokenID string) error {
	ctx := context.Background()
	key := fmt.Sprintf("refresh:%s:%s", userID, tokenID)

	if err := m.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	zap.L().Debug("refresh token revoked",
		zap.String("user_id", userID),
		zap.String("token_id", tokenID),
	)

	return nil
}

// Sentinel errors for the jwt package.
var (
	ErrTokenInvalid        = fmt.Errorf("令牌无效")
	ErrTokenRevoked        = fmt.Errorf("令牌已被撤销")
	ErrRefreshTokenExpired = fmt.Errorf("刷新令牌已过期")
	ErrRefreshTokenReuse   = fmt.Errorf("刷新令牌疑似重用")
)
