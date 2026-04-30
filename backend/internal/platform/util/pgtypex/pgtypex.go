// Package pgtypex 集中处理 pgtype 转换，避免各 handler 重复实现。
package pgtypex

import (
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// OptUUID 把可选 *uuid.UUID 转 pgtype.UUID。nil 或 uuid.Nil 都视为无效。
func OptUUID(p *uuid.UUID) pgtype.UUID {
	if p == nil || *p == uuid.Nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *p, Valid: true}
}

// UUID 把必填 uuid.UUID 转 pgtype.UUID。
func UUID(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

// StrPtr 空串转 nil，否则返回 *string。
func StrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
