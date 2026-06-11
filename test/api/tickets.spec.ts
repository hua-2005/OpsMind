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
 * 申告管理接口集成测试。
 *
 * 覆盖端点：
 *   门户端：
 *     POST  /api/v1/portal/tickets                — 创建申告
 *     GET   /api/v1/portal/tickets                — 我的申告列表
 *     GET   /api/v1/portal/tickets/:id            — 申告详情
 *     PATCH /api/v1/portal/tickets/:id/supplement — 补充信息
 *   后台管理：
 *     GET   /api/v1/admin/tickets                 — 全部申告列表
 *     PATCH /api/v1/admin/tickets/:id/status      — 状态变更
 *     POST  /api/v1/admin/tickets/:id/records     — 处理记录
 *     POST  /api/v1/admin/tickets/:id/knowledge-candidate — 知识候选
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

// ==================== 门户端 ====================

test.describe('门户端申告接口', () => {
  let ticketId: number;

  test.describe('POST /api/v1/portal/tickets', () => {
    test('创建申告成功', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const title = uniqueName('测试申告');
      const resp = await request.post(`${BASE_URL}/api/v1/portal/tickets`, {
        headers: authHeaders(adminToken),
        data: {
          title,
          description: '这是一个自动化测试创建的申告，用于验证接口功能。',
          urgency: 1, // 普通
          impact_scope: 0, // 个人
          affected_systems: ['测试系统'],
          contact_phone: '13800000001',
          contact_email: 'test@opsmind.local',
        },
      });

      const body = await assertSuccess(resp);
      const data = body.data as Record<string, unknown>;
      expect(data.id).toBeGreaterThan(0);
      ticketId = data.id as number;
    });

    test('缺少必填字段返回校验失败', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.post(`${BASE_URL}/api/v1/portal/tickets`, {
        headers: authHeaders(adminToken),
        data: {
          title: '只有标题',
        },
      });

      await assertError(resp, 200, 10003);
    });

    test('无效 urgency 值返回校验失败', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.post(`${BASE_URL}/api/v1/portal/tickets`, {
        headers: authHeaders(adminToken),
        data: {
          title: '测试',
          description: '测试',
          urgency: 99, // 无效值（仅支持 1/2/3）
          contact_phone: '13800000001',
        },
      });

      await assertError(resp, 200, 10003);
    });

    test('无 token 创建返回 401', async ({ request }) => {
      const resp = await request.post(`${BASE_URL}/api/v1/portal/tickets`, {
        data: { title: 'test', description: 'test', urgency: 1, contact_phone: '13800000001' },
      });

      await assertError(resp, 401, 10001);
    });
  });

  test.describe('GET /api/v1/portal/tickets', () => {
    test('返回我的申告列表（分页）', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(`${BASE_URL}/api/v1/portal/tickets?page=1&page_size=10`, {
        headers: authHeaders(adminToken),
      });

      const body = await assertPaginatedResponse(resp);
      const data = body.data as Array<Record<string, unknown>>;
      expect(Array.isArray(data)).toBe(true);

      if (data.length > 0) {
        const ticket = data[0];
        expect(ticket.id).toBeDefined();
        expect(ticket.ticket_no).toBeDefined();
        expect(ticket.title).toBeDefined();
        expect(ticket.urgency).toBeDefined();
        expect(ticket.status).toBeDefined();
        expect(ticket.status_text).toBeDefined();
      }
    });
  });

  test.describe('GET /api/v1/portal/tickets/:id', () => {
    test('申告详情查询成功', async ({ request }) => {
      if (!adminToken || !ticketId) {
        test.skip(true, '未找到认证状态或申告');
        return;
      }

      const resp = await request.get(`${BASE_URL}/api/v1/portal/tickets/${ticketId}`, {
        headers: authHeaders(adminToken),
      });

      const body = await assertSuccess(resp);
      expect((body.data as Record<string, unknown>).id).toBe(ticketId);
    });

    test('不存在的申告返回 404', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(`${BASE_URL}/api/v1/portal/tickets/99999`, {
        headers: authHeaders(adminToken),
      });

      await assertError(resp, 200, 10004);
    });
  });

  test.describe('PATCH /api/v1/portal/tickets/:id/supplement', () => {
    test('非「需补充信息」状态下补充信息应失败', async ({ request }) => {
      if (!adminToken || !ticketId) {
        test.skip(true, '未找到认证状态或申告');
        return;
      }

      const resp = await request.patch(
        `${BASE_URL}/api/v1/portal/tickets/${ticketId}/supplement`,
        {
          headers: authHeaders(adminToken),
          data: { content: '补充信息内容...' },
        },
      );

      // 状态不允许应返回错误
      expect(resp.status()).toBe(200);
      const body = await resp.json();
      expect(body.code).toBeGreaterThan(0);
    });
  });
});

// ==================== 后台管理 ====================

test.describe('后台管理申告接口', () => {
  test.describe('GET /api/v1/admin/tickets', () => {
    test('返回全部申告列表（分页）', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(
        `${BASE_URL}/api/v1/admin/tickets?page=1&page_size=10`,
        { headers: authHeaders(adminToken) },
      );

      await assertPaginatedResponse(resp);
    });

    test('按状态筛选', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      // status=1 待处理
      const resp = await request.get(
        `${BASE_URL}/api/v1/admin/tickets?status=1`,
        { headers: authHeaders(adminToken) },
      );

      await assertPaginatedResponse(resp);
    });

    test('按紧急程度筛选', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      // urgency=2 紧急
      const resp = await request.get(
        `${BASE_URL}/api/v1/admin/tickets?urgency=2`,
        { headers: authHeaders(adminToken) },
      );

      await assertPaginatedResponse(resp);
    });

    test('无效 status 值返回校验失败', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(
        `${BASE_URL}/api/v1/admin/tickets?status=99`,
        { headers: authHeaders(adminToken) },
      );

      await assertError(resp, 200, 10003);
    });
  });

  test.describe('PATCH /api/v1/admin/tickets/:id/status', () => {
    test('无效 action 返回校验失败', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.patch(
        `${BASE_URL}/api/v1/admin/tickets/1/status`,
        {
          headers: authHeaders(adminToken),
          data: { action: 'invalid_action' },
        },
      );

      await assertError(resp, 200, 10003);
    });

    test('不存在的申告返回 404', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.patch(
        `${BASE_URL}/api/v1/admin/tickets/99999/status`,
        {
          headers: authHeaders(adminToken),
          data: { action: 'start', result: '测试' },
        },
      );

      await assertError(resp, 200, 10004);
    });
  });

  test.describe('POST /api/v1/admin/tickets/:id/records', () => {
    test('添加处理记录成功', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      // 先获取一个存在的申告 ID
      const listResp = await request.get(
        `${BASE_URL}/api/v1/admin/tickets?page=1&page_size=1`,
        { headers: authHeaders(adminToken) },
      );
      const listBody = await listResp.json();
      if (listBody.code !== 0 || !listBody.data?.length) {
        test.skip(true, '没有可用的申告');
        return;
      }
      const ticketId = listBody.data[0].id;

      const resp = await request.post(
        `${BASE_URL}/api/v1/admin/tickets/${ticketId}/records`,
        {
          headers: authHeaders(adminToken),
          data: {
            action: 'note',
            content: '自动化测试添加的处理记录',
            detail: '{"source":"playwright_test"}',
          },
        },
      );

      await assertSuccess(resp);
    });

    test('缺少 action 返回校验失败', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.post(
        `${BASE_URL}/api/v1/admin/tickets/1/records`,
        {
          headers: authHeaders(adminToken),
          data: { content: '没有 action' },
        },
      );

      await assertError(resp, 200, 10003);
    });
  });

  test.describe('POST /api/v1/admin/tickets/:id/knowledge-candidate', () => {
    test('生成知识库候选', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      // 先获取可用的知识库 ID
      const kbResp = await request.get(`${BASE_URL}/api/v1/admin/knowledge-bases`, {
        headers: authHeaders(adminToken),
      });
      const kbBody = await kbResp.json();
      if (kbBody.code !== 0 || !kbBody.data?.items?.length) {
        test.skip(true, '没有可用的知识库');
        return;
      }
      const kbId = kbBody.data.items[0].id;

      // 获取一个申告
      const listResp = await request.get(
        `${BASE_URL}/api/v1/admin/tickets?page=1&page_size=1`,
        { headers: authHeaders(adminToken) },
      );
      const listBody = await listResp.json();
      if (listBody.code !== 0 || !listBody.data?.length) {
        test.skip(true, '没有可用的申告');
        return;
      }
      const ticketId = listBody.data[0].id;

      const resp = await request.post(
        `${BASE_URL}/api/v1/admin/tickets/${ticketId}/knowledge-candidate`,
        {
          headers: authHeaders(adminToken),
          data: { kb_id: kbId },
        },
      );

      await assertSuccess(resp);
    });
  });
});

// ==================== 权限验证 ====================

test.describe('权限验证', () => {
  test('无 token 访问后台申告列表返回 401', async ({ request }) => {
    const resp = await request.get(`${BASE_URL}/api/v1/admin/tickets`);
    await assertError(resp, 401, 10001);
  });
});
