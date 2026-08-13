package cmd

import (
	"fmt"
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	"tmeet/internal/log"
	"tmeet/internal/mcpserver"

	"github.com/spf13/cobra"
)

// newMcpCmd 创建 `tmeet mcp` 子命令：以 MCP server（stdio）模式运行，
// 暴露腾讯会议的全部能力为 MCP tool，供 Cursor / Claude Desktop 等 AI Agent 调用。
//
// 该命令本身不要求登录（server 内各 tool 按需校验），
// 退出时机由 MCP client 断开 stdio 决定。
func newMcpCmd(tmeet *internal.Tmeet) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "run as MCP server (stdio), exposing tmeet capabilities to AI agents",
		Long: `Run tmeet as a Model Context Protocol (MCP) server over stdio.

Configure in Cursor / Claude Desktop / other MCP clients:
  command: tmeet
  args:    mcp

All meeting/record/report/contact/control/auth capabilities are exposed as
MCP tools. OAuth login must be completed first via "tmeet auth login" (the
server reads existing local credentials).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := mcpserver.NewServer(tmeet, Version)
			log.Infof(cmd.Context(), "tmeet mcp server starting (version %s)", Version)
			if err := srv.ServeStdio(); err != nil {
				return fmt.Errorf("mcp server: %w", err)
			}
			return nil
		},
	}
	// MCP server 启动不需要登录态，跳过 preCheck。
	cmdutil.InjectSkipPreCheckAnnotation(cmd)
	return cmd
}
