package jwt

import (
	"testing"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeTokenLifecycle(t *testing.T) {
	secret := "my-very-secure-jwt-secret-at-least-32-chars"
	issuer := "niko-admin"
	audience := "niko-admin"
	accessExpire := 3600
	refreshExpire := 86400

	m := NewManager(secret, issuer, audience, accessExpire, refreshExpire, nil)

	// 1. Generate Token
	nodeID := "test-node-uuid-123"
	tokenString, err := m.GenerateNodeToken(nodeID)
	require.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	// 2. Parse Valid Token
	claims, err := m.ParseNodeToken(tokenString)
	require.NoError(t, err)
	assert.Equal(t, nodeID, claims.NodeID)
	assert.Equal(t, "edge-node", claims.Subject)
	assert.Equal(t, issuer, claims.Issuer)

	// Verify expiration is roughly 1 year in the future
	expectedExpiry := time.Now().AddDate(1, 0, 0)
	assert.WithinDuration(t, expectedExpiry, claims.ExpiresAt.Time, 10*time.Second)

	// 3. Parse Invalid Token (e.g. wrong signature)
	mWrong := NewManager("wrong-secret-key-different-hash", issuer, audience, accessExpire, refreshExpire, nil)
	_, err = mWrong.ParseNodeToken(tokenString)
	assert.Error(t, err)

	// 4. Parse User Token with ParseNodeToken should fail
	userClaims := &Claims{
		UserID:   "user-123",
		Username: "alice",
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   "user",
			ExpiresAt: jwtlib.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	userToken, err := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, userClaims).SignedString([]byte(secret))
	require.NoError(t, err)

	_, err = m.ParseNodeToken(userToken)
	assert.Error(t, err)
}
