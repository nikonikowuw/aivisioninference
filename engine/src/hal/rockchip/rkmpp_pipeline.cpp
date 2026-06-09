#include "hal/rockchip/rkmpp_pipeline.h"

#include <iostream>
#include <utility>

namespace aivision {
namespace hal {
namespace rockchip {

RKMPPPipeline::~RKMPPPipeline() {
    Stop();
    EncodeDestroy();
}

bool RKMPPPipeline::Initialize(const std::string& config_json) {
    std::lock_guard<std::mutex> lock(mutex_);
    config_json_ = config_json.empty() ? "{}" : config_json;
    last_status_ = pipeline::HALStatus::Success();
    state_ = pipeline::PipelineState::Idle;
    return true;
}

bool RKMPPPipeline::Start(const std::string& url) {
    std::lock_guard<std::mutex> lock(mutex_);
    stream_url_ = url;

#ifndef AIVISION_WITH_RKMPP
    running_.store(false);
    state_ = pipeline::PipelineState::Error;
    last_status_ = pipeline::HALStatus::Error(
        pipeline::HALStatusCode::Unsupported,
        "RKMPP SDK is not enabled at build time; rebuild with AIVISION_WITH_RKMPP and link MPP/RGA");
    std::cerr << "[RKMPP] " << last_status_.message << std::endl;
    if (state_cb_) state_cb_(state_);
    return false;
#else
    // TODO: 在 RK3588/RK356x 目标机上接入真实实现：
    // 1. RTSP demux/packet reader 输入码流
    // 2. MPP decoder 输出 DMA/NV12 HwBuffer
    // 3. RGA 可选缩放/颜色转换
    // 4. frame_cb_(MakeHwBuffer(desc)) 推入主 Pipeline RingQueue
    running_.store(true);
    state_ = pipeline::PipelineState::Streaming;
    last_status_ = pipeline::HALStatus::Success();
    if (state_cb_) state_cb_(state_);
    return true;
#endif
}

void RKMPPPipeline::Stop() {
    running_.store(false);
    SetState(pipeline::PipelineState::Stopped);
}

void RKMPPPipeline::Pause() {
    SetState(pipeline::PipelineState::Paused);
}

void RKMPPPipeline::Resume() {
    if (running_.load()) SetState(pipeline::PipelineState::Streaming);
}

bool RKMPPPipeline::IsRunning() const {
    return running_.load();
}

void RKMPPPipeline::SetFrameCallback(pipeline::FrameCallback cb) {
    std::lock_guard<std::mutex> lock(mutex_);
    frame_cb_ = std::move(cb);
}

void RKMPPPipeline::SetStateCallback(pipeline::StateCallback cb) {
    std::lock_guard<std::mutex> lock(mutex_);
    state_cb_ = std::move(cb);
}

std::string RKMPPPipeline::GetPipelineType() const {
    return "rockchip-rkmpp";
}

pipeline::PipelineState RKMPPPipeline::GetState() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return state_;
}

bool RKMPPPipeline::EncodeInit(const std::string& config_json) {
    std::lock_guard<std::mutex> lock(mutex_);
#ifndef AIVISION_WITH_RKMPP
    (void)config_json;
    encoder_initialized_ = false;
    last_status_ = pipeline::HALStatus::Error(
        pipeline::HALStatusCode::Unsupported,
        "RKMPP encoder is not enabled at build time");
    return false;
#else
    // TODO: 初始化 MPP encoder，解析 codec/bitrate/fps/gop 等通用配置。
    (void)config_json;
    encoder_initialized_ = true;
    last_status_ = pipeline::HALStatus::Success();
    return true;
#endif
}

bool RKMPPPipeline::EncodeFrameEx(pipeline::HwBufferPtr frame,
                                  uint8_t* data,
                                  size_t size,
                                  size_t& out_size,
                                  pipeline::EncodedPacketDesc& desc) {
    std::lock_guard<std::mutex> lock(mutex_);
    out_size = 0;
#ifndef AIVISION_WITH_RKMPP
    (void)frame;
    (void)data;
    (void)size;
    (void)desc;
    last_status_ = pipeline::HALStatus::Error(
        pipeline::HALStatusCode::Unsupported,
        "RKMPP encoder is not enabled at build time");
    return false;
#else
    // TODO: 调用 MPP encoder，把 DMA/NV12 HwBuffer 编码为 H264/H265 Annex-B。
    // desc.codec/pts/dts/is_key_frame/extra_data 由平台实现填充。
    (void)frame;
    (void)data;
    (void)size;
    desc.codec = pipeline::VideoCodec::H264;
    desc.is_key_frame = false;
    last_status_ = pipeline::HALStatus::Success();
    return false;
#endif
}

void RKMPPPipeline::EncodeDestroy() {
    std::lock_guard<std::mutex> lock(mutex_);
    encoder_initialized_ = false;
}

int RKMPPPipeline::GetDecodeHWType() const {
    return 1;
}

pipeline::HALCapabilities RKMPPPipeline::GetCapabilities() const {
    pipeline::HALCapabilities caps;
    caps.platform = "rockchip-rkmpp";
    caps.decode_codecs = {pipeline::VideoCodec::H264, pipeline::VideoCodec::H265, pipeline::VideoCodec::MJPEG};
    caps.encode_codecs = {pipeline::VideoCodec::H264, pipeline::VideoCodec::H265};
    caps.max_streams = 16;
    caps.max_width = 3840;
    caps.max_height = 2160;
    caps.supports_zero_copy = true;
    caps.supports_hardware_encode = true;
    return caps;
}

pipeline::HALStatus RKMPPPipeline::GetLastStatus() const {
    std::lock_guard<std::mutex> lock(mutex_);
    return last_status_;
}

void RKMPPPipeline::SetState(pipeline::PipelineState state) {
    pipeline::StateCallback cb;
    {
        std::lock_guard<std::mutex> lock(mutex_);
        state_ = state;
        cb = state_cb_;
    }
    if (cb) cb(state);
}

void RKMPPPipeline::SetError(pipeline::HALStatusCode code, std::string message) {
    std::lock_guard<std::mutex> lock(mutex_);
    last_status_ = pipeline::HALStatus::Error(code, std::move(message));
    state_ = pipeline::PipelineState::Error;
}

}  // namespace rockchip
}  // namespace hal
}  // namespace aivision

extern "C" aivision::pipeline::IMediaPipeline* CreatePipeline() {
    return new aivision::hal::rockchip::RKMPPPipeline();
}

extern "C" void DestroyPipeline(aivision::pipeline::IMediaPipeline* pipeline) {
    delete pipeline;
}
