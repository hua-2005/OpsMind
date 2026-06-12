/**
 * 共享 API 响应类型定义
 *
 * 统一项目中所有 API 模块的请求/响应类型，消除分散在各 API 文件中的重复定义。
 * 后端所有接口均返回 { code: number, message: string, data: T } 格式，
 * axios 响应拦截器已将 AxiosResponse 的 data 层提取，因此 T 直接对应后端的 data 字段。
 *
 * 使用方式：
 *   import type { ApiResponse, PageResponse } from '@/types/api'
 *   request.get<ApiResponse<UserListData>>('/api/v1/admin/users', { params })
 */

/** 后端统一响应包装 — 所有 API 端点均使用此格式 */
export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

/** 分页响应 — 列表类接口统一使用此格式 */
export interface PageResponse<T> {
  // TODO(types/api): 后端 SuccessWithPage 实际把 items 放在 data，total/page/page_size 在顶层。
  // 当前 PageResponse<T> 与真实响应结构不一致，是大量 any 解包的根源。
  items: T[]
  total: number
}
