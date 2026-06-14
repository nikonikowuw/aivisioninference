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

func (m *Manager) RefreshTokens(ctx context.Context, refreshToken string) (accessToken string, newRefreshToken string, expiresIn int, err error) {
	pattern := fmt.Sprintf("refresh:*:%s", refreshToken)
	keys, err := m.scanKeys(ctx, pattern)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to scan refresh tokens: %w", err)
	}

	if len(keys) == 0 {
		return "", "", 0, m.handleMissingToken(ctx, refreshToken)
	}

	dataBytes, userID, err := m.consumeRefreshToken(ctx, keys)
	if err != nil {
		return "", "", 0, err
	}

	var data refreshTokenData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		return "", "", 0, fmt.Errorf("failed to unmarshal refresh token data: %w", err)
	}

	if err := m.markTokenUsed(ctx, refreshToken, userID); err != nil {
		return "", "", 0, err
	}

	if data.Username == "" {
		zap.L().Warn("refresh token missing username, will be empty until next login",
			zap.String("user_id", userID),
		)
	}

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

func (m *Manager) scanKeys(ctx context.Context, pattern string) ([]string, error) {
	var keys []string
	var cursor uint64
	for {
		scannedKeys, nextCursor, err := m.redis.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}
		keys = append(keys, scannedKeys...)
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}
	return keys, nil
}

func (m *Manager) handleMissingToken(ctx context.Context, refreshToken string) error {
	reusedBy, err := m.redis.Get(ctx, fmt.Sprintf("refresh:used:%s", refreshToken)).Result()
	if err == nil && reusedBy != "" {
		if revokeErr := m.RevokeAllRefreshTokens(ctx, reusedBy); revokeErr != nil {
			return fmt.Errorf("failed to revoke reused refresh tokens: %w", revokeErr)
		}
		return ErrRefreshTokenReuse
	}
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to check refresh token reuse marker: %w", err)
	}
	return ErrRefreshTokenExpired
}

func (m *Manager) consumeRefreshToken(ctx context.Context, keys []string) ([]byte, string, error) {
	for _, key := range keys {
		dataBytes, err := m.redis.GetDel(ctx, key).Bytes()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, "", fmt.Errorf("failed to consume refresh token data: %w", err)
		}
		if len(dataBytes) == 0 {
			continue
		}

		var data refreshTokenData
		if err := json.Unmarshal(dataBytes, &data); err != nil {
			continue
		}
		return dataBytes, data.UserID, nil
	}

	return nil, "", m.handleMissingToken(ctx, "")
}

func (m *Manager) markTokenUsed(ctx context.Context, refreshToken, userID string) error {
	return m.redis.Set(ctx, 
		fmt.Sprintf("refresh:used:%s", refreshToken), 
		userID, 
		time.Duration(m.refreshExpireSec)*time.Second).Err()
}

func (m *Manager) RevokeAccessToken(ctx context.Context, tokenString string) error {
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

	ttl := m.calculateTTL(claims)
	if ttl <= 0 {
		return nil
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

func (m *Manager) calculateTTL(claims *Claims) time.Duration {
	if claims.ExpiresAt != nil {
		ttl := time.Until(claims.ExpiresAt.Time)
		if ttl <= 0 {
			return 0
		}
		return ttl
	}
	return time.Duration(m.accessExpireSec) * time.Second
}

func (m *Manager) RevokeAllRefreshTokens(ctx context.Context, userID string) error {
	pattern := fmt.Sprintf("refresh:%s:*", userID)
	keys, err := m.scanKeys(ctx, pattern)
	if err != nil {
		return fmt.Errorf("failed to scan refresh tokens for revocation: %w", err)
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

// NodeClaims extends RegisteredClaims for edge node JWT tokens.
type NodeClaims struct {
	jwt.RegisteredClaims
	NodeID string `json:"node_id"`
}

// GenerateNodeToken creates a JWT token for an edge node with 1-year validity.
func (m *Manager) GenerateNodeToken(nodeID string) (string, error) {
	now := time.Now()
	claims := NodeClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   "edge-node",
			Audience:  jwt.ClaimStrings{m.audience},
			ExpiresAt: jwt.NewNumericDate(now.AddDate(1, 0, 0)), // 1 year
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        uuid.New().String(),
		},
		NodeID: nodeID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ParseNodeToken parses and validates a node JWT token, returning the claims.
func (m *Manager) ParseNodeToken(tokenString string) (*NodeClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &NodeClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*NodeClaims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	// Verify subject is "edge-node"
	if claims.Subject != "edge-node" {
		return nil, fmt.Errorf("invalid token subject: %s", claims.Subject)
	}

	return claims, nil
}

// Sentinel errors for the jwt package.
var (
	ErrTokenInvalid        = fmt.Errorf("令牌无效")
	ErrTokenRevoked        = fmt.Errorf("令牌已被撤销")
	ErrRefreshTokenExpired = fmt.Errorf("刷新令牌已过期")
	ErrRefreshTokenReuse   = fmt.Errorf("刷新令牌疑似重用")
)
