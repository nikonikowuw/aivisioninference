#ifndef AIVISION_PIPELINE_ENCODER_STAGE_H
#define AIVISION_PIPELINE_ENCODER_STAGE_H

#include "pipeline.h"
#include "hal.h"
#include <atomic>
#include <condition_variable>
#include <mutex>
#include <queue>
#include <thread>
#include <utility>
#include <vector>

namespace aivision {
namespace pipeline {

/// 编码后的数据包
struct EncodedPacket {
    std::vector<uint8_t> data;
    uint64_t timestamp_ns;
    bool is_key_frame;
};

/// 编码 Stage — 从队列取帧并调用硬件编码器
class EncoderStage : public Stage {
public:
    explicit EncoderStage(IMediaPipeline* hal);
    ~EncoderStage() override;

    bool Init(const StageContext& ctx) override;
    bool Run() override;
    void Stop() override;
    void PushFrame(const FrameContext& frame) override;
    
    std::string GetName() const override { return "EncoderStage"; }
    bool IsRunning() const override { return running_.load(); }

    /// 获取编码后的数据包 (供推流 Stage 使用)
    std::pair<EncodedPacket, bool> PopPacket(uint32_t timeout_ms = 100);

private:
    void Loop();

    IMediaPipeline* hal_ = nullptr;
    StageContext ctx_;
    
    std::atomic<bool> running_{false};
    std::unique_ptr<std::thread> thread_;
    
    // 内部编码包队列
    std::queue<EncodedPacket> packet_queue_;
    std::mutex queue_mutex_;
    std::condition_variable queue_cv_;
    size_t max_packets_ = 30;
};

}  // namespace pipeline
}  // namespace aivision

#endif  // AIVISION_PIPELINE_ENCODER_STAGE_H
