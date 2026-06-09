cmake_minimum_required(VERSION 3.16)
project(aivision-engine VERSION 1.0.0 LANGUAGES CXX C)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

if(APPLE)
    execute_process(COMMAND xcrun --sdk macosx --show-sdk-path OUTPUT_VARIABLE MACOSX_SDK_PATH OUTPUT_STRIP_TRAILING_WHITESPACE)
    include_directories(SYSTEM "${MACOSX_SDK_PATH}/usr/include/c++/v1")
endif()

# ============================================================
# 依赖
# ============================================================

# FlatBuffers
find_package(Flatbuffers REQUIRED)

# Threads
find_package(Threads REQUIRED)

# CURL
find_package(CURL REQUIRED)

# ============================================================
# Rockchip MPP/RGA 硬件加速
#
# 使用方法:
#   # 开发机 (x86_64):  stub 模式，不做硬件加速
#   cmake .. -DAIVISION_WITH_RKMPP=OFF
#
#   # RK3568 目标机: 本地编译，自动检测 MPP/RGA
#   cmake .. -DAIVISION_WITH_RKMPP=ON
#
#   # 交叉编译: 需要 SDK sysroot 包含 MPP/RGA
#   cmake .. -DAIVISION_WITH_RKMPP=ON \
#            -DRK_SDK_PATH=/path/to/sysroot \
#            -DCMAKE_TOOLCHAIN_FILE=cmake/rk3568-toolchain.cmake
# ============================================================

option(AIVISION_WITH_RKMPP "Enable Rockchip MPP/RGA hardware acceleration" OFF)

if(AIVISION_WITH_RKMPP)
    # 如果指定了 RK_SDK_PATH，优先在该路径下查找
    set(_rkmpp_search_hints)
    set(_rkmpp_search_paths)
    if(DEFINED RK_SDK_PATH)
        list(APPEND _rkmpp_search_hints ${RK_SDK_PATH}/usr/include ${RK_SDK_PATH}/usr/local/include)
        list(APPEND _rkmpp_search_paths ${RK_SDK_PATH}/usr/lib ${RK_SDK_PATH}/usr/lib/aarch64-linux-gnu)
    endif()

    # 查找 MPP 库 (支持多种命名变体)
    find_library(ROCKCHIP_MPP_LIB
        NAMES rockchip_mpp mpp rockchip_mpp_static
        PATHS /usr/lib /usr/lib/aarch64-linux-gnu /usr/lib/arm-linux-gnueabihf /usr/local/lib
              ${_rkmpp_search_paths}
    )
    find_path(ROCKCHIP_MPP_INCLUDE
        NAMES rockchip/rk_mpi.h
        PATHS /usr/include /usr/local/include
        HINTS ${_rkmpp_search_hints}
    )

    # 查找 RGA 库
    find_library(RGA_LIB
        NAMES rga librga
        PATHS /usr/lib /usr/lib/aarch64-linux-gnu /usr/lib/arm-linux-gnueabihf /usr/local/lib
              ${_rkmpp_search_paths}
    )
    find_path(RGA_INCLUDE
        NAMES im2d.hpp im2d.h
        PATHS /usr/include /usr/local/include /usr/include/librga
        HINTS ${_rkmpp_search_hints}
    )

    if(ROCKCHIP_MPP_LIB AND ROCKCHIP_MPP_INCLUDE)
        message(STATUS "Found Rockchip MPP: ${ROCKCHIP_MPP_LIB}")
    else()
        message(FATAL_ERROR "Rockchip MPP not found! Install rockchip-mpp-dev on RK3568, or set -DRK_SDK_PATH for cross-compile")
    endif()

    if(RGA_LIB AND RGA_INCLUDE)
        message(STATUS "Found RGA: ${RGA_LIB}")
    else()
        message(FATAL_ERROR "RGA not found! Install librga-dev on RK3568, or set -DRK_SDK_PATH for cross-compile")
    endif()
endif()

# ============================================================
# 源文件
# ============================================================

set(ENGINE_SOURCES
    src/main.cpp
    src/engine.cpp
    src/ipc_server.cpp
    src/pipeline/hw_buffer.cpp
    src/pipeline/ring_queue.cpp
    src/pipeline/worker_pool.cpp
    src/pipeline/snapshot.cpp
    src/pipeline/hal.cpp
    src/pipeline/pipeline.cpp
    src/pipeline/pipeline_manager.cpp
    src/pipeline/encoder_stage.cpp
    src/pipeline/inference_stage.cpp
    src/pipeline/rtsp_push_stage.cpp
    src/algo/so_handle.cpp
    src/algo/algo_instance.cpp
    src/algo/algo_manager.cpp
    src/monitor/metrics_reporter.cpp
)

set(ENGINE_HEADERS
    include/engine.h
    include/ipc/transport.h
    include/ipc/ipc_server.h
    include/ipc/heartbeat.h
    include/pipeline/hw_buffer.h
    include/pipeline/ring_queue.h
    include/pipeline/snapshot.h
    include/pipeline/worker_pool.h
    include/pipeline/hal.h
    include/pipeline/pipeline.h
    include/pipeline/pipeline_manager.h
    include/pipeline/encoder_stage.h
    include/pipeline/inference_stage.h
    include/pipeline/rtsp_push_stage.h
    include/algo/abi_contract.h
    include/algo/so_handle.h
    include/algo/algo_instance.h
    include/algo/algo_manager.h
    include/monitor/metrics_reporter.h
)

# ============================================================
# 编译选项
# ============================================================

if(CMAKE_CXX_COMPILER_ID MATCHES "GNU|Clang")
    add_compile_options(-Wall -Wextra -Wpedantic -Werror=return-type)
    add_compile_options(-O2 -g)
endif()

# ============================================================
# 可执行文件
# ============================================================

add_executable(aivision-engine ${ENGINE_SOURCES})

add_library(aivision-hal-rkmpp SHARED
    src/hal/rockchip/rkmpp_pipeline.cpp
    src/pipeline/hw_buffer.cpp
)

if(APPLE)
add_library(aivision-hal-macos-videotoolbox SHARED
    src/hal/macos/videotoolbox_pipeline.cpp
    src/pipeline/hw_buffer.cpp
)
endif()

target_include_directories(aivision-engine PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/include
    ${CMAKE_CURRENT_SOURCE_DIR}/..
    ${FLATBUFFERS_INCLUDE_DIRS}
)

target_include_directories(aivision-hal-rkmpp PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/include
)

if(APPLE)
target_include_directories(aivision-hal-macos-videotoolbox PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/include
)
endif()

target_link_libraries(aivision-engine PRIVATE
    flatbuffers::flatbuffers
    Threads::Threads
    dl
    ${CURL_LIBRARIES}
)

target_link_libraries(aivision-hal-rkmpp PRIVATE
    Threads::Threads
    dl
)

if(AIVISION_WITH_RKMPP AND ROCKCHIP_MPP_LIB AND RGA_LIB)
    target_compile_definitions(aivision-hal-rkmpp PRIVATE AIVISION_WITH_RKMPP)
    target_include_directories(aivision-hal-rkmpp PRIVATE
        ${ROCKCHIP_MPP_INCLUDE}
        ${RGA_INCLUDE}
    )
    target_link_libraries(aivision-hal-rkmpp PRIVATE
        ${ROCKCHIP_MPP_LIB}
        ${RGA_LIB}
    )
    message(STATUS "RKMPP/RGA enabled for aivision-hal-rkmpp")
else()
    message(STATUS "RKMPP disabled (stub mode)")
endif()

if(APPLE)
target_link_libraries(aivision-hal-macos-videotoolbox PRIVATE
    Threads::Threads
    dl
)
endif()

if(APPLE)
    target_link_libraries(aivision-hal-macos-videotoolbox PRIVATE
        "-framework VideoToolbox"
        "-framework CoreMedia"
        "-framework CoreVideo"
        "-framework CoreFoundation"
    )
endif()

if(NOT APPLE)
    target_link_libraries(aivision-engine PRIVATE rt)
endif()

# ============================================================
# 测试
# ============================================================

if(BUILD_TESTING OR NOT DEFINED BUILD_TESTING)
    enable_testing()

    add_executable(test_rkmpp_pipeline
        tests/test_rkmpp_pipeline.cpp
        src/hal/rockchip/rkmpp_pipeline.cpp
        src/pipeline/hw_buffer.cpp
    )

    target_include_directories(test_rkmpp_pipeline PRIVATE
        ${CMAKE_CURRENT_SOURCE_DIR}/include
    )

    target_link_libraries(test_rkmpp_pipeline PRIVATE
        Threads::Threads
        dl
    )

    if(AIVISION_WITH_RKMPP AND ROCKCHIP_MPP_LIB AND RGA_LIB)
        target_compile_definitions(test_rkmpp_pipeline PRIVATE AIVISION_WITH_RKMPP)
        target_include_directories(test_rkmpp_pipeline PRIVATE
            ${ROCKCHIP_MPP_INCLUDE}
            ${RGA_INCLUDE}
        )
        target_link_libraries(test_rkmpp_pipeline PRIVATE
            ${ROCKCHIP_MPP_LIB}
            ${RGA_LIB}
        )
    endif()

    add_test(NAME rkmpp_pipeline COMMAND test_rkmpp_pipeline)
endif()

# ============================================================
# 安装
# ============================================================

install(TARGETS aivision-engine DESTINATION bin)
install(TARGETS aivision-hal-rkmpp DESTINATION lib/aivision)
install(FILES ${ENGINE_HEADERS} DESTINATION include/aivision-engine)

# 安装 RK3568 相关工具
install(FILES
    ${CMAKE_CURRENT_SOURCE_DIR}/cmake/rk3568-toolchain.cmake
    DESTINATION share/aivision-engine/cmake
)
install(FILES
    ${CMAKE_CURRENT_SOURCE_DIR}/scripts/build-rk3568.sh
    DESTINATION share/aivision-engine/scripts
    PERMISSIONS OWNER_EXECUTE GROUP_EXECUTE WORLD_EXECUTE
)
