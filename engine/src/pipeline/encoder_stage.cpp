#include "pipeline/encoder_stage.h"
#include <iostream>

namespace aivision {
namespace pipeline {

EncoderStage::EncoderStage(IMediaPipeline* hal) : hal_(hal) {}

EncoderStage::~EncoderStage() {
    Stop();
}

bool EncoderStage::Init(const StageContext& ctx) {
    ctx_ = ctx;
    if (!hal_) return false;
    
    // 初始化硬件编码器
    // 默认配置：H.264, 4Mbps, 25fps, GOP 50
    std::string enc_config = R"({"codec":"h264", "bitrate":4000000, "fps":25, "gop":50})";
    return hal_->EncodeInit(enc_config);
}

bool EncoderStage::Run() {
    if (running_.load()) return true;
    running_.store(true);
    thread_ = std::make_unique<std::thread>(&EncoderStage::Loop, this);
    return true;
}

void EncoderStage::Stop() {
    if (!running_.load()) return;
    running_.store(false);
    queue_cv_.notify_all();
    if (thread_ && thread_->joinable()) {
        thread_->join();
    }
    if (hal_) {
        hal_->EncodeDestroy();
    }
}

void EncoderStage::PushFrame(const FrameContext& frame) {
    // 编码 Stage 通常直接从 Pipeline 的 RingQueue 中由 Loop 线程主动 Pop
    // 此处 PushFrame 可用于直接注入（如果不需要 RingQueue 缓冲）
}

std::pair<EncodedPacket, bool> EncoderStage::PopPacket(uint32_t timeout_ms) {
    std::unique_lock<std::mutex> lock(queue_mutex_);
    if (queue_cv_.wait_for(lock, std::chrono::milliseconds(timeout_ms), 
        [this] { return !packet_queue_.empty() || !running_.load(); })) {
        if (packet_queue_.empty()) return {{}, false};
        
        EncodedPacket pkt = std::move(packet_queue_.front());
        packet_queue_.pop();
        return {std::move(pkt), true};
    }
    return {{}, false};
}

void EncoderStage::Loop() {
    std::vector<uint8_t> buffer(1024 * 1024); // 1MB 临时缓冲
    
    while (running_.load()) {
        if (!ctx_.queue) {
            std::this_thread::sleep_for(std::chrono::milliseconds(10));
            continue;
        }

        // 1. 从 Pipeline 队列获取解码后的帧
        auto [frame, ok] = ctx_.queue->TryPop(100);
        if (!ok || !frame.buffer) continue;

        // 2. 调用硬件编码器
        size_t out_size = 0;
        EncodedPacketDesc desc;
        if (hal_->EncodeFrameEx(frame.buffer, buffer.data(), buffer.size(), out_size, desc)) {
            EncodedPacket pkt;
            pkt.data.assign(buffer.data(), buffer.data() + out_size);
            pkt.timestamp_ns = desc.pts_ns != 0 ? desc.pts_ns : frame.timestamp_ns;
            pkt.dts_ns = desc.dts_ns;
            pkt.codec = desc.codec == VideoCodec::Unknown ? VideoCodec::H264 : desc.codec;
            pkt.extra_data = std::move(desc.extra_data);
            // 简单判断是否为 I 帧 (针对 H.264 NALU 类型 5)，优先使用 HAL 元数据
            pkt.is_key_frame = desc.is_key_frame || (out_size > 4 && (buffer[4] & 0x1F) == 5);

            // 3. 入包队列
            {
                std::lock_guard<std::mutex> lock(queue_mutex_);
                if (packet_queue_.size() >= max_packets_) {
                    packet_queue_.pop(); // 丢弃老包
                }
                packet_queue_.push(std::move(pkt));
            }
            queue_cv_.notify_one();
        }
    }
}

}  // namespace pipeline
}  // namespace aivision
