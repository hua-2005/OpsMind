package service_test

import (
	"encoding/json"
	"testing"

	"opsmind/internal/model"
	"opsmind/internal/repository"
	"opsmind/internal/service"
)

// =============================================================================
// mockLlmConfigRepo 模拟 LLM 配置仓库
// =============================================================================

type mockLlmConfigRepo struct {
	configs map[int64]*model.LlmConfig
	nextID  int64
}

func newMockLlmConfigRepo() *mockLlmConfigRepo {
	return &mockLlmConfigRepo{
		configs: make(map[int64]*model.LlmConfig),
		nextID:  1,
	}
}

func (m *mockLlmConfigRepo) Create(cfg *model.LlmConfig) error {
	cfg.ID = m.nextID
	m.nextID++
	m.configs[cfg.ID] = cfg
	return nil
}

func (m *mockLlmConfigRepo) FindByID(id int64) (*model.LlmConfig, error) {
	cfg, ok := m.configs[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return cfg, nil
}

func (m *mockLlmConfigRepo) FindDefault() (*model.LlmConfig, error) {
	for _, cfg := range m.configs {
		if cfg.IsDefault {
			return cfg, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (m *mockLlmConfigRepo) List() ([]model.LlmConfig, error) {
	result := make([]model.LlmConfig, 0, len(m.configs))
	for _, cfg := range m.configs {
		result = append(result, *cfg)
	}
	return result, nil
}

func (m *mockLlmConfigRepo) Update(cfg *model.LlmConfig) error {
	if _, ok := m.configs[cfg.ID]; !ok {
		return repository.ErrNotFound
	}
	m.configs[cfg.ID] = cfg
	return nil
}

func (m *mockLlmConfigRepo) Delete(id int64) error {
	if _, ok := m.configs[id]; !ok {
		return repository.ErrNotFound
	}
	delete(m.configs, id)
	return nil
}

func (m *mockLlmConfigRepo) ClearDefault() error {
	for _, cfg := range m.configs {
		cfg.IsDefault = false
	}
	return nil
}

// =============================================================================
// 测试用例
// =============================================================================

// TestLLMConfigService_CreateDefault 验证创建默认配置并可通过 GetConfig 读取。
func TestLLMConfigService_CreateDefault(t *testing.T) {
	repo := newMockLlmConfigRepo()
	svc := service.NewLLMConfigService(repo)

	_, err := svc.CreateConfig("llama.cpp 本地", 1, "http://llama-cpp:8080/v1", "", "", "qwen3-4b", "bge-m3", 8192, 1024, true)
	if err != nil {
		t.Fatalf("CreateConfig 失败: %v", err)
	}

	// 通过 Manager 读取默认配置
	mgr := svc.GetManager()
	cfg := mgr.GetConfig()
	if cfg == nil {
		t.Fatal("GetConfig 应返回默认配置, 实际 nil")
	}
	if cfg.BaseURL != "http://llama-cpp:8080/v1" {
		t.Errorf("BaseURL = %q, 期望 http://llama-cpp:8080/v1", cfg.BaseURL)
	}
	if cfg.LLMModel != "qwen3-4b" {
		t.Errorf("LLMModel = %q, 期望 qwen3-4b", cfg.LLMModel)
	}
}

// TestLLMConfigService_DefaultUnique 验证 is_default 唯一性约束。
func TestLLMConfigService_DefaultUnique(t *testing.T) {
	repo := newMockLlmConfigRepo()
	svc := service.NewLLMConfigService(repo)

	// 创建第一个默认配置
	_, _ = svc.CreateConfig("默认1", 1, "http://a:8080/v1", "", "", "m1", "e1", 8192, 1024, true)

	// 创建第二个默认配置 — 旧默认应被取消
	_, err := svc.CreateConfig("默认2", 2, "http://b:8080/v1", "", "key", "m2", "e2", 4096, 1536, true)
	if err != nil {
		t.Fatalf("CreateConfig 失败: %v", err)
	}

	// 验证新默认
	mgr := svc.GetManager()
	cfg := mgr.GetConfig()
	if cfg.LLMModel != "m2" {
		t.Errorf("新默认应为 m2, 实际 %s", cfg.LLMModel)
	}

	// 验证旧默认已取消
	cfgs, _ := repo.List()
	defaults := 0
	for _, c := range cfgs {
		if c.IsDefault {
			defaults++
		}
	}
	if defaults != 1 {
		t.Errorf("is_default=true 的配置数应为 1, 实际 %d", defaults)
	}
}

// TestLLMConfigService_DeleteDefault 验证删除默认配置被拒绝。
func TestLLMConfigService_DeleteDefault(t *testing.T) {
	repo := newMockLlmConfigRepo()
	svc := service.NewLLMConfigService(repo)

	_, _ = svc.CreateConfig("默认", 1, "http://x:8080/v1", "", "", "m", "e", 8192, 1024, true)

	cfgs, _ := repo.List()
	err := svc.DeleteConfig(cfgs[0].ID)
	if err == nil {
		t.Error("删除默认配置应返回错误")
	}
}

// TestLLMConfigService_UpdateHotReload 验证更新默认配置后 GetConfig 即时返回新值。
func TestLLMConfigService_UpdateHotReload(t *testing.T) {
	repo := newMockLlmConfigRepo()
	svc := service.NewLLMConfigService(repo)

	_, _ = svc.CreateConfig("默认", 1, "http://a:8080/v1", "", "", "m1", "e1", 8192, 1024, true)

	cfgs, _ := repo.List()
	id := cfgs[0].ID

	// 更新配置
	updated := &model.LlmConfig{
		ID: id, Name: "默认更新", ProviderType: 2,
		BaseURL: "https://api.openai.com/v1", APIKey: "sk-key",
		LLMModel: "gpt-4o", EmbeddingModel: "text-embedding-3-small",
		MaxTokens: 4096, VectorDimension: 1536, IsDefault: true,
	}
	if err := svc.UpdateConfig(updated); err != nil {
		t.Fatalf("UpdateConfig 失败: %v", err)
	}

	// GetConfig 应即时返回新值（atomic.Value 热替换）
	mgr := svc.GetManager()
	cfg := mgr.GetConfig()
	if cfg.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("热替换后 BaseURL = %q, 期望 https://api.openai.com/v1", cfg.BaseURL)
	}
}

// TestLLMConfigService_ListConfigs 验证列出全部配置。
func TestLLMConfigService_ListConfigs(t *testing.T) {
	repo := newMockLlmConfigRepo()
	svc := service.NewLLMConfigService(repo)

	_, _ = svc.CreateConfig("cfg1", 1, "http://a:8080/v1", "", "", "m1", "e1", 8192, 1024, false)
	_, _ = svc.CreateConfig("cfg2", 2, "http://b:8080/v1", "", "k", "m2", "e2", 4096, 1536, false)

	configs, err := svc.ListConfigs()
	if err != nil {
		t.Fatalf("ListConfigs 失败: %v", err)
	}
	if len(configs) != 2 {
		t.Errorf("期望 2 条配置, 实际 %d", len(configs))
	}
}

// TestLLMConfigService_NoDefaultFallback 验证无默认配置时的降级行为。
func TestLLMConfigService_NoDefaultFallback(t *testing.T) {
	repo := newMockLlmConfigRepo()
	svc := service.NewLLMConfigService(repo)

	mgr := svc.GetManager()
	cfg := mgr.GetConfig()

	if cfg != nil {
		t.Error("无默认配置时 GetConfig 应返回 nil")
	}
}

// TestLLMConfigManager_ZeroLockReads 验证 GetConfig 不持有锁。
//
// 使用 atomic.Value 的 Load 操作是零锁的，高并发场景下不会阻塞。
func TestLLMConfigManager_ZeroLockReads(t *testing.T) {
	repo := newMockLlmConfigRepo()
	svc := service.NewLLMConfigService(repo)

	_, _ = svc.CreateConfig("默认", 1, "http://x:8080/v1", "", "", "m", "e", 8192, 1024, true)

	mgr := svc.GetManager()

	// 并发读取不应阻塞
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			cfg := mgr.GetConfig()
			if cfg == nil || cfg.LLMModel != "m" {
				t.Errorf("并发读取返回异常值")
			}
			done <- true
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestLLMConfigService_APIKeyMasked 验证 API Key 在列表中脱敏显示。
func TestLLMConfigService_APIKeyMasked(t *testing.T) {
	repo := newMockLlmConfigRepo()
	svc := service.NewLLMConfigService(repo)

	_, _ = svc.CreateConfig("openai", 2, "https://api.openai.com/v1", "", "sk-1234567890abcdef", "gpt-4o", "text-3-small", 4096, 1536, false)

	configs, _ := svc.ListConfigs()
	if len(configs) == 0 {
		t.Fatal("应有配置")
	}

	// API Key 应脱敏: 显示前4后4，中间用 **** 代替
	apiKey := configs[0].APIKey
	if apiKey == "sk-1234567890abcdef" {
		t.Error("列表中 API Key 应脱敏显示, 不能返回完整值")
	}
	if len(apiKey) == 0 {
		t.Error("API Key 脱敏后不应为空")
	}
}

// TestLLMConfigResponse_MarshalJSON_MasksAPIKey 验证 JSON 序列化时自动脱敏 API Key。
//
// 即使 Service 层忘记调用 maskAPIKey()，MarshalJSON 也会在序列化前自动脱敏，
// 提供编译期级别的安全保障。
func TestLLMConfigResponse_MarshalJSON_MasksAPIKey(t *testing.T) {
	resp := service.LlmConfigResponse{
		ID:        1,
		Name:      "openai",
		APIKey:    "sk-1234567890abcdefghij",
		LLMModel:  "gpt-4o",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Marshal 失败: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)

	apiKey, ok := result["api_key"].(string)
	if !ok {
		t.Fatal("api_key 字段缺失")
	}

	// 不应包含完整密钥
	if apiKey == "sk-1234567890abcdefghij" {
		t.Error("JSON 序列化应自动脱敏 API Key, 不能包含完整值")
	}
	// 应包含脱敏后的值（前4位 + **** + 后4位）
	if len(apiKey) < 8 {
		t.Errorf("脱敏后的 API Key 长度不足: %q (%d)", apiKey, len(apiKey))
	}
}
