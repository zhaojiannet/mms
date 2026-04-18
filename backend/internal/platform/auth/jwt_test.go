package auth

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// 测试前后设置 JWT_SECRET，避免污染其他测试
func withSecret(t *testing.T, secret string) func() {
	t.Helper()
	old := os.Getenv("JWT_SECRET")
	_ = os.Setenv("JWT_SECRET", secret)
	return func() { _ = os.Setenv("JWT_SECRET", old) }
}

// TestSignParseRoundtrip：签发 + 解析后 claims 字段完整保留
func TestSignParseRoundtrip(t *testing.T) {
	defer withSecret(t, strings.Repeat("a", 64))()

	uid := uuid.New()
	tid := uuid.New()

	token, expiresAt, err := SignAccessToken(uid, tid, "u@x.com", "admin", false, 7, nil)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if time.Until(expiresAt) < 23*time.Hour {
		t.Errorf("expiresAt too early: %v", expiresAt)
	}

	claims, err := ParseAccessToken(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.UserID != uid {
		t.Errorf("uid mismatch: got %v want %v", claims.UserID, uid)
	}
	if claims.TenantID != tid {
		t.Errorf("tid mismatch")
	}
	if claims.Role != "admin" {
		t.Errorf("role mismatch: %s", claims.Role)
	}
	if claims.Ver != 7 {
		t.Errorf("ver mismatch: got %d want 7", claims.Ver)
	}
}

// TestRememberTokenLongerTTL：remember=true 签出的 token 过期时间显著更长
func TestRememberTokenLongerTTL(t *testing.T) {
	defer withSecret(t, strings.Repeat("b", 64))()

	_, shortExp, _ := SignAccessToken(uuid.New(), uuid.New(), "u@x", "staff", false, 0, nil)
	_, longExp, _ := SignAccessToken(uuid.New(), uuid.New(), "u@x", "staff", true, 0, nil)

	if !longExp.After(shortExp.Add(24 * time.Hour)) {
		t.Errorf("remember token should be significantly longer than normal, got short=%v long=%v", shortExp, longExp)
	}
}

// TestOriginalIssuedAtPreserved：refresh 场景传入 originalIssuedAt 应保留（防永续攻击）
func TestOriginalIssuedAtPreserved(t *testing.T) {
	defer withSecret(t, strings.Repeat("c", 64))()

	origIat := time.Now().Add(-10 * 24 * time.Hour)
	token, _, err := SignAccessToken(uuid.New(), uuid.New(), "u@x", "staff", false, 0, &origIat)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	claims, err := ParseAccessToken(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.IssuedAt == nil {
		t.Fatal("iat should be set")
	}
	diff := claims.IssuedAt.Time.Sub(origIat).Abs()
	if diff > time.Second {
		t.Errorf("iat not preserved from original: got %v want near %v", claims.IssuedAt.Time, origIat)
	}
}

// TestWrongSignatureRejected：篡改签名的 token 应被拒绝
func TestWrongSignatureRejected(t *testing.T) {
	defer withSecret(t, strings.Repeat("d", 64))()
	token, _, _ := SignAccessToken(uuid.New(), uuid.New(), "u@x", "staff", false, 0, nil)

	// 换 secret 再解析
	_ = os.Setenv("JWT_SECRET", strings.Repeat("e", 64))
	if _, err := ParseAccessToken(token); err == nil {
		t.Error("parse should fail when secret changed")
	}
}

// TestAlgNoneRejected：防御 "alg: none" 攻击 —— 构造 unsigned token 应被拒绝
func TestAlgNoneRejected(t *testing.T) {
	defer withSecret(t, strings.Repeat("f", 64))()

	// 手动构造 alg=none 的 unsigned JWT
	claims := Claims{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Role:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "mms",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	raw, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("unsafe sign: %v", err)
	}
	if _, err := ParseAccessToken(raw); err == nil {
		t.Error("alg=none token must be rejected")
	}
}

// TestExpiredTokenRejected
func TestExpiredTokenRejected(t *testing.T) {
	defer withSecret(t, strings.Repeat("g", 64))()

	// 直接构造一个 exp 在过去的 token（绕过 SignAccessToken 的 now()）
	secret := []byte(strings.Repeat("g", 64))
	claims := Claims{
		UserID:   uuid.New(),
		TenantID: uuid.New(),
		Role:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "mms",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	raw, _ := token.SignedString(secret)

	if _, err := ParseAccessToken(raw); err == nil {
		t.Error("expired token must be rejected")
	}
}

// TestMaxTokenLifetimeConstant：确保常量在合理范围（防误改成过短或过长）
func TestMaxTokenLifetimeConstant(t *testing.T) {
	if MaxTokenLifetime < 7*24*time.Hour {
		t.Errorf("MaxTokenLifetime too short: %v", MaxTokenLifetime)
	}
	if MaxTokenLifetime > 90*24*time.Hour {
		t.Errorf("MaxTokenLifetime too long (security risk): %v", MaxTokenLifetime)
	}
}
