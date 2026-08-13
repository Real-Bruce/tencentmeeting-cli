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

// ListParams 是 meeting list 命令的入参，与 cobra flag 一一对应，
// 供 cobra 层与 MCP 层共用。json tag 供 MCP BindArguments 绑定。
type ListParams struct {
	// Start 查询起始时间（ISO 8601），对应 --start。
	Start string `json:"start,omitempty"`
	// End 查询结束时间（ISO 8601），对应 --end。
	End string `json:"end,omitempty"`
	// ShowAllSub 是否展示所有子会议：0-否，1-是，默认 0。
	ShowAllSub int `json:"show_all_sub,omitempty"`
	// PageToken 分页 token。
	PageToken string `json:"page_token,omitempty"`
	// PageSize 分页大小，最大 20。
	PageSize int `json:"page_size,omitempty"`
}

// pageSizeMaxMeetings meeting list 的 page-size 上限为 20（特殊值，非全局 30）。
const pageSizeMaxMeetings = 20

// List 列出待开始/进行中的会议。业务核心与 cobra/stdout 解耦：
// 组装请求 → 调用 restProxy → 客户端时间范围过滤 → 字段转换 → 返回 result.Result。
// cobra 层据此调用 output.FormatPrint，MCP 层据此直接返回 tool result。
func List(ctx context.Context, tmeet *internal.Tmeet, p ListParams) (*result.Result, error) {
	queryParams := thttp.QueryParams{}
	queryParams.Set("userid", tmeet.UserConfig.OpenId)
	queryParams.Set("instanceid", "1") // PC, fixed value

	// page handler, special page-size max 20
	pageSize, err := clampPageSize(p.PageSize, pageSizeMaxMeetings)
	if err != nil {
		return nil, err
	}
	queryParams.Set("page_size", strconv.Itoa(pageSize))
	queryParams.Set("page_token", p.PageToken)
	queryParams.Set("page_type", strconv.Itoa(cmdutil.PageTypeToken))

	// Filter meeting_info_list by the requested start/end time range.
	var filterStart, filterEnd int64
	if p.Start != "" {
		posTs, err := utils.ISO8601ToTimeStamp(p.Start)
		if err != nil {
			return nil, exception.InvalidArgsError.With("--start format error: %v", err)
		}
		filterStart = posTs
		if p.PageToken == "" {
			queryParams.Set("pos", strconv.FormatInt(posTs, 10))
		}
	}
	if p.End != "" {
		cursoryTs, err := utils.ISO8601ToTimeStamp(p.End)
		if err != nil {
			return nil, exception.InvalidArgsError.With("--end format error: %v", err)
		}
		// Millisecond-level timestamp, needs conversion.
		filterEnd = cursoryTs
	}
	if p.ShowAllSub != 0 {
		queryParams.Set("is_show_all_sub_meetings", strconv.Itoa(p.ShowAllSub))
	}

	req := &thttp.Request{
		ApiURI:      "/v1/meetings",
		QueryParams: queryParams,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}

	data := rsp.Data
	if filterStart > 0 || filterEnd > 0 {
		data = string(utils.FilterMeetingsByTimeRange(
			[]byte(data), "meeting_info_list", "start_time", "end_time", filterStart, filterEnd))
	}

	convertMap := map[string]utils.FieldConverter{
		"start_time":               utils.TimestampConverter,
		"end_time":                 utils.TimestampConverter,
		"meeting_info_list.status": utils.MeetingStatusConverter,
		"meeting_type":             utils.MeetingTypeConverter,
		"recurring_type":           utils.MeetingRecurringTypeConverter,
		"until_date":               utils.TimestampConverter,
		"until_type":               utils.MeetingRecurringUntilTypeConverter,
		"join_meeting_role":        utils.MeetingUserJoinRoleConverter,
		"only_user_join_type":      utils.MeetingJoinTypeConverter,
		"is_show_all_sub_meetings": utils.ShowAllSubMeetingsConverter,
	}
	data = string(utils.ConvertFields([]byte(data), 10, convertMap))

	return &result.Result{
		Data:    data,
		TraceId: rsp.TraceId,
		Message: rsp.Message,
	}, nil
}

// clampPageSize 校验并钳制 page-size，与 cmdutil.ClampingPageSize 等价但不依赖 cobra。
// pageSize < 1 返回错误；超过 max 钳制为 max（静默，不打印告警，告警属于 CLI 层关注）。
func clampPageSize(pageSize, max int) (int, error) {
	if pageSize < 1 {
		return 0, exception.InvalidArgsError.With(
			"illegal page size, must larger equal than 1 and less equal than %d", max)
	}
	if pageSize > max {
		return max, nil
	}
	return pageSize, nil
}
