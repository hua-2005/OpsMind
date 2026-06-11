-- 007_alter_chat_messages.sql
-- chat_messages 表新增 rag_pipeline 字段。
--
-- 存储 RAG 管道各步骤的执行耗时和状态，用于：
--   1. 数据看板的问答质量分析
--   2. 管道性能瓶颈定位
--   3. 用户问答页面的管道步骤回显
--
-- rag_pipeline JSONB 结构示例：
-- {
--   "steps": [
--     {"id": "query_rewrite", "label": "查询改写", "duration_ms": 120, "success": true},
--     {"id": "vector_retrieval", "label": "向量检索", "duration_ms": 45, "success": true},
--     {"id": "rerank", "label": "重排序", "duration_ms": 180, "success": true},
--     {"id": "llm_generate", "label": "LLM 生成", "duration_ms": 2800, "success": true}
--   ],
--   "total_duration_ms": 3145
-- }

ALTER TABLE chat_messages ADD COLUMN IF NOT EXISTS rag_pipeline jsonb;

COMMENT ON COLUMN chat_messages.rag_pipeline IS 'RAG 管道各步骤的执行耗时和状态（JSONB）';
