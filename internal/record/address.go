package record

import (
	"context"
	"net/http"
	"strconv"
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	"tmeet/internal/core/thttp"
	restProxy "tmeet/internal/proxy/rest-proxy"
	"tmeet/internal/result"
)

// AddressParams 是 record address 命令的入参。
type AddressParams struct {
	// MeetingRecordID 录制记录 ID（必填）。
	MeetingRecordID string `json:"meeting_record_id"`
	// Page 旧分页参数（deprecated）。
	Page int `json:"page,omitempty"`
	// PageSize 分页大小，默认 30，最大 30。
	PageSize int `json:"page_size,omitempty"`
	// PageToken 分页 token。
	PageToken string `json:"page_token,omitempty"`
}

// Address 获取录制文件下载地址。
func Address(ctx context.Context, tmeet *internal.Tmeet, p AddressParams) (*result.Result, error) {
	queryParams := thttp.QueryParams{}
	queryParams.Set("meeting_record_id", p.MeetingRecordID)
	queryParams.Set("operator_id", tmeet.UserConfig.OpenId)
	queryParams.Set("operator_id_type", "2")

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

	req := &thttp.Request{
		ApiURI:      "/v1/mcp/addresses",
		QueryParams: queryParams,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}
	return &result.Result{Data: rsp.Data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}

// SmartMinutesParams 是 record smart-minutes 命令的入参。
type SmartMinutesParams struct {
	// RecordFileID 录制文件 ID（必填）。
	RecordFileID string `json:"record_file_id"`
	// Lang 语言：default/zh/en/ja。
	Lang string `json:"lang,omitempty"`
	// Pwd 录制文件访问密码。
	Pwd string `json:"pwd,omitempty"`
}

// SmartMinutes 获取录制文件的智能纪要。
func SmartMinutes(ctx context.Context, tmeet *internal.Tmeet, p SmartMinutesParams) (*result.Result, error) {
	lang := p.Lang
	if lang == "" {
		lang = "default"
	}
	queryParams := thttp.QueryParams{}
	queryParams.Set("operator_id", tmeet.UserConfig.OpenId)
	queryParams.Set("operator_id_type", "2")
	queryParams.Set("lang", lang)
	if p.Pwd != "" {
		queryParams.Set("pwd", p.Pwd)
	}

	req := &thttp.Request{
		ApiURI:      "/v1/smart/minutes/{record_file_id}",
		QueryParams: queryParams,
		PathParams: thttp.PathParams{
			"record_file_id": p.RecordFileID,
		},
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}
	return &result.Result{Data: rsp.Data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}
