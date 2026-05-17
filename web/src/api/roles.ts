import request from './request'

export interface RoleItem {
  id: number
  code: string
  name: string
  status: string
  remark: string | null
}

export interface PermissionNode {
  id: number
  parent_id: number | null
  type: string
  name: string
  code: string
  path: string | null
  method: string | null
  children: PermissionNode[] | null
}

export function getRoles(params?: Record<string, string | number>) {
  return request.get<{ data: RoleItem[]; meta: Record<string, number> }>('/admin/roles', { params })
}

export function getPermissionsTree(params?: Record<string, string>) {
  return request.get<{ data: PermissionNode[] }>('/admin/permissions', { params })
}

export function updateRolePermissions(roleId: number, permissionIds: number[]) {
  return request.patch<{ data: { role_id: number; permission_count: number } }>(
    `/admin/roles/${roleId}/permissions`,
    { permission_ids: permissionIds },
  )
}
