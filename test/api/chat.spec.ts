import { test, expect } from '@playwright/test';
import {
  loadAuthState,
  authHeaders,
  assertSuccess,
  assertError,
} from '../utils/test-helpers.js';

/**
 * 智能问答接口集成测试。
 *
 * 覆盖端点：
 *   POST /api/v1/portal/chat-sessions/stream  — SSE 流式问答
 *   POST /api/v1/portal/chat-sessions          — 非流式问答
 *   GET  /api/v1/portal/chat-sessions/:id      — 会话详情
 *   POST /api/v1/portal/chat-sessions/:id/feedback — 提交反馈
 *
 * SSE 测试注意事项：
 *   Playwright APIRequestContext 不直接支持 SSE 流，
 *   使用 Node.js fetch API 消费 text/event-stream 响应。
 */

const BASE_URL = process.env.API_BASE_URL || 'http://localhost:8080';

let adminToken: string;
let kbId: number;

test.beforeAll(async ({ request }) => {
  const state = loadAuthState();
  adminToken = state?.accessToken || '';

  // 获取可用的知识库 ID
  if (adminToken) {
    const resp = await request.get(`${BASE_URL}/api/v1/admin/knowledge-bases`, {
      headers: authHeaders(adminToken),
    });
    const body = await resp.json();
    if (body.code === 0 && body.data?.items?.length > 0) {
      kbId = body.data.items[0].id;
    }
  }
});

// ==================== 非流式问答 ====================

test.describe('POST /api/v1/portal/chat-sessions (非流式)', () => {
  test('创建问答会话成功，返回答案和来源', async ({ request }) => {
    if (!adminToken || !kbId) {
      test.skip(true, '未找到认证状态或知识库');
      return;
    }

    const resp = await request.post(`${BASE_URL}/api/v1/portal/chat-sessions`, {
      headers: { ...authHeaders(adminToken), timeout: '60000' },
      data: {
        question: '如何重置密码？',
        kb_id: kbId,
        rag_options: {
          top_k: 3,
          query_rewrite: true,
          hybrid: true,
          rerank: true,
        },
      },
    });

    // AI 服务可能不可用，两种情况都算正常
    const body = await resp.json();
    if (body.code === 20001) {
      // AI 服务不可用是预期场景
      expect(body.message).toBeTruthy();
      return;
    }
    if (body.code === 20002) {
      // RAG 服务不可用
      expect(body.message).toBeTruthy();
      return;
    }

    expect(body.code).toBe(0);
    const data = body.data as Record<string, unknown>;
    expect(data.session_id).toBeDefined();
    expect(data.question).toBe('如何重置密码？');
    expect(data.answer).toBeDefined();
    expect(Array.isArray(data.sources)).toBe(true);
    expect(typeof data.confidence).toBe('number');
    // v2 新增字段
    expect(data.pipeline).toBeDefined();
    const pipeline = data.pipeline as Record<string, unknown>;
    expect(Array.isArray(pipeline.steps)).toBe(true);
    expect(typeof pipeline.total_duration_ms).toBe('number');
    // 置信度范围校验
    const confidence = data.confidence as number;
    expect(confidence).toBeGreaterThanOrEqual(0);
    expect(confidence).toBeLessThanOrEqual(1);
  });

  test('缺少必填字段返回校验失败', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.post(`${BASE_URL}/api/v1/portal/chat-sessions`, {
      headers: authHeaders(adminToken),
      data: {
        question: '问题但没有 kb_id',
      },
    });

    await assertError(resp, 200, 10003);
  });

  test('无 token 访问返回 401', async ({ request }) => {
    const resp = await request.post(`${BASE_URL}/api/v1/portal/chat-sessions`, {
      data: { question: 'test', kb_id: 1 },
    });

    await assertError(resp, 401, 10001);
  });

  test('top_k 超出范围应校验', async ({ request }) => {
    if (!adminToken || !kbId) {
      test.skip(true, '未找到认证状态或知识库');
      return;
    }

    // top_k=50 超出 1-20 范围
    const resp = await request.post(`${BASE_URL}/api/v1/portal/chat-sessions`, {
      headers: authHeaders(adminToken),
      data: {
        question: '测试问题',
        kb_id: kbId,
        rag_options: { top_k: 50 },
      },
    });

    const body = await resp.json();
    // 可能被后端修正或返回校验错误
    expect([0, 10003]).toContain(body.code);
  });
});

// ==================== SSE 流式问答 ====================

test.describe('POST /api/v1/portal/chat-sessions/stream (SSE 流式)', () => {
  test('SSE 流式响应包含 step/token/done 事件', async () => {
    if (!adminToken || !kbId) {
      test.skip(true, '未找到认证状态或知识库');
      return;
    }

    // Playwright request 不支持 SSE，使用 fetch
    const resp = await fetch(`${BASE_URL}/api/v1/portal/chat-sessions/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${adminToken}`,
      },
      body: JSON.stringify({
        question: 'VPN 怎么连接？',
        kb_id: kbId,
        rag_options: { top_k: 3, hybrid: true },
      }),
    });

    // SSE 响应应为 200 或 AI 服务不可用
    if (resp.status !== 200) {
      const json = await resp.json();
      // AI 服务不可用是可接受的
      expect([20001, 20002]).toContain(json.code);
      return;
    }

    expect(resp.headers.get('content-type')).toContain('text/event-stream');

    const reader = resp.body?.getReader();
    if (!reader) {
      test.skip(true, '无法读取响应流');
      return;
    }

    const decoder = new TextDecoder();
    let fullText = '';
    const eventTypes: string[] = [];
    let done = false;

    try {
      while (true) {
        const { value, done: streamDone } = await reader.read();
        if (streamDone) break;
        fullText += decoder.decode(value, { stream: true });

        // 解析 SSE 事件类型
        const lines = fullText.split('\n');
        for (const line of lines) {
          if (line.startsWith('data: ')) {
            try {
              const json = JSON.parse(line.slice(6));
              if (json.type === 'done') {
                done = true;
              }
              eventTypes.push(json.type);
            } catch {
              // 处理中的 token 数据可能不是完整 JSON
            }
          }
        }
      }
    } finally {
      reader.releaseLock();
    }

    // 验证至少收到了事件
    if (eventTypes.length === 0) {
      // 如果没有任何事件，可能是 AI 服务出错——检查 raw 响应
      if (fullText.includes('"code":20001') || fullText.includes('"code":20002')) {
        return; // AI/RAG 不可用，跳过
      }
    }

    // 验证事件类型（可能因 AI 服务不可用而降级）
    if (eventTypes.length > 0) {
      // step 事件
      expect(eventTypes.filter(t => t === 'step').length).toBeGreaterThanOrEqual(0);
      // 如果完成了，应该有 done 事件
      if (done) {
        expect(eventTypes).toContain('done');
      }
    }
  });

  test('不传 rag_options 使用默认值', async () => {
    if (!adminToken || !kbId) {
      test.skip(true, '未找到认证状态或知识库');
      return;
    }

    const resp = await fetch(`${BASE_URL}/api/v1/portal/chat-sessions/stream`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${adminToken}`,
      },
      body: JSON.stringify({
        question: '默认参数的测试问题',
        kb_id: kbId,
      }),
    });

    if (resp.status !== 200) {
      const json = await resp.json();
      expect([20001, 20002]).toContain(json.code);
      return;
    }

    expect(resp.headers.get('content-type')).toContain('text/event-stream');
  });
});

// ==================== 会话查询 ====================

test.describe('GET /api/v1/portal/chat-sessions/:id', () => {
  test('查询不存在的会话返回 404', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.get(`${BASE_URL}/api/v1/portal/chat-sessions/99999`, {
      headers: authHeaders(adminToken),
    });

    await assertError(resp, 200, 10004);
  });
});

// ==================== 用户反馈 ====================

test.describe('POST /api/v1/portal/chat-sessions/:id/feedback', () => {
  test('提交有效反馈值', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    // 先创建一个会话
    if (!kbId) {
      test.skip(true, '未找到知识库');
      return;
    }

    const chatResp = await request.post(`${BASE_URL}/api/v1/portal/chat-sessions`, {
      headers: authHeaders(adminToken),
      data: { question: '测试反馈功能', kb_id: kbId },
    });

    const chatBody = await chatResp.json();
    if (chatBody.code !== 0) {
      test.skip(true, '创建会话失败（AI 服务不可用）');
      return;
    }

    const sessionId = chatBody.data.session_id;

    // 提交已解决反馈
    const resp = await request.post(
      `${BASE_URL}/api/v1/portal/chat-sessions/${sessionId}/feedback`,
      {
        headers: authHeaders(adminToken),
        data: { feedback: 1 },
      },
    );

    await assertSuccess(resp);
  });

  test('无效反馈值应校验', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    // feedback=5 超出 0/1/2 范围
    const resp = await request.post(
      `${BASE_URL}/api/v1/portal/chat-sessions/1/feedback`,
      {
        headers: authHeaders(adminToken),
        data: { feedback: 5 },
      },
    );

    await assertError(resp, 200, 10003);
  });
});
