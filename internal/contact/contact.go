package contact

import (
	"context"
	"net/http"
	"tmeet/internal"
	"tmeet/internal/core/thttp"
	"tmeet/internal/exception"
	restProxy "tmeet/internal/proxy/rest-proxy"
	"tmeet/internal/result"
	"tmeet/internal/utils"
)

// SearchParams 是 contact search 命令的入参。
type SearchParams struct {
	// Username 用户名（必填）。
	Username string `json:"username"`
	// JobTitle 职位过滤。
	JobTitle string `json:"job_title,omitempty"`
	// DepartmentName 部门过滤。
	DepartmentName string `json:"department_name,omitempty"`
}

// Search 按用户名搜索通讯录成员。
// 注意：严格限定——仅作为会议邀请/呼叫入会的前置步骤使用，不得单独查询个人信息。
func Search(ctx context.Context, tmeet *internal.Tmeet, p SearchParams) (*result.Result, error) {
	queryParams := thttp.QueryParams{}
	queryParams.Set("username", p.Username)
	queryParams.Set("job_title", p.JobTitle)
	queryParams.Set("department_name", p.DepartmentName)
	queryParams.Set("operator_id", tmeet.UserConfig.OpenId)
	queryParams.Set("operator_id_type", "2")

	req := &thttp.Request{
		ApiURI:      "/v1/contacts/members/search",
		QueryParams: queryParams,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodGet, tmeet, req)
	if err != nil {
		return nil, err
	}

	data := result.ApplyContactSearchLogic(rsp.Data)
	return &result.Result{Data: data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}

// LookupByEmailMax 单次查询邮箱最大数量。
const LookupByEmailMax = 50

// LookupByEmailParams 是 contact lookup-by-email 命令的入参。
type LookupByEmailParams struct {
	// Emails 邮箱列表（必填，最多 50）。
	Emails []string `json:"emails"`
}

// LookupByEmail 按邮箱批量查询用户。
func LookupByEmail(ctx context.Context, tmeet *internal.Tmeet, p LookupByEmailParams) (*result.Result, error) {
	if len(p.Emails) > LookupByEmailMax {
		return nil, exception.InvalidArgsError.With("emails exceed maximum limit of %d", LookupByEmailMax)
	}
	for _, email := range p.Emails {
		if err := utils.ValidateEmail(email); err != nil {
			return nil, err
		}
	}

	body := map[string]interface{}{
		"operator_id":      tmeet.UserConfig.OpenId,
		"operator_id_type": 2, // OpenId
		"users_email":      p.Emails,
	}

	req := &thttp.Request{
		ApiURI: "/v1/contacts/members/lookup-by-email",
		Body:   body,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodPost, tmeet, req)
	if err != nil {
		return nil, err
	}
	return &result.Result{Data: rsp.Data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}

// LookupByPhoneMax 单次查询手机号最大数量。
const LookupByPhoneMax = 50

// LookupByPhoneParams 是 contact lookup-by-phone 命令的入参。
type LookupByPhoneParams struct {
	// Phones 手机号列表（必填，最多 50）。
	Phones []string `json:"phones"`
}

// LookupByPhone 按手机号批量查询用户。
func LookupByPhone(ctx context.Context, tmeet *internal.Tmeet, p LookupByPhoneParams) (*result.Result, error) {
	if len(p.Phones) > LookupByPhoneMax {
		return nil, exception.InvalidArgsError.With("phones exceed maximum limit of %d", LookupByPhoneMax)
	}
	for _, phone := range p.Phones {
		if err := utils.ValidatePhone(phone); err != nil {
			return nil, err
		}
	}

	body := map[string]interface{}{
		"operator_id":      tmeet.UserConfig.OpenId,
		"operator_id_type": 2, // OpenId
		"users_phone":      p.Phones,
	}

	req := &thttp.Request{
		ApiURI: "/v1/contacts/members/lookup-by-phone",
		Body:   body,
	}
	rsp, err := restProxy.RequestProxy(ctx, http.MethodPost, tmeet, req)
	if err != nil {
		return nil, err
	}
	return &result.Result{Data: rsp.Data, TraceId: rsp.TraceId, Message: rsp.Message}, nil
}
