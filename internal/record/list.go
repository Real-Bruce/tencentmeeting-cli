package record

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

// ListParams 是 record list 命令的入参。三组条件必填其一：时间范围 / meeting_id / meeting_code。
type ListParams struct {
	// StartTime 查询起始时间（ISO 8601）。
	StartTime string `json:"start,omitempty"`
	// EndTime 查询结束时间（ISO 8601）。
	EndTime string `json:"end,omitempty"`
	// Page 旧分页参数（deprecated）。
	Page int `json:"page,omitempty"`
	// PageSize 分页大小，默认 30，最大 30。
	PageSize int `json:"page_size,omitempty"`
	// PageToken 分页 token。
	PageToken string `json:"page_token,omitempty"`
	// MeetingID 会议 ID。
	MeetingID string `json:"meeting_id,omitempty"`
	// MeetingCode 会议号。
	MeetingCode string `json:"meeting_code,omitempty"`
}

// List 列出录制记录。
func List(ctx context.Context, tmeet *internal.Tmeet, p ListParams) (*result.Result, error) {
	hasTimeRange := p.StartTime != "" && p.EndTime != ""
	if !hasTimeRange && p.MeetingID == "" && p.MeetingCode == "" {
		return nil, exception.InvalidArgsError.With("one of the following groups is required:\n" +
			"  start + end   (time range)\n" +
			"  meeting_id\n" +
			"  meeting_code")
	}

	queryParams := thttp.QueryParams{}
	queryParams.Set("operator_id", tmeet.UserConfig.OpenId)
	queryParams.Set("operator_id_type", "2") // OpenId

	pageValue, pageType := cmdutil.ChoosePageOrToken(p.Page, p.PageToken)
	queryParams.Set("page_type", strconv.Itoa(pageType))
	if pageType == cmdutil.PageTypeOld {
		queryParams.Set("page", pageValue)
	} else {
		queryParams.Set("page_token", pageValue)
	}
	pageSize, err := cmdutil.ClampPageSize(p.PageSize, cmdutil.PageSizeMaxRecords)
	if err != nil {
		return nil, err
	}
	queryParams.Set("page_size", strconv.Itoa(pageSize))

	if p.MeetingID != "" {
		queryParams.Set("meeting_id", p.MeetingID)
	}
	if p.MeetingCode != "" {
		queryParams.Set("meeting_code", cmdutil.FormatMeetingCode(p.MeetingCode))
	}
	if p.StartTime != "" {
		startTime, err := utils.ISO8601ToTimeStamp(p.StartTime)
		if err != nil {
			return nil, exception.InvalidArgsError.With("start format error: %v", err)
		}
		queryParams.Set("start_time", strconv.FormatInt(startTime, 10))
	}
	if p.EndTime != "" {
		endTime, err := utils.ISO8601ToTimeStamp(p.EndTime)
		if err != nil {
			return nil, exception.InvalidArgsError.With("end format error: %v", err)
		}
		queryParams.Set("end_time", strconv.FormatInt(endTime, 10))
	}

	req := &thttp.Request{
		ApiURI:      "/v1/mcp/records",
		QueryParams: queryParams,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}

	convertMap := map[string]utils.FieldConverter{
		"media_start_time":  utils.TimestampConverter,
		"record_start_time": utils.TimestampConverter,
		"record_end_time":   utils.TimestampConverter,
		"state":             utils.RecordStateConverter,
		"record_type":       utils.RecordTypeConverter,
	}
	data := string(utils.ConvertFields([]byte(rsp.Data), 10, convertMap))
	data = result.StripTotalCountIfZero(data)

	return &result.Result{Data: data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}

// SearchParams 是 record search 命令的入参。
type SearchParams struct {
	// Query 搜索关键字。
	Query string `json:"query,omitempty"`
	// QueryField 搜索字段。
	QueryField string `json:"query_field,omitempty"`
	// FileType 文件类型：video/audio/transcript/upload/external/all。
	FileType string `json:"file_type,omitempty"`
	// MeetingID 会议 ID。
	MeetingID string `json:"meeting_id,omitempty"`
	// MeetingCode 会议号。
	MeetingCode string `json:"meeting_code,omitempty"`
	// From 时间窗下界（ISO 8601）。
	From string `json:"start,omitempty"`
	// To 时间窗上界（ISO 8601）。
	To string `json:"end,omitempty"`
	// PageSize 分页大小，默认 30，最大 30。
	PageSize int `json:"page_size,omitempty"`
	// PageToken 分页 token。
	PageToken string `json:"page_token,omitempty"`
}

// Search 搜索录制记录。
func Search(ctx context.Context, tmeet *internal.Tmeet, p SearchParams) (*result.Result, error) {
	fileType := p.FileType
	if fileType == "" {
		fileType = "all"
	}
	queryParams := thttp.QueryParams{}
	queryParams.Set("operator_id", tmeet.UserConfig.OpenId)
	queryParams.Set("operator_id_type", "2")
	queryParams.Set("file_type", fileType)

	pageSize, err := cmdutil.ClampPageSize(p.PageSize, cmdutil.PageSizeMaxRecords)
	if err != nil {
		return nil, err
	}
	queryParams.Set("page_token", p.PageToken)
	queryParams.Set("page_size", strconv.Itoa(pageSize))

	if p.Query != "" {
		if err := utils.CharacterLimit("query", p.Query, maxQueryLen); err != nil {
			return nil, err
		}
		queryParams.Set("q", p.Query)
		qField := p.QueryField
		if qField == "" {
			qField = "all"
		}
		queryParams.Set("q_fields", qField)
	}
	if p.MeetingID != "" {
		queryParams.Set("meeting_id", p.MeetingID)
	}
	if p.MeetingCode != "" {
		queryParams.Set("meeting_code", cmdutil.FormatMeetingCode(p.MeetingCode))
	}

	var start, end int64
	if p.From != "" {
		start, err = utils.ISO8601ToTimeStamp(p.From)
		if err != nil {
			return nil, exception.InvalidArgsError.With("start format error: %v", err)
		}
		queryParams.Set("from", p.From)
	}
	if p.To != "" {
		end, err = utils.ISO8601ToTimeStamp(p.To)
		if err != nil {
			return nil, exception.InvalidArgsError.With("end format error: %v", err)
		}
		queryParams.Set("to", p.To)
	}
	if start > 0 && end > 0 && start >= end {
		return nil, exception.InvalidArgsError.With("start must be earlier than end")
	}

	req := &thttp.Request{
		ApiURI:      "/v1/mcp/records/search",
		QueryParams: queryParams,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}

	data := result.StripTotalCountIfZero(rsp.Data)
	return &result.Result{Data: data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}

// maxQueryLen 搜索关键字最大长度。
const maxQueryLen = 50
