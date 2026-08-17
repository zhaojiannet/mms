// Package decx 校验请求携带的 decimal 字段。
//
// shopspring/decimal 的 rescale 按指数差构造 big.Int：与任何值比较、Round、
// 加减之前必须先粗筛，否则 "1e1000000000" 这类巨指数载荷（含与 0 比较、
// 系数为 0 的情况）单请求即可放大出数 GB 内存。所有 Bind 进来的 decimal
// 在参与任何运算前必须先过 Sane / Amount。
package decx

import (
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/shopspring/decimal"
)

// MaxAmount 是 NUMERIC(10,2) 能存的最大金额；落库前拦下，不让 pg 溢出变 500
var MaxAmount = decimal.New(9999999999, -2)

// Sane 粗筛：指数 ±10 以内、系数不超 12 位，覆盖一切合法金额与比率。
// 只保证「参与后续 decimal 运算安全」，符号与业务范围由调用点自查
func Sane(d decimal.Decimal) bool {
	return d.Exponent() >= -10 && d.Exponent() <= 10 && d.NumDigits() <= 12
}

// Amount 校验金额字段：粗筛 + NUMERIC(10,2) 上限，返回 400（nil 为通过）。
// 负数是否合法各字段不同，仍由调用点自查
func Amount(name string, d decimal.Decimal) error {
	if !Sane(d) || d.GreaterThan(MaxAmount) {
		return echo.NewHTTPError(http.StatusBadRequest, name+" 超出可受理范围")
	}
	return nil
}
