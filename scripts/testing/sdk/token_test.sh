#!/usr/bin/env bash
# Token 服务测试脚本
# 用途：运行 Token 服务的集成测试

set -eu

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[✅]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[❌]${NC} $1" >&2
}

log_test() {
    echo -e "${CYAN}[🧪]${NC} $1" >&2
}

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SDK_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# 初始化测试环境
log_info "初始化测试环境..."
source "${SCRIPT_DIR}/test_init.sh"
init_test_environment

# 切换到 SDK 目录
cd "${SDK_ROOT}"

# 运行 Token 服务测试
log_test "运行 Token 服务集成测试..."
go test ./test/integration/services/token/... -v -count=1

if [[ $? -eq 0 ]]; then
    log_success "Token 服务测试通过"
    exit 0
else
    log_error "Token 服务测试失败"
    exit 1
fi

