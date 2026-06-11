-- 001_pgvector_extension.sql
-- 安装 pgvector 扩展，启用向量存储能力。
--
-- pgvector 提供 vector/halfvec 类型和余弦距离算子 (<=>)，
-- 用于 knowledge_chunks 表的向量存储和相似度检索。
-- 必须在 postgres:18 + pgvector 扩展镜像或手动安装 pgvector 后执行。

CREATE EXTENSION IF NOT EXISTS vector;

-- 验证扩展是否可用
-- SELECT extname, extversion FROM pg_extension WHERE extname = 'vector';
