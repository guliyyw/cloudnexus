#!/usr/bin/env bash
# =============================================================================
# CloudNexus 自动部署脚本
# 用法:
#   ./deploy/deploy.sh              # 检测变更并部署
#   ./deploy/deploy.sh all          # 全量部署
#   ./deploy/deploy.sh user-file-svc  # 只部署指定服务
#   ./deploy/deploy.sh frontend     # 只部署前端
#
# 前置条件:
#   - SSH 免密登录已配置 (ssh-copy-id user@121.43.145.157)
#   - Go 1.25+ 已安装
#   - Node.js 20+ 已安装
# =============================================================================

set -euo pipefail

# ── 配置 ──
REMOTE_HOST="${DEPLOY_HOST:-121.43.145.157}"
REMOTE_USER="${DEPLOY_USER:-user}"
REMOTE_HOME="/home/${REMOTE_USER}/cloudnexus"
REMOTE_DEPLOY="${REMOTE_HOME}/deploy"
REMOTE_BINS="${REMOTE_DEPLOY}/service-bins"
SSH_CMD="ssh ${REMOTE_USER}@${REMOTE_HOST}"
SCP_CMD="scp -q"

# Go 服务列表 (key=服务名 value=cmd路径)
declare -A GO_SERVICES=(
  ["user-file-svc"]="./cmd/user-file-svc"
  ["im-svc"]="./cmd/im-svc"
  ["docker-svc"]="./cmd/docker-svc"
  ["camera-svc"]="./cmd/camera-svc"
)

# 获取远程 compose 文件名
REMOTE_COMPOSE="${REMOTE_DEPLOY}/docker-compose.single.yml"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
SERVER_DIR="${PROJECT_ROOT}/server"
CLIENT_DIR="${PROJECT_ROOT}/client"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[INFO]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
err()  { echo -e "${RED}[ERR]${NC}  $*"; }

# ── 远程执行 ──
remote() {
  $SSH_CMD "$@"
}

# ── 检测变更 ──
detect_changes() {
  # 比较本地与 origin/master 的差异
  local base
  base=$(git -C "$PROJECT_ROOT" merge-base HEAD origin/master 2>/dev/null || echo "HEAD~1")
  git -C "$PROJECT_ROOT" diff --name-only "$base" HEAD
}

# ── 构建 Go 服务 ──
build_go_service() {
  local svc="$1"
  local cmd_path="${GO_SERVICES[$svc]}"
  local out="/tmp/${svc}"

  log "交叉编译 ${svc} ..."
  cd "$SERVER_DIR"
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o "$out" "$cmd_path"
  echo "$out"
}

# ── 构建前端 ──
build_frontend() {
  log "构建前端 ..."
  cd "$CLIENT_DIR"
  npm run build -- --outDir dist_new 2>&1 | tail -1
  echo "${CLIENT_DIR}/dist_new"
}

# ── 部署 Go 服务 ──
deploy_go_service() {
  local svc="$1"
  local binary="$2"

  log "上传 ${svc} 到服务器 ..."
  $SCP_CMD "$binary" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_BINS}/${svc}"

  log "重启 ${svc} 容器 ..."
  remote "
    cd ${REMOTE_DEPLOY} &&
    docker compose -f ${REMOTE_COMPOSE} stop ${svc} &&
    docker compose -f ${REMOTE_COMPOSE} create ${svc} &&
    docker cp ${REMOTE_BINS}/${svc} \$(docker compose -f ${REMOTE_COMPOSE} ps -q ${svc}):/app/service &&
    docker start \$(docker compose -f ${REMOTE_COMPOSE} ps -a -q ${svc})
  " 2>&1

  # 健康检查
  sleep 2
  local health
  health=$(remote "curl -s -o /dev/null -w '%{http_code}' http://localhost:80/healthz/${svc}" 2>/dev/null || echo "000")
  if [ "$health" = "200" ]; then
    log "${svc} 部署完成 ✓"
  else
    warn "${svc} 健康检查返回 ${health}，请手动确认"
  fi
}

# ── 部署前端 ──
deploy_frontend() {
  local dist="$1"

  log "上传前端文件 ..."
  # 打包上传
  cd "$CLIENT_DIR"
  tar czf /tmp/dist_new.tar.gz -C "$dist" .
  $SCP_CMD /tmp/dist_new.tar.gz "${REMOTE_USER}@${REMOTE_HOST}:/tmp/"

  remote "
    mkdir -p ${REMOTE_HOME}/client/dist_new &&
    tar xzf /tmp/dist_new.tar.gz -C ${REMOTE_HOME}/client/dist_new/ &&
    rm /tmp/dist_new.tar.gz
  "

  log "切换 dist 目录 ..."
  remote "
    cd ${REMOTE_HOME}/client &&
    if [ -d dist_old ]; then rm -rf dist_old; fi &&
    if [ -d dist ]; then mv dist dist_old; fi &&
    mv dist_new dist
  "

  log "重启 nginx ..."
  remote "cd ${REMOTE_DEPLOY} && docker compose -f ${REMOTE_COMPOSE} restart nginx" 2>&1

  log "前端部署完成 ✓"
}

# ── 部署 Nginx 配置 ──
deploy_nginx_conf() {
  log "上传 nginx 配置 ..."
  $SCP_CMD "${PROJECT_ROOT}/deploy/nginx/nginx.conf" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_DEPLOY}/nginx/nginx.conf"
  remote "cd ${REMOTE_DEPLOY} && docker compose -f ${REMOTE_COMPOSE} restart nginx" 2>&1
  log "nginx 配置更新完成 ✓"
}

# ── 重新构建镜像 (完整部署) ──
deploy_full_rebuild() {
  local svc="$1"
  log "远程重新构建 ${svc} 镜像 ..."
  remote "
    cd ${REMOTE_DEPLOY} &&
    docker compose -f ${REMOTE_COMPOSE} build ${svc} &&
    docker compose -f ${REMOTE_COMPOSE} up -d ${svc}
  " 2>&1
  log "${svc} 重建完成 ✓"
}

# ── 主流程 ──
main() {
  local mode="${1:-auto}"

  log "CloudNexus 部署工具"
  echo "  远程: ${REMOTE_USER}@${REMOTE_HOST}"
  echo "  模式: ${mode}"
  echo ""

  # 检查连接
  if ! remote "echo ok" &>/dev/null; then
    err "无法连接到 ${REMOTE_HOST}，请检查 SSH 配置"
    exit 1
  fi

  if [ "$mode" = "all" ]; then
    # ── 全量部署 ──
    dist=$(build_frontend)
    deploy_frontend "$dist"

    for svc in "${!GO_SERVICES[@]}"; do
      bin=$(build_go_service "$svc")
      deploy_go_service "$svc" "$bin"
    done

  elif [ "$mode" = "frontend" ]; then
    # ── 仅前端 ──
    dist=$(build_frontend)
    deploy_frontend "$dist"

  elif [ -n "${GO_SERVICES[$mode]+exists}" ]; then
    # ── 单个 Go 服务 ──
    bin=$(build_go_service "$mode")
    deploy_go_service "$mode" "$bin"

  else
    # ── 自动检测变更 ──
    log "检测文件变更 ..."
    changed=$(detect_changes)

    local deploy_fe=false
    local deploy_nginx=false
    local -a deploy_svcs=()

    while IFS= read -r f; do
      case "$f" in
        client/*)
          deploy_fe=true ;;
        deploy/nginx/*)
          deploy_nginx=true ;;
        server/cmd/user-file-svc/*|server/internal/userfile/*|server/pkg/*)
          deploy_svcs+=("user-file-svc") ;;
        server/cmd/im-svc/*|server/internal/im/*)
          deploy_svcs+=("im-svc") ;;
        server/cmd/docker-svc/*|server/internal/dockermgr/*)
          deploy_svcs+=("docker-svc") ;;
        server/cmd/camera-svc/*|server/internal/camera/*)
          deploy_svcs+=("camera-svc") ;;
      esac
    done <<< "$changed"

    # 去重
    deploy_svcs=($(printf '%s\n' "${deploy_svcs[@]}" | sort -u))

    if [ ${#deploy_svcs[@]} -eq 0 ] && [ "$deploy_fe" = false ] && [ "$deploy_nginx" = false ]; then
      log "未检测到需要部署的变更"
      exit 0
    fi

    echo "变更检测结果:"
    [ "$deploy_fe" = true ]    && echo "  - 前端"
    [ "$deploy_nginx" = true ] && echo "  - nginx 配置"
    for s in "${deploy_svcs[@]}"; do echo "  - ${s}"; done
    echo ""

    # 执行部署
    [ "$deploy_fe" = true ] && { dist=$(build_frontend); deploy_frontend "$dist"; }
    [ "$deploy_nginx" = true ] && deploy_nginx_conf
    for svc in "${deploy_svcs[@]}"; do
      bin=$(build_go_service "$svc")
      deploy_go_service "$svc" "$bin"
    done
  fi

  echo ""
  log "全部部署完成"
}

main "$@"
