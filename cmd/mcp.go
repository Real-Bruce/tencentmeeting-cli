package cmd

import (
	"fmt"
	"tmeet/internal"
	"tmeet/internal/cmdutil"
	"tmeet/internal/log"
	"tmeet/internal/mcpserver"

	"github.com/spf13/cobra"
)

// newMcpCmd 创建 `tmeet mcp` 子命令：以 MCP server 模式运行，
// 暴露腾讯会议的全部能力为 MCP tool，供 AI Agent 调用。
//
// 支持三种传输模式：
//   - stdio（默认）：本地同机调用，Cursor / Claude Desktop 等 spawn 进程
//   - streamable-http：MCP 2025-03-26 规范，适合 Linux 服务器远程部署
//   - sse：旧版 SSE 传输，兼容部分老客户端
//
// 该命令本身不要求登录（server 内各 tool 按需校验）。
func newMcpCmd(tmeet *internal.Tmeet) *cobra.Command {
	var (
		transport string
		addr      string
		endpoint  string
		baseURL   string
		apiKey    string
		tlsCert   string
		tlsKey    string
	)

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "run as MCP server (stdio/http), exposing tmeet capabilities to AI agents",
		Long: `Run tmeet as a Model Context Protocol (MCP) server.

TRANSPORT MODES:

  stdio (default) — local, same-machine:
    Cursor / Claude Desktop / Cline spawn the process and communicate via stdin/stdout.
    Configure in MCP client:
      command: tmeet
      args:    mcp

  streamable-http — remote server deployment (recommended for production):
    Run on a Linux server, AI clients connect via HTTP URL.
      tmeet mcp --transport streamable-http --addr 0.0.0.0:8080
    Client connects to: http://<server>:8080/mcp

  sse — legacy SSE transport (compatibility):
      tmeet mcp --transport sse --addr 0.0.0.0:8080
    Client connects to SSE endpoint: http://<server>:8080/sse

AUTH (for HTTP modes):
  --api-key <key>   Requires clients to send "Authorization: Bearer <key>" header.
                    Leave empty for no auth (local dev only).

TLS (for HTTP modes):
  --tls-cert / --tls-key   Path to cert/key files for HTTPS.

PREREQUISITE:
  OAuth login must be completed first via "tmeet auth login" (the server
  reads existing local credentials).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			srv := mcpserver.NewServer(tmeet, Version)

			switch transport {
			case "stdio", "":
				log.Infof(cmd.Context(), "tmeet mcp server starting (transport=stdio, version=%s)", Version)
				if err := srv.ServeStdio(); err != nil {
					return fmt.Errorf("mcp stdio server: %w", err)
				}
			case "streamable-http":
				log.Infof(cmd.Context(), "tmeet mcp server starting (transport=streamable-http, addr=%s, endpoint=%s)", addr, endpoint)
				if err := srv.ServeStreamableHTTP(mcpserver.ServeHTTPOpts{
					Addr:     addr,
					Endpoint: endpoint,
					APIKey:   apiKey,
					TLSCert:  tlsCert,
					TLSKey:   tlsKey,
				}); err != nil {
					return fmt.Errorf("mcp streamable-http server: %w", err)
				}
			case "sse":
				log.Infof(cmd.Context(), "tmeet mcp server starting (transport=sse, addr=%s, base-url=%s)", addr, baseURL)
				if err := srv.ServeSSE(mcpserver.ServeHTTPOpts{
					Addr:    addr,
					BaseURL: baseURL,
					APIKey:  apiKey,
					TLSCert: tlsCert,
					TLSKey:  tlsKey,
				}); err != nil {
					return fmt.Errorf("mcp sse server: %w", err)
				}
			default:
				return fmt.Errorf("unknown transport %q: must be stdio | streamable-http | sse", transport)
			}
			return nil
		},
	}
	// MCP server 启动不需要登录态，跳过 preCheck。
	cmdutil.InjectSkipPreCheckAnnotation(cmd)

	cmd.Flags().StringVar(&transport, "transport", "stdio", "transport: stdio | streamable-http | sse")
	cmd.Flags().StringVar(&addr, "addr", ":8080", "HTTP listen address (for streamable-http/sse)")
	cmd.Flags().StringVar(&endpoint, "endpoint", "/mcp", "endpoint path (for streamable-http)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "external base URL (for sse, e.g. https://tmeet.example.com)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "bearer token for auth (HTTP modes); empty = no auth")
	cmd.Flags().StringVar(&tlsCert, "tls-cert", "", "TLS cert file path (HTTP modes, optional)")
	cmd.Flags().StringVar(&tlsKey, "tls-key", "", "TLS key file path (HTTP modes, optional)")

	return cmd
}
