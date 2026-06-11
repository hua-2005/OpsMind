import { test, expect } from '@playwright/test';
import {
  loadAuthState,
  authHeaders,
  assertSuccess,
  assertError,
  assertPaginatedResponse,
  uniqueUsername,
  validPassword,
} from '../utils/test-helpers.js';

/**
 * 用户管理接口集成测试。
 *
 * 覆盖端点：
 *   GET   /api/v1/admin/users          — 用户列表
 *   POST  /api/v1/admin/users          — 创建用户
 *   GET   /api/v1/admin/users/:id      — 用户详情
 *   PUT   /api/v1/admin/users/:id      — 更新用户
 *   PATCH /api/v1/admin/users/:id/freeze   — 冻结用户
 *   PATCH /api/v1/admin/users/:id/unfreeze — 恢复用户
 */

const BASE_URL = process.env.API_BASE_URL || 'http://localhost:8080';

let adminToken: string;
let createdUserId: number;
const testUsername = uniqueUsername();

test.beforeAll(() => {
  const state = loadAuthState();
  adminToken = state?.accessToken || '';
  if (!adminToken) {
    console.warn('⚠ 未找到认证状态，需要认证的测试将被跳过');
  }
});

test.describe('GET /api/v1/admin/users', () => {
  test('返回用户列表（分页）', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.get(`${BASE_URL}/api/v1/admin/users?page=1&page_size=10`, {
      headers: authHeaders(adminToken),
    });

    const body = await assertPaginatedResponse(resp, 1);
    const data = body.data as Array<Record<string, unknown>>;
    expect(Array.isArray(data)).toBe(true);

    if (data.length > 0) {
      const user = data[0];
      expect(user.id).toBeDefined();
      expect(user.username).toBeDefined();
      expect(user.real_name).toBeDefined();
      expect(user.status).toBeDefined();
      expect([1, 2]).toContain(user.status as number);
      expect(user.roles).toBeDefined();
      // 敏感字段不应暴露
      expect((user as Record<string, unknown>).password).toBeUndefined();
    }
  });

  test('无 token 访问返回 401', async ({ request }) => {
    const resp = await request.get(`${BASE_URL}/api/v1/admin/users`);
    await assertError(resp, 401, 10001);
  });
});

test.describe('POST /api/v1/admin/users', () => {
  test('创建用户成功', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.post(`${BASE_URL}/api/v1/admin/users`, {
      headers: authHeaders(adminToken),
      data: {
        username: testUsername,
        password: validPassword(),
        real_name: 'Playwright测试用户',
        phone: '13800001001',
        email: `${testUsername}@opsmind.local`,
        role_ids: [4], // 报障人角色
      },
    });

    const body = await assertSuccess(resp);
    const data = body.data as Record<string, unknown>;
    expect(data.id).toBeGreaterThan(0);
    createdUserId = data.id as number;
  });

  test('重复用户名返回 409 冲突', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.post(`${BASE_URL}/api/v1/admin/users`, {
      headers: authHeaders(adminToken),
      data: {
        username: testUsername, // 重复
        password: validPassword(),
        real_name: '重复用户',
        phone: '13800001002',
      },
    });

    await assertError(resp, 200, 10005);
  });

  test('密码不符合策略返回校验失败', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.post(`${BASE_URL}/api/v1/admin/users`, {
      headers: authHeaders(adminToken),
      // 纯数字密码，不符合策略
      data: {
        username: uniqueUsername(),
        password: '12345678',
        real_name: '弱密码用户',
        phone: '13800001003',
      },
    });

    await assertError(resp, 200, 10003);
  });

  test('缺少必填字段返回校验失败', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.post(`${BASE_URL}/api/v1/admin/users`, {
      headers: authHeaders(adminToken),
      data: {
        username: 'incomplete_user',
      },
    });

    await assertError(resp, 200, 10003);
  });
});

test.describe('GET /api/v1/admin/users/:id', () => {
  test('用户详情查询成功', async ({ request }) => {
    if (!adminToken || !createdUserId) {
      test.skip(true, '未找到认证状态或用户');
      return;
    }

    const resp = await request.get(`${BASE_URL}/api/v1/admin/users/${createdUserId}`, {
      headers: authHeaders(adminToken),
    });

    const body = await assertSuccess(resp);
    const data = body.data as Record<string, unknown>;
    expect(data.id).toBe(createdUserId);
    expect(data.username).toBe(testUsername);
  });

  test('不存在的用户返回 404', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.get(`${BASE_URL}/api/v1/admin/users/99999`, {
      headers: authHeaders(adminToken),
    });

    await assertError(resp, 200, 10004);
  });
});

test.describe('PUT /api/v1/admin/users/:id', () => {
  test('更新用户信息成功', async ({ request }) => {
    if (!adminToken || !createdUserId) {
      test.skip(true, '未找到认证状态或用户');
      return;
    }

    const resp = await request.put(`${BASE_URL}/api/v1/admin/users/${createdUserId}`, {
      headers: authHeaders(adminToken),
      data: {
        real_name: '更新后的姓名',
        phone: '13800001999',
        email: 'updated@opsmind.local',
        role_ids: [4],
      },
    });

    await assertSuccess(resp);
  });
});

test.describe('PATCH /api/v1/admin/users/:id/freeze', () => {
  test('冻结用户成功', async ({ request }) => {
    if (!adminToken || !createdUserId) {
      test.skip(true, '未找到认证状态或用户');
      return;
    }

    const resp = await request.patch(
      `${BASE_URL}/api/v1/admin/users/${createdUserId}/freeze`,
      { headers: authHeaders(adminToken) },
    );

    await assertSuccess(resp);
  });

  test('重复冻结应失败', async ({ request }) => {
    if (!adminToken || !createdUserId) {
      test.skip(true, '未找到认证状态或用户');
      return;
    }

    const resp = await request.patch(
      `${BASE_URL}/api/v1/admin/users/${createdUserId}/freeze`,
      { headers: authHeaders(adminToken) },
    );

    // 已冻结再次冻结应报错
    expect(resp.status()).toBe(200);
    const body = await resp.json();
    expect(body.code).toBeGreaterThan(0);
  });
});

test.describe('PATCH /api/v1/admin/users/:id/unfreeze', () => {
  test('恢复用户成功', async ({ request }) => {
    if (!adminToken || !createdUserId) {
      test.skip(true, '未找到认证状态或用户');
      return;
    }

    const resp = await request.patch(
      `${BASE_URL}/api/v1/admin/users/${createdUserId}/unfreeze`,
      { headers: authHeaders(adminToken) },
    );

    await assertSuccess(resp);
  });
});
