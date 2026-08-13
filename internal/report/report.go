package report

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

// ParticipantsParams 是 report participants 命令的入参。
type ParticipantsParams struct {
	// MeetingID 会议 ID（必填）。
	MeetingID string `json:"meeting_id"`
	// SubMeetingID 子会议 ID（周期性会议）。
	SubMeetingID string `json:"sub_meeting_id,omitempty"`
	// Pos 旧分页起始位置（deprecated）。
	Pos int `json:"pos,omitempty"`
	// Size 旧分页大小（deprecated）。
	Size int `json:"size,omitempty"`
	// StartTime 查询起始时间（ISO 8601）。
	StartTime string `json:"start,omitempty"`
	// EndTime 查询结束时间（ISO 8601）。
	EndTime string `json:"end,omitempty"`
	// PageToken 分页 token。
	PageToken string `json:"page_token,omitempty"`
	// PageSize 分页大小，默认 100，最大 100。
	PageSize int `json:"page_size,omitempty"`
}

// Participants 查询会议参会人列表。
func Participants(ctx context.Context, tmeet *internal.Tmeet, p ParticipantsParams) (*result.Result, error) {
	queryParams := thttp.QueryParams{}
	queryParams.Set("operator_id", tmeet.UserConfig.OpenId)
	queryParams.Set("operator_id_type", "2")

	pageValue, pageType := cmdutil.ChoosePosOrToken(p.Pos, p.PageToken)
	queryParams.Set("page_type", strconv.Itoa(pageType))
	if pageType == cmdutil.PageTypeOld {
		queryParams.Set("pos", pageValue)
		queryParams.Set("size", strconv.Itoa(p.Size))
	} else {
		queryParams.Set("page_token", pageValue)
		pageSize, err := cmdutil.ClampPageSize(p.PageSize, cmdutil.PageSizeMaxReports)
		if err != nil {
			return nil, err
		}
		queryParams.Set("page_size", strconv.Itoa(pageSize))
	}

	if p.SubMeetingID != "" {
		queryParams.Set("sub_meeting_id", p.SubMeetingID)
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
		ApiURI:      "/v1/meetings/{meeting_id}/participants",
		PathParams:  thttp.PathParams{"meeting_id": p.MeetingID},
		QueryParams: queryParams,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}

	convertMap := map[string]utils.FieldConverter{
		"join_time":           utils.TimestampConverter,
		"left_time":           utils.TimestampConverter,
		"schedule_start_time": utils.TimestampConverter,
		"schedule_end_time":   utils.TimestampConverter,
		"instanceid":          utils.InstanceIdConverter,
		"user_name":           utils.Base64DecodeConverter,
		"subject":             utils.Base64DecodeConverter,
		"user_role":           utils.MeetingUserRoleConverter,
	}
	data := string(utils.ConvertFields([]byte(rsp.Data), 10, convertMap))
	data = result.StripMetaFilterField(data)

	return &result.Result{Data: data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}

// WaitingRoomLogParams 是 report waiting-room-log 命令的入参。
type WaitingRoomLogParams struct {
	// MeetingID 会议 ID（必填）。
	MeetingID string `json:"meeting_id"`
	// PageSize 分页大小，默认 100，最大 100。
	PageSize int `json:"page_size,omitempty"`
	// Page 旧分页参数（deprecated）。
	Page int `json:"page,omitempty"`
	// PageToken 分页 token。
	PageToken string `json:"page_token,omitempty"`
}

// WaitingRoomLog 查询会议等候室成员日志。
func WaitingRoomLog(ctx context.Context, tmeet *internal.Tmeet, p WaitingRoomLogParams) (*result.Result, error) {
	queryParams := thttp.QueryParams{}
	queryParams.Set("operator_id", tmeet.UserConfig.OpenId)
	queryParams.Set("operator_id_type", "2")

	pageValue, pageType := cmdutil.ChoosePageOrToken(p.Page, p.PageToken)
	queryParams.Set("page_type", strconv.Itoa(pageType))
	if pageType == cmdutil.PageTypeOld {
		queryParams.Set("page", pageValue)
	} else {
		queryParams.Set("page_token", pageValue)
	}
	pageSize, err := cmdutil.ClampPageSize(p.PageSize, cmdutil.PageSizeMaxReports)
	if err != nil {
		return nil, err
	}
	queryParams.Set("page_size", strconv.Itoa(pageSize))

	req := &thttp.Request{
		ApiURI:      "/v1/meeting/{meeting_id}/waiting-room",
		PathParams:  thttp.PathParams{"meeting_id": p.MeetingID},
		QueryParams: queryParams,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}

	convertMap := map[string]utils.FieldConverter{
		"join_time":           utils.TimestampConverter,
		"left_time":           utils.TimestampConverter,
		"schedule_start_time": utils.TimestampConverter,
		"schedule_end_time":   utils.TimestampConverter,
		"instanceid":          utils.InstanceIdConverter,
		"user_name":           utils.Base64DecodeConverter,
		"subject":             utils.Base64DecodeConverter,
	}
	data := string(utils.ConvertFields([]byte(rsp.Data), 10, convertMap))

	return &result.Result{Data: data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}
