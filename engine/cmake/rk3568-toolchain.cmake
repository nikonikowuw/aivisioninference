# =============================================================================
# RK3568 交叉编译工具链 (ARM64/aarch64)
# 使用方法:
#   cmake -B build-rk3568 \
#         -DCMAKE_TOOLCHAIN_FILE=cmake/rk3568-toolchain.cmake \
#         -DAIVISION_WITH_RKMPP=ON \
#         -DRK_SDK_PATH=/path/to/rk3568/sdk/sysroot
# =============================================================================

# 目标架构
set(CMAKE_SYSTEM_NAME Linux)
set(CMAKE_SYSTEM_PROCESSOR aarch64)

# 编译器
set(CMAKE_C_COMPILER aarch64-linux-gnu-gcc)
set(CMAKE_CXX_COMPILER aarch64-linux-gnu-g++)

# 不需要运行时文件检测
set(CMAKE_TRY_COMPILE_TARGET_TYPE STATIC_LIBRARY)

# 指定 sysroot
if(DEFINED RK_SDK_PATH)
    set(CMAKE_SYSROOT ${RK_SDK_PATH})
    set(CMAKE_FIND_ROOT_PATH ${RK_SDK_PATH})
    set(CMAKE_FIND_ROOT_PATH_MODE_PROGRAM NEVER)
    set(CMAKE_FIND_ROOT_PATH_MODE_LIBRARY ONLY)
    set(CMAKE_FIND_ROOT_PATH_MODE_INCLUDE ONLY)
    set(CMAKE_FIND_ROOT_PATH_MODE_PACKAGE ONLY)
endif()

# 编译优化
set(CMAKE_CXX_FLAGS "${CMAKE_CXX_FLAGS} -march=armv8-a+crypto -mtune=cortex-a55")
set(CMAKE_C_FLAGS "${CMAKE_C_FLAGS} -march=armv8-a+crypto -mtune=cortex-a55")
