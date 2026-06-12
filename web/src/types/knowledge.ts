/**
 * 知识库相关类型定义
 *
 * 提供文章状态/处理状态枚举映射。
 */

/** 知识文章状态 */
export enum KnowledgeStatus {
  // TODO(types/knowledge): 知识状态枚举从 0 开始，与后端 ArticleStatusDraft=1 不一致。
  // 需要以前后端共享的 docs/API/knowledge.md 为准统一。
  Draft = 0,
  PendingReview = 1,
  Published = 2,
  Disabled = 3,
}

/** 文档处理状态 */
export enum ProcessStatus {
  // TODO(types/knowledge): 文档处理状态在 API 文档中是字符串，不是数字枚举。
  // 前端应改为 string union，避免和后端 process_status 绑定错位。
  Pending = 0,
  Processing = 1,
  Completed = 2,
  Failed = 3,
}

/** 知识文章状态文本映射 */
export const KNOWLEDGE_STATUS_TEXT: Record<number, string> = {
  [KnowledgeStatus.Draft]: '草稿',
  [KnowledgeStatus.PendingReview]: '待审核',
  [KnowledgeStatus.Published]: '已发布',
  [KnowledgeStatus.Disabled]: '已禁用',
}

/** 文档处理状态文本映射 */
export const PROCESS_STATUS_TEXT: Record<number, string> = {
  [ProcessStatus.Pending]: '待处理',
  [ProcessStatus.Processing]: '处理中',
  [ProcessStatus.Completed]: '已完成',
  [ProcessStatus.Failed]: '失败',
}
