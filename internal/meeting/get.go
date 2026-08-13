package meeting

import (
	"context"
	"net/http"
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	"tmeet/internal/core/thttp"
	"tmeet/internal/exception"
	restProxy "tmeet/internal/proxy/rest-proxy"
	"tmeet/internal/result"
	"tmeet/internal/utils"
)

// GetParams 是 meeting get 命令的入参。MeetingID 优先级高于 MeetingCode，二者必填其一。
type GetParams struct {
	// MeetingID 会议 ID，优先级高于 MeetingCode。
	MeetingID string `json:"meeting_id,omitempty"`
	// MeetingCode 会议号（对外展示用）。
	MeetingCode string `json:"meeting_code,omitempty"`
}

// Get 查询会议详情。按 MeetingID / MeetingCode 分别走不同后端路径（对应 FlagSwitch 路由）。
func Get(ctx context.Context, tmeet *internal.Tmeet, p GetParams) (*result.Result, error) {
	var apiURI string
	pathParams := thttp.PathParams{}
	queryParams := thttp.QueryParams{}
	if p.MeetingID != "" {
		apiURI = "/v1/meetings/{meetingId}"
		pathParams.Set("meetingId", p.MeetingID)
		queryParams.Set("instanceid", "1") // PC, fixed value
		queryParams.Set("operator_id", tmeet.UserConfig.OpenId)
		queryParams.Set("operator_id_type", "2") // OpenId
	} else if p.MeetingCode != "" {
		apiURI = "/v1/meetings"
		pathParams = nil
		queryParams.Set("userid", tmeet.UserConfig.OpenId)
		queryParams.Set("instanceid", "1") // PC, fixed value
		queryParams.Set("meeting_code", cmdutil.FormatMeetingCode(p.MeetingCode))
	} else {
		return nil, exception.InvalidArgsError.With("meeting_id or meeting_code is required")
	}

	req := &thttp.Request{
		ApiURI:      apiURI,
		QueryParams: queryParams,
		PathParams:  pathParams,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}

	convertMap := map[string]utils.FieldConverter{
		"start_time":               utils.TimestampConverter,
		"end_time":                 utils.TimestampConverter,
		"meeting_info_list.status": utils.MeetingStatusConverter,
		"meeting_type":             utils.MeetingTypeConverter,
		"recurring_type":           utils.MeetingRecurringTypeConverter,
		"time_zone":                utils.Base64DecodeConverter,
		"until_date":               utils.TimestampConverter,
		"until_type":               utils.MeetingRecurringUntilTypeConverter,
	}
	data := string(utils.ConvertFields([]byte(rsp.Data), 10, convertMap))

	return &result.Result{
		Data:    data,
		TraceId: rsp.TraceId,
		Message: rsp.Message,
	}, nil
}
