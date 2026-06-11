#!/bin/bash
# OpsMind 一键初始化脚本 — AnythingLLM API Key 自动配置
#
# 两种运行方式：
#   1. 自动模式（默认）：
#        docker compose up -d --build
#      首次部署时 opsmind-setup 服务会自动创建 API Key，无需手动操作。
#
#   2. 手动引导模式（兜底）：
#        bash scripts/setup.sh
#      当自动模式失败或用户偏好手动控制时，本脚本引导完成初始化。
#
# 环境要求：
#   - Docker Desktop 4.x+（含 Docker Compose v2）
#   - Windows: Git Bash / WSL / PowerShell
#   - Linux/macOS: 任意终端
#
# 使用方式：
#   bash scripts/setup.sh           自动检测并初始化
#   bash scripts/setup.sh --reset   强制重新初始化

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ENV_FILE="$PROJECT_ROOT/.env"
COMPOSE_FILE="$PROJECT_ROOT/docker-compose.yml"
AL_CONTAINER="opsmind-anythingllm"
AL_URL="http://localhost:3001"

cd "$PROJECT_ROOT"

# ===== 颜色 =====
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'; NC='\033[0m'
info()    { echo -e "${BLUE}→${NC} $1"; }
success() { echo -e "${GREEN}✓${NC} $1"; }
warn()    { echo -e "${YELLOW}⚠${NC} $1"; }
error()   { echo -e "${RED}✗${NC} $1"; }

# ===== 检查环境 =====
check_prerequisites() {
  if ! command -v docker &> /dev/null; then
    error "未找到 Docker，请先安装 Docker Desktop"
    exit 1
  fi
  if ! docker compose version &> /dev/null; then
    error "需要 Docker Compose v2+"
    exit 1
  fi
}

# ===== 确保 .env 存在 =====
ensure_env_file() {
  if [ ! -f "$ENV_FILE" ]; then
    warn ".env 文件不存在"
    if [ -f "$PROJECT_ROOT/.env.example" ]; then
      cp "$PROJECT_ROOT/.env.example" "$ENV_FILE"
      success "已从 .env.example 创建 .env"
    else
      touch "$ENV_FILE"
      success "已创建空的 .env"
    fi
  fi
}

# ===== 检查是否已配置 =====
is_already_configured() {
  if [ "${1:-}" = "--reset" ]; then
    return 1  # 强制重新初始化
  fi
  if grep -q "^ANYTHINGLLM_API_KEY=.\{10,\}" "$ENV_FILE" 2>/dev/null; then
    return 0
  fi
  return 1
}

# ===== 策略1：通过 Docker 自动初始化（直写 SQLite）=====
try_docker_auto_setup() {
  info "尝试自动初始化（opsmind-setup 服务直写 SQLite）..."

  # 确保 anythingllm 在运行
  if ! docker compose ps anythingllm 2>/dev/null | grep -q "Up"; then
    info "启动 AnythingLLM..."
    docker compose up -d anythingllm
  fi

  # 等待 anythingllm 健康
  info "等待 AnythingLLM 就绪..."
  local attempt=0
  while [ $attempt -lt 60 ]; do
    if docker compose ps anythingllm 2>/dev/null | grep -q "(healthy)"; then
      success "AnythingLLM 已就绪"
      break
    fi
    sleep 2
    attempt=$((attempt + 1))
  done

  # 运行初始化容器
  info "运行初始化容器..."
  docker compose build opsmind-setup 2>/dev/null || true
  docker compose up opsmind-setup 2>&1 | while IFS= read -r line; do
    echo "  $line"
  done

  # 检查初始化结果
  local setup_exit_code=$(docker compose ps -a opsmind-setup 2>/dev/null | grep -o "exited ([0-9]*)" | grep -o "[0-9]*" || echo "1")

  # 从共享卷或容器日志中读取生成的 Key
  local api_key=""

  # 方法1：如果 opsmind-server 正在运行，直接通过共享卷读取
  if docker compose ps opsmind-server 2>/dev/null | grep -q "Up"; then
    api_key=$(docker compose exec -T opsmind-server cat /shared/anythingllm_api_key 2>/dev/null || echo "")
  fi

  # 方法2：从初始化容器日志中提取
  if [ -z "$api_key" ]; then
    api_key=$(docker compose logs opsmind-setup 2>/dev/null | grep -o "API Key 已生成: [a-z0-9-]*" | grep -o "[a-z0-9-]*$" | head -1 || echo "")
  fi

  # 方法3：直接从 anythingllm 的 SQLite 数据库读取
  if [ -z "$api_key" ]; then
    api_key=$(docker exec "$AL_CONTAINER" node -e "
      try {
        const Database = require('better-sqlite3');
        const db = new Database('/app/server/storage/anythingllm.db', { readonly: true });
        const row = db.prepare('SELECT secret FROM api_keys ORDER BY id DESC LIMIT 1').get();
        console.log(row ? row.secret : '');
      } catch(e) { console.log(''); }
    " 2>/dev/null || echo "")
  fi

  # 清理 setup 容器
  docker compose rm -f opsmind-setup 2>/dev/null || true

  if [ -n "$api_key" ]; then
    echo "$api_key"
    return 0
  fi

  return 1
}

# ===== 策略2：引导用户手动创建 =====
guided_manual_setup() {
  echo ""
  info "============================================"
  info "  自动初始化未成功，切换为浏览器引导模式"
  info "============================================"
  echo ""

  # 确保 anythingllm 可访问
  if ! docker compose ps anythingllm 2>/dev/null | grep -q "Up"; then
    info "启动 AnythingLLM..."
    docker compose up -d anythingllm
    info "等待 AnythingLLM 就绪..."
    sleep 10
  fi

  # 自动打开浏览器
  info "正在打开 AnythingLLM 管理页面..."
  if command -v start &> /dev/null; then
    start "http://localhost:3001" 2>/dev/null || true
  elif command -v xdg-open &> /dev/null; then
    xdg-open "http://localhost:3001" 2>/dev/null || true
  elif command -v open &> /dev/null; then
    open "http://localhost:3001" 2>/dev/null || true
  else
    info "请手动打开浏览器访问: ${BLUE}http://localhost:3001${NC}"
  fi

  echo ""
  echo -e "  ${YELLOW}初始化向导（如果尚未完成）：${NC}"
  echo -e "    LLM / Embedding / Vector DB 已通过 .env 预配置"
  echo -e "    点击 Next 确认各步骤即可"
  echo ""
  echo -e "  ${YELLOW}创建 API Key：${NC}"
  echo -e "    ① 点击左下角齿轮 → ${GREEN}Settings${NC}"
  echo -e "    ② 选择 ${GREEN}Developer API${NC}（或 API Keys）"
  echo -e "    ③ 点击 ${GREEN}Create API Key${NC}"
  echo -e "    ④ 复制生成的 Key"
  echo ""

  local api_key=""
  while [ -z "$api_key" ]; do
    read -r -p "  请粘贴 API Key（输入 q 退出）: " api_key
    if [ "$api_key" = "q" ] || [ "$api_key" = "Q" ]; then
      error "用户取消"
      exit 1
    fi
    if [ -z "$api_key" ]; then
      warn "API Key 不能为空"
    fi
  done

  success "已接收 API Key: ${api_key:0:8}...${api_key: -4}"
  echo "$api_key"
}

# ===== 写入 .env =====
write_key_to_env() {
  local api_key="$1"
  info "写入 API Key 到 .env..."

  if grep -q "^ANYTHINGLLM_API_KEY=" "$ENV_FILE" 2>/dev/null; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
      sed -i '' "s|^ANYTHINGLLM_API_KEY=.*|ANYTHINGLLM_API_KEY=$api_key|" "$ENV_FILE"
    else
      sed -i "s|^ANYTHINGLLM_API_KEY=.*|ANYTHINGLLM_API_KEY=$api_key|" "$ENV_FILE"
    fi
  else
    echo "ANYTHINGLLM_API_KEY=$api_key" >> "$ENV_FILE"
  fi

  success ".env 已更新"
}

# ===== 创建默认工作区 =====
create_workspace() {
  local api_key="$1"
  local slug="${2:-opsmind-it-ops}"

  # 如果 anythingllm 端口不可达，通过 docker exec 调用
  local check_result
  check_result=$(curl -s -o /dev/null -w "%{http_code}" \
    -H "Authorization: Bearer $api_key" \
    "${AL_URL}/api/v1/workspace/${slug}" 2>/dev/null || echo "000")

  if [ "$check_result" = "200" ]; then
    success "工作区 '${slug}' 已存在"
    return 0
  fi

  info "创建默认工作区 '${slug}'..."
  curl -s -X POST \
    -H "Authorization: Bearer $api_key" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"opsmind-it-ops\",\"similarityThreshold\":0.6,\"topN\":5,\"chatMode\":\"query\",\"openAiPrompt\":\"你是企业运维数字员工。只能基于知识库内容回答，并输出可执行处理步骤；无法确认时提示用户提交申告。\"}" \
    "${AL_URL}/api/v1/workspace/new" 2>/dev/null > /dev/null

  success "默认工作区创建完成"
}

# ===== 主流程 =====
main() {
  echo ""
  echo -e "${BLUE}╔══════════════════════════════════════════╗${NC}"
  echo -e "${BLUE}║   OpsMind AnythingLLM 初始化           ║${NC}"
  echo -e "${BLUE}╚══════════════════════════════════════════╝${NC}"
  echo ""

  check_prerequisites
  ensure_env_file

  # 检查是否已配置
  if is_already_configured "$@"; then
    local existing
    existing=$(grep "^ANYTHINGLLM_API_KEY=" "$ENV_FILE" | head -1 | cut -d'=' -f2)
    success "已配置 API Key: ${existing:0:8}..."
    echo ""
    info "如需重新初始化，请运行：bash scripts/setup.sh --reset"
    exit 0
  fi

  local api_key=""

  # 策略1：自动（Docker 容器直写 SQLite）
  api_key=$(try_docker_auto_setup)

  # 策略2：手动引导
  if [ -z "$api_key" ]; then
    api_key=$(guided_manual_setup)
  fi

  # 写入 .env
  write_key_to_env "$api_key"

  # 创建默认工作区
  create_workspace "$api_key"

  # 重启 opsmind-server 加载新 Key
  info "重启 OpsMind 后端..."
  docker compose up -d opsmind-server 2>&1 | tail -1

  echo ""
  echo -e "${GREEN}╔══════════════════════════════════════════╗${NC}"
  echo -e "${GREEN}║   初始化完成！                         ║${NC}"
  echo -e "${GREEN}╚══════════════════════════════════════════╝${NC}"
  echo ""
  echo "  访问地址："
  echo "    前端:     http://localhost:5173"
  echo "    后端 API: http://localhost:8080"
  echo "    AnythingLLM: http://localhost:3001（管理用）"
  echo ""
  echo "  下一步："
  echo "    make seed     加载演示数据"
}

main "$@"
