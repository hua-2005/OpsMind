-- 006_create_indexes.sql
-- v2 数据库索引策略。
--
-- 为什么用 HNSW 而非 IVFFlat：
--   HNSW 查询速度更快（10000 级向量规模下 < 50ms vs IVFFlat ~100ms），
--   且 HNSW 构建后无需定期 REINDEX（IVFFlat 在大批量写入后需要重建）。
--   虽然 HNSW 构建时间较长（~1s/10000条），但构建发生于分块写入时，
--   与 embedding API 调用的延迟相比不构成瓶颈。
--
-- 为什么用 halfvec_cosine_ops：
--   cosine 距离是语义相似度的标准度量，halfvec 精度对排序影响 < 0.1%。
--
-- HNSW 参数选择（m=16, ef_construction=200）：
--   m=16: 每个节点的最大连接数（默认值，适合大多数场景）
--   ef_construction=200: 构建时的搜索深度（高于默认值，以构建时间为代价提升检索精度）

-- HNSW 向量索引（knowledge_chunks.embedding）
-- 如果索引已存在，先删除后重建（避免参数不一致）
DROP INDEX IF EXISTS idx_chunks_embedding;
CREATE INDEX idx_chunks_embedding ON knowledge_chunks
    USING hnsw (embedding halfvec_cosine_ops)
    WITH (m = 16, ef_construction = 200);

-- 业务索引：按知识库过滤（配合向量索引做 WHERE kb_id = ? 过滤）
CREATE INDEX IF NOT EXISTS idx_chunks_kb_id ON knowledge_chunks (kb_id);

-- 业务索引：按文章删除向量
CREATE INDEX IF NOT EXISTS idx_chunks_article_id ON knowledge_chunks (article_id);

-- 知识文章状态筛选（审核列表、发布列表按状态查询）
CREATE INDEX IF NOT EXISTS idx_articles_status ON knowledge_articles (status);

-- 文档处理状态筛选（后台管理按处理状态筛选上传文档）
CREATE INDEX IF NOT EXISTS idx_articles_process_status ON knowledge_articles (process_status);
