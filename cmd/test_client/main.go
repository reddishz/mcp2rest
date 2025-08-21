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
	request := mcp.MCPRequest{
		JSONRPC: "2.0",
		ID:      fmt.Sprintf("test_%d", time.Now().UnixNano()),
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
	if _, err := tc.stdin.Write([]byte(requestStr)); err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}
	
	// 读取响应
	responseStr, err := tc.reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	
	// 解析响应
	var response mcp.MCPResponse
	if err := json.Unmarshal([]byte(strings.TrimSpace(responseStr)), &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	
	return &response, nil
}

// Close 关闭客户端
func (tc *TestClient) Close() error {
	if tc.cmd != nil && tc.cmd.Process != nil {
		return tc.cmd.Process.Kill()
	}
	return nil
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
	TestCase   TestCase
	Success    bool
	Error      error
	Response   *mcp.MCPResponse
	Duration   time.Duration
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

func main() {
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
	client, err := NewTestClient(serverPath, configPath)
	if err != nil {
		log.Fatalf("创建测试客户端失败: %v", err)
	}
	defer client.Close()
	
	// 等待服务器启动
	fmt.Println("等待服务器启动...")
	time.Sleep(2 * time.Second)
	
	// 创建测试套件
	testSuite := NewTestSuite(client)
	
	// 添加测试用例
	testSuite.AddTest(TestCase{
		Name:     "测试 BMC 列表查询",
		ToolName: "list",
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
		ToolName: "detail",
		Parameters: map[string]interface{}{
			"id": "test_bmc_001",
		},
		Description: "测试获取指定 BMC 详情功能",
	})
	
	testSuite.AddTest(TestCase{
		Name:     "测试 BMC 搜索",
		ToolName: "search",
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
		ToolName: "create",
		Parameters: map[string]interface{}{
			"id":          "test_bmc_001",
			"title":       "测试 BMC",
			"description": "这是一个测试用的 BMC 数据",
			"bmc": map[string]interface{}{
				"customerSegments": []string{"企业用户", "个人用户"},
				"valuePropositions": []string{"高效解决方案", "优质服务"},
				"channels": []string{"官网", "合作伙伴"},
				"customerRelationships": []string{"长期合作", "技术支持"},
				"keyResources": []string{"技术团队", "品牌声誉"},
				"keyActivities": []string{"产品开发", "市场推广"},
				"keyPartnerships": []string{"技术供应商", "渠道伙伴"},
				"costStructure": []string{"研发成本", "运营成本"},
				"revenueStreams": []string{"产品销售", "服务收费"},
			},
		},
		Description: "测试创建新的 BMC 数据功能",
	})
	
	testSuite.AddTest(TestCase{
		Name:     "测试 BMC 更新",
		ToolName: "update",
		Parameters: map[string]interface{}{
			"id":    "test_bmc_001",
			"title": "更新后的测试 BMC",
		},
		Description: "测试更新现有 BMC 数据功能",
	})
	
	testSuite.AddTest(TestCase{
		Name:     "测试 BMC 删除",
		ToolName: "delete",
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
