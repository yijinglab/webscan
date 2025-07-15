package scanner

import (
	"fmt"
	"testing"
	"time"
)

func TestScannerBasic(t *testing.T) {
	baseURL := "https://httpbin.org" // 使用httpbin.org作为测试目标
	dict := []string{"/status/200", "/status/404", "/delay/1"}
	threadCount := 3
	timeout := 3 * time.Second

	scanner := NewScanner(baseURL, dict, threadCount, timeout)

	go scanner.Start()

	for i := 0; i < len(dict); i++ {
		result := <-scanner.ResultChan
		fmt.Printf("URL: %s | 状态码: %d | 耗时: %v | 错误: %v\n", result.FullURL, result.StatusCode, result.Duration, result.Err)
	}

	scanner.Stop()
} 
