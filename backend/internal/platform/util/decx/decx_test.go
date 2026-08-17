package decx

import (
	"testing"

	"github.com/shopspring/decimal"
)

// 粗筛必须在任何 rescale 前拦下巨指数载荷——这些用例本身要能在瞬间跑完，
// 若粗筛失效，Round/比较会在测试里直接卡死或吃光内存
func TestSane(t *testing.T) {
	ok := []string{"0", "0.01", "380", "99999999.99", "380.000", "1e8", "0.855"}
	for _, s := range ok {
		if !Sane(decimal.RequireFromString(s)) {
			t.Errorf("Sane(%s) = false, want true", s)
		}
	}
	bad := []string{"1e1000000000", "1e-1000000000", "-1e1000000000", "1e11", "1234567890123"}
	for _, s := range bad {
		if Sane(decimal.RequireFromString(s)) {
			t.Errorf("Sane(%s) = true, want false", s)
		}
	}
}

func TestAmount(t *testing.T) {
	if err := Amount("x", decimal.RequireFromString("99999999.99")); err != nil {
		t.Errorf("99999999.99 should pass: %v", err)
	}
	for _, s := range []string{"100000000", "1e1000000000"} {
		if err := Amount("x", decimal.RequireFromString(s)); err == nil {
			t.Errorf("Amount(%s) should fail", s)
		}
	}
	// 负数放行（符号由调用点管），巨负指数仍拦
	if err := Amount("x", decimal.RequireFromString("-5")); err != nil {
		t.Errorf("-5 should pass Amount: %v", err)
	}
}
