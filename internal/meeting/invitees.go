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
)

// --- invitees-add ---

// InviteesAddParams 是 meeting invitees-add 命令的入参。
type InviteesAddParams struct {
	// MeetingID 会议 ID（必填）。
	MeetingID string `json:"meeting_id"`
	// Invitees 要添加的 invitee open_id 列表（必填，最多 100）。
	Invitees []string `json:"invitees"`
}

// InviteesAdd 向会议添加受邀者。
func InviteesAdd(ctx context.Context, tmeet *internal.Tmeet, p InviteesAddParams) (*result.Result, error) {
	if len(p.Invitees) == 0 {
		return nil, exception.InvalidArgsError.With("invitees is required")
	}
	inviteesList, err := cmdutil.PackageApiInviteesUsers("invitees", p.Invitees)
	if err != nil {
		return nil, err
	}
	params := map[string]interface{}{
		"operator_id":      tmeet.UserConfig.OpenId,
		"operator_id_type": 2, // OpenId
		"instanceid":       1, // PC, fixed value
		"add_invitees":     inviteesList,
	}
	req := &thttp.Request{
		ApiURI: "/v1/meetings/{meeting_id}/modify-invitees",
		Body:   params,
		PathParams: thttp.PathParams{
			"meeting_id": p.MeetingID,
		},
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodPut, tmeet, req)
	if err != nil {
		return nil, err
	}
	return &result.Result{Data: rsp.Data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}

// --- invitees-remove ---

// InviteesRemoveParams 是 meeting invitees-remove 命令的入参。
type InviteesRemoveParams struct {
	// MeetingID 会议 ID（必填）。
	MeetingID string `json:"meeting_id"`
	// Invitees 要移除的 invitee open_id 列表（必填，最多 100）。
	Invitees []string `json:"invitees"`
}

// InviteesRemove 从会议移除受邀者。
func InviteesRemove(ctx context.Context, tmeet *internal.Tmeet, p InviteesRemoveParams) (*result.Result, error) {
	if len(p.Invitees) == 0 {
		return nil, exception.InvalidArgsError.With("invitees is required")
	}
	inviteesList, err := cmdutil.PackageApiInviteesUsers("invitees", p.Invitees)
	if err != nil {
		return nil, err
	}
	params := map[string]interface{}{
		"operator_id":      tmeet.UserConfig.OpenId,
		"operator_id_type": 2, // OpenId
		"instanceid":       1, // PC, fixed value
		"delete_invitees":  inviteesList,
	}
	req := &thttp.Request{
		ApiURI: "/v1/meetings/{meeting_id}/modify-invitees",
		Body:   params,
		PathParams: thttp.PathParams{
			"meeting_id": p.MeetingID,
		},
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodPut, tmeet, req)
	if err != nil {
		return nil, err
	}
	return &result.Result{Data: rsp.Data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}

// --- invitees-replace ---

// InviteesReplaceParams 是 meeting invitees-replace 命令的入参。
type InviteesReplaceParams struct {
	// MeetingID 会议 ID（必填）。
	MeetingID string `json:"meeting_id"`
	// Invitees 替换后的完整 invitee open_id 列表，留空表示清空。
	Invitees []string `json:"invitees,omitempty"`
}

// InviteesReplace 替换会议的全部受邀者。
func InviteesReplace(ctx context.Context, tmeet *internal.Tmeet, p InviteesReplaceParams) (*result.Result, error) {
	inviteesList, err := cmdutil.PackageApiInviteesUsers("invitees", p.Invitees)
	if err != nil {
		return nil, err
	}
	params := map[string]interface{}{
		"meeting_id": p.MeetingID,
		"userid":     tmeet.UserConfig.OpenId,
		"instanceid": 1, // PC, fixed value
		"invitees":   inviteesList,
	}
	req := &thttp.Request{
		ApiURI: "/v1/meetings/{meeting_id}/invitees",
		Body:   params,
		PathParams: thttp.PathParams{
			"meeting_id": p.MeetingID,
		},
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodPut, tmeet, req)
	if err != nil {
		return nil, err
	}
	return &result.Result{Data: rsp.Data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}

// --- invitees-list ---

// InviteesListParams 是 meeting invitees-list 命令的入参。
type InviteesListParams struct {
	// MeetingID 会议 ID（必填）。
	MeetingID string `json:"meeting_id"`
	// Pos 旧分页起始位置（deprecated，用 PageToken 代替）。
	Pos int `json:"pos,omitempty"`
	// PageToken 分页 token。
	PageToken string `json:"page_token,omitempty"`
	// PageSize 分页大小，默认 30，最大 30。
	PageSize int `json:"page_size,omitempty"`
}

// InviteesList 查询会议的受邀者列表。
func InviteesList(ctx context.Context, tmeet *internal.Tmeet, p InviteesListParams) (*result.Result, error) {
	queryParams := thttp.QueryParams{}
	queryParams.Set("userid", tmeet.UserConfig.OpenId)
	queryParams.Set("instanceid", "1") // PC, fixed value

	// pagination compatibility
	pageValue, pageType := cmdutil.ChoosePosOrToken(p.Pos, p.PageToken)
	queryParams.Set("page_type", strconv.Itoa(pageType))
	if pageType == cmdutil.PageTypeOld {
		queryParams.Set("pos", pageValue)
	} else {
		queryParams.Set("page_token", pageValue)
		pageSize, err := cmdutil.ClampPageSize(p.PageSize, cmdutil.PageSizeMaxMeetings)
		if err != nil {
			return nil, err
		}
		queryParams.Set("page_size", strconv.Itoa(pageSize))
	}

	req := &thttp.Request{
		ApiURI:      "/v1/meetings/{meeting_id}/invitees",
		QueryParams: queryParams,
		PathParams: thttp.PathParams{
			"meeting_id": p.MeetingID,
		},
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}
	return &result.Result{Data: rsp.Data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}
