package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/mcp2rest/pkg/mcp"
)

// TestClient MCP 测试客户端
type TestClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	reader *bufio.Reader
}

// NewTestClient 创建新的测试客户端
func NewTestClient(serverPath, configPath string) (*TestClient, error) {
	cmd := exec.Command(serverPath, "-config", configPath)

	// 设置管道
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("创建标准输入管道失败: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("创建标准输出管道失败: %w", err)
	}

	// 启动服务器进程
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动服务器失败: %w", err)
	}

	reader := bufio.NewReader(stdout)

	return &TestClient{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		reader: reader,
	}, nil
}

// SendRequest 发送 MCP 请求
func (tc *TestClient) SendRequest(method string, params interface{}) (*mcp.MCPResponse, error) {
	// 创建请求
	idStr := fmt.Sprintf("test_%d", time.Now().UnixNano())
	idBytes, _ := json.Marshal(idStr)
	
	request := mcp.MCPRequest{
		JSONRPC: "2.0",
		ID:      idBytes,
		Method:  method,
	}

	// 序列化参数
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("序列化参数失败: %w", err)
	}
	request.Params = paramsBytes

	// 序列化请求
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送请求
	requestStr := string(requestBytes) + "\n"
	fmt.Printf("DEBUG: 发送请求: %s", requestStr)
	if _, err := tc.stdin.Write([]byte(requestStr)); err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	// 读取响应（带超时）
	responseChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		responseStr, err := tc.reader.ReadString('\n')
		if err != nil {
			errChan <- err
			return
		}
		responseChan <- responseStr
	}()

	select {
	case responseStr := <-responseChan:
		fmt.Printf("DEBUG: 收到响应: %s", responseStr)
		// 解析响应
		var response mcp.MCPResponse
		if err := json.Unmarshal([]byte(strings.TrimSpace(responseStr)), &response); err != nil {
			return nil, fmt.Errorf("解析响应失败: %w, 原始响应: %s", err, strings.TrimSpace(responseStr))
		}
		return &response, nil
	case err := <-errChan:
		return nil, fmt.Errorf("读取响应失败: %w", err)
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("读取响应超时")
	}
}

// Initialize 初始化 MCP 连接
func (tc *TestClient) Initialize() error {
	initParams := map[string]interface{}{
		"protocolVersion": "20241105",
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
		"clientInfo": map[string]interface{}{
			"name":    "MCP2REST-TestClient",
			"version": "1.0.0",
		},
	}

	response, err := tc.SendRequest("initialize", initParams)
	if err != nil {
		return fmt.Errorf("初始化失败: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("初始化错误: %+v", response.Error)
	}

	fmt.Println("✅ MCP 连接初始化成功")
	return nil
}

// SendInitialized 发送初始化完成通知
func (tc *TestClient) SendInitialized() error {
	// 通知类型的方法不需要等待响应
	request := mcp.MCPRequest{
		JSONRPC: "2.0",
		ID:      []byte("null"),
		Method:  "notifications/initialized",
		Params:  []byte("{}"),
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	requestStr := string(requestBytes) + "\n"
	if _, err := tc.stdin.Write([]byte(requestStr)); err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}

	// 等待一小段时间，确保通知被处理
	time.Sleep(100 * time.Millisecond)

	fmt.Println("✅ 初始化完成通知已发送")
	return nil
}

// GetToolsList 获取工具列表
func (tc *TestClient) GetToolsList() ([]map[string]interface{}, error) {
	// 直接发送请求并读取响应
	idStr := fmt.Sprintf("tools_list_%d", time.Now().UnixNano())
	idBytes, _ := json.Marshal(idStr)
	
	request := mcp.MCPRequest{
		JSONRPC: "2.0",
		ID:      idBytes,
		Method:  "tools/list",
		Params:  []byte("{}"),
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	requestStr := string(requestBytes) + "\n"
	fmt.Printf("DEBUG: 发送工具列表请求: %s", requestStr)
	if _, err := tc.stdin.Write([]byte(requestStr)); err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	// 读取响应
	responseStr, err := tc.reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	fmt.Printf("DEBUG: 收到响应: %s\n", strings.TrimSpace(responseStr))

	// 解析响应
	var response mcp.MCPResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(responseStr)), &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w, 原始响应: %s", err, strings.TrimSpace(responseStr))
	}

	if response.Error != nil {
		return nil, fmt.Errorf("获取工具列表错误: %+v", response.Error)
	}

	var result struct {
		Tools []map[string]interface{} `json:"tools"`
	}

	if response.Result != nil {
		if err := json.Unmarshal(response.Result, &result); err != nil {
			return nil, fmt.Errorf("解析工具列表失败: %w", err)
		}
	}

	return result.Tools, nil
}

// Close 关闭客户端
func (tc *TestClient) Close() error {
	if tc.cmd == nil || tc.cmd.Process == nil {
		return nil
	}

	// 先发送 exit 命令给服务器
	fmt.Println("发送 exit 命令给服务器...")
	exitRequest := mcp.MCPRequest{
		JSONRPC: "2.0",
		ID:      []byte(`"exit"`),
		Method:  "exit",
		Params:  []byte("{}"),
	}

	exitBytes, err := json.Marshal(exitRequest)
	if err == nil {
		exitStr := string(exitBytes) + "\n"
		tc.stdin.Write([]byte(exitStr))
		// 给服务器一点时间处理 exit 命令
		time.Sleep(500 * time.Millisecond)
	}

	// 然后尝试优雅关闭
	if err := tc.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		fmt.Printf("发送 SIGTERM 失败: %v\n", err)
	}

	// 等待进程退出
	done := make(chan error, 1)
	go func() {
		_, err := tc.cmd.Process.Wait()
		done <- err
	}()

	select {
	case err := <-done:
		fmt.Printf("进程已退出: %v\n", err)
		return err
	case <-time.After(5 * time.Second):
		fmt.Println("进程退出超时，强制终止...")
		// 超时后强制终止
		return tc.cmd.Process.Kill()
	}
}

// TestCase 测试用例
type TestCase struct {
	Name        string
	ToolName    string
	Parameters  map[string]interface{}
	Description string
}

// TestResult 测试结果
type TestResult struct {
	TestCase TestCase
	Success  bool
	Error    error
	Response *mcp.MCPResponse
	Duration time.Duration
}

// TestSuite 测试套件
type TestSuite struct {
	client *TestClient
	tests  []TestCase
}

// NewTestSuite 创建新的测试套件
func NewTestSuite(client *TestClient) *TestSuite {
	return &TestSuite{
		client: client,
		tests:  make([]TestCase, 0),
	}
}

// AddTest 添加测试用例
func (ts *TestSuite) AddTest(test TestCase) {
	ts.tests = append(ts.tests, test)
}

// RunTests 运行所有测试
func (ts *TestSuite) RunTests() []TestResult {
	results := make([]TestResult, 0, len(ts.tests))

	fmt.Printf("开始运行 %d 个测试用例...\n", len(ts.tests))
	fmt.Println(strings.Repeat("=", 60))

	for i, test := range ts.tests {
		fmt.Printf("测试 %d/%d: %s\n", i+1, len(ts.tests), test.Name)
		fmt.Printf("描述: %s\n", test.Description)
		fmt.Printf("工具: %s\n", test.ToolName)
		fmt.Printf("参数: %+v\n", test.Parameters)

		start := time.Now()
		response, err := ts.client.SendRequest("toolCall", map[string]interface{}{
			"name":       test.ToolName,
			"parameters": test.Parameters,
		})
		duration := time.Since(start)

		result := TestResult{
			TestCase: test,
			Success:  err == nil && response.Error == nil,
			Error:    err,
			Response: response,
			Duration: duration,
		}

		if result.Success {
			fmt.Printf("✅ 成功 (耗时: %v)\n", duration)
			if response.Result != nil {
				var resultData interface{}
				if err := json.Unmarshal(response.Result, &resultData); err == nil {
					fmt.Printf("响应: %+v\n", resultData)
				}
			}
		} else {
			fmt.Printf("❌ 失败 (耗时: %v)\n", duration)
			if err != nil {
				fmt.Printf("错误: %v\n", err)
			}
			if response != nil && response.Error != nil {
				fmt.Printf("MCP错误: %+v\n", response.Error)
			}
		}

		fmt.Println(strings.Repeat("-", 40))
		results = append(results, result)
	}

	return results
}

// PrintSummary 打印测试总结
func (ts *TestSuite) PrintSummary(results []TestResult) {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("测试总结")
	fmt.Println(strings.Repeat("=", 60))

	successCount := 0
	totalDuration := time.Duration(0)

	for _, result := range results {
		if result.Success {
			successCount++
		}
		totalDuration += result.Duration
	}

	fmt.Printf("总测试数: %d\n", len(results))
	fmt.Printf("成功数: %d\n", successCount)
	fmt.Printf("失败数: %d\n", len(results)-successCount)
	fmt.Printf("成功率: %.2f%%\n", float64(successCount)/float64(len(results))*100)
	fmt.Printf("总耗时: %v\n", totalDuration)
	if len(results) > 0 {
		fmt.Printf("平均耗时: %v\n", totalDuration/time.Duration(len(results)))
	}
}

func getProcessCount(name string) int {
	cmd := exec.Command("pgrep", "-f", name)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	return len(strings.Split(strings.TrimSpace(string(output)), "\n"))
}

func main() {
	// 设置环境变量
	os.Setenv("APIKEYAUTH_API_KEY", "ded45a001ffb9c47b1e29fcbdd6bcec6")

	// 配置参数
	serverPath := "./bin/mcp2rest"
	configPath := "./configs/bmc_api.yaml"

	// 检查服务器是否存在
	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		log.Fatalf("服务器可执行文件不存在: %s", serverPath)
	}

	// 检查配置文件是否存在
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("配置文件不存在: %s", configPath)
	}

	// 创建测试客户端
	fmt.Println("创建测试客户端前进程数:", getProcessCount("mcp2rest"))
	client, err := NewTestClient(serverPath, configPath)
	if err != nil {
		log.Fatalf("创建测试客户端失败: %v", err)
	}
	defer func() {
		fmt.Println("关闭客户端前进程数:", getProcessCount("mcp2rest"))
		client.Close()
		fmt.Println("关闭客户端后进程数:", getProcessCount("mcp2rest"))
	}()

	// 等待服务器启动
	fmt.Println("等待服务器启动...")
	time.Sleep(2 * time.Second)
	fmt.Println("服务器启动后进程数:", getProcessCount("mcp2rest"))

	// 测试基本功能
	fmt.Println("=== 测试基本功能 ===")
	
	// 1. 测试初始化
	fmt.Println("1. 测试初始化...")
	if err := client.Initialize(); err != nil {
		log.Fatalf("初始化 MCP 连接失败: %v", err)
	}
	fmt.Println("✅ 初始化成功")

	// 2. 测试发送初始化完成通知
	fmt.Println("2. 测试发送初始化完成通知...")
	if err := client.SendInitialized(); err != nil {
		log.Fatalf("发送初始化完成通知失败: %v", err)
	}
	fmt.Println("✅ 初始化完成通知发送成功")

	// 3. 测试获取工具列表
	fmt.Println("3. 测试获取工具列表...")
	tools, err := client.GetToolsList()
	if err != nil {
		log.Fatalf("获取工具列表失败: %v", err)
	}

	fmt.Printf("✅ 发现 %d 个可用工具:\n", len(tools))
	for i, tool := range tools {
		name := tool["name"].(string)
		description := tool["description"].(string)
		fmt.Printf("  %d. %s: %s\n", i+1, name, description)
	}

	// 4. 测试工具调用
	fmt.Println("4. 测试工具调用...")
	if len(tools) > 0 {
		firstTool := tools[0]
		toolName := firstTool["name"].(string)
		
		fmt.Printf("测试调用工具: %s\n", toolName)
		response, err := client.SendRequest("toolCall", map[string]interface{}{
			"name":       toolName,
			"parameters": map[string]interface{}{},
		})
		
		if err != nil {
			fmt.Printf("❌ 工具调用失败: %v\n", err)
		} else if response.Error != nil {
			fmt.Printf("❌ 工具调用返回错误: %+v\n", response.Error)
		} else {
			fmt.Printf("✅ 工具调用成功\n")
		}
	}

	fmt.Println("=== 基本功能测试完成 ===")

	// 创建测试套件
	testSuite := NewTestSuite(client)

	// 添加测试用例（使用正确的工具名称）
	testSuite.AddTest(TestCase{
		Name:     "测试 BMC 列表查询",
		ToolName: "getList",
		Parameters: map[string]interface{}{
			"page":  1,
			"limit": 10,
			"sort":  "created",
			"order": "desc",
		},
		Description: "测试获取 BMC 数据列表功能",
	})

	testSuite.AddTest(TestCase{
		Name:     "测试 BMC 详情查询",
		ToolName: "getDetail",
		Parameters: map[string]interface{}{
			"id": "test_bmc_001",
		},
		Description: "测试获取指定 BMC 详情功能",
	})

	testSuite.AddTest(TestCase{
		Name:     "测试 BMC 搜索",
		ToolName: "getSearch",
		Parameters: map[string]interface{}{
			"q":      "测试",
			"page":   1,
			"limit":  10,
			"fields": "title,description",
		},
		Description: "测试 BMC 数据搜索功能",
	})

	testSuite.AddTest(TestCase{
		Name:     "测试 BMC 创建",
		ToolName: "postCreate",
		Parameters: map[string]interface{}{
			"id":          "test_bmc_001",
			"title":       "测试 BMC",
			"description": "这是一个测试用的 BMC 数据",
			"bmc": map[string]interface{}{
				"customerSegments":      []string{"企业用户", "个人用户"},
				"valuePropositions":     []string{"高效解决方案", "优质服务"},
				"channels":              []string{"官网", "合作伙伴"},
				"customerRelationships": []string{"长期合作", "技术支持"},
				"keyResources":          []string{"技术团队", "品牌声誉"},
				"keyActivities":         []string{"产品开发", "市场推广"},
				"keyPartnerships":       []string{"技术供应商", "渠道伙伴"},
				"costStructure":         []string{"研发成本", "运营成本"},
				"revenueStreams":        []string{"产品销售", "服务收费"},
			},
		},
		Description: "测试创建新的 BMC 数据功能",
	})

	testSuite.AddTest(TestCase{
		Name:     "测试 BMC 更新",
		ToolName: "postUpdate",
		Parameters: map[string]interface{}{
			"id":    "test_bmc_001",
			"title": "更新后的测试 BMC",
		},
		Description: "测试更新现有 BMC 数据功能",
	})

	testSuite.AddTest(TestCase{
		Name:     "测试 BMC 删除",
		ToolName: "postDelete",
		Parameters: map[string]interface{}{
			"id": "test_bmc_001",
		},
		Description: "测试删除 BMC 数据功能",
	})

	// 运行测试
	results := testSuite.RunTests()

	// 打印总结
	testSuite.PrintSummary(results)

	// 检查是否有失败的测试
	failedCount := 0
	for _, result := range results {
		if !result.Success {
			failedCount++
		}
	}

	if failedCount > 0 {
		fmt.Printf("\n⚠️  有 %d 个测试失败，请检查服务器配置和网络连接\n", failedCount)
		os.Exit(1)
	} else {
		fmt.Printf("\n🎉 所有测试都通过了！\n")
	}
}
