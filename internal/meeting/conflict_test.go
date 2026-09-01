package meeting

import (
	"strings"
	"testing"
	"time"
)

// 查询窗口统一用 2026-08-31T10:20+08:00 ~ 2026-08-31T11:20+08:00（用户示例）。
const (
	conflictQueryStart = "2026-08-31T10:20+08:00"
	conflictQueryEnd   = "2026-08-31T11:20+08:00"
)

func queryWindow(t *testing.T) (int64, int64) {
	t.Helper()
	ts, err := parseISO8601(conflictQueryStart)
	if err != nil {
		t.Fatalf("parse query start: %v", err)
	}
	te, err := parseISO8601(conflictQueryEnd)
	if err != nil {
		t.Fatalf("parse query end: %v", err)
	}
	return ts.Unix(), te.Unix()
}

func TestDetectConflicts(t *testing.T) {
	qStart, qEnd := queryWindow(t)

	// 会议项的时间字段为 List ConvertFields 转换后的 RFC3339 字符串。
	mk := func(start, end string) map[string]interface{} {
		return map[string]interface{}{
			"meeting_id":   "id-" + start,
			"meeting_code": "111-111-111",
			"subject":      "test",
			"start_time":  start,
			"end_time":    end,
		}
	}

	tests := []struct {
		name   string
		items  []map[string]interface{}
		expect int // 期望冲突数
	}{
		{
			name: "overlap crossing right boundary (user example 11:00-11:30)",
			items: []map[string]interface{}{
				mk("2026-08-31T11:00+08:00", "2026-08-31T11:30+08:00"),
			},
			expect: 1,
		},
		{
			name: "fully contained inside query window",
			items: []map[string]interface{}{
				mk("2026-08-31T10:40+08:00", "2026-08-31T11:00+08:00"),
			},
			expect: 1,
		},
		{
			name: "fully containing query window",
			items: []map[string]interface{}{
				mk("2026-08-31T10:00+08:00", "2026-08-31T12:00+08:00"),
			},
			expect: 1,
		},
		{
			name: "overlap crossing left boundary (09:00-10:30)",
			items: []map[string]interface{}{
				mk("2026-08-31T09:00+08:00", "2026-08-31T10:30+08:00"),
			},
			expect: 1,
		},
		{
			name: "adjacent before (ends exactly at query start) — no conflict",
			items: []map[string]interface{}{
				mk("2026-08-31T09:00+08:00", "2026-08-31T10:20+08:00"),
			},
			expect: 0,
		},
		{
			name: "adjacent after (starts exactly at query end) — no conflict",
			items: []map[string]interface{}{
				mk("2026-08-31T11:20+08:00", "2026-08-31T12:00+08:00"),
			},
			expect: 0,
		},
		{
			name: "disjoint before — no conflict",
			items: []map[string]interface{}{
				mk("2026-08-31T08:00+08:00", "2026-08-31T09:00+08:00"),
			},
			expect: 0,
		},
		{
			name: "disjoint after — no conflict",
			items: []map[string]interface{}{
				mk("2026-08-31T14:00+08:00", "2026-08-31T15:00+08:00"),
			},
			expect: 0,
		},
		{
			name: "multiple mixed — only overlapping ones conflict",
			items: []map[string]interface{}{
				mk("2026-08-31T08:00+08:00", "2026-08-31T09:00+08:00"), // disjoint before
				mk("2026-08-31T11:00+08:00", "2026-08-31T11:30+08:00"), // overlap
				mk("2026-08-31T11:20+08:00", "2026-08-31T12:00+08:00"), // adjacent after
				mk("2026-08-31T10:00+08:00", "2026-08-31T12:00+08:00"), // containing
			},
			expect: 2,
		},
		{
			name: "missing start_time — skipped",
			items: []map[string]interface{}{
				{"end_time": "2026-08-31T11:30+08:00"},
			},
			expect: 0,
		},
		{
			name: "invalid time string — skipped",
			items: []map[string]interface{}{
				mk("not-a-time", "2026-08-31T11:30+08:00"),
			},
			expect: 0,
		},
		{
			name:   "empty list",
			items:  nil,
			expect: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectConflicts(tt.items, qStart, qEnd)
			if len(got) != tt.expect {
				t.Fatalf("detectConflicts: expected %d conflicts, got %d (%v)", tt.expect, len(got), got)
			}
		})
	}
}

func TestParseISO8601AndDayStart(t *testing.T) {
	// 验证 +08:00 时区下 dayStart 计算为当天 00:00。
	ts, err := parseISO8601("2026-08-31T12:30+08:00")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	loc := ts.Location()
	dayStart := time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, loc)
	want := "2026-08-31T00:00:00+08:00"
	if got := dayStart.Format(time.RFC3339); got != want {
		t.Fatalf("dayStart: want %s, got %s", want, got)
	}

	// 无秒格式同样可解析。
	if _, err := parseISO8601("2026-08-31T12:30+08:00"); err != nil {
		t.Fatalf("parse no-seconds format: %v", err)
	}
	// 缺失时区应解析失败。
	if _, err := parseISO8601("2026-08-31T12:30:00"); err == nil {
		t.Fatalf("expected error for missing timezone, got nil")
	}
}

func TestMarshalNoEscape(t *testing.T) {
	// 验证 & 不被转义为 &（遵循输出不变量）。
	out, err := marshalNoEscape(map[string]interface{}{"subject": "A & B < C>"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "A & B < C>") {
		t.Fatalf("expected raw HTML chars preserved, got %s", got)
	}
}
