# 发布清单

升级版本时按顺序执行：

1. 更新 `CHANGELOG.md`（遵循 Keep a Changelog，参考既有 v1.0.x 条目格式）
2. `make sync-version VERSION=v1.0.x` 同步 `package.json`
3. `make build VERSION=v1.0.x` 交叉编译到 `dist/`（5 个目标：darwin-amd64/arm64, linux-amd64/arm64, windows-amd64）
4. 更新 `skills/tmeet-skill/SKILL.md` 顶部版本号，以及 `references/tmeet-*.md` 中受影响的命令文档
5. 提交 PR，按 [CONTRIBUTING.md](../CONTRIBUTING.md) 走分支策略（`feat/*` / `fix/*`）
