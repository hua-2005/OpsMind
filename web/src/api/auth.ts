import request from './request'

export interface LoginParams {
  username: string
  password: string
}

export interface LoginResult {
  access_token: string
  expires_in: number
  user: { id: number; username: string; real_name: string; status: string }
  roles: string[]
  permissions: string[]
}

export interface Profile {
  id: number
  username: string
  real_name: string
  phone: string | null
  email: string | null
  status: string
  roles: string[]
}

export interface PermissionsData {
  menus: MenuItem[]
  buttons: string[]
  apis: string[] | null
}

export interface MenuItem {
  name: string
  code: string
  path: string
  children: MenuItem[] | null
}

export function login(params: LoginParams) {
  return request.post<{ data: LoginResult }>('/auth/login', params)
}

export function refresh() {
  return request.post<{ data: { access_token: string; expires_in: number } }>('/auth/refresh')
}

export function logout() {
  return request.post('/auth/logout')
}

export function getProfile() {
  return request.get<{ data: Profile }>('/auth/profile')
}

export function getPermissions() {
  return request.get<{ data: PermissionsData }>('/auth/permissions')
}
