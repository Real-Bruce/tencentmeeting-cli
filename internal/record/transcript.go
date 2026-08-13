package record

import (
	"context"
	"net/http"
	"tmeet/internal"
	"tmeet/internal/core/thttp"
	restProxy "tmeet/internal/proxy/rest-proxy"
	"tmeet/internal/result"
	"tmeet/internal/utils"
)

// TranscriptGetParams 是 record transcript-get 命令的入参。
type TranscriptGetParams struct {
	// RecordFileID 录制文件 ID（必填）。
	RecordFileID string `json:"record_file_id"`
	// MeetingID 会议 ID。
	MeetingID string `json:"meeting_id,omitempty"`
	// Pid 起始段落 ID。
	Pid string `json:"pid,omitempty"`
	// Limit 查询段落数量。
	Limit string `json:"limit,omitempty"`
}

// TranscriptGet 获取转写详情。
func TranscriptGet(ctx context.Context, tmeet *internal.Tmeet, p TranscriptGetParams) (*result.Result, error) {
	queryParams := thttp.QueryParams{}
	queryParams.Set("operator_id", tmeet.UserConfig.OpenId)
	queryParams.Set("operator_id_type", "2")
	queryParams.Set("record_file_id", p.RecordFileID)
	if p.MeetingID != "" {
		queryParams.Set("meeting_id", p.MeetingID)
	}
	if p.Pid != "" {
		queryParams.Set("pid", p.Pid)
	}
	if p.Limit != "" {
		queryParams.Set("limit", p.Limit)
	}

	req := &thttp.Request{
		ApiURI:      "/v1/records/mcp/transcripts/details",
		QueryParams: queryParams,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}

	convertMap := map[string]utils.FieldConverter{
		"start_time": utils.HHMMSSConverter,
		"end_time":   utils.HHMMSSConverter,
	}
	data := string(utils.ConvertFields([]byte(rsp.Data), 10, convertMap))
	return &result.Result{Data: data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}

// TranscriptParagraphsParams 是 record transcript-paragraphs 命令的入参。
type TranscriptParagraphsParams struct {
	// RecordFileID 录制文件 ID（必填）。
	RecordFileID string `json:"record_file_id"`
	// MeetingID 会议 ID。
	MeetingID string `json:"meeting_id,omitempty"`
}

// TranscriptParagraphs 获取转写段落列表。
func TranscriptParagraphs(ctx context.Context, tmeet *internal.Tmeet, p TranscriptParagraphsParams) (*result.Result, error) {
	queryParams := thttp.QueryParams{}
	queryParams.Set("operator_id", tmeet.UserConfig.OpenId)
	queryParams.Set("operator_id_type", "2")
	queryParams.Set("record_file_id", p.RecordFileID)
	if p.MeetingID != "" {
		queryParams.Set("meeting_id", p.MeetingID)
	}

	req := &thttp.Request{
		ApiURI:      "/v1/records/mcp/transcripts/paragraphs",
		QueryParams: queryParams,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}

	convertMap := map[string]utils.FieldConverter{
		"start_time":   utils.HHMMSSConverter,
		"end_time":     utils.HHMMSSConverter,
		"audio_detect": utils.RecordAudioDetectConverter,
	}
	data := string(utils.ConvertFields([]byte(rsp.Data), 10, convertMap))
	return &result.Result{Data: data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}

// TranscriptSearchParams 是 record transcript-search 命令的入参。
type TranscriptSearchParams struct {
	// RecordFileID 录制文件 ID（必填）。
	RecordFileID string `json:"record_file_id"`
	// MeetingID 会议 ID。
	MeetingID string `json:"meeting_id,omitempty"`
	// Text 搜索文本（必填）。
	Text string `json:"text"`
}

// TranscriptSearch 搜索转写内容。
func TranscriptSearch(ctx context.Context, tmeet *internal.Tmeet, p TranscriptSearchParams) (*result.Result, error) {
	queryParams := thttp.QueryParams{}
	queryParams.Set("operator_id", tmeet.UserConfig.OpenId)
	queryParams.Set("operator_id_type", "2")
	queryParams.Set("record_file_id", p.RecordFileID)
	queryParams.Set("text", p.Text)
	if p.MeetingID != "" {
		queryParams.Set("meeting_id", p.MeetingID)
	}

	req := &thttp.Request{
		ApiURI:      "/v1/records/mcp/transcripts/search",
		QueryParams: queryParams,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}

	convertMap := map[string]utils.FieldConverter{
		"start_time": utils.HHMMSSConverter,
	}
	data := string(utils.ConvertFields([]byte(rsp.Data), 10, convertMap))
	return &result.Result{Data: data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}
