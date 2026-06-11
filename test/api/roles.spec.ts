import { test, expect } from '@playwright/test';
import {
  loadAuthState,
  authHeaders,
  assertSuccess,
  assertError,
  assertPaginatedResponse,
  uniqueName,
} from '../utils/test-helpers.js';

/**
 * 角色与菜单管理接口集成测试。
 *
 * 覆盖端点：
 *   GET    /api/v1/admin/roles          — 角色列表
 *   POST   /api/v1/admin/roles          — 创建角色
 *   GET    /api/v1/admin/roles/:id      — 角色详情
 *   PUT    /api/v1/admin/roles/:id      — 更新角色
 *   DELETE /api/v1/admin/roles/:id      — 删除角色
 *   GET    /api/v1/admin/menus          — 菜单列表
 *   PUT    /api/v1/admin/roles/:id/menus — 更新角色菜单权限
 */

const BASE_URL = process.env.API_BASE_URL || 'http://localhost:8080';

let adminToken: string;

test.beforeAll(() => {
  const state = loadAuthState();
  adminToken = state?.accessToken || '';
  if (!adminToken) {
    console.warn('⚠ 未找到认证状态，需要认证的测试将被跳过');
  }
});

// ==================== 角色 CRUD ====================

test.describe('角色管理', () => {
  let createdRoleId: number;
  const roleName = uniqueName('测试角色');

  test.describe('GET /api/v1/admin/roles', () => {
    test('返回角色列表（分页）', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(`${BASE_URL}/api/v1/admin/roles?page=1&page_size=10`, {
        headers: authHeaders(adminToken),
      });

      const body = await assertPaginatedResponse(resp, 1);
      const data = body.data as Array<Record<string, unknown>>;
      expect(Array.isArray(data)).toBe(true);

      if (data.length > 0) {
        const role = data[0];
        expect(role.id).toBeDefined();
        expect(role.name).toBeDefined();
        expect(Array.isArray(role.permissions)).toBe(true);
      }
    });
  });

  test.describe('POST /api/v1/admin/roles', () => {
    test('创建角色成功', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.post(`${BASE_URL}/api/v1/admin/roles`, {
        headers: authHeaders(adminToken),
        data: {
          name: roleName,
          description: 'Playwright 测试创建的角色',
          permissions: ['ticket:manage', 'knowledge:create'],
        },
      });

      const body = await assertSuccess(resp);
      const data = body.data as Record<string, unknown>;
      expect(data.id).toBeGreaterThan(0);
      createdRoleId = data.id as number;
    });

    test('重复角色名返回 409', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.post(`${BASE_URL}/api/v1/admin/roles`, {
        headers: authHeaders(adminToken),
        data: {
          name: roleName, // 重复
          description: '重复的角色',
          permissions: [],
        },
      });

      await assertError(resp, 200, 10005);
    });

    test('缺少名称返回校验失败', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.post(`${BASE_URL}/api/v1/admin/roles`, {
        headers: authHeaders(adminToken),
        data: {
          description: '没有名称',
          permissions: [],
        },
      });

      await assertError(resp, 200, 10003);
    });
  });

  test.describe('GET /api/v1/admin/roles/:id', () => {
    test('角色详情查询成功', async ({ request }) => {
      if (!adminToken || !createdRoleId) {
        test.skip(true, '未找到认证状态或角色');
        return;
      }

      const resp = await request.get(`${BASE_URL}/api/v1/admin/roles/${createdRoleId}`, {
        headers: authHeaders(adminToken),
      });

      const body = await assertSuccess(resp);
      const data = body.data as Record<string, unknown>;
      expect(data.id).toBe(createdRoleId);
      expect(data.name).toBe(roleName);
    });

    test('不存在的角色返回 404', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(`${BASE_URL}/api/v1/admin/roles/99999`, {
        headers: authHeaders(adminToken),
      });

      await assertError(resp, 200, 10004);
    });
  });

  test.describe('PUT /api/v1/admin/roles/:id', () => {
    test('更新角色成功（全量替换 permissions）', async ({ request }) => {
      if (!adminToken || !createdRoleId) {
        test.skip(true, '未找到认证状态或角色');
        return;
      }

      const resp = await request.put(`${BASE_URL}/api/v1/admin/roles/${createdRoleId}`, {
        headers: authHeaders(adminToken),
        data: {
          name: `${roleName}_v2`,
          description: '更新后的描述',
          permissions: ['ticket:manage', 'knowledge:manage', 'audit:view'],
        },
      });

      await assertSuccess(resp);
    });
  });

  test.describe('DELETE /api/v1/admin/roles/:id', () => {
    test('删除角色成功', async ({ request }) => {
      if (!adminToken || !createdRoleId) {
        test.skip(true, '未找到认证状态或角色');
        return;
      }

      const resp = await request.delete(`${BASE_URL}/api/v1/admin/roles/${createdRoleId}`, {
        headers: authHeaders(adminToken),
      });

      await assertSuccess(resp);
    });

    test('重复删除返回 404', async ({ request }) => {
      if (!adminToken || !createdRoleId) {
        test.skip(true, '未找到认证状态或角色');
        return;
      }

      const resp = await request.delete(`${BASE_URL}/api/v1/admin/roles/${createdRoleId}`, {
        headers: authHeaders(adminToken),
      });

      await assertError(resp, 200, 10004);
    });
  });
});

// ==================== 菜单管理 ====================

test.describe('菜单管理', () => {
  test.describe('GET /api/v1/admin/menus', () => {
    test('返回全部菜单（树形结构）', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(`${BASE_URL}/api/v1/admin/menus`, {
        headers: authHeaders(adminToken),
      });

      const body = await assertSuccess(resp);
      const data = body.data as Array<Record<string, unknown>>;
      expect(Array.isArray(data)).toBe(true);

      if (data.length > 0) {
        const menu = data[0];
        expect(menu.id).toBeDefined();
        expect(menu.name).toBeDefined();
        expect(menu.path).toBeDefined();
        expect(menu.icon).toBeDefined();
        expect(menu.sort_order).toBeDefined();
      }
    });
  });

  test.describe('PUT /api/v1/admin/roles/:id/menus', () => {
    test('更新角色菜单权限（全量替换）', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      // 获取菜单 ID 列表
      const menuResp = await request.get(`${BASE_URL}/api/v1/admin/menus`, {
        headers: authHeaders(adminToken),
      });
      const menuBody = await menuResp.json();
      if (menuBody.code !== 0 || !menuBody.data?.length) {
        test.skip(true, '没有可用菜单');
        return;
      }
      const menuIds = menuBody.data.slice(0, 3).map((m: Record<string, unknown>) => m.id);

      // 获取运维人员角色（通常 ID=3 或 role 列表中存在）
      const roleResp = await request.get(`${BASE_URL}/api/v1/admin/roles`, {
        headers: authHeaders(adminToken),
      });
      const roleBody = await roleResp.json();
      if (roleBody.code !== 0 || !roleBody.data?.length) {
        test.skip(true, '没有可用角色');
        return;
      }
      const roleId = roleBody.data[0].id;

      const resp = await request.put(`${BASE_URL}/api/v1/admin/roles/${roleId}/menus`, {
        headers: authHeaders(adminToken),
        data: { menu_ids: menuIds },
      });

      await assertSuccess(resp);
    });

    test('无效角色 ID 返回 404', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.put(`${BASE_URL}/api/v1/admin/roles/99999/menus`, {
        headers: authHeaders(adminToken),
        data: { menu_ids: [1, 2] },
      });

      await assertError(resp, 200, 10004);
    });
  });
});

// ==================== 权限验证 ====================

test.describe('权限验证', () => {
  test('无 token 访问角色列表返回 401', async ({ request }) => {
    const resp = await request.get(`${BASE_URL}/api/v1/admin/roles`);
    await assertError(resp, 401, 10001);
  });
});
