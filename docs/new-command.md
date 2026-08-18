# 新增一个子命令

> 以新增 `meeting foo` 为例。修改前先读 [invariants.md](invariants.md)。

## 步骤

1. **注册命令**：在 `cmd/meeting/base.go:16` 的 `AddCommand(...)` 调用里加 `newFooCmd(tmeet)`。

2. **实现命令**：新建 `cmd/meeting/foo.go`，套用洋葱模型：

   ```go
   type FooOptions struct {
       tmeet      *internal.Tmeet
       MeetingID  string
       // ...其他 flag
   }

   func newFooCmd(tmeet *internal.Tmeet) *cobra.Command {
       opts := &FooOptions{tmeet: tmeet}
       cmd := &cobra.Command{
           Use:   "foo",
           Short: "do foo",
           RunE: middleWare.Chain(
               opts.Run,
               middleWare.WithApiCmd(cmdutil.StaticApiCmd(cmdutil.ApiCmdMeetingFoo)),  // 或 FlagSwitch
               middleWare.WithCompact(tmeet),
           ),
       }
       cmd.Flags().StringVar(&opts.MeetingID, "meeting-id", "", "meeting ID")
       // ...
       return cmd
   }

   func (o *FooOptions) Run(cmd *cobra.Command, args []string) error {
       // 1. 组装 thttp.Request
       // 2. restProxy.RequestProxy(cmd.Context(), http.MethodGet, o.tmeet, req)
       // 3. （可选）utils.ConvertFields 字段转换
       // 4. output.FormatPrint(cmd, rsp.TraceId, rsp.Message, rsp.Data, output.WithConvert(convertMap))
       return nil
   }
   ```

3. **注册 ApiCmd 常量**：在 `internal/cmdutil/api_schema.go:14` 起的 `ApiCmd*` 列表里加 `ApiCmdMeetingFoo = "meeting_foo"`。

4. **不需要登录的命令**（如 `auth login` 自身）：调用 `cmdutil.InjectSkipPreCheckAnnotation(cmd)` 跳过 `preCheck`；或用 `InjectSkipPreCheckFlagAnnotation(cmd, "flagName")` 实现按 flag 跳过（参考 `tshoot log --upload`）。

5. **分页命令**：用 `cmdutil.ClampingPageSize(cmd, pageSize, PageSizeMaxMeetings)` 钳制 page-size；用 `ChoosePageOrToken` / `ChoosePosOrToken` 选分页参数；`page_type=PageTypeToken`（新模式）或 `PageTypeOld`（旧模式）。

6. **写操作（POST/PUT/DELETE）**：参考 `cmd/meeting/create.go`。涉及权限申请两步流的，参考 `cmd/record/permission_apply_prepare.go` + `permission_apply_commit.go`（prepare 获取预览 → 用户确认 → commit 提交）。

7. **时间参数**：用 `utils.ISO8601ToTimeStamp`（`internal/utils/time.go`）转 Unix 秒，错误返回 `exception.InvalidArgsError.With("--start format error: %v", err)`。

8. **会议号归一化**：任何接受 `--meeting-code` 的命令，必须用 `cmdutil.FormatMeetingCode`（`internal/cmdutil/meeting_help.go`）预处理，兼容 `#腾讯会议：295-150-176` / `295 150 176` 等多种复制形式。

9. **更新 SKILL**：`skills/tmeet-skill/references/` 下对应 `tmeet-<domain>.md` 增补命令文档，并在 `SKILL.md` 主文件更新命令总览表。SKILL 版本号与 CLI 同步。
