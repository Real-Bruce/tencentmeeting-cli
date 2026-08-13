package meeting

import (
	"context"
	"net/http"
	"strconv"
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	"tmeet/internal/core/thttp"
	"tmeet/internal/exception"
	restProxy "tmeet/internal/proxy/rest-proxy"
	"tmeet/internal/result"
	"tmeet/internal/utils"
)

// maxQueryLen 搜索关键字最大长度。
const maxQueryLen = 50

// SearchParams 是 meeting search 命令的入参。
type SearchParams struct {
	// Query 搜索关键字。
	Query string `json:"query,omitempty"`
	// QueryField 搜索字段：subject/creator/note/all。
	QueryField string `json:"query_field,omitempty"`
	// MeetingCode 按会议号精确过滤。
	MeetingCode string `json:"meeting_code,omitempty"`
	// From 搜索时间窗下界（ISO 8601）。
	From string `json:"start,omitempty"`
	// To 搜索时间窗上界（ISO 8601）。
	To string `json:"end,omitempty"`
	// PageSize 分页大小，默认 30，最大 30。
	PageSize int `json:"page_size,omitempty"`
	// PageToken 分页 token。
	PageToken string `json:"page_token,omitempty"`
}

// Search 按关键字/会议号/时间范围/状态搜索会议。
func Search(ctx context.Context, tmeet *internal.Tmeet, p SearchParams) (*result.Result, error) {
	queryParams := thttp.QueryParams{}
	queryParams.Set("operator_id", tmeet.UserConfig.OpenId)
	queryParams.Set("operator_id_type", "2") // OpenId

	// page handler
	pageSize, err := cmdutil.ClampPageSize(p.PageSize, cmdutil.PageSizeMaxMeetings)
	if err != nil {
		return nil, err
	}
	queryParams.Set("page_token", p.PageToken)
	queryParams.Set("page_size", strconv.Itoa(pageSize))

	// other params
	if p.Query != "" {
		if err := utils.CharacterLimit("--query", p.Query, maxQueryLen); err != nil {
			return nil, err
		}
		queryParams.Set("q", p.Query)
		queryParams.Set("q_fields", p.QueryField)
	}
	if p.MeetingCode != "" {
		queryParams.Set("meeting_code", cmdutil.FormatMeetingCode(p.MeetingCode))
	}

	// time range
	var start, end int64
	if p.From != "" {
		start, err = utils.ISO8601ToTimeStamp(p.From)
		if err != nil {
			return nil, exception.InvalidArgsError.With("--start format error: %v", err)
		}
		queryParams.Set("from", p.From)
	}
	if p.To != "" {
		end, err = utils.ISO8601ToTimeStamp(p.To)
		if err != nil {
			return nil, exception.InvalidArgsError.With("--end format error: %v", err)
		}
		queryParams.Set("to", p.To)
	}
	if start > 0 && end > 0 && start >= end {
		return nil, exception.InvalidArgsError.With("--start must be earlier than --end")
	}

	req := &thttp.Request{
		ApiURI:      "/v1/meetings/search-meetings",
		QueryParams: queryParams,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}

	convertMap := map[string]utils.FieldConverter{
		"meetings.meeting_type": utils.MeetingTypeConverter,
		"meetings.status":       utils.MeetingStatusInSearchConverter,
	}
	data := string(utils.ConvertFields([]byte(rsp.Data), 10, convertMap))
	// total_count=0 被剥离（服务器返回 0 通常是占位，不能展示给用户）。
	data = result.StripTotalCountIfZero(data)

	return &result.Result{
		Data:    data,
		TraceId: rsp.TraceId,
		Message: rsp.Message,
	}, nil
}
