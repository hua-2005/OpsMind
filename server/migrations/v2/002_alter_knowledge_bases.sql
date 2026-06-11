-- 002_alter_knowledge_bases.sql
-- knowledge_bases 表 v1→v2 变更：
--   删除 rag_workspace_slug（v1 用于关联 AnythingLLM workspace）
--   新增 llm_config_id（关联 llm_configs.id，v2 用于指定 LLM/Embedding 配置）
--
-- 为什么删除 rag_workspace_slug：
-- v2 不再依赖 AnythingLLM，知识库不需要 workspace 概念。
-- 向量存储由 pgvector 在 knowledge_chunks 表中统一管理。

ALTER TABLE knowledge_bases DROP COLUMN IF EXISTS rag_workspace_slug;
ALTER TABLE knowledge_bases ADD COLUMN IF NOT EXISTS llm_config_id bigint;
