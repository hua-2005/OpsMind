import { APIRequestContext, APIResponse, expect } from '@playwright/test';
import * as fs from 'fs';
import * as path from 'path';
import { fileURLToPath } from 'url';

/**
 * OpsMind API 测试共享工具函数。
 *
 * 提供统一的响应校验、认证 token 管理、分页参数构建等功能，
 * 避免各测试文件重复编写相同的校验逻辑。
 */

// ---- 类型定义 ----

/** OpsMind 统一 API 响应格式 */
export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
  total?: number;
  page?: number;
  page_size?: number;
}

/** 登录响应中保存的认证信息 */
export interface AuthState {
  accessToken: string;
  refreshToken: string;
  userId: number;
  username: string;
  roles: string[];
  expiresAt: number; // epoch ms
}

/** 创建资源后返回的 ID */
export interface CreatedEntity {
  id: number;
}

// ---- 认证状态管理 ----

// ESM 兼容的 __dirname 替代方案
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const AUTH_STATE_PATH = path.resolve(__dirname, '..', 'auth', 'auth-state.json');

/**
 * 从文件读取已保存的认证状态。
 * 如果 token 已过期（超过 1.5 小时），返回 null 以便重新登录。
 */
export function loadAuthState(): AuthState | null {
  try {
    if (!fs.existsSync(AUTH_STATE_PATH)) return null;
    const raw = fs.readFileSync(AUTH_STATE_PATH, 'utf-8');
    const state: AuthState = JSON.parse(raw);
    // Token 有效期 2 小时，提前 30 分钟刷新
    if (Date.now() > state.expiresAt - 30 * 60 * 1000) return null;
    return state;
  } catch {
    return null;
  }
}

/**
 * 保存认证状态到文件。
 */
export function saveAuthState(state: AuthState): void {
  const dir = path.dirname(AUTH_STATE_PATH);
  if (!fs.existsSync(dir)) fs.mkdirSync(dir, { recursive: true });
  fs.writeFileSync(AUTH_STATE_PATH, JSON.stringify(state, null, 2), 'utf-8');
}

// ---- 请求辅助函数 ----

/**
 * 返回带 JWT 认证头的 headers 对象。
 */
export function authHeaders(token: string): Record<string, string> {
  return {
    'Content-Type': 'application/json',
    Authorization: `Bearer ${token}`,
  };
}

/**
 * 返回 multipart/form-data 的认证头（不含 Content-Type，让 Playwright 自动设置 boundary）。
 */
export function authHeadersMultipart(token: string): Record<string, string> {
  return {
    Authorization: `Bearer ${token}`,
  };
}

// ---- 响应校验函数 ----

/**
 * 校验响应 HTTP 状态码和 OpsMind 业务 code。
 */
export async function assertSuccess(response: APIResponse, expectedCode = 0): Promise<ApiResponse> {
  expect(response.status()).toBe(200);
  const body: ApiResponse = await response.json();
  expect(body.code).toBe(expectedCode);
  if (expectedCode === 0) {
    expect(body.message).toBe('success');
  }
  return body;
}

/**
 * 校验失败响应（HTTP 非 200 或业务 code 非 0）。
 */
export async function assertError(
  response: APIResponse,
  expectedHttpStatus: number,
  expectedCode: number,
): Promise<ApiResponse> {
  expect(response.status()).toBe(expectedHttpStatus);
  const body: ApiResponse = await response.json();
  expect(body.code).toBe(expectedCode);
  expect(body.message).toBeTruthy();
  return body;
}

/**
 * 校验分页响应结构。
 */
export async function assertPaginatedResponse(
  response: APIResponse,
  minTotal = 0,
): Promise<ApiResponse> {
  const body = await assertSuccess(response);
  if (body.total !== undefined) {
    expect(typeof body.total).toBe('number');
    expect(body.total).toBeGreaterThanOrEqual(minTotal);
  }
  return body;
}

/**
 * 校验响应 data 不为 null/undefined。
 */
export function assertDataNotNull<T>(body: ApiResponse<T>): T {
  expect(body.data).not.toBeNull();
  expect(body.data).not.toBeUndefined();
  return body.data!;
}

/**
 * 构建分页查询参数。
 */
export function paginationParams(page = 1, pageSize = 10): Record<string, string> {
  return {
    page: String(page),
    page_size: String(pageSize),
  };
}

// ---- 测试数据工厂 ----

let seq = 0;
export function uniqueName(prefix: string): string {
  seq++;
  return `${prefix}_${Date.now()}_${seq}`;
}

export function uniqueUsername(): string {
  return `testuser_${Date.now()}_${++seq}`;
}

/**
 * 生成符合密码策略的测试密码（大小写字母+数字，8位以上）。
 */
export function validPassword(): string {
  return `Test${Date.now()}!1`;
}
