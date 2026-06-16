// Package request 定义智能问答相关请求 DTO。
package request

// RAGOptions 检索参数。
type RAGOptions struct {
	// TODO(dto/chat): bool 字段无法区分未传和显式 false。
	// 若不传 rag_options，后端应使用默认 true；若传 false，才关闭对应步骤。
	TopK         int  `json:"top_k"`
	QueryRewrite bool `json:"query_rewrite"`
	MultiRoute   bool `json:"multi_route"`
	Hybrid       bool `json:"hybrid"`
	Rerank       bool `json:"rerank"`
	// TODO(dto/chat): 缺少 route_count/rerank_count 字段，与 rag.RAGOptions 类型不同步。
	RouteCount   int  `json:"route_count"`
	RerankCount  int  `json:"rerank_count"`
}

// CreateChatRequest 创建问答会话请求。
type CreateChatRequest struct {
	Question   string      `json:"question" binding:"required"` // 用户问题
	KBID       int64       `json:"kb_id" binding:"required"`    // 目标知识库 ID
	SessionID  int64       `json:"session_id"`                  // 会话 ID（0=新会话, >0=追加到已有会话）
	RAGOptions *RAGOptions `json:"rag_options"`                 // RAG 管道参数（可选）
}
