package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mcp2rest/internal/config"
	"github.com/mcp2rest/internal/handler"
	"github.com/mcp2rest/pkg/mcp"
	"github.com/mcp2rest/internal/logging"
	"context"
)

// Server MCP服务器
type Server struct {
	config      *config.Config
	openAPISpec *config.OpenAPISpec
	handler     *handler.RequestHandler
	httpServer  *http.Server
	connections map[*websocket.Conn]bool
	mu          sync.Mutex
	upgrader    websocket.Upgrader
	ctx         context.Context
	cancel      context.CancelFunc
	done        chan struct{}
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
		config:      cfg,
		openAPISpec: spec,
		handler:     reqHandler,
		connections: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源的WebSocket连接
			},
		},
		ctx:    ctx,
		cancel: cancel,
		done:   make(chan struct{}),
	}, nil
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
	logging.Logger.Println("正在停止服务器...")
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
	
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Done 返回完成通道
func (s *Server) Done() <-chan struct{} {
	return s.done
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

	logging.Logger.Printf("WebSocket服务器启动在 %s", addr)
	return s.httpServer.ListenAndServe()
}

// handleWebSocket 处理WebSocket连接
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.Logger.Printf("升级WebSocket连接失败: %v", err)
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

	logging.Logger.Printf("新的WebSocket连接: %s", conn.RemoteAddr())

	for {
		// 读取消息
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logging.Logger.Printf("WebSocket读取错误: %v", err)
			}
			break
		}

		// 处理MCP请求
		response, err := s.handleMCPRequest(message)
		if err != nil {
			logging.Logger.Printf("处理MCP请求失败: %v", err)
			continue
		}

		// 发送响应
		if err := conn.WriteMessage(websocket.TextMessage, response); err != nil {
			logging.Logger.Printf("WebSocket写入错误: %v", err)
			break
		}
	}
}

// startStdioServer 启动标准输入/输出服务器
func (s *Server) startStdioServer() error {
	logging.Logger.Println("启动标准输入/输出服务器")
	
	// 创建带缓冲的读取器和写入器
	reader := bufio.NewReaderSize(os.Stdin, 64*1024) // 64KB 缓冲区
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
			select {
			case <-s.ctx.Done():
				logging.Logger.Println("读取协程收到关闭信号")
				return
			default:
				// 读取一行（JSON消息）
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						logging.Logger.Println("标准输入已关闭，退出服务器")
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
		}
	}()
	
	// 等待上下文取消
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
	// 设置请求超时
	ctx, cancel := context.WithTimeout(s.ctx, s.config.Global.Timeout)
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
	select {
	case <-ctx.Done():
		logging.Logger.Printf("请求处理超时")
		// 直接使用 os.Stdout
		errResp := mcp.NewErrorResponse("", -32001, "Request timed out")
		if response, err := json.Marshal(errResp); err == nil {
			os.Stdout.Write(response)
			os.Stdout.Write([]byte("\n"))
		}
	case res := <-resultChan:
		if res.err != nil {
			logging.Logger.Printf("处理MCP请求失败: %v", res.err)
			// 直接使用 os.Stdout
			errResp := mcp.NewErrorResponse("", -32603, fmt.Sprintf("处理请求失败: %v", res.err))
			if response, err := json.Marshal(errResp); err == nil {
				os.Stdout.Write(response)
				os.Stdout.Write([]byte("\n"))
			}
			return
		}
		
		// 直接使用 os.Stdout
		os.Stdout.Write(res.response)
		os.Stdout.Write([]byte("\n"))
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
	case "toolCall":
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
			Tools      map[string]interface{} `json:"tools"`
			Resources  map[string]interface{} `json:"resources"`
			Logging    map[string]interface{} `json:"logging"`
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
		"protocolVersion": "2025-03-26",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": true,
			},
			"resources": map[string]interface{}{
				"subscribe": true,
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
			"name":    "MCP2REST",
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
	
	// 异步关闭服务器
	go func() {
		time.Sleep(100 * time.Millisecond) // 给响应发送一点时间
		logging.Logger.Printf("执行退出操作")
		s.Stop()
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
	
	// 记录工具调用信息
	logging.Logger.Printf("工具调用: %s, 参数: %+v", toolParams.Name, toolParams.Parameters)
	
	// 处理请求
	result, err := s.handler.HandleRequest(toolParams)
	if err != nil {
		logging.Logger.Printf("处理工具调用失败: %v", err)
		errResp := mcp.NewErrorResponse(request.GetIDString(), -32603, fmt.Sprintf("内部错误: %v", err))
		return json.Marshal(errResp)
	}
	
	// 创建成功响应
	response, err := mcp.NewSuccessResponse(request.GetIDString(), result)
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