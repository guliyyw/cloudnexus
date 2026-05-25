package auth

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
		// 验证签名算法，防止算法混淆攻击
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}

// SetTokenCookies 将 access_token 和 refresh_token 设置为 httpOnly Cookie
// SECURITY: httpOnly Cookie 防止 XSS 攻击窃取 Token
func SetTokenCookies(c *gin.Context, pair *TokenPair, accessTTL, refreshTTL time.Duration) {
	isSecure := c.Request.TLS != nil
	// Access Token Cookie - 短有效期
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("access_token", pair.AccessToken, int(accessTTL.Seconds()), "/", "", isSecure, true)
	// Refresh Token Cookie - 长有效期
	c.SetCookie("refresh_token", pair.RefreshToken, int(refreshTTL.Seconds()), "/", "", isSecure, true)
}

// ClearTokenCookies 清除 Token Cookie（用于登出）
func ClearTokenCookies(c *gin.Context) {
	isSecure := c.Request.TLS != nil
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("access_token", "", -1, "/", "", isSecure, true)
	c.SetCookie("refresh_token", "", -1, "/", "", isSecure, true)
}

// GetRefreshTokenFromCookie 从 Cookie 中获取 refresh_token
func GetRefreshTokenFromCookie(c *gin.Context) string {
	if token, err := c.Cookie("refresh_token"); err == nil {
		return token
	}
	return ""
}
