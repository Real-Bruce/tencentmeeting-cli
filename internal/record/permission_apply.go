package record

import (
	"context"
	"net/http"
	"tmeet/internal"
	"tmeet/internal/core/thttp"
	restProxy "tmeet/internal/proxy/rest-proxy"
	"tmeet/internal/result"
)

// PermissionApplyPrepareParams 是 record permission-apply-prepare 命令的入参。
type PermissionApplyPrepareParams struct {
	// MeetingRecordID 录制记录 ID（必填）。
	MeetingRecordID string `json:"meeting_record_id"`
	// MeetingID 会议 ID。
	MeetingID string `json:"meeting_id,omitempty"`
}

// PermissionApplyPrepare 预览录制权限申请（prepare 步骤，两步流第一步）。
func PermissionApplyPrepare(ctx context.Context, tmeet *internal.Tmeet, p PermissionApplyPrepareParams) (*result.Result, error) {
	queryParams := thttp.QueryParams{}
	queryParams.Set("operator_id", tmeet.UserConfig.OpenId)
	queryParams.Set("operator_id_type", "2")
	queryParams.Set("meeting_record_id", p.MeetingRecordID)
	if p.MeetingID != "" {
		queryParams.Set("meeting_id", p.MeetingID)
	}

	req := &thttp.Request{
		ApiURI:      "/v1/records/mcp/permission-apply/prepare",
		QueryParams: queryParams,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}
	return &result.Result{Data: rsp.Data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}

// PermissionApplyCommitParams 是 record permission-apply-commit 命令的入参。
type PermissionApplyCommitParams struct {
	// MeetingRecordID 录制记录 ID（必填）。
	MeetingRecordID string `json:"meeting_record_id"`
	// MeetingID 会议 ID。
	MeetingID string `json:"meeting_id,omitempty"`
}

// PermissionApplyCommit 提交录制权限申请（commit 步骤，两步流第二步，用户确认后调用）。
func PermissionApplyCommit(ctx context.Context, tmeet *internal.Tmeet, p PermissionApplyCommitParams) (*result.Result, error) {
	queryParams := thttp.QueryParams{}
	queryParams.Set("operator_id", tmeet.UserConfig.OpenId)
	queryParams.Set("operator_id_type", "2")
	queryParams.Set("meeting_record_id", p.MeetingRecordID)
	if p.MeetingID != "" {
		queryParams.Set("meeting_id", p.MeetingID)
	}

	req := &thttp.Request{
		ApiURI:      "/v1/records/mcp/permission-apply/commit",
		QueryParams: queryParams,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}
	return &result.Result{Data: rsp.Data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}
