package tshoot

import (
	"context"
	"net/http"
	"tmeet/internal"
	"tmeet/internal/core/thttp"
	restProxy "tmeet/internal/proxy/rest-proxy"
	"tmeet/internal/result"
	"tmeet/internal/utils"
)

// FeedbackParams 是 tshoot feedback 命令的入参。
type FeedbackParams struct {
	// Category 反馈分类（必填）。
	Category string `json:"category"`
	// Intent 原始意图（必填，最多 200 字符）。
	Intent string `json:"intent"`
	// ActionsTried 已尝试的操作（最多 500 字符）。
	ActionsTried string `json:"actions_tried,omitempty"`
	// Result 结果或阻塞点（最多 500 字符）。
	Result string `json:"result,omitempty"`
	// ToolName 使用的工具/命令名。
	ToolName string `json:"tool_name,omitempty"`
	// ErrorCode 工具返回的错误码。
	ErrorCode string `json:"error_code,omitempty"`
}

// 长度限制
const (
	maxIntentLen       = 200
	maxActionsTriedLen = 500
	maxResultLen       = 500
)

// Feedback 向平台反馈问题（工具缺失/失败/能力不足等）。
func Feedback(ctx context.Context, tmeet *internal.Tmeet, p FeedbackParams) (*result.Result, error) {
	if err := utils.CharacterLimit("intent", p.Intent, maxIntentLen); err != nil {
		return nil, err
	}
	if err := utils.CharacterLimit("actions_tried", p.ActionsTried, maxActionsTriedLen); err != nil {
		return nil, err
	}
	if err := utils.CharacterLimit("result", p.Result, maxResultLen); err != nil {
		return nil, err
	}

	params := map[string]interface{}{
		"operator_id":      tmeet.UserConfig.OpenId,
		"operator_id_type": "2", // OpenId
		"category":         p.Category,
		"intent":           p.Intent,
		"actions_tried":    p.ActionsTried,
		"result":           p.Result,
		"tool_name":        p.ToolName,
		"error_code":       p.ErrorCode,
		"from_source":      "CLI", // fixed
	}

	req := &thttp.Request{
		ApiURI: "/v1/api/feedback",
		Body:   params,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodPost, tmeet, req)
	if err != nil {
		return nil, err
	}
	return &result.Result{Data: rsp.Data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}
