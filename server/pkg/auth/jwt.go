package auth

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID      uint64   `json:"user_id"`
	Username    string   `json:"username"`
	IsAdmin     bool     `json:"is_admin"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	JTI         string   `json:"jti"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type Config struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

func GenerateTokenPair(cfg Config, userID uint64, username string, isAdmin bool, roles, permissions []string) (*TokenPair, error) {
	now := time.Now()
	jti := make([]byte, 16)
	if _, err := rand.Read(jti); err != nil {
		return nil, err
	}
	jtiStr := hex.EncodeToString(jti)

	if roles == nil {
		roles = []string{}
	}
	if permissions == nil {
		permissions = []string{}
	}

	accessClaims := &Claims{
		UserID:      userID,
		Username:    username,
		IsAdmin:     isAdmin,
		Roles:       roles,
		Permissions: permissions,
		JTI:         jtiStr,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.AccessTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jtiStr + "_a",
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(cfg.AccessSecret))
	if err != nil {
		return nil, err
	}

	refreshClaims := &Claims{
		UserID:      userID,
		Username:    username,
		IsAdmin:     isAdmin,
		Roles:       roles,
		Permissions: permissions,
		JTI:         jtiStr,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.RefreshTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        jtiStr + "_r",
		},
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(cfg.RefreshSecret))
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(cfg.AccessTTL.Seconds()),
	}, nil
}

func ParseToken(tokenStr string, secret string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}
