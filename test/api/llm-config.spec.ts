import { test, expect } from '@playwright/test';
import {
  loadAuthState,
  authHeaders,
  assertSuccess,
  assertError,
  uniqueName,
} from '../utils/test-helpers.js';

/**
 * LLM 配置接口集成测试。
 *
 * 覆盖端点：
 *   GET    /api/v1/admin/llm-configs        — LLM 配置列表
 *   POST   /api/v1/admin/llm-configs        — 创建 LLM 配置
 *   GET    /api/v1/admin/llm-configs/:id    — LLM 配置详情
 *   PUT    /api/v1/admin/llm-configs/:id    — 更新 LLM 配置
 *   DELETE /api/v1/admin/llm-configs/:id    — 删除 LLM 配置
 *   POST   /api/v1/admin/llm-configs/:id/test — 测试连接
 *
 * v2 变更说明：
 *   本 API 统一了 v1 的 llm-configs 和 embedding-configs，
 *   embedding 模型通过 embedding_model 和 vector_dimension 字段管理。
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

test.describe('GET /api/v1/admin/llm-configs', () => {
  test('返回 LLM 配置列表', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.get(`${BASE_URL}/api/v1/admin/llm-configs`, {
      headers: authHeaders(adminToken),
    });

    const body = await assertSuccess(resp);
    const data = body.data as Array<Record<string, unknown>>;
    expect(Array.isArray(data)).toBe(true);

    if (data.length > 0) {
      const config = data[0];
      // 基础字段
      expect(config.id).toBeDefined();
      expect(config.name).toBeDefined();
      expect(config.provider_type).toBeDefined();
      expect([1, 2]).toContain(config.provider_type as number);
      expect(config.provider_type_text).toBeDefined();
      // LLM 字段
      expect(config.base_url).toBeDefined();
      expect(config.llm_model).toBeDefined();
      expect(config.max_tokens).toBeDefined();
      // Embedding 字段（v2 新增）
      expect(config.embedding_model).toBeDefined();
      expect(config.vector_dimension).toBeDefined();
      // API Key 应掩码显示
      expect(config.api_key_masked).toBeDefined();
      // 默认配置标记
      expect(typeof config.is_default).toBe('boolean');
    }
  });

  test('无 token 访问返回 401', async ({ request }) => {
    const resp = await request.get(`${BASE_URL}/api/v1/admin/llm-configs`);
    await assertError(resp, 401, 10001);
  });
});

test.describe('POST /api/v1/admin/llm-configs', () => {
  let createdConfigId: number;

  test('创建 llama.cpp 类型配置成功', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const name = uniqueName('测试LLM配置');
    const resp = await request.post(`${BASE_URL}/api/v1/admin/llm-configs`, {
      headers: authHeaders(adminToken),
      data: {
        name,
        provider_type: 1, // llama.cpp
        base_url: 'http://llama-cpp:8080/v1',
        api_key: '',
        llm_model: 'qwen3-4b',
        embedding_model: 'bge-m3',
        max_tokens: 8192,
        vector_dimension: 1024,
        is_default: false,
      },
    });

    const body = await assertSuccess(resp);
    const data = body.data as Record<string, unknown>;
    expect(data.id).toBeGreaterThan(0);
    createdConfigId = data.id as number;
  });

  test('创建 OpenAI-compatible 类型配置成功', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const name = uniqueName('OpenAI配置');
    const resp = await request.post(`${BASE_URL}/api/v1/admin/llm-configs`, {
      headers: authHeaders(adminToken),
      data: {
        name,
        provider_type: 2, // OpenAI-compatible
        base_url: 'https://api.openai.com/v1',
        api_key: 'sk-test-key-1234567890',
        llm_model: 'gpt-4o-mini',
        embedding_model: 'text-embedding-3-small',
        max_tokens: 16384,
        vector_dimension: 1536,
        is_default: false,
      },
    });

    const body = await assertSuccess(resp);
    const data = body.data as Record<string, unknown>;
    expect(data.id).toBeGreaterThan(0);
    // API key 应掩码显示
    const detailResp = await request.get(
      `${BASE_URL}/api/v1/admin/llm-configs/${data.id}`,
      { headers: authHeaders(adminToken) },
    );
    const detailBody = await detailResp.json();
    if (detailBody.code === 0 && detailBody.data) {
      const detail = detailBody.data as Record<string, unknown>;
      expect(detail.api_key_masked).toBeDefined();
      // 掩码格式：sk-****... 或 空字符串
      const masked = detail.api_key_masked as string;
      if (masked) {
        expect(masked).toContain('****');
      }
    }

    // 清理
    await request.delete(`${BASE_URL}/api/v1/admin/llm-configs/${data.id}`, {
      headers: authHeaders(adminToken),
    });
  });

  test('缺少必填字段返回校验失败', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.post(`${BASE_URL}/api/v1/admin/llm-configs`, {
      headers: authHeaders(adminToken),
      data: {
        name: '不完整的配置',
      },
    });

    await assertError(resp, 200, 10003);
  });

  test('无效 provider_type 返回校验失败', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.post(`${BASE_URL}/api/v1/admin/llm-configs`, {
      headers: authHeaders(adminToken),
      data: {
        name: '无效类型',
        provider_type: 99,
        base_url: 'http://localhost:8080/v1',
        llm_model: 'test',
        embedding_model: 'test',
        max_tokens: 1000,
        vector_dimension: 100,
      },
    });

    await assertError(resp, 200, 10003);
  });

  test.afterAll(async ({ request }) => {
    if (createdConfigId && adminToken) {
      await request.delete(`${BASE_URL}/api/v1/admin/llm-configs/${createdConfigId}`, {
        headers: authHeaders(adminToken),
      });
    }
  });
});

test.describe('GET /api/v1/admin/llm-configs/:id', () => {
  test('LLM 配置详情查询成功', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    // 先获取列表取得第一个配置的 ID
    const listResp = await request.get(`${BASE_URL}/api/v1/admin/llm-configs`, {
      headers: authHeaders(adminToken),
    });
    const listBody = await listResp.json();
    if (listBody.code !== 0 || !listBody.data?.length) {
      test.skip(true, '没有可用的 LLM 配置');
      return;
    }
    const configId = listBody.data[0].id;

    const resp = await request.get(`${BASE_URL}/api/v1/admin/llm-configs/${configId}`, {
      headers: authHeaders(adminToken),
    });

    const body = await assertSuccess(resp);
    const data = body.data as Record<string, unknown>;
    expect(data.id).toBe(configId);
    expect(data.name).toBeDefined();
    expect(data.base_url).toBeDefined();
    expect(data.llm_model).toBeDefined();
    expect(data.embedding_model).toBeDefined();
  });

  test('不存在的配置返回 404', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    const resp = await request.get(`${BASE_URL}/api/v1/admin/llm-configs/99999`, {
      headers: authHeaders(adminToken),
    });

    await assertError(resp, 200, 10004);
  });
});

test.describe('PUT /api/v1/admin/llm-configs/:id', () => {
  test('更新 LLM 配置成功', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    // 创建一个临时配置用于更新测试
    const createResp = await request.post(`${BASE_URL}/api/v1/admin/llm-configs`, {
      headers: authHeaders(adminToken),
      data: {
        name: uniqueName('更新测试'),
        provider_type: 2,
        base_url: 'https://api.deepseek.com/v1',
        api_key: 'sk-deepseek-key-xxxx',
        llm_model: 'deepseek-chat',
        embedding_model: 'text-embedding-3-small',
        max_tokens: 8192,
        vector_dimension: 1536,
        is_default: false,
      },
    });
    const createBody = await createResp.json();
    if (createBody.code !== 0) {
      test.skip(true, '创建临时配置失败');
      return;
    }
    const configId = createBody.data.id;

    const resp = await request.put(`${BASE_URL}/api/v1/admin/llm-configs/${configId}`, {
      headers: authHeaders(adminToken),
      data: {
        name: 'DeepSeek v3 (更新后)',
        provider_type: 2,
        base_url: 'https://api.deepseek.com/v1',
        api_key: '', // 不传 api_key 表示保留原密钥
        llm_model: 'deepseek-chat',
        embedding_model: 'text-embedding-3-small',
        max_tokens: 16384,
        vector_dimension: 1536,
        is_default: false,
      },
    });

    await assertSuccess(resp);

    // 验证更新结果
    const detailResp = await request.get(
      `${BASE_URL}/api/v1/admin/llm-configs/${configId}`,
      { headers: authHeaders(adminToken) },
    );
    const detailBody = await detailResp.json();
    expect(detailBody.data.name).toBe('DeepSeek v3 (更新后)');
    expect(detailBody.data.max_tokens).toBe(16384);

    // 清理
    await request.delete(`${BASE_URL}/api/v1/admin/llm-configs/${configId}`, {
      headers: authHeaders(adminToken),
    });
  });
});

test.describe('DELETE /api/v1/admin/llm-configs/:id', () => {
  test('不能删除默认配置', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    // 找到默认配置
    const listResp = await request.get(`${BASE_URL}/api/v1/admin/llm-configs`, {
      headers: authHeaders(adminToken),
    });
    const listBody = await listResp.json();
    if (listBody.code !== 0 || !listBody.data?.length) {
      test.skip(true, '没有可用的 LLM 配置');
      return;
    }
    const defaultConfig = listBody.data.find(
      (c: Record<string, unknown>) => c.is_default === true,
    );
    if (!defaultConfig) {
      test.skip(true, '没有默认配置');
      return;
    }

    const resp = await request.delete(
      `${BASE_URL}/api/v1/admin/llm-configs/${defaultConfig.id}`,
      { headers: authHeaders(adminToken) },
    );

    await assertError(resp, 200, 10003);
  });
});

test.describe('POST /api/v1/admin/llm-configs/:id/test', () => {
  test('测试 LLM 连接', async ({ request }) => {
    if (!adminToken) {
      test.skip(true, '未找到认证状态');
      return;
    }

    // 获取一个配置
    const listResp = await request.get(`${BASE_URL}/api/v1/admin/llm-configs`, {
      headers: authHeaders(adminToken),
    });
    const listBody = await listResp.json();
    if (listBody.code !== 0 || !listBody.data?.length) {
      test.skip(true, '没有可用的 LLM 配置');
      return;
    }
    const configId = listBody.data[0].id;

    const resp = await request.post(
      `${BASE_URL}/api/v1/admin/llm-configs/${configId}/test`,
      { headers: authHeaders(adminToken) },
    );

    // 测试连接可能成功也可能失败（取决于服务是否可达）
    expect(resp.status()).toBe(200);
    const body = await resp.json();
    expect(body.code).toBe(0);
    const data = body.data as Record<string, unknown>;
    expect(typeof data.success).toBe('boolean');
    expect(typeof data.latency_ms).toBe('number');

    if (data.success) {
      expect(data.model).toBeDefined();
      expect(data.test_message).toBeDefined();
    } else {
      expect(data.error).toBeDefined();
    }
  });
});
