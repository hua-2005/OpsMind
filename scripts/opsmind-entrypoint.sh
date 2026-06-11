#!/bin/sh
# OpsMind Server 入口包装脚本
#
# 在启动 Go 二进制前检查共享卷中是否有 setup 服务写入的 API Key。
# 如果环境变量 OPSMIND_ANYTHINGLLM_API_KEY 未设置，自动从共享卷读取。
#
# 这使 docker compose up -d 首次运行也能正常工作：
# opsmind-setup 容器自动生成 Key → 写入共享卷 → 本脚本读取 → 启动后端

set -e

KEY_FILE="/shared/anythingllm_api_key"

# 如果环境变量未设置但共享卷中存在 Key 文件，自动导入
if [ -z "$OPSMIND_ANYTHINGLLM_API_KEY" ] && [ -f "$KEY_FILE" ]; then
  API_KEY=$(cat "$KEY_FILE" 2>/dev/null || echo "")
  if [ -n "$API_KEY" ]; then
    export OPSMIND_ANYTHINGLLM_API_KEY="$API_KEY"
    echo "[opsmind-server] 已从共享卷加载 AnythingLLM API Key"
  fi
fi

# 启动 Go 后端
exec ./opsmind-server
