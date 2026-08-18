package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"tmeet/internal"
	"tmeet/internal/config"
	"tmeet/internal/exception"
	"tmeet/internal/log"
	"tmeet/internal/result"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server 是 tmeet 的 MCP server，持有单例 Tmeet 客户端，
// 启动时注册全部 tool，通过 stdio 与 MCP client（如 Cursor/Claude Desktop）通信。
type Server struct {
	tmeet  *internal.Tmeet
	mcpSrv *server.MCPServer
}

// NewServer 创建 MCP server。tmeet 应在调用方创建一次后传入（含已加载的用户凭证）。
func NewServer(tmeet *internal.Tmeet, version string) *Server {
	s := &Server{
		tmeet:  tmeet,
		mcpSrv: server.NewMCPServer("hdtmeet-mcp", version),
	}
	s.registerTools()
	return s
}

// registerTools 注册所有域的 MCP tool。新增域时在此追加调用。
func (s *Server) registerTools() {
	s.registerAuthTools()
	s.registerMeetingTools()
	s.registerRecordTools()
	s.registerReportTools()
	s.registerContactTools()
	s.registerControlTools()
	s.registerTshootTools()
}

// ServeStdio 启动 stdio 传输，阻塞至 client 断开。
func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.mcpSrv)
}

// ServeStreamableHTTP 启动 Streamable HTTP 传输（MCP 2025-03-26 规范）。
// 适合 Linux 服务器部署，远程 AI 客户端通过 HTTP URL 接入。
// 客户端连接地址为 http(s)://<host>:<addr>/<endpoint>。
func (s *Server) ServeStreamableHTTP(opts ServeHTTPOpts) error {
	transportOpts := []server.StreamableHTTPOption{
		server.WithEndpointPath(opts.Endpoint),
	}
	if opts.TLSCert != "" && opts.TLSKey != "" {
		transportOpts = append(transportOpts, server.WithTLSCert(opts.TLSCert, opts.TLSKey))
	}

	httpSrv := server.NewStreamableHTTPServer(s.mcpSrv, transportOpts...)
	handler := authMiddleware(httpSrv, opts.APIKey)

	srv := &http.Server{
		Addr:    opts.Addr,
		Handler: handler,
	}

	log.Infof(context.Background(), "tmeet mcp streamable-http server listening on %s%s", opts.Addr, opts.Endpoint)
	if opts.TLSCert != "" && opts.TLSKey != "" {
		return srv.ListenAndServeTLS(opts.TLSCert, opts.TLSKey)
	}
	return srv.ListenAndServe()
}

// ServeSSE 启动 SSE 传输（旧版 MCP 规范，兼容部分老客户端）。
// 客户端连接 SSE 端点 http(s)://<host>:<addr>/sse 获取会话，再向 /message 发送请求。
func (s *Server) ServeSSE(opts ServeHTTPOpts) error {
	sseOpts := []server.SSEOption{
		server.WithSSEEndpoint("/sse"),
		server.WithMessageEndpoint("/message"),
	}
	if opts.BaseURL != "" {
		sseOpts = append(sseOpts, server.WithBaseURL(opts.BaseURL))
	}

	sseSrv := server.NewSSEServer(s.mcpSrv, sseOpts...)
	mux := http.NewServeMux()
	mux.Handle("/sse", authMiddleware(sseSrv.SSEHandler(), opts.APIKey))
	mux.Handle("/message", authMiddleware(sseSrv.MessageHandler(), opts.APIKey))

	srv := &http.Server{
		Addr:    opts.Addr,
		Handler: mux,
	}

	log.Infof(context.Background(), "tmeet mcp sse server listening on %s (sse: /sse, message: /message)", opts.Addr)
	if opts.TLSCert != "" && opts.TLSKey != "" {
		return srv.ListenAndServeTLS(opts.TLSCert, opts.TLSKey)
	}
	return srv.ListenAndServe()
}

// ServeHTTPOpts 是 HTTP 传输的部署选项。
type ServeHTTPOpts struct {
	// Addr 监听地址，如 ":8080" 或 "0.0.0.0:8080"。
	Addr string
	// Endpoint 端点路径（仅 StreamableHTTP），默认 "/mcp"。
	Endpoint string
	// BaseURL 对外可访问的基础 URL（仅 SSE，反向代理后端时用），如 "https://tmeet.example.com"。
	BaseURL string
	// APIKey Bearer token 鉴权。非空时客户端必须在 Authorization 头传 "Bearer <key>"。留空则不鉴权。
	APIKey string
	// TLSCert TLS 证书文件路径（可选）。
	TLSCert string
	// TLSKey TLS 私钥文件路径（可选）。
	TLSKey string
}

// authMiddleware 在 APIKey 非空时校验 Authorization: Bearer <key> 头。
// APIKey 为空时直接放行（本地开发场景）。
func authMiddleware(handler http.Handler, apiKey string) http.Handler {
	if apiKey == "" {
		return handler
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		expected := "Bearer " + apiKey
		if auth != expected {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32001,"message":"unauthorized: invalid or missing API key"}}`))
			return
		}
		handler.ServeHTTP(w, r)
	})
}

// toolFunc 是业务 tool 的核心签名：入参为已绑定好的 params，返回 result.Result。
// 由各域注册器用具体 params 类型包装成 ToolHandlerFunc。
type toolFunc[P any] func(ctx context.Context, tmeet *internal.Tmeet, params P) (*result.Result, error)

// requireLogin 为需要登录的 tool 做 preCheck：本地凭证为空时返回可操作错误。
// 跳过登录校验的 tool（如 auth_*）不调用本函数。
func (s *Server) requireLogin(ctx context.Context) error {
	usr, err := config.GetUserConfig()
	if err != nil {
		// 读取凭证出错，可能文件损坏；交给上层处理，不静默。
		return err
	}
	if usr == nil {
		return exception.GetUserConfigEmptyError
	}
	// 同步到内存中的 tmeet，确保长进程持有最新凭证引用。
	s.tmeet.UserConfig = usr
	return nil
}

// wrap 把一个具体类型的 toolFunc 包装成 mcp.ToolHandlerFunc，统一处理：
// traceID 注入、CmdPath 设置、（可选）登录校验、错误映射。
func wrap[P any](s *Server, cmdPath string, skipLogin bool, fn toolFunc[P]) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// 每次调用注入独立 traceID，与请求头 Tmeet-Trace 关联。
		ctx = context.WithValue(ctx, log.CtxTraceIDKey, log.GenerateTraceID())
		s.tmeet.CmdPath = "mcp/" + cmdPath

		if !skipLogin {
			if err := s.requireLogin(ctx); err != nil {
				return errorResult(err), nil
			}
		}

		var params P
		if err := req.BindArguments(&params); err != nil {
			return errorResult(exception.InvalidArgsError.With("bind arguments: %v", err)), nil
		}

		r, err := fn(ctx, s.tmeet, params)
		if err != nil {
			return errorResult(err), nil
		}
		return mcp.NewToolResultText(r.Data), nil
	}
}

// errorResult 把 TmeetError（或其他错误）转成 MCP tool error result，
// 返回可操作的 message（含错误码），让模型据此引导用户恢复。
func errorResult(err error) *mcp.CallToolResult {
	var tmeetErr *exception.TmeetError
	if errors.As(err, &tmeetErr) {
		return mcp.NewToolResultErrorf("[code %d] %s", tmeetErr.Code, tmeetErr.Error())
	}
	return mcp.NewToolResultErrorf("%v", err)
}

// emitJSON 把任意结构体 marshal 成 JSON 字符串（不转义 HTML）。
// 当某 tool 需要返回结构化数据而非 result.Result.Data 时可用。
func emitJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal json: %w", err)
	}
	return string(b), nil
}
