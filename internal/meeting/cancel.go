package meeting

import (
	"context"
	"net/http"
	"tmeet/internal"
	"tmeet/internal/core/thttp"
	restProxy "tmeet/internal/proxy/rest-proxy"
	"tmeet/internal/result"
)

// CancelParams 是 meeting cancel 命令的入参。
type CancelParams struct {
	// MeetingID 会议 ID（必填）。
	MeetingID string `json:"meeting_id"`
	// SubMeetingID 子会议 ID，取消周期性会议的某一场。
	SubMeetingID string `json:"sub_meeting_id,omitempty"`
	// MeetingType 会议类型：0-普通/单个子会议，1-整个周期性会议。
	MeetingType int `json:"meeting_type,omitempty"`
}

// Cancel 取消会议。
func Cancel(ctx context.Context, tmeet *internal.Tmeet, p CancelParams) (*result.Result, error) {
	params := map[string]interface{}{
		"userid":       tmeet.UserConfig.OpenId,
		"reason_code":  0, // fixed value
		"instanceid":   1, // PC, fixed value
		"meetingId":    p.MeetingID,
		"meeting_type": p.MeetingType,
	}
	if p.SubMeetingID != "" {
		params["sub_meeting_id"] = p.SubMeetingID
	}

	req := &thttp.Request{
		ApiURI: "/v1/meetings/{meetingId}/cancel",
		Body:   params,
		PathParams: thttp.PathParams{
			"meetingId": p.MeetingID,
		},
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodPost, tmeet, req)
	if err != nil {
		return nil, err
	}

	return &result.Result{
		Data:    rsp.Data,
		TraceId: rsp.TraceId,
		Message: rsp.Message,
	}, nil
}
