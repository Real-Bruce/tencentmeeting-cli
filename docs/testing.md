# 测试约定

## 约定

- 涉及真实 keychain / 文件系统的测试用 `_integration_test.go` 后缀 + `//go:build integration`（见 `internal/config/user_integration_test.go`）
- `internal/core/keychain/mock.go` 提供 `NewMockKeychain()`，通过 `config.SetKeychain(mock)` 注入，避免测试触达系统

## 命令

```bash
go test ./...                            # 全量单元测试
go test -v ./internal/<pkg>/...          # 单个包，带详细输出
go test -tags=integration ./...          # 集成测试（涉及真实 keychain / 文件系统，谨慎）
```
