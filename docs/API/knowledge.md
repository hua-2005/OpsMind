# 知识库管理接口

> 基础路径：`/api/v1/admin` | 认证：JWT + RBAC

## 知识文章状态

```
草稿(1) → 已提交审核(2) → 审核通过(3) → 已发布(4) → 已停用(5)
               ↓                         ↓
          审核驳回(6)              同步失败(可重试)
```

## 知识库 CRUD

### 1. 知识库列表

```http
GET /api/v1/admin/knowledge-bases
Authorization: Bearer <token>
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "id": 1,
        "name": "IT 运维 FAQ",
        "description": "常见的 IT 运维问题和解决方案",
        "rag_workspace_slug": "opsmind-it-ops",
        "embedding_model": "bge-m3",
        "vector_dimension": 1024,
        "created_by": 1,
        "created_at": "2026-06-11 19:27:43",
        "updated_at": "2026-06-11 19:27:43"
      }
    ]
  }
}
```

### 2. 创建知识库

```http
POST /api/v1/admin/knowledge-bases
Authorization: Bearer <token>
```

**请求体：**

```json
{
  "name": "网络运维 FAQ",
  "description": "网络相关的运维知识",
  "embedding_model": "bge-m3"
}
```

> 创建时会在 AnythingLLM 中自动创建对应的 workspace。

### 3. 更新知识库

```http
PUT /api/v1/admin/knowledge-bases/:id
Authorization: Bearer <token>
```

---

## 知识文章 CRUD

### 4. 文章列表

```http
GET /api/v1/admin/knowledge-bases/:kb_id/articles?page=1&page_size=10
Authorization: Bearer <token>
```

**响应：**

```json
{
  "code": 0,
  "data": {
    "articles": [
      {
        "id": 1,
        "kb_id": 1,
        "kb_name": "IT 运维 FAQ",
        "question": "如何重置 VPN 密码？",
        "answer": "请登录 VPN 自助服务平台...",
        "category": "网络与VPN",
        "tags": ["VPN", "密码", "自助"],
        "status": 4,
        "status_text": "已发布",
        "sync_status": "synced",
        "created_at": "2026-06-11T19:27:43Z",
        "updated_at": "2026-06-11T19:27:43Z"
      }
    ],
    "total": 5
  }
}
```

### 5. 创建文章

```http
POST /api/v1/admin/knowledge-bases/:kb_id/articles
Authorization: Bearer <token>
```

**请求体：**

```json
{
  "question": "公司 VPN 连接超时怎么办？",
  "answer": "1. 检查网络连接\n2. 尝试备用线路 vpn2.company.com\n3. 联系 IT 服务台（分机 8888）",
  "category": "网络与VPN",
  "tags": ["VPN", "连接", "超时"],
  "kb_id": 1
}
```

> 状态初始为「草稿(1)」。

### 6. 更新文章

```http
PUT /api/v1/admin/articles/:id
Authorization: Bearer <token>
```

### 7. 文章详情

```http
GET /api/v1/admin/articles/:id
Authorization: Bearer <token>
```

**响应含切片信息：**

```json
{
  "code": 0,
  "data": {
    "id": 1,
    "question": "如何重置 VPN 密码？",
    "answer": "...",
    "chunks": [
      {
        "id": 1,
        "content": "如何重置 VPN 密码？请登录 VPN 自助服务平台。",
        "embedding_model": "bge-m3",
        "vector_dimension": 1024,
        "sync_status": "synced"
      }
    ]
  }
}
```

---

## 审核流程

### 8. 提交审核

```http
POST /api/v1/admin/articles/:id/submit-review
Authorization: Bearer <token>
```

> 状态：草稿(1) → 已提交审核(2)

### 9. 审核操作

```http
POST /api/v1/admin/articles/:id/review
Authorization: Bearer <token>
```

**请求体：**

```json
{
  "approved": true,
  "review_comment": "内容准确，通过审核"
}
```

> `approved=true` → 审核通过(3)，否则 → 审核驳回(6)
>
> **业务规则：** 审核人不能是文章创建人。

---

## 发布与停用

### 10. 发布

```http
POST /api/v1/admin/articles/:id/publish
Authorization: Bearer <token>
```

> 状态：审核通过(3) → 已发布(4)
>
> 发布时同步调用 AnythingLLM API 将文档内容写入指定 workspace，触发 embedding 和向量写入。

### 11. 停用

```http
POST /api/v1/admin/articles/:id/disable
Authorization: Bearer <token>
```

> 状态：已发布(4) → 已停用(5)
>
> 停用时调用 AnythingLLM API 从 workspace 中删除对应文档的向量。

### 12. 重试同步

```http
POST /api/v1/admin/articles/:id/retry-sync
Authorization: Bearer <token>
```

> 当发布时同步 AnythingLLM 失败（`sync_status=failed`），可重试。

---

## Embedding 配置

### 13. Embedding 配置列表

```http
GET /api/v1/admin/embedding-configs
Authorization: Bearer <token>
```

### 14. 创建配置

```http
POST /api/v1/admin/embedding-configs
Authorization: Bearer <token>
```

**请求体：**

```json
{
  "name": "BGE-M3 本地",
  "model_type": 2,
  "local_path": "/models/bge-m3",
  "vector_dimension": 1024,
  "is_default": true
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | ✓ | 配置名称 |
| model_type | int | ✓ | 1=API（远程）, 2=本地 |
| api_endpoint | string | | API 地址（model_type=1 时） |
| api_key | string | | API Key（model_type=1 时） |
| local_path | string | | 本地模型路径（model_type=2 时） |
| vector_dimension | int | ✓ | 向量维度 |
| is_default | bool | | 是否设为默认 |

### 15-16. 更新 / 删除配置

```http
PUT /api/v1/admin/embedding-configs/:id
DELETE /api/v1/admin/embedding-configs/:id
```
