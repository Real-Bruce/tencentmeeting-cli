# 配置与环境变量

## 环境变量

| 环境变量 | 默认 | 用途 |
|---|---|---|
| `TMEET_CLI_CONFIG_DIR` | `~/.tmeet/` | config.json / agent.json / logs/ / token.lock |
| `TMEET_CLI_DATA_DIR` | 平台相关 | keychain 加密数据目录（Linux/macOS .enc 文件） |
| `TMEET_AGENT` | — | AI Agent 标识（Cursor / Claude Desktop / Cline / …），写入 agent.json |
| `TMEET_MODEL` | — | LLM 模型名（Claude 3.5 Sonnet / GPT-4o / …），写入 agent.json |
| `GOPROXY` | `https://proxy.golang.org,direct` | 国内必须改 `https://goproxy.cn,direct`（见 [debugging.md](debugging.md)） |

## 配置文件结构

`config.json` 结构（明文）：`{ "active_open_id": "..." }`，仅用于定位 keychain 条目。

`agent.json` 结构（明文）：`{ "agent": "...", "model": "..." }`，与用户身份生命周期独立，**logout 不删除**。

> 凭证加解密与 keychain 操作属于不变量，见 [invariants.md](invariants.md#凭证安全)。
