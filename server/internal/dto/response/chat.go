// Package response 定义智能问答相关响应 DTO。
package response

// ChatSessionResponse 问答会话响应（含答案和来源）。
type ChatSessionResponse struct {
	SessionID       int64        `json:"session_id"`
	Question        string       `json:"question"`
	Answer          string       `json:"answer"`
	Sources         []SourceItem `json:"sources"`
	Confidence      float64      `json:"confidence"`
	CanSubmitTicket bool         `json:"can_submit_ticket"`
	DurationMS      int          `json:"duration_ms"`
	Feedback        int16        `json:"feedback"`
	CreatedAt       string       `json:"created_at"`
	// TODO(dto/chat): 增加 pipeline 字段保存管道指标。
	// 当前前端期待 pipeline_metrics，后端文档期待 pipeline，命名需要统一。
}

// SourceItem 知识来源条目。
type SourceItem struct {
	DocName      string  `json:"doc_name"`
	ChunkContent string  `json:"chunk_content"`
	Confidence   float64 `json:"confidence"`
}
