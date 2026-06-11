-- 003_alter_knowledge_articles.sql
-- knowledge_articles 表 v1→v2 变更：统一文章模型。
--
-- v1 模型（FAQ 结构）：question + answer + rag_document_location
-- v2 模型（统一文章）：title + content + source_type + 文档元数据
--
-- 为什么改为统一模型：
-- v2 支持手动输入和文档上传两种来源，统一文章模型避免了
-- FAQ 结构字段和文档元数据字段的分离，简化检索和展示逻辑。
--
-- 迁移策略：
--   1. 保留已有数据——answer 列重命名为 content（不丢失数据）
--   2. 删除 question 列——原 question 内容由应用层迁移脚本写入 title
--   3. 新增 v2 字段——全部使用 ADD COLUMN IF NOT EXISTS 支持幂等执行

-- 删除 v1 专属字段
ALTER TABLE knowledge_articles DROP COLUMN IF EXISTS question;
ALTER TABLE knowledge_articles DROP COLUMN IF EXISTS rag_document_location;

-- answer → content（保留原有数据）
-- 注意：如果 content 列已存在（重命名已执行），跳过此步骤
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
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS title varchar(255) NOT NULL DEFAULT '';
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS source_type smallint NOT NULL DEFAULT 1;
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS word_count integer NOT NULL DEFAULT 0;
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS chunk_count integer NOT NULL DEFAULT 0;
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS file_type varchar(16);
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS minio_path varchar(512);
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS process_status varchar(16) NOT NULL DEFAULT 'completed';
ALTER TABLE knowledge_articles ADD COLUMN IF NOT EXISTS process_error text;

-- 为已有数据填充默认值：title 取 content 前 50 字符（如果 title 仍为空）
UPDATE knowledge_articles SET title = LEFT(COALESCE(content, '未命名文章'), 50) WHERE title = '';

COMMENT ON COLUMN knowledge_articles.source_type IS '来源类型：1=manual(手动输入), 2=upload(文档上传)';
COMMENT ON COLUMN knowledge_articles.process_status IS '文档处理状态：pending/parsing/chunking/embedding/completed/failed';
COMMENT ON COLUMN knowledge_articles.file_type IS '文档格式：pdf/docx/md/txt，仅 source_type=upload';
COMMENT ON COLUMN knowledge_articles.minio_path IS 'MinIO 对象存储路径，仅 source_type=upload';
