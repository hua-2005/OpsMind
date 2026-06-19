// Package adapter 提供外部服务的适配层。
//
// llm_client.go 定义 LLMClient 接口和 OpenAI-compatible HTTP 实现。
// 所有 LLM 调用（文本生成、流式输出）必须通过此适配层，禁止直接 HTTP 调用。
//
// 接口设计决策（ADR-V2-002）：
// ChatCompletion 和 ChatCompletionStream 是两个独立方法，不通过参数切换。
// 调用方在编译时就知道自己需要流式还是非流式，分离方法比运行时判断更清晰。
package adapter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// =============================================================================
// 接口定义
// =============================================================================

// LLMClient 定义 LLM 调用接口（OpenAI-compatible 协议）。
//
// 支持任意 OpenAI-compatible API：
//   - llama.cpp server    → http://llama-cpp:8080/v1
//   - OpenAI              → https://api.openai.com/v1
//   - DeepSeek / Moonshot → 各服务商地址
type LLMClient interface {
	// ChatCompletion 同步对话 — 用于查询改写、多路路由、重排序等非流式场景。
	ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// ChatCompletionStream 流式对话 — 用于对用户的 SSE 实时回答。
	// 返回 channel 逐 token 输出，调用方通过 range channel 消费。
	// channel 在流式结束后由实现方关闭。
	ChatCompletionStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error)
}

// =============================================================================
// 请求/响应类型
// =============================================================================

// ChatRequest 对话请求。
//
// Stream 字段由 ChatCompletion（硬编码 false）和 ChatCompletionStream（硬编码 true）
// 在内部分别控制，不暴露给调用方——调用方通过方法选择（同步/流式）隐式决定是否流式。
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

// ChatMessage 对话消息。
type ChatMessage struct {
	Role    string `json:"role"`    // "system" | "user" | "assistant"
	Content string `json:"content"`
}

// ChatResponse 同步对话响应。
type ChatResponse struct {
	Content      string `json:"content"`       // 完整回复文本
	FinishReason string `json:"finish_reason"` // "stop" | "length"
	TokensUsed   int    `json:"tokens_used"`
}

// StreamChunk SSE 流式的单个 token 块。
type StreamChunk struct {
	Content      string `json:"content"`       // token 文本
	FinishReason string `json:"finish_reason"` // "stop" | "length" | ""（空表示未结束）
	Error        error  `json:"-"`             // 流式传输错误（channel 关闭前发送）
}

// =============================================================================
// OpenAI-compatible 实现
// =============================================================================

const (
	// defaultMaxRetries HTTP 请求最大重试次数
	defaultMaxRetries = 3
	// retryBaseDelay 重试基础延迟
	retryBaseDelay = 500 * time.Millisecond
)

// OpenAIClient 实现 LLMClient，对接 OpenAI-compatible API。
//
// 为什么使用标准 net/http 而非第三方 SDK：
// OpenAI-compatible API 足够简单（两个端点），标准库即可满足需求，避免引入额外依赖。
type OpenAIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	maxRetries int // 可重试的 HTTP 状态码（429/503）最大重试次数
}

// NewOpenAIClient 创建 OpenAIClient 实例。
//
// 校验 baseURL 非空且是合法 URL，避免空字符串在请求阶段产生难读的 "unsupported protocol scheme" 错误。
func NewOpenAIClient(baseURL, apiKey string, timeout time.Duration) *OpenAIClient {
	_ = validateBaseURL(baseURL) // 仅告警，创建阶段不阻断（兼容加载配置验证独立性）
	return &OpenAIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		maxRetries: defaultMaxRetries,
	}
}

// validateBaseURL 校验 baseURL 非空且是合法 URL，无效时记录告警。
func validateBaseURL(baseURL string) error {
	if baseURL == "" {
		slog.Warn("LLM baseURL 为空，后续请求将失败")
		return fmt.Errorf("baseURL 不能为空")
	}
	if _, err := url.Parse(baseURL); err != nil {
		slog.Warn("LLM baseURL 格式不合法", "url", baseURL, "error", err)
		return fmt.Errorf("baseURL 格式不合法: %w", err)
	}
	return nil
}

// =============================================================================
// ChatCompletion — 同步调用
// =============================================================================

// openAICompletionRequest OpenAI /chat/completions 请求体。
type openAICompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream"`
}

// openAICompletionResponse OpenAI /chat/completions 响应体。
type openAICompletionResponse struct {
	Choices []struct {
		Index int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// ChatCompletion 发送同步对话请求。
func (c *OpenAIClient) ChatCompletion(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		return nil, fmt.Errorf("ChatCompletion: req.Model 不能为空")
	}
	start := time.Now()
	body := openAICompletionRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}

	respBody, err := c.doRequest(ctx, "/chat/completions", body)
	if err != nil {
		slog.Error("LLM 同步调用失败", "model", req.Model, "latency_ms", time.Since(start).Milliseconds(), "error", err)
		return nil, err
	}

	var apiResp openAICompletionResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("解析 LLM 响应失败: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("LLM 返回空 choices")
	}

	slog.Info("LLM 同步调用完成", "model", req.Model, "tokens", apiResp.Usage.TotalTokens, "latency_ms", time.Since(start).Milliseconds())
	return &ChatResponse{
		Content:      apiResp.Choices[0].Message.Content,
		FinishReason: apiResp.Choices[0].FinishReason,
		TokensUsed:   apiResp.Usage.TotalTokens,
	}, nil
}

// =============================================================================
// ChatCompletionStream — 流式调用
// =============================================================================

// openAIStreamChunk OpenAI 流式响应的单个 SSE data 块。
type openAIStreamChunk struct {
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

// ChatCompletionStream 发送流式对话请求，返回 token channel。
//
// 为什么使用 buffered channel（容量 100）：
// HTTP 读取 goroutine 将解析后的 token 写入 channel，调用方从 channel 读取。
// buffered channel 避免网络抖动时 reader goroutine 阻塞，减少延迟。
func (c *OpenAIClient) ChatCompletionStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	body := openAICompletionRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("序列化流式请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建流式请求失败: %w", err)
	}
	c.setHeaders(httpReq)
	httpReq.Header.Set("Accept", "text/event-stream")

	slog.Info("LLM 流式调用开始", "model", req.Model)
	resp, err := c.streamRequestWithRetry(ctx, httpReq, req.Model)
	if err != nil {
		slog.Error("LLM 流式请求失败", "model", req.Model, "error", err)
		return nil, fmt.Errorf("流式请求 %s 失败: %w", c.baseURL, err)
	}

	ch := make(chan StreamChunk, 100)
	go c.readSSEStream(ctx, resp, ch)

	return ch, nil
}

// streamRequestWithRetry 对流式请求执行 429/503 重试。
//
// 流式请求不能复用 doRequest（doRequest 读取完整 body 后返回，与流式语义冲突），
// 但 429/503 仍应重试——本地 llama.cpp 可能在加载模型时返回 503。
func (c *OpenAIClient) streamRequestWithRetry(ctx context.Context, httpReq *http.Request, model string) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryBaseDelay * time.Duration(1<<(attempt-1))
			if delay > 8*time.Second {
				delay = 8 * time.Second
			}
			slog.Warn("LLM 流式请求重试中", "attempt", attempt, "delay_ms", delay.Milliseconds(), "error", lastErr)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		// 每次重试需要重新创建 HTTP 请求（Body 已被消费）
		newReq := httpReq.Clone(ctx)
		resp, err := c.httpClient.Do(newReq)
		if err != nil {
			lastErr = fmt.Errorf("流式请求 %s 失败: %w", c.baseURL, err)
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			slog.Warn("LLM 流式 API 返回可重试状态码", "model", model, "status", resp.StatusCode)
			lastErr = fmt.Errorf("LLM API 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
			continue
		}

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			slog.Error("LLM 流式 API 返回错误", "model", model, "status", resp.StatusCode)
			return nil, fmt.Errorf("LLM API 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
		}

		return resp, nil
	}

	return nil, fmt.Errorf("流式请求重试 %d 次后仍失败: %w", c.maxRetries, lastErr)
}

// readSSEStream 读取 SSE 流式响应，解析 data: 行并通过 channel 发送。
//
// 为什么在 goroutine 中读取而非调用方直接读取 Body：
// 流式读取需要持续占用 goroutine，channel 模式将「网络 IO」和「业务处理」解耦，
// 调用方可以用 range channel 消费 token，同时检测 ctx.Done() 实现断连处理。
//
// 所有 ch <- send 都通过 sendChunk 辅助函数执行，
// 当 ctx 取消或 channel 满且消费者已断开时，goroutine 优雅退出而非永久阻塞。
func (c *OpenAIClient) readSSEStream(ctx context.Context, resp *http.Response, ch chan<- StreamChunk) {
	defer close(ch)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	// 扩展 buffer 到 1MB，防止较大的 SSE data 行触发 ErrTooLong。
	// 例如 LLM 返回含大段代码块的 token 时，单行可能远超默认 64KB。
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// 移除 "data: " 前缀
		data := strings.TrimPrefix(line, "data: ")
		// 流式结束标记
		if data == "[DONE]" {
			return
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			// 解析失败：发送错误 token 并继续（非致命）
			if !sendChunk(ctx, ch, StreamChunk{Error: fmt.Errorf("解析 SSE chunk 失败: %w", err)}) {
				return
			}
			continue
		}

		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			var finishReason string
			if chunk.Choices[0].FinishReason != nil {
				finishReason = *chunk.Choices[0].FinishReason
			}
			if content != "" || finishReason != "" {
				if !sendChunk(ctx, ch, StreamChunk{
					Content:      content,
					FinishReason: finishReason,
				}) {
					return
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		sendChunk(ctx, ch, StreamChunk{Error: fmt.Errorf("读取 SSE 流失败: %w", err)})
	}
}

// sendChunk 安全地向 channel 发送 chunk，ctx 取消时返回 false。
//
// 使用 select 同时监听 ctx.Done() 和 channel send，
// 消费者断开连接时 goroutine 不会永久阻塞在 channel send 上。
func sendChunk(ctx context.Context, ch chan<- StreamChunk, chunk StreamChunk) bool {
	select {
	case <-ctx.Done():
		return false
	case ch <- chunk:
		return true
	}
}

// =============================================================================
// 辅助方法
// =============================================================================

// doRequest 发送 HTTP 请求并返回响应体。
//
// 对 429（限流）和 503（服务不可用）执行指数退避重试，最多 maxRetries 次。
func (c *OpenAIClient) doRequest(ctx context.Context, path string, body interface{}) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避: 500ms * 2^(attempt-1)，最大 8 秒
			delay := retryBaseDelay * time.Duration(1<<(attempt-1))
			if delay > 8*time.Second {
				delay = 8 * time.Second
			}
			slog.Warn("LLM HTTP 请求重试中", "attempt", attempt, "delay_ms", delay.Milliseconds(), "error", lastErr)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		respBody, err := c.tryRequest(ctx, path, jsonBody)
		if err == nil {
			return respBody, nil
		}
		lastErr = err

		// 仅 429/503 可重试，其他状态码直接返回
		if !isRetryable(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("重试 %d 次后仍失败: %w", c.maxRetries, lastErr)
}

// tryRequest 执行单次 HTTP 请求（不含重试逻辑）。
func (c *OpenAIClient) tryRequest(ctx context.Context, path string, jsonBody []byte) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 %s 失败: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
		return nil, &retryableError{statusCode: resp.StatusCode, body: string(respBody)}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// retryableError 可重试的 HTTP 错误（429/503）。
type retryableError struct {
	statusCode int
	body       string
}

func (e *retryableError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.statusCode, e.body)
}

// isRetryable 判断错误是否可重试。
func isRetryable(err error) bool {
	_, ok := err.(*retryableError)
	return ok
}

// doHTTPRequest 包级共享 HTTP 请求辅助函数，供 Embedding 客户端复用。
//
// 封装 setHeaders + HTTP 发送 + 状态码检查，消除 llm_client 与 embedding_client 的重复代码。
// 对 429/503 返回 retryableError，与 OpenAIClient.tryRequest 保持一致，使 EmbeddingClient.isRetryable 能正确识别。
func doHTTPRequest(ctx context.Context, baseURL, apiKey, path string, jsonBody []byte, client *http.Client) ([]byte, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 %s 失败: %w", baseURL, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
		return nil, &retryableError{statusCode: resp.StatusCode, body: string(respBody)}
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// setHeaders 设置通用请求头。
func (c *OpenAIClient) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
}
