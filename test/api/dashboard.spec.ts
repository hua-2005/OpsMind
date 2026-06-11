import { test, expect } from '@playwright/test';
import {
  loadAuthState,
  authHeaders,
  assertSuccess,
  assertError,
} from '../utils/test-helpers.js';

/**
 * 数据看板接口集成测试。
 *
 * 覆盖端点：
 *   GET /api/v1/admin/dashboard/stats  — 统计数据
 *   GET /api/v1/admin/dashboard/trends — 趋势数据
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

test.describe('GET /api/v1/admin/dashboard/stats', () => {
  test('返回看板统计数据', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.get(`${BASE_URL}/api/v1/admin/dashboard/stats`, {
      headers: authHeaders(adminToken),
    });

    const body = await assertSuccess(resp);
    const data = body.data as Record<string, unknown>;

    // 验证所有统计字段类型
    expect(typeof data.today_tickets).toBe('number');
    expect(typeof data.pending_tickets).toBe('number');
    expect(typeof data.processing_tickets).toBe('number');
    expect(typeof data.resolved_tickets).toBe('number');
    expect(typeof data.today_chats).toBe('number');
    expect(typeof data.avg_confidence).toBe('number');
    expect(typeof data.knowledge_count).toBe('number');

    // 值合理性校验
    expect((data.today_tickets as number)).toBeGreaterThanOrEqual(0);
    expect((data.pending_tickets as number)).toBeGreaterThanOrEqual(0);
    expect((data.processing_tickets as number)).toBeGreaterThanOrEqual(0);
    expect((data.resolved_tickets as number)).toBeGreaterThanOrEqual(0);
    expect((data.today_chats as number)).toBeGreaterThanOrEqual(0);
    // 置信度应在 0-1 之间
    const avgConf = data.avg_confidence as number;
    if (avgConf > 0) {
      expect(avgConf).toBeGreaterThanOrEqual(0);
      expect(avgConf).toBeLessThanOrEqual(1);
    }
    expect((data.knowledge_count as number)).toBeGreaterThanOrEqual(0);
  });

  test('无 token 访问返回 401', async ({ request }) => {
    const resp = await request.get(`${BASE_URL}/api/v1/admin/dashboard/stats`);
    await assertError(resp, 401, 10001);
  });
});

test.describe('GET /api/v1/admin/dashboard/trends', () => {
  test('返回趋势数据（day 粒度）', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.get(
      `${BASE_URL}/api/v1/admin/dashboard/trends?start_date=2026-06-01&end_date=2026-06-11&granularity=day`,
      { headers: authHeaders(adminToken) },
    );

    const body = await assertSuccess(resp);
    const data = body.data as Record<string, unknown>;
    const dataPoints = data.data_points as Array<Record<string, unknown>>;
    expect(Array.isArray(dataPoints)).toBe(true);

    if (dataPoints.length > 0) {
      const point = dataPoints[0];
      expect(point.date).toBeDefined();
      // 日期格式 YYYY-MM-DD
      expect((point.date as string)).toMatch(/^\d{4}-\d{2}-\d{2}$/);
      expect(typeof point.ticket_count).toBe('number');
      expect(typeof point.chat_count).toBe('number');
    }
  });

  test('缺失日期参数返回校验失败', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.get(`${BASE_URL}/api/v1/admin/dashboard/trends`, {
      headers: authHeaders(adminToken),
    });

    await assertError(resp, 200, 10003);
  });

  test('日期格式错误返回校验失败', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.get(
      `${BASE_URL}/api/v1/admin/dashboard/trends?start_date=invalid&end_date=2026-06-11`,
      { headers: authHeaders(adminToken) },
    );

    await assertError(resp, 200, 10003);
  });

  test('结束日期早于开始日期返回校验失败', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.get(
      `${BASE_URL}/api/v1/admin/dashboard/trends?start_date=2026-06-11&end_date=2026-06-01`,
      { headers: authHeaders(adminToken) },
    );

    await assertError(resp, 200, 10003);
  });
});
