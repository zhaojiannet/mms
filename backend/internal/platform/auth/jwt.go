package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Claims 是 access token 的载荷
//   - TenantID 防止 token 跨租户串用（中间件验 token 时必须对齐 Host 解析到的 tenant）
//   - Role 来自 users.role，业务层权限判断直接读 claim，避免每次回库
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	jwt.RegisteredClaims
}

func jwtSecret() ([]byte, error) {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		return nil, errors.New("JWT_SECRET not set")
	}
	return []byte(s), nil
}

func accessTTL() time.Duration {
	if s := os.Getenv("JWT_ACCESS_TTL"); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			return d
		}
	}
	return 30 * time.Minute
}

// SignAccessToken 用 HS256 签发 access token
func SignAccessToken(userID, tenantID uuid.UUID, email, role string) (string, time.Time, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(accessTTL())
	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "mms",
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return signed, expiresAt, nil
}

// ParseAccessToken 验签并返回 claims（含过期检查）
func ParseAccessToken(raw string) (*Claims, error) {
	secret, err := jwtSecret()
	if err != nil {
		return nil, err
	}
	token, err := jwt.ParseWithClaims(raw, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("mms"))
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
