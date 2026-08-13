package meeting

import "encoding/json"

// parseRoot 把 JSON 字符串解析成 map，用于按条件剥离字段（如 total_count=0）。
// 解析失败时返回 error，调用方按需忽略。
func parseRoot(data string) (map[string]interface{}, error) {
	var m map[string]interface{}
	err := json.Unmarshal([]byte(data), &m)
	return m, err
}
