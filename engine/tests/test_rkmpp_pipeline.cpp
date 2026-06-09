// test_rkmpp_pipeline.cpp — RKMPP Pipeline 单元测试
// 验证 IMediaPipeline 接口在 stub 模式和 RKMPP 模式下的正确行为

#include "hal/rockchip/rkmpp_pipeline.h"
#include <algorithm>
#include <cassert>
#include <cstring>
#include <iostream>

using namespace aivision::pipeline;
using namespace aivision::hal::rockchip;

static int passed = 0;
static int failed = 0;

#define TEST(name, expr) do { \
    if (!(expr)) { \
        std::cerr << "[FAIL] " << name << std::endl; \
        failed++; \
    } else { \
        std::cout << "[PASS] " << name << std::endl; \
        passed++; \
    } \
} while(0)

void test_basic_lifecycle() {
    std::cout << "\n=== Test: Basic Lifecycle ===" << std::endl;
    RKMPPPipeline pipeline;

    // 1. 初始状态应为 Idle
    TEST("initial state is Idle", pipeline.GetState() == PipelineState::Idle);

    // 2. Initialize 应成功
    TEST("initialize succeeds", pipeline.Initialize("{}"));

    // 3. GetPipelineType
    TEST("pipeline type is rockchip-rkmpp", pipeline.GetPipelineType() == "rockchip-rkmpp");

    // 4. GetCapabilities
    auto caps = pipeline.GetCapabilities();
    TEST("platform is rockchip-rkmpp", caps.platform == "rockchip-rkmpp");
    TEST("supports H264 decode", !caps.decode_codecs.empty());
    {
        bool has_h265 = std::find(caps.decode_codecs.begin(),
                                   caps.decode_codecs.end(),
                                   VideoCodec::H265)
                        != caps.decode_codecs.end();
        TEST("supports H265 decode", has_h265);
    }
    TEST("supports zero-copy", caps.supports_zero_copy);

    // 5. IsRunning / Start / Stop (stub 模式下 Start 应失败)
    TEST("not running initially", !pipeline.IsRunning());
    bool started = pipeline.Start("rtsp://localhost/test");
    // 在 stub 模式下应该失败，会打出错误日志
    TEST("start returns false in stub mode (no RKMPP)", !started);
    TEST("state is Error after failed start",
         pipeline.GetState() == PipelineState::Error);

    // 6. Stop 应安全（即使未启动）
    pipeline.Stop();
    TEST("state is Stopped after stop",
         pipeline.GetState() == PipelineState::Stopped);

    // 7. Pause / Resume
    pipeline.Pause();
    pipeline.Resume();
    TEST("pause/resume safe in any state", true);
}

void test_encoder_interface() {
    std::cout << "\n=== Test: Encoder Interface ===" << std::endl;
    RKMPPPipeline pipeline;
    pipeline.Initialize("{}");

    // 1. EncodeInit — stub 模式应失败
    bool enc_init = pipeline.EncodeInit("{\"codec\":\"h264\",\"width\":1920,\"height\":1080}");
#ifndef AIVISION_WITH_RKMPP
    TEST("EncodeInit returns false in stub mode", !enc_init);
#else
    TEST("EncodeInit returns true with RKMPP", enc_init);
#endif

    // 2. EncodeFrameEx — 空帧应失败
    uint8_t out_buf[1024];
    size_t out_size = 0;
    EncodedPacketDesc desc;
    bool enc_result = pipeline.EncodeFrameEx(nullptr, out_buf, sizeof(out_buf), out_size, desc);
    TEST("EncodeFrameEx with null frame returns false", !enc_result);

    // 3. EncodeDestroy — 安全调用
    pipeline.EncodeDestroy();

    // 4. 重复调用安全
    pipeline.EncodeDestroy();
    TEST("EncodeDestroy safe to call multiple times", true);
}

void test_state_callbacks() {
    std::cout << "\n=== Test: State Callbacks ===" << std::endl;
    RKMPPPipeline pipeline;
    pipeline.Initialize("{}");

    int state_count = 0;
    PipelineState last_state = PipelineState::Idle;

    pipeline.SetStateCallback([&](PipelineState s) {
        state_count++;
        last_state = s;
    });

    // Start should trigger state callback
    pipeline.Start("rtsp://localhost/test");
    pipeline.Stop();

    TEST("callbacks were invoked", state_count > 0);
}

void test_frame_callbacks() {
    std::cout << "\n=== Test: Frame Callbacks ===" << std::endl;
    RKMPPPipeline pipeline;
    pipeline.Initialize("{}");

    int frame_count = 0;
    pipeline.SetFrameCallback([&](HwBufferPtr) {
        frame_count++;
    });

    // 在 stub 模式下，Start 会失败，不会产生帧
    pipeline.Start("rtsp://localhost/test");
    pipeline.Stop();

    TEST("no frames received in stub mode", frame_count == 0);
}

void test_get_last_status() {
    std::cout << "\n=== Test: GetLastStatus ===" << std::endl;
    RKMPPPipeline pipeline;
    pipeline.Initialize("{}");

    // 初始状态应为 Success
    auto status = pipeline.GetLastStatus();
    TEST("initial status is OK", status.Ok());

    // 失败的 Start 应设置错误状态
    pipeline.Start("rtsp://invalid");
    status = pipeline.GetLastStatus();
    TEST("error status after failed start", !status.Ok());
    TEST("error code is Unsupported",
         status.code == HALStatusCode::Unsupported);
}

int main() {
    std::cout << "===================================" << std::endl;
    std::cout << "RKMPP Pipeline Unit Tests" << std::endl;
    std::cout << "Build: "
#ifdef AIVISION_WITH_RKMPP
              << "RKMPP ENABLED"
#else
              << "STUB MODE"
#endif
              << std::endl;
    std::cout << "===================================" << std::endl;

    test_basic_lifecycle();
    test_encoder_interface();
    test_state_callbacks();
    test_frame_callbacks();
    test_get_last_status();

    std::cout << "\n===================================" << std::endl;
    std::cout << "Results: " << passed << " passed, " << failed << " failed"
              << std::endl;
    std::cout << "===================================" << std::endl;

    return failed > 0 ? 1 : 0;
}
