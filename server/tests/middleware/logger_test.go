// Package middleware_test 测试中间件的导出 API。
//
// 本文件测试请求日志中间件的日志输出。
package middleware_test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"opsmind/internal/middleware"
)

// setupLoggerRouter 创建用于测试日志中间件的 Gin 路由
func setupLoggerRouter(buf *bytes.Buffer) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.LoggerWithWriter(buf))
	r.GET("/test", func(c *gin.Context) {
		c.String(200, "ok")
	})
	r.POST("/test", func(c *gin.Context) {
		c.String(201, "created")
	})
	return r
}

// TestLogger_Method 测试日志记录请求方法
func TestLogger_Method(t *testing.T) {
	var buf bytes.Buffer
	r := setupLoggerRouter(&buf)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// 解析日志输出
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("解析日志失败: %v\n日志内容: %s", err, buf.String())
	}

	// 验证 method 字段
	method, ok := logEntry["method"].(string)
	if !ok {
		t.Fatalf("日志缺少 method 字段: %v", logEntry)
	}
	if method != "GET" {
		t.Errorf("期望 method=GET，实际 %s", method)
	}
}

// TestLogger_Path 测试日志记录请求路径
func TestLogger_Path(t *testing.T) {
	var buf bytes.Buffer
	r := setupLoggerRouter(&buf)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("解析日志失败: %v", err)
	}

	// 验证 path 字段
	path, ok := logEntry["path"].(string)
	if !ok {
		t.Fatalf("日志缺少 path 字段: %v", logEntry)
	}
	if path != "/test" {
		t.Errorf("期望 path=/test，实际 %s", path)
	}
}

// TestLogger_StatusCode 测试日志记录状态码
func TestLogger_StatusCode(t *testing.T) {
	var buf bytes.Buffer
	r := setupLoggerRouter(&buf)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("解析日志失败: %v", err)
	}

	// 验证 status_code 字段
	statusCode, ok := logEntry["status_code"].(float64)
	if !ok {
		t.Fatalf("日志缺少 status_code 字段: %v", logEntry)
	}
	if statusCode != 200 {
		t.Errorf("期望 status_code=200，实际 %v", statusCode)
	}
}

// TestLogger_Latency 测试日志记录耗时
func TestLogger_Latency(t *testing.T) {
	var buf bytes.Buffer
	r := setupLoggerRouter(&buf)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("解析日志失败: %v", err)
	}

	// 验证 latency 字段存在
	if _, ok := logEntry["latency"]; !ok {
		t.Fatalf("日志缺少 latency 字段: %v", logEntry)
	}
}

// TestLogger_ClientIP 测试日志记录客户端 IP
func TestLogger_ClientIP(t *testing.T) {
	var buf bytes.Buffer
	r := setupLoggerRouter(&buf)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "192.168.1.1")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("解析日志失败: %v", err)
	}

	// 验证 client_ip 字段存在
	if _, ok := logEntry["client_ip"]; !ok {
		t.Fatalf("日志缺少 client_ip 字段: %v", logEntry)
	}
}

// TestLogger_JSONFormat 测试日志输出为 JSON 格式
func TestLogger_JSONFormat(t *testing.T) {
	var buf bytes.Buffer
	r := setupLoggerRouter(&buf)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	// 验证输出是有效的 JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("日志输出不是有效的 JSON: %v\n日志内容: %s", err, buf.String())
	}
}

// TestLogger_DifferentStatusCodes 测试不同状态码的日志
func TestLogger_DifferentStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		statusCode int
	}{
		{"GET 200", "GET", "/test", 200},
		{"POST 201", "POST", "/test", 201},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			r := setupLoggerRouter(&buf)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			r.ServeHTTP(w, req)

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("解析日志失败: %v", err)
			}

			statusCode, ok := logEntry["status_code"].(float64)
			if !ok {
				t.Fatalf("日志缺少 status_code 字段: %v", logEntry)
			}
			if statusCode != float64(tt.statusCode) {
				t.Errorf("期望 status_code=%d，实际 %v", tt.statusCode, statusCode)
			}
		})
	}
}
