import { test, expect } from '@playwright/test';
import {
  loadAuthState,
  authHeaders,
  assertSuccess,
  assertError,
  validPassword,
} from '../utils/test-helpers.js';

/**
 * 认证接口集成测试。
 *
 * 覆盖端点：
 *   POST /api/v1/auth/login          — 登录
 *   POST /api/v1/auth/refresh        — 刷新令牌
 *   POST /api/v1/auth/change-password — 修改密码
 *   POST /api/v1/auth/logout         — 登出
 */

const BASE_URL = process.env.API_BASE_URL || 'http://localhost:8080';

test.describe('POST /api/v1/auth/login', () => {
  test('正确凭据登录成功，返回 token、用户信息和角色', async ({ request }) => {
    const resp = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: { username: 'admin', password: 'Admin@123' },
    });

    const body = await assertSuccess(resp);
    expect(body.data).toBeDefined();
    const data: Record<string, unknown> = body.data as Record<string, unknown>;
    // 必须返回 access_token
    expect(data.access_token).toBeDefined();
    expect(typeof data.access_token).toBe('string');
    expect((data.access_token as string).length).toBeGreaterThan(50);
    // 必须返回 refresh_token
    expect(data.refresh_token).toBeDefined();
    // 必须返回用户信息
    expect(data.user).toBeDefined();
    const user = data.user as Record<string, unknown>;
    expect(user.id).toBeGreaterThan(0);
    expect(user.username).toBe('admin');
    // 必须返回角色
    expect(Array.isArray(data.roles)).toBe(true);
    // 必须返回权限
    expect(Array.isArray(data.permissions)).toBe(true);
    // 必须返回菜单
    expect(Array.isArray(data.menus)).toBe(true);
  });

  test('错误密码返回 10003', async ({ request }) => {
    const resp = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: { username: 'admin', password: 'WrongPassword123' },
    });

    await assertError(resp, 200, 10003);
  });

  test('不存在的用户名返回 10003', async ({ request }) => {
    const resp = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: { username: 'nonexistent_user_xyz', password: 'Test1234' },
    });

    await assertError(resp, 200, 10003);
  });

  test('缺少必填字段返回参数校验失败', async ({ request }) => {
    const resp = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: { username: 'admin' },
    });

    // 可能返回 400 或 200 + 10003
    const body = await resp.json();
    expect([400, 200]).toContain(resp.status());
    expect([10003]).toContain(body.code);
  });
});

test.describe('POST /api/v1/auth/refresh', () => {
  test('有效 refresh_token 刷新成功', async ({ request }) => {
    // 先登录获取 refresh_token
    const loginResp = await request.post(`${BASE_URL}/api/v1/auth/login`, {
      data: { username: 'admin', password: 'Admin@123' },
    });
    const loginBody = await loginResp.json();
    const refreshToken = loginBody.data.refresh_token;

    const resp = await request.post(`${BASE_URL}/api/v1/auth/refresh`, {
      data: { refresh_token: refreshToken },
    });

    const body = await assertSuccess(resp);
    const data = body.data as Record<string, unknown>;
    expect(data.access_token).toBeDefined();
    expect(data.refresh_token).toBeDefined();
    // 新 token 不能和旧的一样
    expect(data.access_token).not.toBe(refreshToken);
  });

  test('无效 refresh_token 返回认证错误', async ({ request }) => {
    const resp = await request.post(`${BASE_URL}/api/v1/auth/refresh`, {
      data: { refresh_token: 'invalid_refresh_token_12345' },
    });

    await assertError(resp, 401, 10001);
  });

  test('缺少 refresh_token 返回参数校验失败', async ({ request }) => {
    const resp = await request.post(`${BASE_URL}/api/v1/auth/refresh`, {
      data: {},
    });

    const body = await resp.json();
    expect([400, 401]).toContain(resp.status());
    expect(body.code).toBeGreaterThan(0);
  });
});

test.describe('POST /api/v1/auth/change-password', () => {
  test('无 token 访问返回 401', async ({ request }) => {
    const resp = await request.post(`${BASE_URL}/api/v1/auth/change-password`, {
      data: { old_password: 'OldPass123', new_password: 'NewPass456' },
    });

    await assertError(resp, 401, 10001);
  });

  test('密码不符合策略返回校验失败', async ({ request }) => {
    const state = loadAuthState();
    if (!state) {
      test.skip(true, '未找到认证状态，请先运行 auth-setup');
      return;
    }

    const resp = await request.post(`${BASE_URL}/api/v1/auth/change-password`, {
      headers: authHeaders(state.accessToken),
      // 新密码纯数字，不符合「大小写字母+数字」策略
      data: { old_password: 'Admin@123', new_password: '12345678' },
    });

    await assertError(resp, 200, 10003);
  });

  test('新密码短于 8 位返回校验失败', async ({ request }) => {
    const state = loadAuthState();
    if (!state) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.post(`${BASE_URL}/api/v1/auth/change-password`, {
      headers: authHeaders(state.accessToken),
      data: { old_password: 'Admin@123', new_password: 'Ab1' },
    });

    await assertError(resp, 200, 10003);
  });
});

test.describe('POST /api/v1/auth/logout', () => {
  test('已认证用户登出成功', async ({ request }) => {
    const state = loadAuthState();
    if (!state) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.post(`${BASE_URL}/api/v1/auth/logout`, {
      headers: authHeaders(state.accessToken),
    });

    await assertSuccess(resp);
  });

  test('无 token 登出返回 401', async ({ request }) => {
    const resp = await request.post(`${BASE_URL}/api/v1/auth/logout`);

    await assertError(resp, 401, 10001);
  });
});
