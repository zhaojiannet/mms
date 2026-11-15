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
	// 巨指数直接构造：decimal 解析字符串时已拒绝超过 MaxDecodeExponent 的指数，
	// 但这个上限只管解析，指数在它之内的值和非解析来源的值仍要靠 Sane 拦
	bad := []decimal.Decimal{
		decimal.New(1, 1_000_000_000), decimal.New(1, -1_000_000_000), decimal.New(-1, 1_000_000_000),
		decimal.RequireFromString("1e5000"), decimal.RequireFromString("1e11"), decimal.RequireFromString("1234567890123"),
	}
	for _, d := range bad {
		if Sane(d) {
			t.Errorf("Sane(%se%d) = true, want false", d.Coefficient(), d.Exponent())
		}
	}
}

func TestAmount(t *testing.T) {
	if err := Amount("x", decimal.RequireFromString("99999999.99")); err != nil {
		t.Errorf("99999999.99 should pass: %v", err)
	}
	for _, d := range []decimal.Decimal{decimal.RequireFromString("100000000"), decimal.New(1, 1_000_000_000)} {
		if err := Amount("x", d); err == nil {
			t.Errorf("Amount(%se%d) should fail", d.Coefficient(), d.Exponent())
		}
	}
	// 负数放行（符号由调用点管），巨负指数仍拦
	if err := Amount("x", decimal.RequireFromString("-5")); err != nil {
		t.Errorf("-5 should pass Amount: %v", err)
	}
}
