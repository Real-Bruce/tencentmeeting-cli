package control

import (
	"context"
	"net/http"
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	"tmeet/internal/core/thttp"
	"tmeet/internal/exception"
	restProxy "tmeet/internal/proxy/rest-proxy"
	"tmeet/internal/result"
)

// CallParams 是 control call 命令的入参。
type CallParams struct {
	// MeetingID 会议 ID（必填）。
	MeetingID string `json:"meeting_id"`
	// Users 要呼叫的用户 open_id 列表（必填，最多 20）。
	Users []string `json:"users"`
}

// Call 呼叫成员入会（会中控制）。
func Call(ctx context.Context, tmeet *internal.Tmeet, p CallParams) (*result.Result, error) {
	if len(p.Users) == 0 {
		return nil, exception.InvalidArgsError.With("users is required")
	}
	callList, err := cmdutil.PackageMeetingControlUsers("users", p.Users)
	if err != nil {
		return nil, err
	}

	params := map[string]interface{}{
		"operator_id":      tmeet.UserConfig.OpenId,
		"operator_id_type": 2, // OpenId
		"users":            callList,
	}

	req := &thttp.Request{
		ApiURI: "/v1/meetings/{meeting_id}/batch-call",
		Body:   params,
		PathParams: thttp.PathParams{
			"meeting_id": p.MeetingID,
		},
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodPost, tmeet, req)
	if err != nil {
		return nil, err
	}
	return &result.Result{Data: rsp.Data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}

// KickParams 是 control kick 命令的入参。
type KickParams struct {
	// MeetingID 会议 ID（必填）。
	MeetingID string `json:"meeting_id"`
	// AllowRejoin 被踢出后是否允许重新入会，默认 true。
	AllowRejoin bool `json:"allow_rejoin,omitempty"`
	// Users 要踢出的用户 open_id 列表。
	Users []string `json:"users,omitempty"`
	// SipUsers 要踢出的 sip 用户 ms_open_id 列表。
	SipUsers []string `json:"sip_users,omitempty"`
	// PstnUsers 要踢出的 pstn 用户 ms_open_id 列表。
	PstnUsers []string `json:"pstn_users,omitempty"`
}

// Kick 踢出会议成员（会中控制）。users/sip_users/pstn_users 至少填一组，总数最多 20。
func Kick(ctx context.Context, tmeet *internal.Tmeet, p KickParams) (*result.Result, error) {
	if len(p.Users) == 0 && len(p.SipUsers) == 0 && len(p.PstnUsers) == 0 {
		return nil, exception.InvalidArgsError.With("users/sip_users/pstn_users, at least one of them is required")
	}
	if total := len(p.Users) + len(p.SipUsers) + len(p.PstnUsers); total > cmdutil.MeetingControlUsersListMax {
		return nil, exception.InvalidArgsError.With(
			"total number of users/sip_users/pstn_users is too long, max is %d, got %d",
			cmdutil.MeetingControlUsersListMax, total)
	}

	kickList, err := cmdutil.PackageMeetingControlUsers("users", p.Users)
	if err != nil {
		return nil, err
	}
	kickSipList, err := cmdutil.PackageMeetingControlSpecialUsers("sip_users", p.SipUsers, 9)
	if err != nil {
		return nil, err
	}
	kickPstnList, err := cmdutil.PackageMeetingControlSpecialUsers("pstn_users", p.PstnUsers, 0)
	if err != nil {
		return nil, err
	}
	kickList = append(kickList, kickSipList...)
	kickList = append(kickList, kickPstnList...)

	params := map[string]interface{}{
		"operator_id":      tmeet.UserConfig.OpenId,
		"operator_id_type": 2, // OpenId
		"instanceid":       1, // PC, fixed value
		"allow_rejoin":     p.AllowRejoin,
		"users":            kickList,
	}

	req := &thttp.Request{
		ApiURI: "/v1/real-control/meetings/{meeting_id}/kickout",
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
