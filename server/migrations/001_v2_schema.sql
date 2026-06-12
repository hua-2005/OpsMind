-- OpsMind v2 数据库迁移（单体脚本）
--
-- 包含以下变更（原 v2/001~007 合并）：
--   1. pgvector 扩展安装
--   2. knowledge_bases 表结构变更（移除 rag_workspace_slug，新增 llm_config_id）
--   3. knowledge_articles 统一文章模型（question+answer → title+content+source_type）
--   4. knowledge_chunks pgvector 向量存储（halfvec + HNSW 索引）
--   5. llm_configs 表创建（LLM/Embedding 统一配置）
--   6. 业务索引和 HNSW 向量索引
--   7. chat_messages 新增 rag_pipeline 字段
--
-- 所有 DDL 使用 IF NOT EXISTS / DROP IF EXISTS 保证幂等。
-- 必须在 postgres:18 + pgvector 扩展镜像下执行。

-- =============================================================================
-- 1. pgvector 扩展
-- =============================================================================
CREATE EXTENSION IF NOT EXISTS vector;

-- =============================================================================
-- 2. knowledge_bases v1→v2
-- =============================================================================
-- 删除 v1 的 AnythingLLM workspace 关联，新增 LLM 配置外键
ALTER TABLE knowledge_bases DROP COLUMN IF EXISTS rag_workspace_slug;
ALTER TABLE knowledge_bases ADD COLUMN IF NOT EXISTS llm_config_id bigint;

-- =============================================================================
-- 3. knowledge_articles v1→v2：统一文章模型
-- =============================================================================
-- 删除 v1 FAQ 结构专属字段
ALTER TABLE knowledge_articles DROP COLUMN IF EXISTS question;
ALTER TABLE knowledge_articles DROP COLUMN IF EXISTS rag_document_location;

-- answer → content（保留数据）
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'knowledge_articles' AND column_name = 'answer'
    ) THEN
        ALTER TABLE knowledge_articles RENAME COLUMN answer TO content;
    END IF;
END $$;

-- v2 新增字段
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS title         varchar(255) NOT NULL DEFAULT '';
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS source_type  smallint NOT NULL DEFAULT 1;
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS word_count   integer NOT NULL DEFAULT 0;
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS chunk_count  integer NOT NULL DEFAULT 0;
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS file_type    varchar(16);
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS minio_path   varchar(512);
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS process_status varchar(16) NOT NULL DEFAULT 'completed';
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS process_error text;

-- 已有数据填充 title
UPDATE knowledge_articles SET title = LEFT(COALESCE(content, '未命名文章'), 50) WHERE title = '';

COMMENT ON COLUMN knowledge_articles.source_type IS '来源类型：1=manual(手动输入), 2=upload(文档上传)';
COMMENT ON COLUMN knowledge_articles.process_status IS '文档处理状态：pending/parsing/chunking/embedding/completed/failed';
COMMENT ON COLUMN knowledge_articles.file_type IS '文档格式：pdf/docx/md/txt，仅 source_type=upload';
COMMENT ON COLUMN knowledge_articles.minio_path IS 'MinIO 对象存储路径，仅 source_type=upload';

-- =============================================================================
-- 4. knowledge_chunks v1→v2：pgvector 向量存储
-- =============================================================================
-- 删除 v1 的 AnythingLLM 同步状态字段
ALTER TABLE knowledge_chunks DROP COLUMN IF EXISTS sync_status;
ALTER TABLE knowledge_chunks DROP COLUMN IF EXISTS sync_error;
ALTER TABLE knowledge_chunks DROP COLUMN IF EXISTS synced_at;

-- v2 新增字段
ALTER TABLE knowledge_chunks ADD COLUMN IF NOT EXISTS kb_id        bigint NOT NULL DEFAULT 0;
ALTER TABLE knowledge_chunks ADD COLUMN IF NOT EXISTS chunk_index  integer NOT NULL DEFAULT 0;

-- 向量列：halfvec(1024)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'knowledge_chunks' AND column_name = 'embedding'
    ) THEN
        ALTER TABLE knowledge_chunks ADD COLUMN embedding halfvec(1024);
    END IF;
END $$;

COMMENT ON COLUMN knowledge_chunks.kb_id IS '冗余字段：加速按知识库的向量检索过滤';
COMMENT ON COLUMN knowledge_chunks.chunk_index IS '分块序号，从 0 开始递增';
COMMENT ON COLUMN knowledge_chunks.embedding IS 'halfvec(1024) 半精度向量，pgvector 余弦相似度检索';

-- =============================================================================
-- 5. llm_configs — LLM/Embedding 提供商配置
-- =============================================================================
CREATE TABLE IF NOT EXISTS llm_configs (
    id               bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    name             varchar(128) NOT NULL,
    provider_type    smallint NOT NULL DEFAULT 1,
    base_url         varchar(512) NOT NULL,
    api_key          varchar(512),
    llm_model        varchar(128) NOT NULL,
    embedding_model  varchar(128) NOT NULL,
    max_tokens       integer NOT NULL DEFAULT 8192,
    vector_dimension integer NOT NULL DEFAULT 1024,
    is_default       boolean NOT NULL DEFAULT false,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE llm_configs IS 'LLM/Embedding 提供商配置。支持 llama.cpp server 和 OpenAI-compatible API';
COMMENT ON COLUMN llm_configs.provider_type IS '1=llama.cpp, 2=OpenAI-compatible';
COMMENT ON COLUMN llm_configs.api_key IS 'API 密钥（AES-256 加密存储）；llama.cpp 本地部署可为空';
COMMENT ON COLUMN llm_configs.vector_dimension IS 'Embedding 向量维度（bge-m3=1024, text-embedding-3-small=1536）';
COMMENT ON COLUMN llm_configs.is_default IS '是否系统默认配置。最多一条记录为 true';

CREATE UNIQUE INDEX IF NOT EXISTS idx_llm_configs_default ON llm_configs (is_default) WHERE is_default = true;

-- =============================================================================
-- 6. 索引
-- =============================================================================
-- HNSW 向量索引（重建以保证参数一致）
DROP INDEX IF EXISTS idx_chunks_embedding;
CREATE INDEX idx_chunks_embedding ON knowledge_chunks
    USING hnsw (embedding halfvec_cosine_ops)
    WITH (m = 16, ef_construction = 200);

-- 业务索引
CREATE INDEX IF NOT EXISTS idx_chunks_kb_id           ON knowledge_chunks (kb_id);
CREATE INDEX IF NOT EXISTS idx_chunks_article_id      ON knowledge_chunks (article_id);
CREATE INDEX IF NOT EXISTS idx_articles_status        ON knowledge_articles (status);
CREATE INDEX IF NOT EXISTS idx_articles_process_status ON knowledge_articles (process_status);

-- =============================================================================
-- 7. chat_messages 扩展
-- =============================================================================
ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS rag_pipeline jsonb;
COMMENT ON COLUMN chat_messages.rag_pipeline IS 'RAG 管道各步骤的执行耗时和状态（JSONB）';
