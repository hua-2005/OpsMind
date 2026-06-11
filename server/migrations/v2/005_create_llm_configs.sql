-- 005_create_llm_configs.sql
-- 创建 llm_configs 表 — LLM/Embedding 提供商配置。
--
-- v2 新增：统一管理 llama.cpp server 和 OpenAI-compatible API 的连接参数。
-- 替代 v1 分散在环境变量和 embedding_configs 表中的配置方式。
--
-- 为什么统一 LLM 和 Embedding 配置：
-- llama.cpp 和 OpenAI-compatible API 都通过同一 Base URL 提供服务，
-- LLM 和 Embedding 只是模型名称不同（如 /v1/chat/completions vs /v1/embeddings），
-- 统一配置减少用户的理解成本和配置错误。

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

-- 唯一部分索引：确保最多一个默认配置
CREATE UNIQUE INDEX IF NOT EXISTS idx_llm_configs_default ON llm_configs (is_default) WHERE is_default = true;
