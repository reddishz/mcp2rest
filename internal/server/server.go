package server

import (
	"bufio"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mcp2rest/internal/config"
	"github.com/mcp2rest/internal/debug"
	"github.com/mcp2rest/internal/handler"
	"github.com/mcp2rest/internal/logging"
	"github.com/mcp2rest/pkg/mcp"
)

// Server MCP服务器
type Server struct {
	config      *config.Config
	openAPISpec *config.OpenAPISpec
	handler     *handler.RequestHandler
	httpServer  *http.Server
	ctx         context.Context
	cancel      context.CancelFunc
	done        chan struct{}
	// SSE 连接管理
	sseConnections map[string]*SSEConnection
	sseMutex       sync.RWMutex
	// 会话管理
	sessions map[string]*MCPSession
	sessionMutex sync.RWMutex
}

// SSEConnection SSE连接
type SSEConnection struct {
	ID         string
	Writer     http.ResponseWriter
	Flusher    http.Flusher
	Context    context.Context
	Cancel     context.CancelFunc
	RemoteAddr string
	SessionID  string
}

// MCPSession MCP会话
type MCPSession struct {
	ID           string
	ClientID     string
	Endpoint     string
	CreatedAt    time.Time
	LastActivity time.Time
}

// NewServer 创建新的服务器实例
func NewServer(cfg *config.Config, spec *config.OpenAPISpec) (*Server, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建请求处理器
	reqHandler, err := handler.NewRequestHandler(cfg, spec)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("创建请求处理器失败: %w", err)
	}

	return &Server{
		config:         cfg,
		openAPISpec:    spec,
		handler:        reqHandler,
		ctx:            ctx,
		cancel:         cancel,
		done:           make(chan struct{}),
		sseConnections: make(map[string]*SSEConnection),
		sessions:       make(map[string]*MCPSession),
	}, nil
}

// Start 启动服务器
func (s *Server) Start() error {
	switch s.config.Server.Mode {
	case "sse":
		return s.startSSEServer()
	case "stdio":
		return s.startStdioServer()
	default:
		return fmt.Errorf("不支持的服务器模式: %s (支持: stdio, sse)", s.config.Server.Mode)
	}
}

// Stop 停止服务器
func (s *Server) Stop() error {
	logging.Logger.Println("正在停止服务器...")
	s.cancel()

	// 关闭HTTP服务器
	if s.httpServer != nil {
		return s.httpServer.Shutdown(context.Background())
	}

	// 安全关闭 done 通道
	select {
	case <-s.done:
		// 通道已经关闭
	default:
		close(s.done)
	}

	return nil
}

// StopWithContext 使用上下文停止服务器
func (s *Server) StopWithContext(ctx context.Context) error {
	logging.Logger.Println("正在停止服务器...")
	s.cancel()

	// 等待 done 通道或上下文超时
	select {
	case <-s.done:
		logging.Logger.Println("服务器正常停止")
		return nil
	case <-ctx.Done():
		logging.Logger.Printf("服务器停止超时: %v", ctx.Err())
		// 强制关闭 done 通道，防止重复关闭
		select {
		case <-s.done:
			// 通道已经关闭
		default:
			close(s.done)
		}
		return ctx.Err()
	}
}

// Done 返回完成通道
func (s *Server) Done() <-chan struct{} {
	return s.done
}

// Cancel 取消服务器上下文
func (s *Server) Cancel() {
	s.cancel()
}

// getServerName 根据模式获取服务器名称
func getServerName(mode string) string {
	switch mode {
	case "stdio":
		return "MCP2REST-STDIO"
	case "sse":
		return "MCP2REST-SSE"
	default:
		return "MCP2REST"
	}
}

// startSSEServer 启动SSE服务器
func (s *Server) startSSEServer() error {
	mux := http.NewServeMux()

	// 按照 MCP SSE 规范设置端点
	mux.HandleFunc("/sse", s.handleSSEConnection)           // GET: 建立 SSE 连接
	mux.HandleFunc("/messages/", s.handleMCPMessages)       // POST: 处理 MCP 消息

	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	logging.Logger.Printf("SSE服务器启动在 %s", addr)
	logging.Logger.Printf("SSE连接端点: %s/sse", addr)
	logging.Logger.Printf("消息处理端点: %s/messages/", addr)
	return s.httpServer.ListenAndServe()
}

// handleSSEConnection 处理SSE连接建立 (GET /sse)
func (s *Server) handleSSEConnection(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 设置SSE头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// 创建SSE写入器
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	// 创建客户端连接标识
	clientID := fmt.Sprintf("%s-%d", r.RemoteAddr, time.Now().UnixNano())
	
	// 创建会话ID
	sessionID := s.generateSessionID()
	
	// 创建连接上下文
	connCtx, connCancel := context.WithCancel(r.Context())

	// 创建SSE连接
	conn := &SSEConnection{
		ID:         clientID,
		Writer:     w,
		Flusher:    flusher,
		Context:    connCtx,
		Cancel:     connCancel,
		RemoteAddr: r.RemoteAddr,
		SessionID:  sessionID,
	}

	// 创建会话
	session := &MCPSession{
		ID:           sessionID,
		ClientID:     clientID,
		Endpoint:     fmt.Sprintf("/messages/?session_id=%s", sessionID),
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	// 注册连接和会话
	s.sseMutex.Lock()
	s.sseConnections[clientID] = conn
	s.sseMutex.Unlock()

	s.sessionMutex.Lock()
	s.sessions[sessionID] = session
	s.sessionMutex.Unlock()

	logging.Logger.Printf("SSE客户端连接: %s, 会话: %s", clientID, sessionID)

	// 记录调试信息
	debug.LogInfo("SSE连接建立", map[string]interface{}{
		"remote_addr": r.RemoteAddr,
		"method":      r.Method,
		"url":         r.URL.String(),
		"client_id":   clientID,
		"session_id":  sessionID,
		"headers":     r.Header,
	})

	// 按照 MCP 规范发送专用消息端点
	endpointMessage := fmt.Sprintf("event: endpoint\ndata: %s\n\n", session.Endpoint)
	fmt.Fprint(w, endpointMessage)
	flusher.Flush()

	// 保持连接活跃
	for {
		select {
		case <-s.ctx.Done():
			logging.Logger.Printf("服务器关闭，SSE连接关闭: %s", clientID)
			s.removeSSEConnection(clientID)
			return
		case <-connCtx.Done():
			logging.Logger.Printf("客户端断开连接: %s", clientID)
			s.removeSSEConnection(clientID)
			return
		case <-time.After(30 * time.Second):
			// 每30秒发送一次心跳，保持连接活跃
			s.sseMutex.RLock()
			if currentConn, exists := s.sseConnections[clientID]; exists {
				heartbeatMessage := fmt.Sprintf("event: heartbeat\ndata: {\"timestamp\":\"%s\",\"session_id\":\"%s\"}\n\n",
					time.Now().Format(time.RFC3339), sessionID)
				fmt.Fprint(currentConn.Writer, heartbeatMessage)
				currentConn.Flusher.Flush()
			}
			s.sseMutex.RUnlock()
		}
	}
}

// generateSessionID 生成会话ID
func (s *Server) generateSessionID() string {
	return fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%d-%s", time.Now().UnixNano(), uuid.New().String()))))
}

// handleMCPMessages 处理MCP消息 (POST /messages/?session_id=xxx)
func (s *Server) handleMCPMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理 OPTIONS 预检请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 解析会话ID
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "Missing session_id", http.StatusBadRequest)
		return
	}

	// 验证会话
	s.sessionMutex.RLock()
	session, exists := s.sessions[sessionID]
	s.sessionMutex.RUnlock()

	if !exists {
		http.Error(w, "Invalid session_id", http.StatusBadRequest)
		return
	}

	// 更新会话活动时间
	s.sessionMutex.Lock()
	session.LastActivity = time.Now()
	s.sessionMutex.Unlock()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logging.Logger.Printf("读取请求体失败: %v", err)
		debug.LogError("读取MCP请求体失败", err)
		http.Error(w, "读取请求体失败", http.StatusBadRequest)
		return
	}

	// 记录请求详情
	debug.LogRequest("POST", r.URL.Path, map[string]string{
		"Content-Type": r.Header.Get("Content-Type"),
		"User-Agent":   r.Header.Get("User-Agent"),
		"Session-ID":   sessionID,
	}, body)

	// 处理MCP请求
	response, err := s.handleMCPRequest(body)
	if err != nil {
		logging.Logger.Printf("处理MCP请求失败: %v", err)
		debug.LogError("处理MCP请求失败", err)
		http.Error(w, "处理请求失败", http.StatusInternalServerError)
		return
	}

	// 记录响应详情
	responseHeaders := make(map[string]string)
	for key, values := range w.Header() {
		responseHeaders[key] = values[0] // 取第一个值
	}
	debug.LogResponse(200, responseHeaders, response)

	// 按照 MCP 规范，返回 "Accepted" 状态码
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"Accepted"}`))

	// 通过 SSE 连接推送实际响应
	s.pushMessageToSession(sessionID, response)
}

// pushMessageToSession 向指定会话推送消息
func (s *Server) pushMessageToSession(sessionID string, message []byte) {
	s.sessionMutex.RLock()
	session, exists := s.sessions[sessionID]
	s.sessionMutex.RUnlock()

	if !exists {
		logging.Logger.Printf("会话不存在: %s", sessionID)
		return
	}

	s.sseMutex.RLock()
	conn, exists := s.sseConnections[session.ClientID]
	s.sseMutex.RUnlock()

	if !exists {
		logging.Logger.Printf("连接不存在: %s", session.ClientID)
		return
	}

	// 按照 MCP 规范发送消息
	messageEvent := fmt.Sprintf("event: message\ndata: %s\n\n", string(message))
	fmt.Fprint(conn.Writer, messageEvent)
	conn.Flusher.Flush()

	logging.Logger.Printf("向会话 %s 推送消息", sessionID)
}

// removeSSEConnection 移除SSE连接
func (s *Server) removeSSEConnection(clientID string) {
	s.sseMutex.Lock()
	defer s.sseMutex.Unlock()

	if conn, exists := s.sseConnections[clientID]; exists {
		conn.Cancel()
		delete(s.sseConnections, clientID)
		
		// 同时清理会话
		s.sessionMutex.Lock()
		for sessionID, session := range s.sessions {
			if session.ClientID == clientID {
				delete(s.sessions, sessionID)
				logging.Logger.Printf("会话已移除: %s", sessionID)
				break
			}
		}
		s.sessionMutex.Unlock()
		
		logging.Logger.Printf("SSE连接已移除: %s", clientID)
	}
}

// startStdioServer 启动标准输入/输出服务器
func (s *Server) startStdioServer() error {
	logging.Logger.Println("启动标准输入/输出服务器")

	// 创建带缓冲的读取器和写入器
	reader := bufio.NewReaderSize(os.Stdin, 64*1024)   // 64KB 缓冲区
	writer := bufio.NewWriterSize(os.Stdout, 256*1024) // 256KB 缓冲区
	defer writer.Flush()

	// 创建请求通道，用于并发处理
	requestChan := make(chan *requestTask, 100) // 缓冲通道

	// 使用 WaitGroup 确保所有协程正确退出
	var wg sync.WaitGroup

	// 启动工作协程池
	workerCount := 4 // 可以根据需要调整
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			logging.Logger.Printf("启动工作协程 %d", workerID)
			s.stdioWorker(requestChan)
			logging.Logger.Printf("工作协程 %d 已退出", workerID)
		}(i)
	}

	// 启动读取协程
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(requestChan) // 确保在读取协程退出时关闭通道
		defer func() {
			if r := recover(); r != nil {
				logging.Logger.Printf("标准输入/输出服务器发生panic: %v", r)
			}
			logging.Logger.Println("读取协程已退出")
			// 确保在读取协程退出时关闭服务器
			s.cancel()
		}()

		logging.Logger.Println("启动读取协程")

		for {
			// 首先检查上下文是否已取消
			select {
			case <-s.ctx.Done():
				logging.Logger.Println("读取协程收到关闭信号")
				return
			default:
				// 继续读取
			}

			// 直接读取，不使用超时，让系统自然处理 EOF
			line, err := reader.ReadString('\n')

			if err != nil {
				if err == io.EOF {
					logging.Logger.Println("标准输入已关闭 (EOF)，这是最重要的关闭信号")
					s.cancel()
					return
				}
				logging.Logger.Printf("从标准输入读取失败: %v", err)
				// 发送错误响应
				s.sendErrorResponse(writer, "", -32700, fmt.Sprintf("读取输入失败: %v", err))
				continue
			}

			// 去除换行符和空白字符
			line = strings.TrimSpace(line)
			if line == "" {
				continue // 跳过空行
			}

			// 创建请求任务
			task := &requestTask{
				data: []byte(line),
			}

			// 发送到工作协程池
			select {
			case requestChan <- task:
				// 任务已发送
			case <-s.ctx.Done():
				return
			default:
				// 通道已满，直接处理
				logging.Logger.Printf("工作协程池已满，直接处理请求")
				s.processRequest(task)
			}
		}
	}()

	// 等待上下文取消
	logging.Logger.Println("等待服务器停止信号...")
	<-s.ctx.Done()
	logging.Logger.Println("标准输入/输出服务器收到停止信号")

	// 等待所有协程退出
	logging.Logger.Println("等待所有协程退出...")
	wg.Wait()
	logging.Logger.Println("所有协程已退出")

	// 安全关闭 done 通道
	select {
	case <-s.done:
		// 通道已经关闭
	default:
		close(s.done)
	}

	logging.Logger.Println("标准输入/输出服务器已停止")
	return nil
}

// requestTask 请求任务
type requestTask struct {
	data []byte
}

// stdioWorker 标准输入/输出工作协程
func (s *Server) stdioWorker(requestChan <-chan *requestTask) {
	for {
		select {
		case <-s.ctx.Done():
			return
		case task, ok := <-requestChan:
			if !ok {
				return
			}
			s.processRequest(task)
		}
	}
}

// processRequest 处理单个请求
func (s *Server) processRequest(task *requestTask) {
	// 记录请求详情
	debug.LogRequest("STDIO", "stdin", map[string]string{
		"Content-Type": "application/json",
	}, task.data)

	// 解析MCP请求以获取详细信息
	var mcpRequest mcp.MCPRequest
	if err := json.Unmarshal(task.data, &mcpRequest); err == nil {
		debug.LogMCPRequest(fmt.Sprintf("%v", mcpRequest.ID), mcpRequest.Method, mcpRequest.Params)
	}

	// 设置请求超时
	logging.Logger.Printf("处理请求，超时配置: %v", s.config.Global.Timeout)
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Global.Timeout)
	defer cancel()

	// 使用通道进行超时控制，减少协程使用
	type result struct {
		response []byte
		err      error
	}

	resultChan := make(chan result, 1)

	// 启动处理协程
	go func() {
		response, err := s.handleMCPRequest(task.data)
		resultChan <- result{response: response, err: err}
	}()

	// 等待处理完成或超时
	logging.Logger.Printf("等待请求处理完成...")
	select {
	case <-ctx.Done():
		logging.Logger.Printf("请求处理超时，超时时间: %v", s.config.Global.Timeout)
		// 直接使用 os.Stdout
		errResp := mcp.NewErrorResponse("", -32001, "Request timed out")
		if response, err := json.Marshal(errResp); err == nil {
			os.Stdout.Write(response)
			os.Stdout.Write([]byte("\n"))
		}
	case res := <-resultChan:
		logging.Logger.Printf("请求处理完成")
		if res.err != nil {
			logging.Logger.Printf("处理MCP请求失败: %v", res.err)
			debug.LogError("处理MCP请求失败", res.err)
			// 直接使用 os.Stdout
			errResp := mcp.NewErrorResponse("", -32603, fmt.Sprintf("处理请求失败: %v", res.err))
			if response, err := json.Marshal(errResp); err == nil {
				os.Stdout.Write(response)
				os.Stdout.Write([]byte("\n"))
			}
			return
		}

		// 检查响应是否为空（通知类型的请求）
		if res.response == nil {
			logging.Logger.Printf("通知类型请求，无需发送响应")
			return
		}

		// 记录响应详情
		debug.LogResponse(200, map[string]string{
			"Content-Type": "application/json",
		}, res.response)

		// 直接使用 os.Stdout，并检查写入错误
		logging.Logger.Printf("发送响应: %s", string(res.response))
		if _, err := os.Stdout.Write(res.response); err != nil {
			logging.Logger.Printf("写入 stdout 失败: %v，Client 可能已断开连接")
			debug.LogError("写入stdout失败", err)
			s.cancel() // 触发关闭流程
			return
		}
		if _, err := os.Stdout.Write([]byte("\n")); err != nil {
			logging.Logger.Printf("写入换行符失败: %v，Client 可能已断开连接")
			debug.LogError("写入换行符失败", err)
			s.cancel() // 触发关闭流程
			return
		}
		logging.Logger.Printf("响应发送完成")
	}
}

// writeResponse 写入响应到标准输出
func (s *Server) writeResponse(writer *bufio.Writer, response []byte) error {
	// 写入响应数据
	if _, err := writer.Write(response); err != nil {
		return fmt.Errorf("写入响应数据失败: %w", err)
	}

	// 立即刷新，确保数据被写入
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("刷新缓冲区失败: %w", err)
	}

	// 写入换行符
	if err := writer.WriteByte('\n'); err != nil {
		return fmt.Errorf("写入换行符失败: %w", err)
	}

	// 再次刷新，确保换行符被写入
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("刷新缓冲区失败: %w", err)
	}

	return nil
}

// sendErrorResponse 发送错误响应
func (s *Server) sendErrorResponse(writer *bufio.Writer, id string, code int, message string) {
	errResp := mcp.NewErrorResponse(id, code, message)
	response, err := json.Marshal(errResp)
	if err != nil {
		logging.Logger.Printf("序列化错误响应失败: %v", err)
		return
	}

	if err := s.writeResponse(writer, response); err != nil {
		logging.Logger.Printf("发送错误响应失败: %v", err)
	}
}

// handleMCPRequest 处理MCP请求
func (s *Server) handleMCPRequest(data []byte) ([]byte, error) {
	// 解析请求
	var request mcp.MCPRequest
	if err := json.Unmarshal(data, &request); err != nil {
		logging.Logger.Printf("解析MCP请求失败: %v, 数据: %s", err, string(data))
		errResp := mcp.NewErrorResponse("", -32700, "解析请求失败")
		return json.Marshal(errResp)
	}

	// 记录请求信息
	logging.Logger.Printf("收到MCP请求: ID=%s, Method=%s", request.GetIDString(), request.Method)

	// 验证请求格式
	if request.JSONRPC != "2.0" {
		logging.Logger.Printf("不支持的JSON-RPC版本: %s", request.JSONRPC)
		errResp := mcp.NewErrorResponse(request.GetIDString(), -32600, "不支持的JSON-RPC版本")
		return json.Marshal(errResp)
	}

	// 处理不同的方法
	switch request.Method {
	case "initialize":
		return s.handleInitialize(request)
	case "notifications/initialized":
		return s.handleInitialized(request)
	case "notifications/cancelled":
		return s.handleCancelled(request)
	case "tools/list":
		return s.handleToolsList(request)
	case "toolCall", "tools/call":
		return s.handleToolCall(request)
	case "exit":
		return s.handleExit(request)
	default:
		logging.Logger.Printf("不支持的方法: %s", request.Method)
		errResp := mcp.NewErrorResponse(request.GetIDString(), -32601, "不支持的方法")
		return json.Marshal(errResp)
	}
}

// handleInitialize 处理初始化请求
func (s *Server) handleInitialize(request mcp.MCPRequest) ([]byte, error) {
	logging.Logger.Printf("处理初始化请求")

	// 解析初始化参数
	var initParams struct {
		ProtocolVersion string `json:"protocolVersion"`
		Capabilities    struct {
			Tools          map[string]interface{} `json:"tools"`
			Resources      map[string]interface{} `json:"resources"`
			Logging        map[string]interface{} `json:"logging"`
			StreamableHttp map[string]interface{} `json:"streamableHttp"`
		} `json:"capabilities"`
		ClientInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"clientInfo"`
	}

	if err := json.Unmarshal(request.Params, &initParams); err != nil {
		logging.Logger.Printf("解析初始化参数失败: %v", err)
		errResp := mcp.NewErrorResponse(request.GetIDString(), -32602, "无效的初始化参数")
		return json.Marshal(errResp)
	}

	logging.Logger.Printf("客户端信息: %s v%s", initParams.ClientInfo.Name, initParams.ClientInfo.Version)
	logging.Logger.Printf("协议版本: %s", initParams.ProtocolVersion)

	// 构建初始化响应
	initResult := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": true,
			},
			"resources": map[string]interface{}{
				"subscribe":   true,
				"unsubscribe": true,
			},
			"logging": map[string]interface{}{
				"logMessage": true,
			},
			"streamableHttp": map[string]interface{}{
				"request": true,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    getServerName(s.config.Server.Mode),
			"version": "1.0.0",
		},
	}

	response, err := mcp.NewSuccessResponse(request.GetIDString(), initResult)
	if err != nil {
		logging.Logger.Printf("创建初始化响应失败: %v", err)
		errResp := mcp.NewErrorResponse(request.GetIDString(), -32603, "创建响应失败")
		return json.Marshal(errResp)
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		logging.Logger.Printf("序列化初始化响应失败: %v", err)
		errResp := mcp.NewErrorResponse(request.GetIDString(), -32603, "序列化响应失败")
		return json.Marshal(errResp)
	}

	logging.Logger.Printf("初始化响应发送成功")
	return responseBytes, nil
}

// handleInitialized 处理初始化完成通知
func (s *Server) handleInitialized(request mcp.MCPRequest) ([]byte, error) {
	logging.Logger.Printf("处理初始化完成通知")

	// 对于通知类型的请求，不需要返回响应
	return nil, nil
}

// handleCancelled 处理取消通知
func (s *Server) handleCancelled(request mcp.MCPRequest) ([]byte, error) {
	logging.Logger.Printf("处理取消通知")

	// 对于通知类型的请求，不需要返回响应
	return nil, nil
}

// handleExit 处理退出请求
func (s *Server) handleExit(request mcp.MCPRequest) ([]byte, error) {
	logging.Logger.Printf("收到退出请求，准备关闭服务器")

	// 发送退出响应
	response, err := mcp.NewSuccessResponse(request.GetIDString(), nil)
	if err != nil {
		logging.Logger.Printf("创建退出响应失败: %v", err)
		return nil, err
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		logging.Logger.Printf("序列化退出响应失败: %v", err)
		return nil, err
	}

	// 立即关闭服务器
	logging.Logger.Printf("执行退出操作")
	go func() {
		// 给响应发送一点时间
		time.Sleep(50 * time.Millisecond)
		s.Stop()
		// 强制退出进程
		os.Exit(0)
	}()

	return responseBytes, nil
}

// handleToolsList 处理工具列表请求
func (s *Server) handleToolsList(request mcp.MCPRequest) ([]byte, error) {
	logging.Logger.Printf("处理工具列表请求")

	// 获取所有可用的工具名称
	tools := s.handler.GetAvailableTools()

	// 构建工具列表响应
	toolsListResult := map[string]interface{}{
		"tools": tools,
	}

	response, err := mcp.NewSuccessResponse(request.GetIDString(), toolsListResult)
	if err != nil {
		logging.Logger.Printf("创建工具列表响应失败: %v", err)
		errResp := mcp.NewErrorResponse(request.GetIDString(), -32603, "创建响应失败")
		return json.Marshal(errResp)
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		logging.Logger.Printf("序列化工具列表响应失败: %v", err)
		errResp := mcp.NewErrorResponse(request.GetIDString(), -32603, "序列化响应失败")
		return json.Marshal(errResp)
	}

	logging.Logger.Printf("工具列表响应发送成功，包含 %d 个工具", len(tools))
	return responseBytes, nil
}

// handleToolCall 处理工具调用请求
func (s *Server) handleToolCall(request mcp.MCPRequest) ([]byte, error) {
	// 记录请求开始时间
	startTime := time.Now()

	// 解析工具调用参数
	toolParams, err := mcp.ParseToolCallParams(request.Params)
	if err != nil {
		logging.Logger.Printf("解析工具调用参数失败: %v", err)
		errResp := mcp.NewErrorResponse(request.GetIDString(), -32602, fmt.Sprintf("无效的参数: %v", err))
		return json.Marshal(errResp)
	}

	// 处理工具名称前缀
	originalName := toolParams.Name
	if strings.HasPrefix(toolParams.Name, "mcp_") {
		toolParams.Name = strings.TrimPrefix(toolParams.Name, "mcp_")
		logging.Logger.Printf("检测到 mcp_ 前缀，将工具名称从 %s 改为 %s", originalName, toolParams.Name)
	}

	// 记录工具调用信息
	logging.Logger.Printf("工具调用: %s (原始名称: %s), 参数: %+v", toolParams.Name, originalName, toolParams.Parameters)

	// 处理请求
	result, err := s.handler.HandleRequest(toolParams)
	if err != nil {
		logging.Logger.Printf("处理工具调用失败: %v", err)
		errResp := mcp.NewErrorResponse(request.GetIDString(), -32603, fmt.Sprintf("内部错误: %v", err))
		return json.Marshal(errResp)
	}

	// 按照 MCP 规范构建工具调用响应
	// 工具调用响应应该包含 content 数组字段
	var toolCallResponse map[string]interface{}
	
	if result.Type == "error" {
		// 错误响应
		toolCallResponse = map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": fmt.Sprintf("错误: %v", result.Result),
				},
			},
			"isError": true,
		}
	} else {
		// 成功响应
		// 将结果转换为文本格式
		resultText := ""
		if result.Result != nil {
			if resultBytes, err := json.MarshalIndent(result.Result, "", "  "); err == nil {
				resultText = string(resultBytes)
			} else {
				resultText = fmt.Sprintf("%v", result.Result)
			}
		}
		
		toolCallResponse = map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": resultText,
				},
			},
			"isError": false,
		}
	}

	// 创建成功响应
	response, err := mcp.NewSuccessResponse(request.GetIDString(), toolCallResponse)
	if err != nil {
		logging.Logger.Printf("创建成功响应失败: %v", err)
		errResp := mcp.NewErrorResponse(request.GetIDString(), -32603, fmt.Sprintf("创建响应失败: %v", err))
		return json.Marshal(errResp)
	}

	// 序列化响应
	responseBytes, err := json.Marshal(response)
	if err != nil {
		logging.Logger.Printf("序列化响应失败: %v", err)
		errResp := mcp.NewErrorResponse(request.GetIDString(), -32603, fmt.Sprintf("序列化响应失败: %v", err))
		return json.Marshal(errResp)
	}

	// 记录处理时间
	duration := time.Since(startTime)
	logging.Logger.Printf("工具调用处理完成: ID=%s, 耗时=%v", request.GetIDString(), duration)

	return responseBytes, nil
}
