/**
 * Token 存取工具函数
 *
 * 操作 localStorage 存储 JWT token。
 * key 固定为 'opsmind_token'，与后端约定一致。
 */

const TOKEN_KEY = 'opsmind_token'
// TODO(web/auth): 只保存 access_token，refresh_token 没有持久化和自动刷新流程。
// access token 过期后用户会直接回登录页，体验和后端双令牌设计不匹配。
// TODO(web/auth): localStorage 容易受 XSS 影响。
// 若威胁模型要求更高，可评估 httpOnly cookie 或至少缩短 token 生命周期并加强 CSP。

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
