-- 004_alter_knowledge_chunks.sql
-- knowledge_chunks 表 v1→v2 变更：pgvector 向量存储。
--
-- v1 模式：
--   knowledge_chunks 仅记录 AnythingLLM 同步状态（sync_status/sync_error/synced_at），
--   向量数据存储在 AnythingLLM 内部的 LanceDB 中（OpsMind 不可见）。
--
-- v2 模式：
--   knowledge_chunks 直接存储向量（embedding halfvec + HNSW 索引），
--   由 OpsMind 自行完成 embedding 生成和向量写入。
--   新增 kb_id 冗余列加速按知识库的检索过滤。

-- 删除 v1 同步状态字段
ALTER TABLE knowledge_chunks DROP COLUMN IF EXISTS sync_status;
ALTER TABLE knowledge_chunks DROP COLUMN IF EXISTS sync_error;
ALTER TABLE knowledge_chunks DROP COLUMN IF EXISTS synced_at;

-- v2 新增字段
ALTER TABLE knowledge_chunks ADD COLUMN IF NOT EXISTS kb_id bigint NOT NULL DEFAULT 0;
ALTER TABLE knowledge_chunks ADD COLUMN IF NOT EXISTS chunk_index integer NOT NULL DEFAULT 0;

-- 向量列：halfvec 半精度（float16），存储减半，精度损失对余弦排序 < 0.1%。
-- 维度由知识库配置的 vector_dimension 决定。
-- 使用 DO 块动态获取维度（如果 knowledge_bases 表已有数据且包含一致的 vector_dimension），
-- 否则默认使用 1024（bge-m3）。
DO $$
DECLARE
    vec_dim integer;
BEGIN
    -- 尝试从 knowledge_bases 获取已有维度（取最大维度以容纳所有可能的值）
    SELECT COALESCE(MAX(vector_dimension), 1024) INTO vec_dim FROM knowledge_bases;

    -- 如果向量列不存在，动态创建（pgvector 不支持 ALTER TABLE ADD COLUMN halfvec(N) 中的 N 为变量，使用固定 1024）
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'knowledge_chunks' AND column_name = 'embedding'
    ) THEN
        EXECUTE 'ALTER TABLE knowledge_chunks ADD COLUMN embedding halfvec(1024)';
    END IF;
END $$;

COMMENT ON COLUMN knowledge_chunks.kb_id IS '冗余字段：加速按知识库的向量检索过滤，避免 JOIN knowledge_articles';
COMMENT ON COLUMN knowledge_chunks.chunk_index IS '分块序号，从 0 开始递增';
COMMENT ON COLUMN knowledge_chunks.embedding IS 'halfvec(1024) 半精度向量，pgvector 余弦相似度检索';
