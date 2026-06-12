import request from '@/utils/request'
import type { ApiResponse } from '@/types/api'
import type { MenuItem } from '@/stores/auth'

interface LoginParams {
  username: string
  password: string
}

interface LoginResponse {
  access_token: string
  refresh_token: string
  user: {
    id: number
    username: string
    real_name: string
    phone: string
    email: string
    first_login: boolean
  }
  roles: string[]
  permissions: string[]
  menus: MenuItem[]
}

interface ChangePasswordParams {
  old_password: string
  new_password: string
}

export function login(data: LoginParams) {
  return request.post<ApiResponse<LoginResponse>>('/api/v1/auth/login', data)
}

export function refreshToken(refresh_token: string) {
  return request.post<ApiResponse<{ access_token: string; refresh_token: string }>>('/api/v1/auth/refresh', { refresh_token })
}

export function changePassword(data: ChangePasswordParams) {
  // TODO(api/auth): 后端路由是 /api/v1/auth/me/change-password，不是 /api/v1/auth/change-password。
  // 当前调用会命中公开 auth 前缀之外的不存在路由。
  return request.post<ApiResponse<null>>('/api/v1/auth/change-password', data)
}

export function logout() {
  // TODO(api/auth): 后端路由是 /api/v1/auth/me/logout。
  // 前后端路径不一致会导致退出接口 404，虽然客户端也可本地清 token。
  return request.post<ApiResponse<null>>('/api/v1/auth/logout')
}
