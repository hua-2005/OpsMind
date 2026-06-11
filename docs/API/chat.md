# 智能问答接口

> 基础路径：`/api/v1/portal` | 认证：JWT | 功能：RAG 增强问答 + SSE 流式输出

## 1. 创建问答会话（流式 SSE）

```http
POST /api/v1/portal/chat-sessions/stream
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体：**

```json
{
  "question": "如何重置 VPN 密码？",
  "kb_id": 1
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| question | string | ✓ | 用户问题 |
| kb_id | int64 | ✓ | 目标知识库 ID |

**SSE 事件流：**

响应类型为 `text/event-stream`，分两类事件：

**token 事件** — 逐块流式发送答案文本：

```
data: {"type":"token","content":"VPN 密码"}
data: {"type":"token","content":"重置步骤"}
data: {"type":"token","content":"如下：\n1."}
```

每次发送约 5 个字符，间隔约 30ms，模拟打字机效果。

**done 事件** — 流式结束，含完整元数据：

```
data: {"type":"done","metadata":{"session_id":42,"question":"如何重置 VPN 密码？","answer":"VPN 密码重置步骤：1. 登录自助平台...","sources":[{"doc_name":"VPN 密码重置 FAQ","chunk_content":"...","confidence":0.85}],"confidence":0.85,"can_submit_ticket":false,"duration_ms":3200,"feedback":0,"created_at":"2026-06-11 20:30:00"}}
```

**错误降级（非 SSE）：**

当 AI 服务不可用时，直接返回 JSON 错误：

```json
{
  "code": 20001,
  "message": "当前 AI 服务暂不可用，请提交申告由人工处理",
  "data": null
}
```

**前端消费示例：**

```typescript
import { streamChatSession } from '@/api/chat'

await streamChatSession(
  { question: '如何重置 VPN 密码？', kb_id: 1 },
  {
    onToken(content: string) {
      // 逐字符追加到 UI
      assistantMessage.content += content
    },
    onDone(session: ChatSessionResponse) {
      // 流式完成，更新来源和反馈按钮
      currentSession.value = session
    },
    onError(error: string) {
      // 显示错误提示
      showError(error)
    }
  }
)
```

---

## 2. 创建问答会话（非流式）

```http
POST /api/v1/portal/chat-sessions
Authorization: Bearer <token>
```

**请求体：** 同流式接口

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "session_id": 42,
    "question": "如何重置 VPN 密码？",
    "answer": "VPN 密码重置步骤：1. 登录自助平台...",
    "sources": [
      {
        "doc_name": "VPN 密码重置 FAQ",
        "chunk_content": "问题：如何重置 VPN 密码？答案：...",
        "confidence": 0.85
      }
    ],
    "confidence": 0.85,
    "can_submit_ticket": false,
    "duration_ms": 3200,
    "feedback": 0,
    "created_at": "2026-06-11 20:30:00"
  }
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| session_id | int64 | 会话 ID |
| question | string | 用户问题 |
| answer | string | AI 答案（或降级兜底文本） |
| sources | array | 知识来源列表 |
| sources[].doc_name | string | 来源文档名称 |
| sources[].chunk_content | string | 匹配的切片内容 |
| sources[].confidence | float | 该来源置信度 (0-1) |
| confidence | float | 整体置信度（取 sources 最高分） |
| can_submit_ticket | bool | 是否建议转人工申告 |
| duration_ms | int | RAG 查询耗时（毫秒） |

---

## 3. 查询会话详情

```http
GET /api/v1/portal/chat-sessions/:id
Authorization: Bearer <token>
```

**响应：** 同创建会话响应

---

## 4. 提交反馈

```http
POST /api/v1/portal/chat-sessions/:id/feedback
Authorization: Bearer <token>
```

**请求体：**

```json
{
  "feedback": 1
}
```

| 值 | 说明 |
|----|------|
| 0 | 未评价（默认） |
| 1 | 已解决 |
| 2 | 未解决 |

**成功响应 (200)：**

```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

## 降级规则

| 场景 | 行为 |
|------|------|
| AnythingLLM 容器不可达 | 返回 `code=20001`，提示 AI 不可用 |
| RAG 返回 sources 为空 | 返回兜底答案 + `can_submit_ticket=true` |
| 置信度 < 0.6 | 返回兜底答案 + `can_submit_ticket=true` |

**兜底文本：** 「暂未找到足够匹配的知识，建议提交申告由运维人员人工处理」

## 置信度阈值

默认阈值 `0.6`，可通过后台系统配置接口修改：

```http
PUT /api/v1/admin/configs/ai_confidence_threshold
```
