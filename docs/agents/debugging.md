# 调试

## 国内开发：配置 GOPROXY

本机默认 `proxy.golang.org` 在国内不可达，**安装 gopls / dlv / 拉依赖**都会超时。已写入用户级 go env：

```bash
go env -w GOPROXY=https://goproxy.cn,direct
```

> 同样原因：`go install gopls@latest` 超时 → 改 GOPROXY 即可解决。

## VSCode 调试

`../../.vscode/launch.json` 已配置 6 个调试入口（`auth login` / `auth status` / `meeting list` / `record list` / `-V` / **custom args**）。日常调试用最后一个，直接改 `args` 数组 → 保存 → F5。

> **CLI 调试模型提示**：CLI 工具每次 F5 = 一个进程 = 一条预写死在 `args` 里的命令。**没有"交互式终端输命令命中断点"的体验**。换命令就改 `args` 再 F5。

推荐断点：

- `cmd/<domain>/<cmd>.go` 的 `Run` 方法 → 看参数解析
- `internal/proxy/rest-proxy/proxy.go:46` `RequestProxy` → 看 token 刷新 + HTTP 请求
- `internal/proxy/rest-proxy/proxy.go:110` → 看 HTTP 状态码分支
- `internal/output/print.go:58` `FormatPrint` → 看输出格式化
