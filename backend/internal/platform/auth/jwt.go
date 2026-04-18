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
//   - Role 来自 users.role，业务层权限判断直接读 claim
//   - Ver（token_version）：与 users.token_version 比对；改密/停用/降级后旧 token 失效
//   - 绝对寿命：IssuedAt + MaxTokenLifetime 后不允许 refresh（防永续会话）
type Claims struct {
	UserID   uuid.UUID `json:"user_id"`
	TenantID uuid.UUID `json:"tenant_id"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	Ver      int32     `json:"ver"`
	jwt.RegisteredClaims
}

// MaxTokenLifetime token 绝对寿命：即使不断 refresh，也不能超过此时长
// 超过后必须重新登录；防被盗 token 永续攻击
const MaxTokenLifetime = 30 * 24 * time.Hour

func jwtSecret() ([]byte, error) {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		return nil, errors.New("JWT_SECRET not set")
	}
	return []byte(s), nil
}

// accessTTL 返回默认登录 TTL（一天），可由 JWT_ACCESS_TTL 环境变量覆盖
func accessTTL() time.Duration {
	if s := os.Getenv("JWT_ACCESS_TTL"); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			return d
		}
	}
	return 24 * time.Hour
}

// rememberTTL 返回"信任此设备"登录 TTL（7 天），可由 JWT_REMEMBER_TTL 环境变量覆盖
func rememberTTL() time.Duration {
	if s := os.Getenv("JWT_REMEMBER_TTL"); s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			return d
		}
	}
	return 7 * 24 * time.Hour
}

// SignAccessToken 用 HS256 签发 access token
//   - remember=true 时签发长期 token（默认 7 天）；否则短期（默认 24 小时）
//   - ver: 必填 users.token_version，被盗后增版本可立即失效
//   - originalIssuedAt: refresh 场景传入原 token 的 iat 保持绝对寿命窗不被重置；新登录传 nil 即可
func SignAccessToken(userID, tenantID uuid.UUID, email, role string, remember bool, ver int32, originalIssuedAt *time.Time) (string, time.Time, error) {
	secret, err := jwtSecret()
	if err != nil {
		return "", time.Time{}, err
	}
	ttl := accessTTL()
	if remember {
		ttl = rememberTTL()
	}
	expiresAt := time.Now().Add(ttl)
	iat := time.Now()
	if originalIssuedAt != nil {
		iat = *originalIssuedAt
	}
	claims := Claims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		Role:     role,
		Ver:      ver,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "mms",
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(iat),
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
