/**
 * Token 存取工具函数
 *
 * 操作 localStorage 存储 JWT token。
 * key 固定为 'opsmind_token'，与后端约定一致。
 */

const TOKEN_KEY = 'opsmind_token'

/**
 * 获取存储的 token
 * @returns token 字符串，不存在时返回 null
 */
export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

/**
 * 存储 token
 * @param token JWT token 字符串
 */
export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

/**
 * 移除存储的 token
 */
export function removeToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}
