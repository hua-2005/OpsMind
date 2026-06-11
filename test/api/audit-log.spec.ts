import { test, expect } from '@playwright/test';
import {
  loadAuthState,
  authHeaders,
  assertSuccess,
  assertError,
  assertPaginatedResponse,
} from '../utils/test-helpers.js';

/**
 * 审计日志 + 系统配置 + 站内消息 + 健康检查 接口集成测试。
 *
 * 覆盖端点：
 *   GET  /api/v1/admin/audit-logs             — 查询审计日志
 *   GET  /api/v1/admin/configs/:key           — 获取系统配置
 *   PUT  /api/v1/admin/configs/:key           — 更新系统配置
 *   GET  /api/v1/portal/messages              — 站内消息列表
 *   PUT  /api/v1/portal/messages/:id/read     — 标记已读
 *   GET  /api/v1/portal/messages/unread-count  — 未读计数
 *   GET  /health                              — 健康检查
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

// ==================== 审计日志 ====================

test.describe('审计日志', () => {
  test.describe('GET /api/v1/admin/audit-logs', () => {
    test('返回审计日志列表（分页）', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(
        `${BASE_URL}/api/v1/admin/audit-logs?page=1&page_size=10`,
        { headers: authHeaders(adminToken) },
      );

      const body = await assertPaginatedResponse(resp);
      const data = body.data as Array<Record<string, unknown>>;
      expect(Array.isArray(data)).toBe(true);

      if (data.length > 0) {
        const log = data[0];
        expect(log.id).toBeDefined();
        expect(log.operator_id).toBeDefined();
        expect(log.operator_name).toBeDefined();
        expect(log.action).toBeDefined();
        expect(log.target_type).toBeDefined();
        expect(log.created_at).toBeDefined();
      }
    });

    test('按操作人筛选', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(
        `${BASE_URL}/api/v1/admin/audit-logs?operator_id=1`,
        { headers: authHeaders(adminToken) },
      );

      await assertPaginatedResponse(resp);
    });

    test('按操作类型筛选', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(
        `${BASE_URL}/api/v1/admin/audit-logs?action=knowledge:publish`,
        { headers: authHeaders(adminToken) },
      );

      await assertPaginatedResponse(resp);
    });

    test('无 token 访问返回 401', async ({ request }) => {
      const resp = await request.get(`${BASE_URL}/api/v1/admin/audit-logs`);
      await assertError(resp, 401, 10001);
    });
  });
});

// ==================== 系统配置 ====================

test.describe('系统配置', () => {
  test.describe('GET /api/v1/admin/configs/:key', () => {
    test('获取应用名称配置', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(`${BASE_URL}/api/v1/admin/configs/app_name`, {
        headers: authHeaders(adminToken),
      });

      const body = await assertSuccess(resp);
      const data = body.data as Record<string, unknown>;
      expect(data.key).toBe('app_name');
      expect(data.value).toBeDefined();
    });

    test('不存在的配置键返回 404', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(
        `${BASE_URL}/api/v1/admin/configs/non_existent_key`,
        { headers: authHeaders(adminToken) },
      );

      await assertError(resp, 200, 10004);
    });
  });

  test.describe('PUT /api/v1/admin/configs/:key', () => {
    test('更新配置成功', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.put(`${BASE_URL}/api/v1/admin/configs/app_name`, {
        headers: authHeaders(adminToken),
        data: { value: 'OpsMind Test' },
      });

      await assertSuccess(resp);

      // 验证更新结果
      const getResp = await request.get(`${BASE_URL}/api/v1/admin/configs/app_name`, {
        headers: authHeaders(adminToken),
      });
      const getBody = await getResp.json();
      expect(getBody.data.value).toBe('OpsMind Test');

      // 恢复原值
      await request.put(`${BASE_URL}/api/v1/admin/configs/app_name`, {
        headers: authHeaders(adminToken),
        data: { value: 'OpsMind' },
      });
    });
  });
});

// ==================== 站内消息 ====================

test.describe('站内消息', () => {
  test.describe('GET /api/v1/portal/messages', () => {
    test('返回消息列表（分页）', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(
        `${BASE_URL}/api/v1/portal/messages?page=1&page_size=10`,
        { headers: authHeaders(adminToken) },
      );

      const body = await assertPaginatedResponse(resp);
      const data = body.data as Array<Record<string, unknown>>;
      expect(Array.isArray(data)).toBe(true);

      if (data.length > 0) {
        const msg = data[0];
        expect(msg.id).toBeDefined();
        expect(msg.title).toBeDefined();
        expect(msg.type).toBeDefined();
        expect(typeof msg.is_read).toBe('boolean');
      }
    });

    test('无 token 访问返回 401', async ({ request }) => {
      const resp = await request.get(`${BASE_URL}/api/v1/portal/messages`);
      await assertError(resp, 401, 10001);
    });
  });

  test.describe('GET /api/v1/portal/messages/unread-count', () => {
    test('返回未读计数', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(`${BASE_URL}/api/v1/portal/messages/unread-count`, {
        headers: authHeaders(adminToken),
      });

      const body = await assertSuccess(resp);
      const data = body.data as Record<string, unknown>;
      expect(typeof data.count).toBe('number');
      expect((data.count as number)).toBeGreaterThanOrEqual(0);
    });
  });

  test.describe('PUT /api/v1/portal/messages/:id/read', () => {
    test('不存在的消息返回 404', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.put(`${BASE_URL}/api/v1/portal/messages/99999/read`, {
        headers: authHeaders(adminToken),
      });

      await assertError(resp, 200, 10004);
    });
  });
});

// ==================== 健康检查 ====================

test.describe('GET /health', () => {
  test('健康检查无需认证，返回 ok', async ({ request }) => {
    const resp = await request.get(`${BASE_URL}/health`);

    expect(resp.status()).toBe(200);
    const body = await resp.json();
    expect(body.status).toBe('ok');
  });
});
