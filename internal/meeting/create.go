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

// CreateParams 是 meeting create 命令的入参。
// 可选设置字段使用指针类型：nil 表示未提供（等价于 CLI 的 flag 未 Changed），
// 非 nil 表示显式设置。MCP 的 BindArguments 在 JSON 缺该 key 时保持 nil，天然对应。
type CreateParams struct {
	// Subject 会议主题（必填）。
	Subject string `json:"subject"`
	// StartTime 会议开始时间（ISO 8601，必填）。
	StartTime string `json:"start"`
	// EndTime 会议结束时间（ISO 8601，必填）。
	EndTime string `json:"end"`
	// Password 会议密码（4-6 位数字，可选）。
	Password string `json:"password,omitempty"`
	// Timezone 时区，如 Asia/Shanghai。
	Timezone string `json:"timezone,omitempty"`
	// MeetingType 会议类型：0-普通，1-周期性。
	MeetingType int `json:"meeting_type,omitempty"`
	// OnlyUserJoinType 入会限制：1-所有成员，2-仅受邀，3-仅内部。
	OnlyUserJoinType int `json:"join_type,omitempty"`
	// AutoInWaitingRoom 是否启用等候室。
	AutoInWaitingRoom bool `json:"waiting_room,omitempty"`
	// RecurringType 周期类型（meeting_type=1 时必填）：0-每天，1-工作日，2-每周，3-每两周，4-每月。
	RecurringType int `json:"recurring_type,omitempty"`
	// UntilType 周期结束类型（meeting_type=1 时必填）：0-按日期，1-按次数。
	UntilType int `json:"until_type,omitempty"`
	// UntilCount 周期次数（meeting_type=1，默认 7）。
	UntilCount int `json:"until_count,omitempty"`
	// UntilDate 周期结束日期（ISO 8601，meeting_type=1）。
	UntilDate string `json:"until_date,omitempty"`
	// Invitees 受邀参与者 open_id 列表（最多 100）。
	Invitees []string `json:"invitees,omitempty"`
	// WaterMarkType 文字水印：0-单行，1-双行，2-关闭。nil=未设置。
	WaterMarkType *int `json:"water_mark_type,omitempty"`
	// AudioWatermark 音频水印。nil=未设置。
	AudioWatermark *bool `json:"audio_watermark,omitempty"`
	// AutoRecordType 主持人入会自动录制：none/local/cloud。nil=未设置。
	AutoRecordType *string `json:"auto_record_type,omitempty"`
	// AutoASR 自动语音识别。nil=未设置。
	AutoASR *bool `json:"auto_asr,omitempty"`
}

// Create 创建会议。
func Create(ctx context.Context, tmeet *internal.Tmeet, p CreateParams) (*result.Result, error) {
	// Time conversion.
	startTime, err := utils.ISO8601ToTimeStamp(p.StartTime)
	if err != nil {
		return nil, exception.InvalidArgsError.With("--start format error: %v", err)
	}
	endTime, err := utils.ISO8601ToTimeStamp(p.EndTime)
	if err != nil {
		return nil, exception.InvalidArgsError.With("--end format error: %v", err)
	}

	params := map[string]interface{}{
		"subject":      p.Subject,
		"start_time":   strconv.FormatInt(startTime, 10),
		"end_time":     strconv.FormatInt(endTime, 10),
		"password":     p.Password,
		"timezone":     p.Timezone,
		"meeting_type": p.MeetingType,
		"type":         0, // scheduled meeting, fixed value
		"instanceid":   1, // PC, fixed value
		"userid":       tmeet.UserConfig.OpenId,
	}
	if p.MeetingType == 1 {
		// Recurring meeting, add recurring parameters.
		recurringRule := map[string]interface{}{
			"until_type":     p.UntilType,
			"until_count":    p.UntilCount,
			"recurring_type": p.RecurringType,
		}
		if p.UntilDate != "" {
			untilDate, err := utils.ISO8601ToTimeStamp(p.UntilDate)
			if err != nil {
				return nil, exception.InvalidArgsError.With("--until-date format error: %v", err)
			}
			recurringRule["until_date"] = untilDate
		}
		params["recurring_rule"] = recurringRule
	}
	// setting param
	settings := make(map[string]interface{}, 0)
	if p.OnlyUserJoinType > 0 {
		settings["only_user_join_type"] = p.OnlyUserJoinType
	}
	if p.AutoInWaitingRoom {
		settings["auto_in_waiting_room"] = p.AutoInWaitingRoom
	}
	if p.WaterMarkType != nil {
		if *p.WaterMarkType != 2 {
			settings["water_mark_type"] = *p.WaterMarkType
		}
		settings["allow_screen_shared_watermark"] = *p.WaterMarkType != 2
	}
	if p.AudioWatermark != nil {
		settings["audio_watermark"] = *p.AudioWatermark
	}
	if p.AutoRecordType != nil {
		settings["auto_record_type"] = *p.AutoRecordType
	}
	if p.AutoASR != nil {
		settings["auto_asr"] = *p.AutoASR
	}
	if len(settings) > 0 {
		params["settings"] = settings
	}

	// Invitees: validated, deduped and capped (max 100) by the shared helper.
	if len(p.Invitees) > 0 {
		inviteesList, err := cmdutil.PackageApiInviteesUsers("--invitees", p.Invitees)
		if err != nil {
			return nil, err
		}
		params["invitees"] = inviteesList
	}

	req := &thttp.Request{
		ApiURI: "/v1/meetings",
		Body:   params,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodPost, tmeet, req)
	if err != nil {
		return nil, err
	}

	convertMap := map[string]utils.FieldConverter{
		"start_time":          utils.TimestampConverter,
		"end_time":            utils.TimestampConverter,
		"only_user_join_type": utils.MeetingJoinTypeConverter,
	}
	data := string(utils.ConvertFields([]byte(rsp.Data), 10, convertMap))

	return &result.Result{
		Data:    data,
		TraceId: rsp.TraceId,
		Message: rsp.Message,
	}, nil
}

// GenerateMeetingSettingsHints 生成会议设置的企业锁提示。
// MCP 层可选调用以附加提示信息；cobra 层通过 output.WithHints 使用。
func GenerateMeetingSettingsHints(data string) []string {
	return cmdutil.GenerateMeetingSettingsHints(data)
}
