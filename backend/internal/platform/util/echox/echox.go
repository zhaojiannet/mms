// Package echox 集中处理 echo 请求参数解析，避免各 handler 重复实现。
package echox

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

// ParseInt32 解析查询参数：空串/非法/<=0 都返回 def，超过 max 截断到 max。
func ParseInt32(s string, def, max int32) int32 {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return def
	}
	if int32(v) > max {
		return max
	}
	return int32(v)
}

// Paging 一次解出 page/limit/offset（page 默认 1，limit 范围 [def, max]）。
func Paging(c *echo.Context, def, max int32) (page, limit, offset int32) {
	page = ParseInt32(c.QueryParam("page"), 1, 100000)
	limit = ParseInt32(c.QueryParam("limit"), def, max)
	offset = (page - 1) * limit
	return
}

// ParamUUID 解析路径参数为 uuid.UUID，失败返回 400。
func ParamUUID(c *echo.Context, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, "invalid "+name)
	}
	return id, nil
}
