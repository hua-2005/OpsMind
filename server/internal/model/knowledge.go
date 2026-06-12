package model

import (
	"time"

	"gorm.io/datatypes"
)

// KnowledgeBase 知识库表
type KnowledgeBase struct {
	ID               int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name             string    `gorm:"type:varchar(128);not null" json:"name"`
	Description      string    `gorm:"type:text" json:"description"`
	// TODO(model/knowledge): RAGWorkspaceSlug 是 v1 AnythingLLM 残留字段。
	// v2 已由 pgvector 管理向量数据，应确认是否删除或迁移为兼容字段。
	RAGWorkspaceSlug string    `gorm:"type:varchar(128);uniqueIndex;column:rag_workspace_slug" json:"rag_workspace_slug"`
	EmbeddingModel   string    `gorm:"type:varchar(128);not null;column:embedding_model" json:"embedding_model"`
	VectorDimension  int       `gorm:"not null;column:vector_dimension" json:"vector_dimension"`
	CreatedBy        int64     `gorm:"column:created_by" json:"created_by"`
	CreatedAt        time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt        time.Time `gorm:"not null" json:"updated_at"`
}

func (KnowledgeBase) TableName() string { return "knowledge_bases" }

// KnowledgeArticle 知识文章表
type KnowledgeArticle struct {
	ID                  int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	KBID                int64          `gorm:"not null;column:kb_id" json:"kb_id"`
	KnowledgeBase       KnowledgeBase  `gorm:"foreignKey:KBID;references:ID" json:"knowledge_base,omitempty"`
	// TODO(model/knowledge): Question/Answer 与 v2 的 title/content 统一文章模型不一致。
	// 应做一次 schema/DTO/前端字段迁移，减少双套术语。
	Question            string         `gorm:"type:text;not null" json:"question"`
	Answer              string         `gorm:"type:text;not null" json:"answer"`
	Category            string         `gorm:"type:varchar(64)" json:"category"`
	Tags                datatypes.JSON `gorm:"type:jsonb" json:"tags"`
	Status              int16          `gorm:"not null;default:1;index:idx_articles_status" json:"status"`
	CreatedBy           int64          `gorm:"column:created_by" json:"created_by"`
	ReviewedBy          *int64         `gorm:"column:reviewed_by" json:"reviewed_by"`
	PublishedBy         *int64         `gorm:"column:published_by" json:"published_by"`
	ReviewComment       string         `gorm:"type:text;column:review_comment" json:"review_comment"`
	// TODO(model/knowledge): RAGDocumentLocation 仍是 v1 同步位置语义。
	// 文档上传更需要 source_type/file_type/minio_path/process_status/process_error 等字段。
	RAGDocumentLocation string         `gorm:"type:varchar(512);column:rag_document_location" json:"rag_document_location"`
	CreatedAt           time.Time      `gorm:"not null" json:"created_at"`
	UpdatedAt           time.Time      `gorm:"not null" json:"updated_at"`
}

func (KnowledgeArticle) TableName() string { return "knowledge_articles" }

// KnowledgeChunk 知识切片表。
// 记录知识条目发布时的切片内容和 pgvector 向量。
// embedding 向量以 halfvec 类型存储在 pgvector column 中（由 VectorStore 适配器通过 SQL 管理，不走 GORM）。
type KnowledgeChunk struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ArticleID       int64     `gorm:"not null;column:article_id;index:idx_chunks_article_id" json:"article_id"`
	// TODO(model/knowledge): kb_id/chunk_index 已补到 GORM 模型，但 Service/DTO 仍有旧 sync_status 语义残留。
	// 需要继续向上清理响应字段，确保 API 不再暴露 AnythingLLM 同步概念。
	KBID            int64     `gorm:"not null;default:0;column:kb_id;index:idx_chunks_kb_id" json:"kb_id"`
	Content         string    `gorm:"type:text;not null" json:"content"`
	ChunkIndex      int       `gorm:"not null;default:0;column:chunk_index" json:"chunk_index"`
	EmbeddingModel  string    `gorm:"type:varchar(128);not null;column:embedding_model" json:"embedding_model"`
	VectorDimension int       `gorm:"not null;column:vector_dimension" json:"vector_dimension"`
	CreatedAt       time.Time `gorm:"not null" json:"created_at"`
	// embedding halfvec(1024) — 由 VectorStore 适配器通过 SQL 直接管理，不走 GORM
}

func (KnowledgeChunk) TableName() string { return "knowledge_chunks" }
