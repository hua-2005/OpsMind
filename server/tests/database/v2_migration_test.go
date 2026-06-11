//go:build integration

// Package database_test 验证 v2 数据库迁移 schema。
//
// 本测试在真实 pgvector 实例上执行 v2 迁移 SQL，
// 验证表结构、字段类型、索引是否符合 TECHv2 §3.2 定义。
//
// 运行方式（需 Docker pgvector 运行中）：
//
//	go test ./tests/database/... -v -tags=integration -run TestV2
package database_test

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// =============================================================================
// 测试辅助函数
// =============================================================================

// dbConn 连接 opsmind_test 数据库（迁移在此数据库中执行和验证）。
func dbConn() (*sql.DB, error) {
	host := "localhost"
	user := "opsmind"
	password := "opsmind123"
	dbname := "opsmind_test"
	if env := os.Getenv("DB_HOST"); env != "" {
		host = env
	}
	if env := os.Getenv("DB_USER"); env != "" {
		user = env
	}
	if env := os.Getenv("DB_PASSWORD"); env != "" {
		password = env
	}
	dsn := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=%s sslmode=disable",
		host, user, password, dbname)
	return sql.Open("postgres", dsn)
}

// runMigrationSQLs 依次执行 v2 迁移 SQL 文件。
func runMigrationSQLs(t *testing.T, db *sql.DB) {
	t.Helper()
	dir := "../../migrations/v2"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("跳过迁移测试：无法读取迁移目录 (%v)。请确保在 server/ 或项目根目录运行测试。", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		path := fmt.Sprintf("%s/%s", dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("读取迁移文件 %s 失败: %v", path, err)
		}
		sqlStr := string(data)
		// 按语句分割并逐条执行（忽略空行和纯注释行）。
		for _, stmt := range strings.Split(sqlStr, ";") {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" || strings.HasPrefix(stmt, "--") {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				// CREATE EXTENSION 重复执行会报错，跳过即可
				if strings.Contains(err.Error(), "already exists") {
					continue
				}
				t.Fatalf("执行迁移 %s 失败: %v\nSQL: %s", entry.Name(), err, stmt)
			}
		}
	}
}

// columnExists 检查表中是否存在指定列。
func columnExists(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_name = $1 AND column_name = $2)",
		table, column,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("查询列存在性失败 (%s.%s): %v", table, column, err)
	}
	return exists
}

// indexExists 检查指定索引是否存在。
func indexExists(t *testing.T, db *sql.DB, indexName string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM pg_indexes WHERE indexname = $1)",
		indexName,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("查询索引存在性失败 (%s): %v", indexName, err)
	}
	return exists
}

// extensionExists 检查 PostgreSQL 扩展是否已安装。
func extensionExists(t *testing.T, db *sql.DB, extName string) bool {
	t.Helper()
	var exists bool
	err := db.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM pg_extension WHERE extname = $1)",
		extName,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("查询扩展存在性失败 (%s): %v", extName, err)
	}
	return exists
}

// =============================================================================
// 测试用例
// =============================================================================

// TestV2Migration_RunAll 执行全部 v2 迁移，验证 schema。
func TestV2Migration_RunAll(t *testing.T) {
	db, err := dbConn()
	if err != nil {
		t.Skipf("跳过集成测试：无法连接数据库 (%v)", err)
		return
	}
	defer db.Close()

	// 确认数据库连通
	if err := db.Ping(); err != nil {
		t.Skipf("跳过集成测试：数据库 Ping 失败 (%v)", err)
		return
	}

	runMigrationSQLs(t, db)

	// === 验证 pgvector 扩展 ===
	t.Run("pgvector_extension", func(t *testing.T) {
		if !extensionExists(t, db, "vector") {
			t.Error("pgvector 扩展未安装 — CREATE EXTENSION IF NOT EXISTS vector 应已执行")
		}
	})

	// === 验证 knowledge_bases 表变更 ===
	t.Run("knowledge_bases", func(t *testing.T) {
		if columnExists(t, db, "knowledge_bases", "rag_workspace_slug") {
			t.Error("knowledge_bases.rag_workspace_slug 应已删除")
		}
		if !columnExists(t, db, "knowledge_bases", "llm_config_id") {
			t.Error("knowledge_bases.llm_config_id 应已新增")
		}
	})

	// === 验证 knowledge_articles 表变更 ===
	t.Run("knowledge_articles", func(t *testing.T) {
		if columnExists(t, db, "knowledge_articles", "question") {
			t.Error("knowledge_articles.question 应已删除")
		}
		if columnExists(t, db, "knowledge_articles", "rag_document_location") {
			t.Error("knowledge_articles.rag_document_location 应已删除")
		}
		// answer → content
		if !columnExists(t, db, "knowledge_articles", "content") {
			t.Error("knowledge_articles.content 应存在 (原 answer 列改名)")
		}
		// v2 新增字段
		for _, col := range []string{"title", "source_type", "word_count", "chunk_count", "file_type", "minio_path", "process_status", "process_error"} {
			if !columnExists(t, db, "knowledge_articles", col) {
				t.Errorf("knowledge_articles.%s 应已新增", col)
			}
		}
	})

	// === 验证 knowledge_chunks 表变更 ===
	t.Run("knowledge_chunks", func(t *testing.T) {
		for _, col := range []string{"sync_status", "sync_error", "synced_at"} {
			if columnExists(t, db, "knowledge_chunks", col) {
				t.Errorf("knowledge_chunks.%s 应已删除", col)
			}
		}
		// v2 新增字段
		for _, col := range []string{"kb_id", "chunk_index"} {
			if !columnExists(t, db, "knowledge_chunks", col) {
				t.Errorf("knowledge_chunks.%s 应已新增", col)
			}
		}
		if !columnExists(t, db, "knowledge_chunks", "embedding") {
			t.Error("knowledge_chunks.embedding (halfvec) 应已新增")
		}
	})

	// === 验证 llm_configs 表 ===
	t.Run("llm_configs", func(t *testing.T) {
		required := map[string]string{
			"id":               "integer",
			"name":             "character varying",
			"provider_type":    "smallint",
			"base_url":         "character varying",
			"api_key":          "character varying",
			"llm_model":        "character varying",
			"embedding_model":  "character varying",
			"max_tokens":       "integer",
			"vector_dimension": "integer",
			"is_default":       "boolean",
			"created_at":       "timestamp with time zone",
			"updated_at":       "timestamp with time zone",
		}
		for col, expectedType := range required {
			if !columnExists(t, db, "llm_configs", col) {
				t.Errorf("llm_configs.%s 应存在", col)
			} else {
				// 验证类型
				var dataType string
				db.QueryRow(
					"SELECT data_type FROM information_schema.columns WHERE table_name = 'llm_configs' AND column_name = $1",
					col,
				).Scan(&dataType)
				if !strings.Contains(dataType, expectedType) {
					t.Errorf("llm_configs.%s 类型为 %s，期望包含 %s", col, dataType, expectedType)
				}
			}
		}
	})

	// === 验证 chat_messages 表新增字段 ===
	t.Run("chat_messages", func(t *testing.T) {
		if !columnExists(t, db, "chat_messages", "rag_pipeline") {
			t.Error("chat_messages.rag_pipeline (jsonb) 应已新增")
		}
	})

	// === 验证 HNSW 向量索引 ===
	t.Run("hnsw_index", func(t *testing.T) {
		if !indexExists(t, db, "idx_chunks_embedding") {
			t.Error("idx_chunks_embedding HNSW 向量索引应存在")
		}
		if !indexExists(t, db, "idx_chunks_kb_id") {
			t.Error("idx_chunks_kb_id 索引应存在")
		}
		if !indexExists(t, db, "idx_chunks_article_id") {
			t.Error("idx_chunks_article_id 索引应存在")
		}
	})

	// === 验证 system_configs 表仍存在 ===
	t.Run("preserved_tables", func(t *testing.T) {
		preserved := []string{
			"users", "roles", "user_roles", "menus", "role_menus",
			"tickets", "ticket_records", "chat_sessions", "chat_messages",
			"audit_logs", "messages", "system_configs",
		}
		for _, table := range preserved {
			var exists bool
			db.QueryRow(
				"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = $1)",
				table,
			).Scan(&exists)
			if !exists {
				t.Errorf("表 %s 应保留（迁移不应删除 v2 未涉及的旧表）", table)
			}
		}
	})

	// === 清理：删除 llm_configs 表的唯一部分索引验证（通过 SQL 查询验证）===
	t.Run("llm_configs_default_index", func(t *testing.T) {
		if !indexExists(t, db, "idx_llm_configs_default") {
			t.Error("idx_llm_configs_default 唯一部分索引应存在")
		}
	})
}

// TestV2Migration_Idempotent 验证迁移可重复执行（幂等性）。
func TestV2Migration_Idempotent(t *testing.T) {
	db, err := dbConn()
	if err != nil {
		t.Skipf("跳过集成测试：无法连接数据库 (%v)", err)
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("跳过集成测试：数据库 Ping 失败 (%v)", err)
		return
	}

	// 第一次执行
	runMigrationSQLs(t, db)
	// 第二次执行不应报错（幂等）
	// 使用 recover 包住以防 panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("迁移幂等性失败：第二次执行触发 panic: %v", r)
			}
		}()
		runMigrationSQLs(t, db)
	}()
}

// TestV2Migration_SeedFileExecutes 验证 seed.sql 可执行。
func TestV2Migration_SeedFileExecutes(t *testing.T) {
	db, err := dbConn()
	if err != nil {
		t.Skipf("跳过集成测试：无法连接数据库 (%v)", err)
		return
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("跳过集成测试：数据库 Ping 失败 (%v)", err)
		return
	}

	// 先执行迁移
	runMigrationSQLs(t, db)

	// 执行种子数据
	// 使用 psql 执行 seed SQL（参考现有 make seed 方式）
	cmd := exec.Command("psql",
		"-U", "opsmind",
		"-d", "opsmind_test",
		"-h", "localhost",
		"-f", "../../migrations/seed.sql",
	)
	cmd.Env = append(os.Environ(), "PGPASSWORD=opsmind123")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("seed.sql 执行失败: %v\n输出: %s", err, output)
	}

	// 验证 knowledge_articles 数据按 v2 格式存在
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM knowledge_articles WHERE title IS NOT NULL AND title != ''").Scan(&count); err != nil {
		t.Fatalf("验证 seed 数据失败: %v", err)
	}
	if count == 0 {
		t.Error("seed 后 knowledge_articles 应包含 title 非空的记录")
	}

	// 验证 llm_configs 默认配置存在
	var configCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM llm_configs WHERE is_default = true").Scan(&configCount); err != nil {
		t.Fatalf("验证 llm_configs 默认配置失败: %v", err)
	}
	if configCount != 1 {
		t.Errorf("seed 后应有 1 条默认 LLM 配置，实际 %d 条", configCount)
	}
}
