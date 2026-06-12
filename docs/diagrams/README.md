# OpsMind 架构与业务流程图

> 基于实际代码函数调用链绘制。最后更新：2026-06-12

## 图表索引

| 文件 | 核心文件 | 说明 |
|------|----------|------|
| [architecture.md](architecture.md) | `main.go`, `router/`, `middleware/` | 系统架构总览 — 分层全景 + 请求生命周期 + 模块依赖 |
| [chat-rag-flow.md](chat-rag-flow.md) | `handler/chat.go`, `service/chat_service.go`, `rag/pipeline.go`, `adapter/llm_client.go` | 智能问答 RAG 管道 — SSE 流式 + 非流式 + 降级矩阵 |
| [knowledge-publish-flow.md](knowledge-publish-flow.md) | `handler/knowledge.go`, `service/knowledge_service.go`, `rag/chunker.go`, `rag/embedder.go`, `adapter/vector_store.go` | 知识文章生命周期 — 创建→审核→发布(pgvector 管道)→停用 |
| [document-upload-flow.md](document-upload-flow.md) | `handler/knowledge.go`, `service/knowledge_service.go`, `rag/document_parser.go`, `rag/processor.go` | 文档上传与异步处理 — PDF/DOCX/MD/TXT 解析→分块→embedding |
| [llm-config-flow.md](llm-config-flow.md) | `handler/llm_config.go`, `service/llm_config_service.go`, `repository/llm_config_repo.go` | LLM 配置管理 — CRUD + 测试连接 + atomic.Value 热替换 |
| [ticket-lifecycle.md](ticket-lifecycle.md) | `handler/ticket.go`, `service/ticket_service.go`, `scheduler.go` | 申告完整生命周期 — 创建→状态机 5 态转换→超时自动关闭 |
| [ticket-state-machine.md](ticket-state-machine.md) | `service/ticket_service.go` | 申告状态机 — 状态转换规则与守卫条件 |
| [auth-flow.md](auth-flow.md) | `handler/auth.go`, `service/auth_service.go`, `middleware/auth.go` | 认证流程 — 登录/JWT/RBAC 权限校验 |
| [user-rbac-flow.md](user-rbac-flow.md) | `handler/user.go`, `handler/role.go`, `service/user_service.go`, `service/role_service.go` | 用户管理 + 角色权限 |
| [dashboard-audit-flow.md](dashboard-audit-flow.md) | `handler/dashboard.go`, `service/dashboard_service.go`, `handler/audit.go`, `service/audit_service.go` | 看板统计 + 审计日志 |
| [request-lifecycle.md](request-lifecycle.md) | `middleware/`, `router/` | 请求生命周期 — 中间件链、路由分组、错误处理 |

## 架构层次对应

```
Handler 层   →  handler/xxx.go      请求绑定、响应格式化
Service 层   →  service/xxx.go      业务逻辑、事务管理
Repository   →  repository/xxx.go   数据访问（GORM）
RAG 引擎     →  rag/xxx.go          Pipeline / BM25 / HybridFuse / Rerank / Chunker / Embedder / Processor
Adapter 层   →  adapter/xxx.go      LLMClient / EmbeddingClient / VectorStore(pgvector) / StorageClient(MinIO)
Middleware   →  middleware/xxx.go    RequestID / CORS / Logger / JWTAuth / RBAC
```

## 关键函数速查

| 流程 | 入口 (Handler) | 核心 (Service) | 数据面 |
|------|---------------|----------------|--------|
| 智能问答 | `ChatHandler.CreateChatSession` / `StreamChatSession` | `ChatService.CreateChatSession` → `Pipeline.Execute` → `LLMClient.ChatCompletion` | `chat_sessions`, `chat_messages` |
| 知识发布 | `KnowledgeHandler.Publish` | `KnowledgeService.Publish` → `Chunker.Split` → `Embedder.Embed` → `VectorStore.BatchInsert` | `knowledge_articles`, `knowledge_chunks` |
| 文档上传 | `KnowledgeHandler.UploadDocuments` | `KnowledgeService.UploadDocuments` → `DocParser.Parse` → `Processor.Submit` | `knowledge_articles`, MinIO |
| 申告管理 | `TicketHandler.UpdateStatus` | `TicketService.UpdateStatus` → 状态机校验 | `tickets`, `ticket_records` |
| LLM 配置 | `LLMConfigHandler.TestConnection` | `LLMConfigService.TestConnection` → `LLMClient.ChatCompletion` | `llm_configs` |
| 认证 | `AuthHandler.Login` | `AuthService.Login` → `bcrypt.CompareHashAndPassword` | `users`, `user_roles` |
