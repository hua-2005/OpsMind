/**
 * 知识库管理 API — 全部 17 个端点已通过 ApiResponse&lt;T&gt; 补全泛型类型。
 */
import request from '@/utils/request'
import type { ApiResponse, PageResponse } from '@/types/api'

// =============================================================================
// 知识库 (KnowledgeBase)
// =============================================================================

interface CreateKBParams {
  name: string
  description?: string
  embedding_model?: string
}

interface UpdateKBParams {
  name: string
  description?: string
}

/** 知识库数据模型 */
interface KnowledgeBaseItem {
  id: number
  name: string
  description?: string
  embedding_model?: string
  created_at?: string
}

/** 知识库列表（后台管理用，需要 admin 权限） */
export function listKnowledgeBases() {
  // TODO(api/knowledge): 后端 ListKBs 当前直接返回数组，部分页面却按 data.items 解包。
  // 应统一为 PageResponse/KBListResponse 或直接数组，移除调用方兼容代码。
  return request.get<ApiResponse<KnowledgeBaseItem[]>>('/api/v1/admin/knowledge-bases')
}

/** 知识库列表（门户端用，无需 admin 权限 — Chat 页知识库下拉框） */
export function listKnowledgeBasesForPortal() {
  return request.get<ApiResponse<{ items: KnowledgeBaseItem[] }>>('/api/v1/portal/knowledge-bases')
}

export function createKnowledgeBase(data: CreateKBParams) {
  return request.post<ApiResponse<KnowledgeBaseItem>>('/api/v1/admin/knowledge-bases', data)
}

export function updateKnowledgeBase(id: number, data: UpdateKBParams) {
  return request.put<ApiResponse<KnowledgeBaseItem>>(`/api/v1/admin/knowledge-bases/${id}`, data)
}

// =============================================================================
// 知识文章 (KnowledgeArticle)
// =============================================================================

interface CreateArticleParams {
  // TODO(api/knowledge): 前端发送 title/content，但后端 DTO 仍要求 question/answer。
  // 在后端迁移前，这里会创建失败；应尽快统一 v2 文章模型字段。
  kb_id: number
  title: string
  content: string
  source_type?: number  // 1=手动, 2=上传
  category?: string
  tags?: string[]
}

interface UpdateArticleParams {
  title: string
  content: string
  category?: string
  tags?: string[]
}

interface ArticleListParams {
  page?: number
  page_size?: number
  status?: number
}

/** 知识文章数据模型 */
interface KnowledgeArticleItem {
  id: number
  kb_id: number
  title: string
  content: string
  source_type: number
  category?: string
  tags?: string[]
  status: number
  process_status?: number
  process_error?: string
  created_at?: string
  updated_at?: string
}

export function listArticles(kbID: number, params: ArticleListParams) {
  return request.get<ApiResponse<PageResponse<KnowledgeArticleItem>>>(`/api/v1/admin/knowledge-bases/${kbID}/articles`, { params })
}

export function getArticleDetail(id: number) {
  return request.get<ApiResponse<KnowledgeArticleItem>>(`/api/v1/admin/articles/${id}`)
}

export function createArticle(kbID: number, data: CreateArticleParams) {
  return request.post<ApiResponse<KnowledgeArticleItem>>(`/api/v1/admin/knowledge-bases/${kbID}/articles`, data)
}

export function updateArticle(id: number, data: UpdateArticleParams) {
  return request.put<ApiResponse<KnowledgeArticleItem>>(`/api/v1/admin/articles/${id}`, data)
}

export function submitReview(id: number) {
  return request.post<ApiResponse<null>>(`/api/v1/admin/articles/${id}/submit-review`)
}

export function reviewArticle(id: number, data: { approved: boolean; review_comment?: string }) {
  return request.post<ApiResponse<null>>(`/api/v1/admin/articles/${id}/review`, data)
}

export function publishArticle(id: number) {
  return request.post<ApiResponse<null>>(`/api/v1/admin/articles/${id}/publish`)
}

export function disableArticle(id: number) {
  return request.post<ApiResponse<null>>(`/api/v1/admin/articles/${id}/disable`)
}

export function enableArticle(id: number) {
  return request.post<ApiResponse<null>>(`/api/v1/admin/articles/${id}/enable`)
}

export function retrySyncArticle(id: number) {
  return request.post<ApiResponse<null>>(`/api/v1/admin/articles/${id}/retry-sync`)
}

// =============================================================================
// v2 文档上传/处理（替代旧 RAG 同步）
// =============================================================================

/** 上传文档到知识库（multipart form） */
export function uploadDocuments(kbID: number, formData: FormData) {
  // TODO(api/knowledge): 后端当前读取字段 file，API 文档要求 files，多文件上传也未对齐。
  // 前端 FormData append 的 key 必须和后端统一，否则上传接口不可用。
  return request.post<ApiResponse<KnowledgeArticleItem>>(`/api/v1/admin/knowledge-bases/${kbID}/documents/upload`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

/** 查询文档处理状态 */
export function getDocumentStatus(kbID: number, articleID: number) {
  // TODO(api/knowledge): 后端返回 process_status 字符串，类型这里写 status/status_text number。
  // 需要按 docs/API/knowledge.md 对齐为 process_status/process_error/progress。
  return request.get<ApiResponse<{ status: number; status_text: string }>>(`/api/v1/admin/knowledge-bases/${kbID}/documents/${articleID}/status`)
}

/** 重试文档处理 */
export function retryDocument(kbID: number, articleID: number) {
  return request.post<ApiResponse<null>>(`/api/v1/admin/knowledge-bases/${kbID}/documents/${articleID}/retry`)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Embedding 配置 API 已迁移至 llm_config.ts（v2 Task 6.5 清理）
//
// 旧接口 (embedding-configs) 已被后端移除，前端使用新的 LLM 配置 API 替代。
// 详见：web/src/api/llm_config.ts
// ═══════════════════════════════════════════════════════════════════════════════
