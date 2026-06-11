import { test, expect } from '@playwright/test';
import {
  loadAuthState,
  authHeaders,
  authHeadersMultipart,
  assertSuccess,
  assertError,
  assertPaginatedResponse,
  assertDataNotNull,
  uniqueName,
} from '../utils/test-helpers.js';

/**
 * 知识库管理接口集成测试。
 *
 * 覆盖端点：
 *   GET    /api/v1/portal/knowledge-bases               — 门户端知识库列表
 *   GET    /api/v1/admin/knowledge-bases                 — 知识库列表
 *   POST   /api/v1/admin/knowledge-bases                 — 创建知识库
 *   PUT    /api/v1/admin/knowledge-bases/:id             — 更新知识库
 *   DELETE /api/v1/admin/knowledge-bases/:id             — 删除知识库
 *   GET    /api/v1/admin/knowledge-bases/:kb_id/articles  — 文章列表
 *   POST   /api/v1/admin/knowledge-bases/:kb_id/articles  — 创建文章
 *   PUT    /api/v1/admin/articles/:id                    — 更新文章
 *   GET    /api/v1/admin/articles/:id                    — 文章详情
 *   POST   /api/v1/admin/articles/:id/submit-review      — 提交审核
 *   POST   /api/v1/admin/articles/:id/review             — 审核操作
 *   POST   /api/v1/admin/articles/:id/publish            — 发布
 *   POST   /api/v1/admin/articles/:id/disable            — 停用
 *   POST   /api/v1/admin/articles/:id/enable             — 启用
 *   POST   /api/v1/admin/knowledge-bases/:kb_id/documents/upload — 文档上传
 *   GET    /api/v1/admin/knowledge-bases/:kb_id/documents/:id/status — 处理状态
 *   POST   /api/v1/admin/knowledge-bases/:kb_id/documents/:id/retry — 重试失败文档
 */

const BASE_URL = process.env.API_BASE_URL || 'http://localhost:8080';

// 所有需要认证的测试共享 token
let adminToken: string;

test.beforeAll(() => {
  const state = loadAuthState();
  adminToken = state?.accessToken || '';
  if (!adminToken) {
    console.warn('⚠ 未找到认证状态，需要认证的测试将被跳过');
  }
});

// ==================== 知识库 CRUD ====================

test.describe('知识库 CRUD', () => {
  test.describe('GET /api/v1/portal/knowledge-bases', () => {
    test('门户端返回知识库列表（仅 id + name）', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(`${BASE_URL}/api/v1/portal/knowledge-bases`, {
        headers: authHeaders(adminToken),
      });

      const body = await assertSuccess(resp);
      const data = body.data as Array<Record<string, unknown>>;
      expect(Array.isArray(data)).toBe(true);
      if (data.length > 0) {
        const item = data[0];
        expect(item.id).toBeDefined();
        expect(item.name).toBeDefined();
        // 门户端不应暴露管理字段
        expect(item.embedding_model).toBeUndefined();
        expect(item.vector_dimension).toBeUndefined();
      }
    });

    test('无 token 访问返回 401', async ({ request }) => {
      const resp = await request.get(`${BASE_URL}/api/v1/portal/knowledge-bases`);
      await assertError(resp, 401, 10001);
    });
  });

  test.describe('GET /api/v1/admin/knowledge-bases', () => {
    test('返回知识库列表（含管理字段）', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(`${BASE_URL}/api/v1/admin/knowledge-bases`, {
        headers: authHeaders(adminToken),
      });

      const body = await assertSuccess(resp);
      const data = body.data as Record<string, unknown>;
      expect(data).toBeDefined();
      const items = data.items as Array<Record<string, unknown>>;
      expect(Array.isArray(items)).toBe(true);
    });
  });

  test.describe('POST /api/v1/admin/knowledge-bases', () => {
    let createdKbId: number;

    test('创建知识库成功', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const name = uniqueName('测试知识库');
      const resp = await request.post(`${BASE_URL}/api/v1/admin/knowledge-bases`, {
        headers: authHeaders(adminToken),
        data: {
          name,
          description: 'Playwright 自动化测试创建的知识库',
          embedding_model: 'bge-m3',
          vector_dimension: 1024,
          llm_config_id: 1,
        },
      });

      const body = await assertSuccess(resp);
      expect(body.data).toBeDefined();
      const data = body.data as Record<string, unknown>;
      expect(data.id).toBeGreaterThan(0);
      createdKbId = data.id as number;
    });

    test('缺少必填字段返回校验失败', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.post(`${BASE_URL}/api/v1/admin/knowledge-bases`, {
        headers: authHeaders(adminToken),
        data: {
          name: '只有名称没有其它字段',
        },
      });

      await assertError(resp, 200, 10003);
    });

    test.afterAll(async ({ request }) => {
      // 清理创建的测试知识库
      if (createdKbId && adminToken) {
        await request.delete(`${BASE_URL}/api/v1/admin/knowledge-bases/${createdKbId}`, {
          headers: authHeaders(adminToken),
        });
      }
    });
  });

  test.describe('PUT /api/v1/admin/knowledge-bases/:id', () => {
    test('更新不存在的知识库返回 404', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.put(`${BASE_URL}/api/v1/admin/knowledge-bases/99999`, {
        headers: authHeaders(adminToken),
        data: { name: '不存在的知识库' },
      });

      await assertError(resp, 200, 10004);
    });
  });
});

// ==================== 知识文章 CRUD ====================

test.describe('知识文章 CRUD', () => {
  let kbId: number;
  let articleId: number;

  test.beforeAll(async ({ request }) => {
    if (!adminToken) return;

    // 获取一个已有的知识库 ID
    const resp = await request.get(`${BASE_URL}/api/v1/admin/knowledge-bases`, {
      headers: authHeaders(adminToken),
    });
    const body = await resp.json();
    if (body.code === 0) {
      const items = body.data?.items || [];
      if (items.length > 0) {
        kbId = items[0].id;
      }
    }
  });

  test.describe('POST /api/v1/admin/knowledge-bases/:kb_id/articles', () => {
    test('创建文章成功（手动输入模式）', async ({ request }) => {
      if (!adminToken || !kbId) {
        test.skip(true, '未找到认证状态或知识库');
        return;
      }

      const title = uniqueName('测试文章');
      const resp = await request.post(
        `${BASE_URL}/api/v1/admin/knowledge-bases/${kbId}/articles`,
        {
          headers: authHeaders(adminToken),
          data: {
            title,
            content: '这是测试文章的内容，用于验证知识文章创建功能。\n\n## 第二段\n\n更多内容...',
            source_type: 1,
            category: '测试分类',
            tags: ['测试', '自动化'],
          },
        },
      );

      const body = await assertSuccess(resp);
      const data = body.data as Record<string, unknown>;
      expect(data.id).toBeGreaterThan(0);
      articleId = data.id as number;
    });

    test('缺少标题返回校验失败', async ({ request }) => {
      if (!adminToken || !kbId) {
        test.skip(true, '未找到认证状态或知识库');
        return;
      }

      const resp = await request.post(
        `${BASE_URL}/api/v1/admin/knowledge-bases/${kbId}/articles`,
        {
          headers: authHeaders(adminToken),
          data: {
            content: '只有内容没有标题',
          },
        },
      );

      await assertError(resp, 200, 10003);
    });
  });

  test.describe('GET /api/v1/admin/knowledge-bases/:kb_id/articles', () => {
    test('返回文章列表（分页）', async ({ request }) => {
      if (!adminToken || !kbId) {
        test.skip(true, '未找到认证状态或知识库');
        return;
      }

      const resp = await request.get(
        `${BASE_URL}/api/v1/admin/knowledge-bases/${kbId}/articles?page=1&page_size=10`,
        { headers: authHeaders(adminToken) },
      );

      const body = await assertPaginatedResponse(resp);
      const data = body.data as Record<string, unknown>;
      const articles = data.articles as Array<Record<string, unknown>>;
      expect(Array.isArray(articles)).toBe(true);

      if (articles.length > 0) {
        const article = articles[0];
        // v2 字段验证
        expect(article.title).toBeDefined();
        expect(article.content).toBeDefined();
        expect(article.source_type).toBeDefined();
        expect(article.word_count).toBeDefined();
        expect(article.status).toBeDefined();
        expect(article.status_text).toBeDefined();
      }
    });

    test('按状态筛选', async ({ request }) => {
      if (!adminToken || !kbId) {
        test.skip(true, '未找到认证状态或知识库');
        return;
      }

      const resp = await request.get(
        `${BASE_URL}/api/v1/admin/knowledge-bases/${kbId}/articles?status=1`,
        { headers: authHeaders(adminToken) },
      );

      await assertPaginatedResponse(resp);
    });
  });

  test.describe('GET /api/v1/admin/articles/:id', () => {
    test('返回文章详情', async ({ request }) => {
      if (!adminToken || !articleId) {
        test.skip(true, '未找到认证状态或文章');
        return;
      }

      const resp = await request.get(`${BASE_URL}/api/v1/admin/articles/${articleId}`, {
        headers: authHeaders(adminToken),
      });

      const body = await assertSuccess(resp);
      const data = body.data as Record<string, unknown>;
      expect(data.id).toBe(articleId);
      expect(data.kb_id).toBe(kbId);
      expect(data.title).toBeDefined();
      expect(data.content).toBeDefined();
       // v2 字段
      expect(data.source_type).toBeDefined();
      expect(data.word_count).toBeDefined();
      expect(data.chunks).toBeDefined();
    });

    test('不存在的文章返回 404', async ({ request }) => {
      if (!adminToken) {
        test.skip(true, '未找到认证状态');
        return;
      }

      const resp = await request.get(`${BASE_URL}/api/v1/admin/articles/99999`, {
        headers: authHeaders(adminToken),
      });

      await assertError(resp, 200, 10004);
    });
  });

  test.describe('PUT /api/v1/admin/articles/:id', () => {
    test('更新草稿文章成功', async ({ request }) => {
      if (!adminToken || !articleId) {
        test.skip(true, '未找到认证状态或文章');
        return;
      }

      const resp = await request.put(`${BASE_URL}/api/v1/admin/articles/${articleId}`, {
        headers: authHeaders(adminToken),
        data: {
          title: `${uniqueName('更新后')}`,
          content: '更新后的内容...',
          category: '更新分类',
          tags: ['更新'],
        },
      });

      await assertSuccess(resp);
    });
  });

  // ==================== 审核流程 ====================

  test.describe('审核流程', () => {
    test('提交审核成功 (草稿 → 已提交审核)', async ({ request }) => {
      if (!adminToken || !articleId) {
        test.skip(true, '未找到认证状态或文章');
        return;
      }

      const resp = await request.post(
        `${BASE_URL}/api/v1/admin/articles/${articleId}/submit-review`,
        { headers: authHeaders(adminToken) },
      );

      await assertSuccess(resp);
    });

    test('重复提交审核应失败', async ({ request }) => {
      if (!adminToken || !articleId) {
        test.skip(true, '未找到认证状态或文章');
        return;
      }

      const resp = await request.post(
        `${BASE_URL}/api/v1/admin/articles/${articleId}/submit-review`,
        { headers: authHeaders(adminToken) },
      );

      // 状态不是草稿，提交审核应失败
      expect(resp.status()).toBe(200);
      const body = await resp.json();
      expect(body.code).toBeGreaterThan(0);
    });
  });
});

// ==================== 文档上传（v2 新增） ====================

test.describe('文档上传与处理', () => {
  let kbId: number;

  test.beforeAll(async ({ request }) => {
    if (!adminToken) return;
    const resp = await request.get(`${BASE_URL}/api/v1/admin/knowledge-bases`, {
      headers: authHeaders(adminToken),
    });
    const body = await resp.json();
    if (body.code === 0 && body.data?.items?.length > 0) {
      kbId = body.data.items[0].id;
    }
  });

  test('上传不支持的格式返回校验失败', async ({ request }) => {
    if (!adminToken || !kbId) {
      test.skip(true, '未找到认证状态或知识库');
      return;
    }

    // 发送一个假的 .exe 文件
    const boundary = '----TestBoundary';
    const body = [
      `--${boundary}`,
      'Content-Disposition: form-data; name="files"; filename="test.exe"',
      'Content-Type: application/octet-stream',
      '',
      'fake exe content',
      `--${boundary}--`,
    ].join('\r\n');

    const resp = await request.post(
      `${BASE_URL}/api/v1/admin/knowledge-bases/${kbId}/documents/upload`,
      {
        headers: {
          ...authHeadersMultipart(adminToken),
          'Content-Type': `multipart/form-data; boundary=${boundary}`,
        },
        data: body,
      },
    );

    await assertError(resp, 200, 10003);
  });

  test('上传不存在的知识库返回 404', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const boundary = '----TestBoundary2';
    const body = [
      `--${boundary}`,
      'Content-Disposition: form-data; name="files"; filename="test.md"',
      'Content-Type: text/markdown',
      '',
      '# 测试文档\n\n这是测试内容。',
      `--${boundary}--`,
    ].join('\r\n');

    const resp = await request.post(
      `${BASE_URL}/api/v1/admin/knowledge-bases/99999/documents/upload`,
      {
        headers: {
          ...authHeadersMultipart(adminToken),
          'Content-Type': `multipart/form-data; boundary=${boundary}`,
        },
        data: body,
      },
    );

    await assertError(resp, 200, 10004);
  });

  test('查询文档处理状态（不存在的文档）', async ({ request }) => {
    if (!adminToken || !kbId) {
      test.skip(true, '未找到认证状态或知识库');
      return;
    }

    const resp = await request.get(
      `${BASE_URL}/api/v1/admin/knowledge-bases/${kbId}/documents/99999/status`,
      { headers: authHeaders(adminToken) },
    );

    await assertError(resp, 200, 10004);
  });

  test('重试不存在的文档返回 404', async ({ request }) => {
    if (!adminToken || !kbId) {
      test.skip(true, '未找到认证状态或知识库');
      return;
    }

    const resp = await request.post(
      `${BASE_URL}/api/v1/admin/knowledge-bases/${kbId}/documents/99999/retry`,
      { headers: authHeaders(adminToken) },
    );

    await assertError(resp, 200, 10004);
  });
});

// ==================== 发布与停用 ====================

test.describe('发布与停用流程', () => {
  test('发布不存在的文章返回 404', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.post(`${BASE_URL}/api/v1/admin/articles/99999/publish`, {
      headers: authHeaders(adminToken),
    });

    await assertError(resp, 200, 10004);
  });

  test('停用不存在的文章返回 404', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.post(`${BASE_URL}/api/v1/admin/articles/99999/disable`, {
      headers: authHeaders(adminToken),
    });

    await assertError(resp, 200, 10004);
  });
});

// ==================== 权限验证 ====================

test.describe('权限验证', () => {
  test('无 token 访问管理端知识库接口返回 401', async ({ request }) => {
    const resp = await request.get(`${BASE_URL}/api/v1/admin/knowledge-bases`);
    await assertError(resp, 401, 10001);
  });
});
