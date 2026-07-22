#!/bin/bash
# =============================================================================
# RK3568 构建脚本
#
# 用法:
#   # 在 RK3568 目标机上本地编译 (推荐)
#   ./scripts/build-rk3568.sh
#
#   # 在 x86 开发机上交叉编译 (需要 SDK sysroot)
#   ./scripts/build-rk3568.sh /path/to/rk3568/sysroot
# =============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

if [ $# -ge 1 ]; then
    # === 交叉编译模式 ===
    SYSROOT="$1"
    BUILD_DIR="${SCRIPT_DIR}/build-rk3568-cross"
    TOOLCHAIN_FILE="${SCRIPT_DIR}/cmake/rk3568-toolchain.cmake"

    echo "=== RK3568 交叉编译 ==="
    echo "  Sysroot: ${SYSROOT}"

    if [ ! -f "${TOOLCHAIN_FILE}" ]; then
        echo "Error: toolchain file not found: ${TOOLCHAIN_FILE}"
        exit 1
    fi

    mkdir -p "${BUILD_DIR}"
    cmake -B "${BUILD_DIR}" \
        -DCMAKE_TOOLCHAIN_FILE="${TOOLCHAIN_FILE}" \
        -DRK_SDK_PATH="${SYSROOT}" \
        -DAIVISION_WITH_RKMPP=ON \
        -DCMAKE_BUILD_TYPE=Release
    cmake --build "${BUILD_DIR}" -j$(nproc)

    echo ""
    echo "=== 交叉编译完成 ==="
    echo "  库: ${BUILD_DIR}/libaivision-hal-rkmpp.so"
else
    # === 本地编译模式 (在 RK3568 上执行) ===
    BUILD_DIR="${SCRIPT_DIR}/build-rk3568"

    echo "=== RK3568 本地编译 ==="
    echo "  请确保已安装: sudo apt install rockchip-mpp-dev librga-dev"

    mkdir -p "${BUILD_DIR}"
    cmake -B "${BUILD_DIR}" \
        -DAIVISION_WITH_RKMPP=ON \
        -DCMAKE_BUILD_TYPE=Release
    cmake --build "${BUILD_DIR}" -j$(nproc)

    echo ""
    echo "=== 编译完成 ==="
    echo "  引擎: ${BUILD_DIR}/aivision-engine (含 HAL 静态链接)"
fi
