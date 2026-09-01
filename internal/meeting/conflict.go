package meeting

import (
	"bytes"
	"context"
	"encoding/json"
	"time"
	"tmeet/internal"
	"tmeet/internal/exception"
	"tmeet/internal/result"
	"tmeet/internal/utils"
)

// ConflictParams 是 meeting_conflict 工具的入参，供 MCP BindArguments 绑定。
// Start/End 均为必填，使用 ISO 8601 带时区格式（如 2026-08-31T12:30+08:00）。
type ConflictParams struct {
	// Start 待检测时间段的起始时间（ISO 8601 带时区）。
	Start string `json:"start,omitempty"`
	// End 待检测时间段的结束时间（ISO 8601 带时区，必须晚于 Start）。
	End string `json:"end,omitempty"`
}

// conflictMaxPages 分页拉取会议的安全上限，防止服务端异常时死循环。
// 单页最大 20 条，50 页 ≈ 1000 场，远超单日容量。
const conflictMaxPages = 50

// Conflict 检测给定时间段是否与当前用户的待开始/进行中会议存在时间冲突。
//
// 业务核心与 cobra/stdout 解耦：复用 meeting.List 拉取查询日全天会议，
// 在客户端按"重叠"谓词判定冲突（而非 List 的包含过滤），构造自定义 JSON 返回。
// 仅覆盖待开始/进行中会议（后端 /v1/meetings 不返回已结束历史会议）。
//
// 冲突判定：meetingStart < queryEnd && meetingEnd > queryStart（严格不等号，边界相接不算冲突）。
func Conflict(ctx context.Context, tmeet *internal.Tmeet, p ConflictParams) (*result.Result, error) {
	if p.Start == "" || p.End == "" {
		return nil, exception.InvalidArgsError.With("start and end are required for conflict detection")
	}

	// 解析查询时段，保留时区 Location 用于计算当天边界。
	tStart, err := parseISO8601(p.Start)
	if err != nil {
		return nil, exception.InvalidArgsError.With("--start format error: %v", err)
	}
	tEnd, err := parseISO8601(p.End)
	if err != nil {
		return nil, exception.InvalidArgsError.With("--end format error: %v", err)
	}
	if !tEnd.After(tStart) {
		return nil, exception.InvalidArgsError.With("end must be after start: start=%s end=%s", p.Start, p.End)
	}

	queryStartTs := tStart.Unix()
	queryEndTs := tEnd.Unix()

	// 把查询起始日扩展为当天 00:00（同 Location），作为 meeting.List 的 pos 游标
	// 与客户端 filterStart，拉取当天及以后的全部待开始/进行中会议。
	// 不传 End，避免 List 的 filterEnd 包含过滤误删跨越午夜的会议。
	loc := tStart.Location()
	dayStart := time.Date(tStart.Year(), tStart.Month(), tStart.Day(), 0, 0, 0, 0, loc)
	dayStartISO := dayStart.Format(time.RFC3339)

	// 分页遍历 meeting.List，累加当天全部会议项。
	var allItems []map[string]interface{}
	var traceID string
	pageToken := ""
	for i := 0; i < conflictMaxPages; i++ {
		r, err := List(ctx, tmeet, ListParams{
			Start:     dayStartISO,
			PageToken: pageToken,
			PageSize:  20,
		})
		if err != nil {
			return nil, err
		}
		if traceID == "" {
			traceID = r.TraceId
		}
		items, nextToken, ok := extractMeetingItems(r.Data)
		if !ok {
			break
		}
		allItems = append(allItems, items...)
		if nextToken == "" {
			break
		}
		pageToken = nextToken
	}

	// 客户端重叠判定，收集冲突会议。
	conflicts := detectConflicts(allItems, queryStartTs, queryEndTs)

	// 构造自定义输出，SetEscapeHTML(false) 遵循输出不变量。
	out := map[string]interface{}{
		"meet_conflict":     len(conflicts) > 0,
		"total_count":       len(conflicts),
		"conflict_meetings": conflicts,
	}
	data, err := marshalNoEscape(out)
	if err != nil {
		return nil, exception.InvalidArgsError.With("marshal conflict result failed: %v", err)
	}

	return &result.Result{
		Data:    string(data),
		TraceId: traceID,
		Message: "",
	}, nil
}

// detectConflicts 是冲突判定的纯函数（便于单测，不触网）。
// 对每个会议项解析 start_time/end_time（已是 RFC3339 字符串，由 List 的 ConvertFields 转换），
// 应用重叠谓词 meetingStart < queryEnd && meetingEnd > queryStart。
// 缺失或无法解析时间的项跳过（不计为冲突）。
func detectConflicts(items []map[string]interface{}, queryStartTs, queryEndTs int64) []map[string]interface{} {
	conflicts := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		startStr, _ := item["start_time"].(string)
		endStr, _ := item["end_time"].(string)
		if startStr == "" || endStr == "" {
			continue
		}
		mStart, err := utils.ISO8601ToTimeStamp(startStr)
		if err != nil || mStart <= 0 {
			continue
		}
		mEnd, err := utils.ISO8601ToTimeStamp(endStr)
		if err != nil || mEnd <= 0 {
			continue
		}
		// 严格重叠：边界相接（end==queryStart 或 start==queryEnd）不算冲突。
		if mStart < queryEndTs && mEnd > queryStartTs {
			conflicts = append(conflicts, item)
		}
	}
	return conflicts
}

// extractMeetingItems 从 meeting.List 返回的 Data JSON 中提取 meeting_info_list 与 next_page_token。
// 解析失败时返回 ok=false，由调用方决定是否中止遍历。
func extractMeetingItems(data string) (items []map[string]interface{}, nextToken string, ok bool) {
	var root map[string]interface{}
	if err := json.Unmarshal([]byte(data), &root); err != nil {
		return nil, "", false
	}
	rawList, found := root["meeting_info_list"].([]interface{})
	if !found {
		// 列表字段缺失视为空结果页（非错误），正常结束遍历。
		return nil, "", true
	}
	items = make([]map[string]interface{}, 0, len(rawList))
	for _, raw := range rawList {
		if m, ok := raw.(map[string]interface{}); ok {
			items = append(items, m)
		}
	}
	if t, ok := root["next_page_token"].(string); ok {
		nextToken = t
	}
	return items, nextToken, true
}

// parseISO8601 解析 ISO 8601 带时区的时间字符串，返回保留 Location 的 time.Time。
// 与 utils.ISO8601ToTimeStamp 支持的格式一致：RFC3339 与无秒格式。
func parseISO8601(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339,             // 2006-01-02T15:04:05Z07:00
		"2006-01-02T15:04Z07:00", // 2026-03-12T14:00+08:00（无秒）
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, exception.InvalidArgsError.With("failed to parse time format: %s, supported formats: 2006-01-02T15:04:05Z07:00 or 2006-01-02T15:04Z07:00", s)
}

// marshalNoEscape 将值编码为 JSON 且不转义 HTML 字符（& < >），
// 镜像 internal/output/print.go 的 marshalNoEscape，遵循输出不变量。
func marshalNoEscape(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	// json.Encoder.Encode 末尾会追加换行，去除以保持 Data 紧凑。
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
