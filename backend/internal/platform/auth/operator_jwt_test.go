package auth

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestOperatorTokenRoundtrip：平台 token 签发解析闭环
func TestOperatorTokenRoundtrip(t *testing.T) {
	defer withSecret(t, strings.Repeat("a", 64))()

	oid := uuid.New()
	token, _, err := SignOperatorToken(oid, "op@x.com", 3)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	claims, err := ParseOperatorToken(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if claims.OperatorID != oid || claims.Email != "op@x.com" || claims.Ver != 3 {
		t.Errorf("claims mismatch: %+v", claims)
	}
}

// TestTokenIssuerIsolation：商户 token 与平台 token 互相拒绝（issuer 隔离是两套身份的边界）
func TestTokenIssuerIsolation(t *testing.T) {
	defer withSecret(t, strings.Repeat("a", 64))()

	merchantToken, _, err := SignAccessToken(uuid.New(), uuid.New(), "u@x.com", "admin", false, 1, nil)
	if err != nil {
		t.Fatalf("sign merchant: %v", err)
	}
	if _, err := ParseOperatorToken(merchantToken); err == nil {
		t.Error("ParseOperatorToken 竟然接受了商户 token")
	}

	operatorToken, _, err := SignOperatorToken(uuid.New(), "op@x.com", 1)
	if err != nil {
		t.Fatalf("sign operator: %v", err)
	}
	if _, err := ParseAccessToken(operatorToken); err == nil {
		t.Error("ParseAccessToken 竟然接受了平台 token")
	}
}
