# OpsMind 架构与业务流程图

> 基于实际代码函数调用链绘制。最后更新：2026-06-11

## v2 架构

v2 系统架构图见 [`docs/v2/TECHv2.md §1.1`](../v2/TECHv2.md#11-系统架构图)。

## 图表索引

### v2 适用（业务逻辑不变）

以下图表描述的业务流程在 v2 中仍然正确——这些模块不受 RAG 重构影响：

| 文件 | 核心文件 | 说明 |
|------|----------|------|
| [auth-flow.md](auth-flow.md) | `auth_service.go`, `middleware/auth.go` | 登录/刷新/修改密码 + JWT 中间件 + RBAC 权限校验 |
| [ticket-lifecycle.md](ticket-lifecycle.md) | `ticket_service.go`, `scheduler.go`, `message_service.go` | 申告完整生命周期：创建 → 状态机 5 态转换 → 补充信息 → 7 天自动关闭 |
| [ticket-state-machine.md](ticket-state-machine.md) | `ticket_service.go` | 申告状态机 — 状态转换规则与守卫条件 |
| [user-rbac-flow.md](user-rbac-flow.md) | `user_service.go`, `role_service.go` | 用户 CRUD + 角色权限 |
| [dashboard-audit-flow.md](dashboard-audit-flow.md) | `dashboard_service.go`, `audit_repo.go` | 看板统计 + 审计日志 + 系统配置 |
| [request-lifecycle.md](request-lifecycle.md) | 全栈 | 请求生命周期 — 中间件链、路由分组、错误处理 |

### v1 已归档（AnythingLLM 架构，与 v2 不对应）

| 图表 | 说明 |
|------|------|
| [architecture.md](architecture.md) | ~~v1 系统架构（含 AnythingLLM/vLLM 组件，v2 已移除）~~ |
| [chat-rag-flow.md](chat-rag-flow.md) | ~~v1 RAG 链路：AnythingLLM → vLLM。v2 对应流程见 `docs/v2/TECHv2.md §2.1.3`~~ |
| [knowledge-publish-flow.md](knowledge-publish-flow.md) | ~~v1 知识发布：AnythingLLM 同步。v2 对应流程见 `docs/API/knowledge.md §13`~~ |

## 架构层次对应（v2）

```
Handler 层  →  handler/xxx.go      请求绑定、响应格式化
Service 层  →  service/xxx.go      业务逻辑、事务管理
Repository  →  repository/xxx.go   数据访问（GORM）
RAG 引擎    →  rag/xxx.go          RAG 管道/检索/分块/embedding (v2 新增)
Adapter 层  →  adapter/xxx.go      外部服务（LLMClient / EmbeddingClient / VectorStore(pgvector) / StorageClient(MinIO)）
Middleware   →  middleware/xxx.go   认证/权限/日志
```
