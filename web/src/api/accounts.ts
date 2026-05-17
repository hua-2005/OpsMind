import request from './request'

export interface AccountItem {
  id: number
  username: string
  real_name: string
  phone?: string
  email?: string
  status: string
  roles?: { code: string; name: string }[]
  last_login_at?: string
  created_at: string
}

export interface AccountDetail extends AccountItem {
  remark?: string
  roles: { id: number; code: string; name: string }[]
}

export interface ListResult<T> {
  data: T[]
  meta: { total: number; page: number; per_page: number; total_pages: number }
}

export function getAccounts(params: Record<string, string | number>) {
  return request.get<ListResult<AccountItem>>('/admin/accounts', { params })
}

export function createAccount(data: {
  username: string
  password: string
  real_name: string
  phone?: string
  email?: string
  role_ids: number[]
  remark?: string
}) {
  return request.post<{ data: AccountItem }>('/admin/accounts', data)
}

export function getAccountDetail(id: number) {
  return request.get<{ data: AccountDetail }>(`/admin/accounts/${id}`)
}

export function updateAccount(id: number, data: Record<string, unknown>) {
  return request.patch<{ data: { id: number; real_name: string; updated_at: string } }>(`/admin/accounts/${id}`, data)
}

export function freezeAccount(id: number, reason: string) {
  return request.post<{ data: { id: number; status: string } }>(`/admin/accounts/${id}/freeze`, { reason })
}

export function restoreAccount(id: number, reason: string) {
  return request.post<{ data: { id: number; status: string } }>(`/admin/accounts/${id}/restore`, { reason })
}
