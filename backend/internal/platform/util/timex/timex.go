// Package timex 集中处理时间/日期解析，避免各 handler 重复实现。
package timex

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// ParseRFC3339Tz 解析 *string（可空 / 空串视为 NULL）为 pgtype.Timestamptz。
// 用于 handler 接收 JSON 里的 ISO 时间字段。
func ParseRFC3339Tz(s *string) (pgtype.Timestamptz, error) {
	if s == nil || *s == "" {
		return pgtype.Timestamptz{Valid: false}, nil
	}
	t, err := time.Parse(time.RFC3339, *s)
	if err != nil {
		return pgtype.Timestamptz{}, err
	}
	return pgtype.Timestamptz{Time: t, Valid: true}, nil
}

// ParseRFC3339TzStr 同 ParseRFC3339Tz，参数为裸字符串（空串视为 NULL）。
func ParseRFC3339TzStr(s string) (pgtype.Timestamptz, error) {
	if s == "" {
		return pgtype.Timestamptz{Valid: false}, nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return pgtype.Timestamptz{}, err
	}
	return pgtype.Timestamptz{Time: t, Valid: true}, nil
}

// ParseDate 解析 *string（YYYY-MM-DD，空串视为 NULL）为 pgtype.Date。
func ParseDate(s *string) (pgtype.Date, error) {
	if s == nil || *s == "" {
		return pgtype.Date{Valid: false}, nil
	}
	t, err := time.Parse("2006-01-02", *s)
	if err != nil {
		return pgtype.Date{}, err
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}
