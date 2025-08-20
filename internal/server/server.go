package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/mcp2rest/internal/config"
	"github.com/mcp2rest/internal/handler"
	"github.com/mcp2rest/pkg/mcp"
)

// Server 表示MCP2REST服务器
type Server struct {
	config      *config.Config
	handler     *handler.RequestHandler
	httpServer  *http.Server
	connections map[*websocket.Conn]bool
	mu          sync.Mutex
	upgrader    websocket.Upgrader
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewServer 创建新的服务器实例
func NewServer(cfg *config.Config) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	reqHandler, err := handler.NewRequestHandler(cfg)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("创建请求处理器失败: %w", err)
	}

	server := &Server{
		config:      cfg,
		handler:     reqHandler,
		connections: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源的WebSocket连接
			},
		},
		ctx:    ctx,
		cancel: cancel,
	}

	return server, nil
}

// Start 启动服务器
func (s *Server) Start() error {
	switch s.config.Server.Mode {
	case "websocket":
		return s.startWebSocketServer()
	case "stdio":
		return s.startStdioServer()
	default:
		return fmt.Errorf("不支持的服务器模式: %s", s.config.Server.Mode)
	}
}

// Stop 停止服务器
func (s *Server) Stop() error {
	s.cancel()
	
	// 关闭所有WebSocket连接
	s.mu.Lock()
	for conn := range s.connections {
		conn.Close()
	}
	s.mu.Unlock()

	// 关闭HTTP服务器
	if s.httpServer != nil {
		return s.httpServer.Shutdown(context.Background())
	}
	
	return nil
}

// startWebSocketServer 启动WebSocket服务器
func (s *Server) startWebSocketServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleWebSocket)

	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	log.Printf("WebSocket服务器启动在 %s", addr)
	return s.httpServer.ListenAndServe()
}

// handleWebSocket 处理WebSocket连接
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("升级WebSocket连接失败: %v", err)
		return
	}
	defer conn.Close()

	// 添加连接到映射
	s.mu.Lock()
	s.connections[conn] = true
	s.mu.Unlock()

	// 在函数返回时删除连接
	defer func() {
		s.mu.Lock()
		delete(s.connections, conn)
		s.mu.Unlock()
	}()

	log.Printf("新的WebSocket连接: %s", conn.RemoteAddr())

	for {
		// 读取消息
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket读取错误: %v", err)
			}
			break
		}

		// 处理MCP请求
		response, err := s.handleMCPRequest(message)
		if err != nil {
			log.Printf("处理MCP请求失败: %v", err)
			continue
		}

		// 发送响应
		if err := conn.WriteMessage(websocket.TextMessage, response); err != nil {
			log.Printf("WebSocket写入错误: %v", err)
			break
		}
	}
}

// startStdioServer 启动标准输入/输出服务器
func (s *Server) startStdioServer() error {
	log.Println("启动标准输入/输出服务器")
	
	go func() {
		var buffer [4096]byte
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				// 从标准输入读取
				n, err := fmt.Stdin.Read(buffer[:])
				if err != nil {
					log.Printf("从标准输入读取失败: %v", err)
					continue
				}

				// 处理MCP请求
				response, err := s.handleMCPRequest(buffer[:n])
				if err != nil {
					log.Printf("处理MCP请求失败: %v", err)
					continue
				}

				// 写入标准输出
				if _, err := fmt.Stdout.Write(response); err != nil {
					log.Printf("写入标准输出失败: %v", err)
				}
				fmt.Println() // 添加换行符
			}
		}
	}()

	// 等待上下文取消
	<-s.ctx.Done()
	return nil
}

// handleMCPRequest 处理MCP请求
func (s *Server) handleMCPRequest(data []byte) ([]byte, error) {
	var request mcp.MCPRequest
	if err := json.Unmarshal(data, &request); err != nil {
		errResp := mcp.NewErrorResponse("", -32700, "解析请求失败")
		return json.Marshal(errResp)
	}

	// 只处理工具调用
	if request.Method != "toolCall" {
		errResp := mcp.NewErrorResponse(request.ID, -32601, "不支持的方法")
		return json.Marshal(errResp)
	}

	// 解析工具调用参数
	toolParams, err := mcp.ParseToolCallParams(request.Params)
	if err != nil {
		errResp := mcp.NewErrorResponse(request.ID, -32602, fmt.Sprintf("无效的参数: %v", err))
		return json.Marshal(errResp)
	}

	// 处理请求
	result, err := s.handler.HandleRequest(toolParams)
	if err != nil {
		errResp := mcp.NewErrorResponse(request.ID, -32603, fmt.Sprintf("内部错误: %v", err))
		return json.Marshal(errResp)
	}

	// 创建成功响应
	response, err := mcp.NewSuccessResponse(request.ID, result)
	if err != nil {
		errResp := mcp.NewErrorResponse(request.ID, -32603, fmt.Sprintf("创建响应失败: %v", err))
		return json.Marshal(errResp)
	}

	return json.Marshal(response)
}