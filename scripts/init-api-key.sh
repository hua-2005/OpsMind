#!/bin/sh
# OpsMind AnythingLLM 自动初始化 — 在容器内运行
#
# 功能：
# 1. 等待 AnythingLLM 完成数据库初始化（api_keys 表已创建）
# 2. 检查是否已存在 API Key，有则直接输出
# 3. 无则生成新 Key 并写入 SQLite 数据库
# 4. 将 Key 写入共享卷供 opsmind-server 读取
#
# 表结构兼容性：脚本会先探测 api_keys 表的实际列名，适配不同版本

set -e

DB_PATH="/storage/anythingllm.db"
OUTPUT_FILE="/shared/anythingllm_api_key"
MAX_WAIT=120

echo "[opsmind-setup] 等待 AnythingLLM 数据库就绪..."

# 等待数据库文件创建且 api_keys 表存在
ELAPSED=0
while [ $ELAPSED -lt $MAX_WAIT ]; do
  if [ -f "$DB_PATH" ]; then
    # 检查 api_keys 表是否已创建（数据库迁移完成）
    if sqlite3 "$DB_PATH" "SELECT name FROM sqlite_master WHERE type='table' AND name='api_keys';" 2>/dev/null | grep -q "api_keys"; then
      echo "[opsmind-setup] AnythingLLM 数据库已就绪"
      break
    fi
  fi
  sleep 3
  ELAPSED=$((ELAPSED + 3))
done

if [ ! -f "$DB_PATH" ]; then
  echo "[opsmind-setup] 错误：等待超时，数据库文件未创建"
  exit 1
fi

# 检查是否已有 API Key
EXISTING=$(sqlite3 "$DB_PATH" "SELECT secret FROM api_keys LIMIT 1;" 2>/dev/null || echo "")

if [ -n "$EXISTING" ] && [ "$EXISTING" != "" ]; then
  echo "[opsmind-setup] 检测到已有 API Key: ${EXISTING:0:8}..."
  echo "$EXISTING" > "$OUTPUT_FILE"
  echo "[opsmind-setup] 已写入 $OUTPUT_FILE"
  exit 0
fi

# 探测 api_keys 表结构（兼容不同版本）
echo "[opsmind-setup] 探测 api_keys 表结构..."
COLUMNS=$(sqlite3 "$DB_PATH" "PRAGMA table_info('api_keys');" 2>/dev/null)

# 生成 API Key（格式与 AnythingLLM 一致：sk- 前缀 + 随机 hex）
API_KEY="sk-$(openssl rand -hex 24)"
NOW=$(date -u +"%Y-%m-%dT%H:%M:%S.000Z")

# 根据表结构选择插入方式
if echo "$COLUMNS" | grep -q "secret"; then
  # 标准结构：id, secret, created_at, ...
  if echo "$COLUMNS" | grep -q "createdAt"; then
    sqlite3 "$DB_PATH" "INSERT INTO api_keys (secret, createdAt) VALUES ('$API_KEY', '$NOW');"
  elif echo "$COLUMNS" | grep -q "created_at"; then
    sqlite3 "$DB_PATH" "INSERT INTO api_keys (secret, created_at) VALUES ('$API_KEY', '$NOW');"
  else
    sqlite3 "$DB_PATH" "INSERT INTO api_keys (secret) VALUES ('$API_KEY');"
  fi
  echo "[opsmind-setup] API Key 已写入数据库: ${API_KEY:0:8}..."
else
  # 降级：列名不匹配，尝试直接写入已知的默认结构
  echo "[opsmind-setup] 警告：无法探测列名，尝试默认结构..."
  sqlite3 "$DB_PATH" "INSERT INTO api_keys (secret, createdAt) VALUES ('$API_KEY', '$NOW');" 2>/dev/null || {
    echo "[opsmind-setup] 错误：无法写入数据库，请手动创建 API Key"
    exit 1
  }
fi

# 写入共享文件供 opsmind-server 读取
echo "$API_KEY" > "$OUTPUT_FILE"
echo "[opsmind-setup] ============================================"
echo "[opsmind-setup] API Key 已生成: ${API_KEY:0:8}...${API_KEY: -4}"
echo "[opsmind-setup] 已写入 $OUTPUT_FILE"
echo "[opsmind-setup] ============================================"
