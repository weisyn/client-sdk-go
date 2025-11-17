#!/usr/bin/env bash
# SDK 测试环境初始化脚本
# 用途：初始化 SDK 测试环境，确保 WES 节点运行

set -eu

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[✅]${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}[⚠️]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[❌]${NC} $1" >&2
}

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SDK_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
WES_ROOT="$(cd "${SDK_ROOT}/../weisyn.git" && pwd)"

# 节点配置
NODE_ENDPOINT="http://localhost:8080/jsonrpc"
NODE_STARTUP_TIMEOUT=60
NODE_CHECK_INTERVAL=2

# 检查节点是否运行
check_node_running() {
    if curl -sf "${NODE_ENDPOINT}" >/dev/null 2>&1 || \
       curl -sf "http://localhost:8080/health" >/dev/null 2>&1; then
        return 0
    fi
    return 1
}

# 启动节点
start_node() {
    log_info "正在启动 WES 节点..."

    if [[ ! -d "${WES_ROOT}" ]]; then
        log_error "WES 节点目录不存在: ${WES_ROOT}"
        exit 1
    fi

    cd "${WES_ROOT}"

    # 使用测试初始化脚本启动节点
    if [[ -f "scripts/testing/common/test_init.sh" ]]; then
        log_info "使用测试初始化脚本启动节点..."
        bash scripts/testing/common/test_init.sh
    else
        log_warning "未找到测试初始化脚本，尝试直接启动节点..."
        go run ./cmd/testing --api-only > /tmp/wes-node.log 2>&1 &
        NODE_PID=$!
        log_info "节点进程已启动 (PID: ${NODE_PID})"
    fi

    # 等待节点启动
    log_info "等待节点启动（最多 ${NODE_STARTUP_TIMEOUT} 秒）..."
    local waited=0
    while [[ ${waited} -lt ${NODE_STARTUP_TIMEOUT} ]]; do
        if check_node_running; then
            log_success "节点已启动并运行"
            return 0
        fi
        sleep ${NODE_CHECK_INTERVAL}
        waited=$((waited + NODE_CHECK_INTERVAL))
        echo -n "." >&2
    done
    echo "" >&2

    log_error "节点启动超时"
    return 1
}

# 主函数：初始化测试环境
init_test_environment() {
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "SDK 测试环境初始化"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

    # 检查节点是否运行
    if check_node_running; then
        log_success "节点已运行"
    else
        log_warning "节点未运行，正在启动..."
        if ! start_node; then
            log_error "节点启动失败"
            exit 1
        fi
    fi

    log_success "测试环境初始化完成"
}

# 如果直接运行此脚本，执行初始化
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    init_test_environment
fi

