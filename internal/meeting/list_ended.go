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

// ListEndedParams 是 meeting list-ended 命令的入参。
type ListEndedParams struct {
	// StartTime 查询起始时间（ISO 8601）。
	StartTime string `json:"start,omitempty"`
	// EndTime 查询结束时间（ISO 8601）。
	EndTime string `json:"end,omitempty"`
	// Page 旧分页参数（从 1 开始，deprecated，用 PageToken 代替）。
	Page int `json:"page,omitempty"`
	// PageSize 分页大小，默认 30，最大 30。
	PageSize int `json:"page_size,omitempty"`
	// PageToken 分页 token。
	PageToken string `json:"page_token,omitempty"`
}

// ListEnded 列出已结束的会议。
func ListEnded(ctx context.Context, tmeet *internal.Tmeet, p ListEndedParams) (*result.Result, error) {
	queryParams := thttp.QueryParams{}

	// pagination compatibility
	pageValue, pageType := cmdutil.ChoosePageOrToken(p.Page, p.PageToken)
	queryParams.Set("page_type", strconv.Itoa(pageType))
	if pageType == cmdutil.PageTypeOld {
		queryParams.Set("page", pageValue)
	} else {
		queryParams.Set("page_token", pageValue)
	}
	pageSize, err := clampPageSize(p.PageSize, cmdutil.PageSizeMaxMeetings)
	if err != nil {
		return nil, err
	}
	queryParams.Set("page_size", strconv.Itoa(pageSize))

	if p.StartTime != "" {
		startTs, err := utils.ISO8601ToTimeStamp(p.StartTime)
		if err != nil {
			return nil, exception.InvalidArgsError.With("--start format error: %v", err)
		}
		queryParams.Set("start_time", strconv.FormatInt(startTs, 10))
	}
	if p.EndTime != "" {
		endTs, err := utils.ISO8601ToTimeStamp(p.EndTime)
		if err != nil {
			return nil, exception.InvalidArgsError.With("--end format error: %v", err)
		}
		queryParams.Set("end_time", strconv.FormatInt(endTs, 10))
	}

	req := &thttp.Request{
		ApiURI:      "/v1/history/meetings/{userid}",
		QueryParams: queryParams,
		PathParams:  thttp.PathParams{"userid": tmeet.UserConfig.OpenId},
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}

	convertMap := map[string]utils.FieldConverter{
		"start_time":   utils.TimestampConverter,
		"end_time":     utils.TimestampConverter,
		"meeting_type": utils.MeetingTypeConverter,
	}
	data := string(utils.ConvertFields([]byte(rsp.Data), 10, convertMap))

	return &result.Result{
		Data:    data,
		TraceId: rsp.TraceId,
		Message: rsp.Message,
	}, nil
}
