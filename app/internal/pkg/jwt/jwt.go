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
	UserID  string   `json:"user_id"`
	RoleIDs []string `json:"role_ids"`
	IsRoot  bool     `json:"is_root"`
}

// refreshTokenData is the JSON payload stored in Redis for refresh tokens.
type refreshTokenData struct {
	UserID    string   `json:"user_id"`
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
func (m *Manager) GenerateTokenPair(userID string, roleIDs []string, isRoot bool) (accessToken string, refreshToken string, expiresIn int, err error) {
	expiresIn = m.accessExpireSec
	now := time.Now()

	// Build access token claims
	claims := &Claims{
		UserID:  userID,
		RoleIDs: roleIDs,
		IsRoot:  isRoot,
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
		return nil, fmt.Errorf("%w: %v", appErrTokenInvalid, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, appErrTokenInvalid
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
		return nil, appErrTokenRevoked
	}

	return claims, nil
}

// RefreshTokens rotates the refresh token and issues a new token pair.
// If the provided refresh token has been reused (already deleted), it revokes
// all refresh tokens for that user as a security measure.
func (m *Manager) RefreshTokens(ctx context.Context, refreshToken string) (accessToken string, newRefreshToken string, expiresIn int, err error) {

	// Find the refresh token in Redis by scanning for the token value
	// We need to look up the token across all users. The key format is
	// refresh:{user_id}:{token_id}, so we scan.
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
		reusedBy, getErr := m.redis.Get(ctx, fmt.Sprintf("refresh:used:%s", refreshToken)).Result()
		if getErr == nil && reusedBy != "" {
			if revokeErr := m.RevokeAllRefreshTokens(ctx, reusedBy); revokeErr != nil {
				return "", "", 0, fmt.Errorf("failed to revoke reused refresh tokens: %w", revokeErr)
			}
			return "", "", 0, appErrRefreshTokenReuse
		}
		if getErr != nil && getErr != redis.Nil {
			return "", "", 0, fmt.Errorf("failed to check refresh token reuse marker: %w", getErr)
		}
		return "", "", 0, appErrRefreshTokenExpired
	}

	var (
		oldKey   string
		dataBytes []byte
	)
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
		reusedBy, getErr := m.redis.Get(ctx, fmt.Sprintf("refresh:used:%s", refreshToken)).Result()
		if getErr == nil && reusedBy != "" {
			if revokeErr := m.RevokeAllRefreshTokens(ctx, reusedBy); revokeErr != nil {
				return "", "", 0, fmt.Errorf("failed to revoke reused refresh tokens: %w", revokeErr)
			}
			return "", "", 0, appErrRefreshTokenReuse
		}
		if getErr != nil && getErr != redis.Nil {
			return "", "", 0, fmt.Errorf("failed to check refresh token reuse marker: %w", getErr)
		}
		return "", "", 0, appErrRefreshTokenExpired
	}

	var data refreshTokenData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return "", "", 0, fmt.Errorf("failed to unmarshal refresh token data: %w", err)
	}

	userID := data.UserID
	if err := m.redis.Set(ctx, fmt.Sprintf("refresh:used:%s", refreshToken), userID, time.Duration(m.refreshExpireSec)*time.Second).Err(); err != nil {
		return "", "", 0, fmt.Errorf("failed to set refresh token reuse marker: %w", err)
	}

	// old token is already consumed by GETDEL during rotation.

	// Generate new token pair.
	accessToken, newRefreshToken, expiresIn, err = m.GenerateTokenPair(userID, data.RoleIDs, data.IsRoot)
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
	// Parse without validation to extract expiry
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return m.secret, nil
	}, jwt.WithoutClaimsValidation())
	if err != nil {
		return fmt.Errorf("failed to parse token for revocation: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return appErrTokenInvalid
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
	appErrTokenInvalid         = fmt.Errorf("令牌无效")
	appErrTokenRevoked         = fmt.Errorf("令牌已被撤销")
	appErrRefreshTokenExpired  = fmt.Errorf("刷新令牌已过期")
	appErrRefreshTokenReuse    = fmt.Errorf("刷新令牌疑似重用")
)
