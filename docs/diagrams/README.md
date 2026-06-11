# OpsMind 架构与业务流程图

> 基于实际代码函数调用链绘制，精确到 Handler → Service → Repository → Adapter 各层函数名。
> 最后更新：2026-06-11

---

## 架构图

| 文件 | 说明 |
|------|------|
| [architecture.md](architecture.md) | 系统架构总览 — 分层全景、模块依赖、ER 图、目录结构 |
| [request-lifecycle.md](request-lifecycle.md) | 请求生命周期 — 中间件链、路由分组、错误处理路径、启动顺序 |

---

## 业务数据流

| 文件 | 核心文件 | 说明 |
|------|----------|------|
| [auth-flow.md](auth-flow.md) | `auth_service.go`, `middleware/auth.go`, `middleware/rbac.go` | 登录/刷新/修改密码 + JWT 中间件 + RBAC 权限校验 + buildLoginResponse 菜单树构建 |
| [chat-rag-flow.md](chat-rag-flow.md) | `chat_service.go`, `adapter/rag_client.go` | 完整 RAG 链路：AnythingLLM 检索 → vLLM 生成 → 置信度判断 → 降级兜底 + 反馈 |
| [ticket-lifecycle.md](ticket-lifecycle.md) | `ticket_service.go`, `scheduler.go`, `message_service.go` | 申告完整生命周期：创建 → 状态机 5 态转换 → 补充信息循环 → 7 天自动关闭 → 站内消息 |
| [ticket-state-machine.md](ticket-state-machine.md) | `ticket_service.go` | 申告状态机（聚焦参考）— 状态转换规则与守卫条件 |
| [knowledge-publish-flow.md](knowledge-publish-flow.md) | `knowledge_service.go`, `adapter/rag_client.go` | 知识库创建 → 审核状态机 → AnythingLLM 同步 → 停用/重试 |
| [user-rbac-flow.md](user-rbac-flow.md) | `user_service.go`, `role_service.go`, `auth_service.go` | 用户 CRUD + 角色权限 + ER 数据模型 + buildLoginResponse 完整链路 |
| [dashboard-audit-flow.md](dashboard-audit-flow.md) | `dashboard_service.go`, `audit_repo.go`, `config_service.go` | 看板 7 项统计 SQL + 趋势数据 + 审计日志分散写入 + 系统配置 Upsert |

---

## 图例说明

| 符号 | 含义 |
|------|------|
| `actor` | 用户/外部角色 |
| `participant` | 系统组件（精确到文件名） |
| `par ... and ... end` | 并行执行 |
| `alt ... else ... end` | 条件分支 |
| `rect rgb(...)` | 子流程/内部调用 |
| `Note over` | 说明性注释 |
| `loop ... end` | 循环操作 |
| `stateDiagram-v2` | 状态机图 |
| `flowchart TD/LR` | 流程图（上下/左右） |
| `erDiagram` | 实体关系图 |
| `classDiagram` | 类图（方法总览） |

---

## 架构层次对应

```
Handler 层  →  handler/xxx.go      请求绑定、响应格式化
Service 层  →  service/xxx.go      业务逻辑、事务管理
Repository  →  repository/xxx.go   数据访问（GORM）
Adapter 层  →  adapter/xxx.go      外部服务（RagClient / StorageClient）
Middleware   →  middleware/xxx.go   认证/权限/日志
```
