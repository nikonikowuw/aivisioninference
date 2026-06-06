#include "pipeline/rtsp_push_stage.h"
#include <iostream>
#include <chrono>

namespace aivision {
namespace pipeline {

RtspPushStage::RtspPushStage(const std::string& push_url, std::shared_ptr<EncoderStage> encoder)
    : push_url_(push_url), encoder_(std::move(encoder)) {}

RtspPushStage::~RtspPushStage() {
    Stop();
}

bool RtspPushStage::Init(const StageContext& ctx) {
    ctx_ = ctx;
    return encoder_ != nullptr;
}

bool RtspPushStage::Run() {
    if (running_.load()) return true;
    running_.store(true);
    thread_ = std::make_unique<std::thread>(&RtspPushStage::Loop, this);
    return true;
}

void RtspPushStage::Stop() {
    if (!running_.load()) return;
    running_.store(false);
    if (thread_ && thread_->joinable()) {
        thread_->join();
    }
    Disconnect();
}

void RtspPushStage::PushFrame(const FrameContext& frame) {
    // RTSP Stage 通过 PopPacket 获取数据，不处理原始帧
}

void RtspPushStage::Loop() {
    uint32_t reconnect_delay_ms = 5000;
    
    while (running_.load()) {
        if (!connected_.load()) {
            if (Connect()) {
                connected_.store(true);
                reconnect_delay_ms = 5000;
                std::cout << "[RTSP] Connected to " << push_url_ << std::endl;
            } else {
                std::this_thread::sleep_for(std::chrono::milliseconds(reconnect_delay_ms));
                // 指数退避 (简单实现)
                reconnect_delay_ms = std::min(reconnect_delay_ms * 2, 120000u);
                continue;
            }
        }

        // 获取编码包
        auto [pkt, ok] = encoder_->PopPacket(100);
        if (!ok) continue;

        // 发送数据
        if (!SendPacket(pkt)) {
            std::cerr << "[RTSP] Send failed, reconnecting..." << std::endl;
            connected_.store(false);
            Disconnect();
        }
    }
}

bool RtspPushStage::Connect() {
    // TODO: 实现具体的 RTSP ANNOUNCE/SETUP/PLAY 流程 (针对 ZLM)
    // 模拟连接成功
    return true;
}

void RtspPushStage::Disconnect() {
    // TODO: 发送 TEARDOWN 并关闭 Socket
    connected_.store(false);
}

bool RtspPushStage::SendPacket(const EncodedPacket& pkt) {
    // TODO: 实现 RTP 打包 (H.264/H.265 over TCP)
    // 模拟发送成功
    return true;
}

}  // namespace pipeline
}  // namespace aivision
