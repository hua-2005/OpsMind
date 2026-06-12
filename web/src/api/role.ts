import request from '@/utils/request'
import type { ApiResponse } from '@/types/api'

export interface RoleItem {
  id: number
  name: string
  description: string
  permissions: string[]
  created_at?: string
  updated_at?: string
}

export function getRoleList() {
  // TODO(api/role): 后端角色列表使用分页响应，当前类型声明为 RoleItem[]。
  // 调用方大量使用 res?.data || res 兼容，根因是这里类型与真实响应不一致。
  return request.get<ApiResponse<RoleItem[]>>('/api/v1/admin/roles')
}
