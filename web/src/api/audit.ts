/**
 * 审计日志 API 封装（后台管理端）
 */
import request from '@/utils/request'
import type { ApiResponse, PageResponse } from '@/types/api'

export interface AuditLogItem {
  id: number
  username?: string
  operator_name?: string
  action: string
  target_type?: string
  target_id?: number
  detail?: string
  // TODO(api/audit): 后端字段是 ip_address，前端这里写 ip。
  // 字段不一致会导致审计列表 IP 列为空。
  ip?: string
  created_at?: string
}

export interface AuditLogListParams {
  page?: number
  page_size?: number
}

export function listAuditLogs(params?: AuditLogListParams) {
  return request.get<ApiResponse<PageResponse<AuditLogItem>>>('/api/v1/admin/audit-logs', { params })
}
