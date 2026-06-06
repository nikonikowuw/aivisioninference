cmake_minimum_required(VERSION 3.16)
project(aivision-engine VERSION 1.0.0 LANGUAGES CXX C)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

# ============================================================
# 依赖
# ============================================================

# FlatBuffers
find_package(Flatbuffers REQUIRED)

# Threads
find_package(Threads REQUIRED)

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

target_include_directories(aivision-engine PRIVATE
    ${CMAKE_CURRENT_SOURCE_DIR}/include
    ${CMAKE_CURRENT_SOURCE_DIR}/..
    ${FLATBUFFERS_INCLUDE_DIRS}
)

target_link_libraries(aivision-engine PRIVATE
    ${FLATBUFFERS_LIBRARIES}
    Threads::Threads
    dl
    rt
)

# ============================================================
# 安装
# ============================================================

install(TARGETS aivision-engine DESTINATION bin)
install(FILES ${ENGINE_HEADERS} DESTINATION include/aivision-engine)
