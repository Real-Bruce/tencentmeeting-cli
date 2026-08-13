package meeting

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	"tmeet/internal/core/thttp"
	"tmeet/internal/exception"
	restProxy "tmeet/internal/proxy/rest-proxy"
	"tmeet/internal/result"
	"tmeet/internal/utils"
)

// inviteesOperateTypeMap 映射 invitees 操作策略字符串到服务端枚举值。
var inviteesOperateTypeMap = map[string]int{
	"replace": 3,
	"add":     1,
	"remove":  2,
}

// UpdateParams 是 meeting update 命令的入参。
// 可选字段使用指针类型：nil=未提供（等价 CLI flag 未 Changed），非 nil=显式设置。
// MCP 的 BindArguments 在 JSON 缺 key 时保持 nil，cobra 层按 Changed 设置指针。
type UpdateParams struct {
	// MeetingID 会议 ID（必填）。
	MeetingID string `json:"meeting_id"`
	// Subject 会议主题。
	Subject string `json:"subject,omitempty"`
	// StartTime 会议开始时间（ISO 8601）。
	StartTime string `json:"start,omitempty"`
	// EndTime 会议结束时间（ISO 8601）。
	EndTime string `json:"end,omitempty"`
	// Password 会议密码（4-6 位数字）。
	Password string `json:"password,omitempty"`
	// Timezone 时区。
	Timezone string `json:"timezone,omitempty"`
	// MeetingType 会议类型。nil=未设置，非 nil=显式设置（0-普通，1-周期性）。
	MeetingType *int `json:"meeting_type,omitempty"`
	// OnlyUserJoinType 入会限制：1-所有，2-仅受邀，3-仅内部。
	OnlyUserJoinType int `json:"join_type,omitempty"`
	// AutoInWaitingRoom 是否启用等候室。
	AutoInWaitingRoom bool `json:"waiting_room,omitempty"`
	// RecurringType 周期类型。nil=未提供（用于 sub_meeting_id 冲突检查）。
	RecurringType *int `json:"recurring_type,omitempty"`
	// UntilType 周期结束类型。nil=未提供。
	UntilType *int `json:"until_type,omitempty"`
	// UntilCount 周期次数。nil=未提供。
	UntilCount *int `json:"until_count,omitempty"`
	// UntilDate 周期结束日期。nil=未提供。
	UntilDate *string `json:"until_date,omitempty"`
	// SubMeetingID 子会议 ID：仅修改单场子会议时间，与周期规则字段互斥。
	SubMeetingID string `json:"sub_meeting_id,omitempty"`
	// Invitees 受邀者 open_id 列表。nil=未提供（与 InviteesType 必须同时提供）。
	Invitees []string `json:"invitees,omitempty"`
	// InviteesType 受邀者操作策略：replace/add/remove。空=未提供。
	InviteesType string `json:"invitees_type,omitempty"`
	// WaterMarkType 文字水印。nil=未设置。
	WaterMarkType *int `json:"water_mark_type,omitempty"`
	// AudioWatermark 音频水印。nil=未设置。
	AudioWatermark *bool `json:"audio_watermark,omitempty"`
	// AutoRecordType 自动录制。nil=未设置。
	AutoRecordType *string `json:"auto_record_type,omitempty"`
	// AutoASR 自动语音识别。nil=未设置。
	AutoASR *bool `json:"auto_asr,omitempty"`
}

// Update 更新会议。
func Update(ctx context.Context, tmeet *internal.Tmeet, p UpdateParams) (*result.Result, error) {
	params := map[string]interface{}{
		"userid":     tmeet.UserConfig.OpenId,
		"instanceid": 1, // PC, fixed value
	}

	if p.MeetingType != nil {
		params["meeting_type"] = *p.MeetingType
	}
	if p.Subject != "" {
		params["subject"] = p.Subject
	}
	if p.Password != "" {
		params["password"] = p.Password
	}
	if p.Timezone != "" {
		params["timezone"] = p.Timezone
	}
	if p.StartTime != "" {
		startTime, err := utils.ISO8601ToTimeStamp(p.StartTime)
		if err != nil {
			return nil, exception.InvalidArgsError.With("--start format error: %v", err)
		}
		params["start_time"] = strconv.FormatInt(startTime, 10)
	}
	if p.EndTime != "" {
		endTime, err := utils.ISO8601ToTimeStamp(p.EndTime)
		if err != nil {
			return nil, exception.InvalidArgsError.With("--end format error: %v", err)
		}
		params["end_time"] = strconv.FormatInt(endTime, 10)
	}
	if p.MeetingType != nil && *p.MeetingType == 1 {
		recurringRule, err := buildRecurringRule(p)
		if err != nil {
			return nil, err
		}
		params["recurring_rule"] = recurringRule
	}
	// setting parameters
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

	// invitees / invitees-type
	if err := applyInvitees(p, params); err != nil {
		return nil, err
	}

	req := &thttp.Request{
		ApiURI: "/v1/meetings/" + p.MeetingID,
		Body:   params,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodPut, tmeet, req)
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

// buildRecurringRule 组装 recurring_rule。当 SubMeetingID 非空时只修改单场子会议时间，
// 且不得与周期规则字段同时设置；否则按周期规则字段更新整个周期性会议。
func buildRecurringRule(p UpdateParams) (map[string]interface{}, error) {
	if p.SubMeetingID != "" {
		conflicting := []struct {
			name string
			set  bool
		}{
			{"recurring-type", p.RecurringType != nil},
			{"until-type", p.UntilType != nil},
			{"until-count", p.UntilCount != nil},
			{"until-date", p.UntilDate != nil},
		}
		for _, c := range conflicting {
			if c.set {
				return nil, exception.InvalidArgsError.With(
					"sub_meeting_id cannot be used together with %s: "+
						"either update a single sub-meeting's time, or update the recurring rule", c.name)
			}
		}
		return map[string]interface{}{"sub_meeting_id": p.SubMeetingID}, nil
	}

	recurringRule := map[string]interface{}{
		"until_type":     intOr(p.UntilType, 0),
		"until_count":    intOr(p.UntilCount, 7),
		"recurring_type": intOr(p.RecurringType, 0),
	}
	if p.UntilDate != nil && *p.UntilDate != "" {
		untilDate, err := utils.ISO8601ToTimeStamp(*p.UntilDate)
		if err != nil {
			return nil, exception.InvalidArgsError.With("--until-date format error: %v", err)
		}
		recurringRule["until_date"] = untilDate
	}
	return recurringRule, nil
}

// applyInvitees 校验 invitees / invitees-type 并写入 params。
// 两者必须同时提供；策略值必须是 replace/add/remove 之一。
func applyInvitees(p UpdateParams, params map[string]interface{}) error {
	inviteesSet := p.Invitees != nil
	typeSet := p.InviteesType != ""
	if !inviteesSet && !typeSet {
		return nil
	}
	if !typeSet {
		return exception.InvalidArgsError.With(
			"invitees_type is required when invitees is set (replace | add | remove)")
	}
	if !inviteesSet {
		return exception.InvalidArgsError.With(
			"invitees is required when invitees_type is set")
	}

	strategy := strings.ToLower(strings.TrimSpace(p.InviteesType))
	operateType, ok := inviteesOperateTypeMap[strategy]
	if !ok {
		return exception.InvalidArgsError.With(
			"invitees_type must be one of: replace, add, remove (got %q)", p.InviteesType)
	}

	inviteesList, err := cmdutil.PackageApiInviteesUsers("--invitees", p.Invitees)
	if err != nil {
		return err
	}

	params["invitees"] = inviteesList
	params["invitees_operate_type"] = operateType
	return nil
}

// intOr 返回指针指向的值，指针为 nil 时返回默认值。
func intOr(p *int, def int) int {
	if p != nil {
		return *p
	}
	return def
}
