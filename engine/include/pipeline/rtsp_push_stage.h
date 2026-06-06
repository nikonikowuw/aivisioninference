#ifndef AIVISION_PIPELINE_RTSP_PUSH_STAGE_H
#define AIVISION_PIPELINE_RTSP_PUSH_STAGE_H

#include "pipeline.h"
#include "encoder_stage.h"
#include <thread>
#include <atomic>
#include <string>
#include <memory>

namespace aivision {
namespace pipeline {

/// RTSP 推流 Stage — 获取编码包并通过 RTSP 协议推送到远程服务器 (如 ZLM)
class RtspPushStage : public Stage {
public:
    RtspPushStage(const std::string& push_url, std::shared_ptr<EncoderStage> encoder);
    ~RtspPushStage() override;

    bool Init(const StageContext& ctx) override;
    bool Run() override;
    void Stop() override;
    void PushFrame(const FrameContext& frame) override;
    
    std::string GetName() const override { return "RtspPushStage"; }
    bool IsRunning() const override { return running_.load(); }

private:
    void Loop();
    bool Connect();
    void Disconnect();
    bool SendPacket(const EncodedPacket& pkt);

    std::string push_url_;
    std::shared_ptr<EncoderStage> encoder_;
    StageContext ctx_;
    
    std::atomic<bool> running_{false};
    std::atomic<bool> connected_{false};
    std::unique_ptr<std::thread> thread_;
    
    // TCP Socket 或 RTSP Client 句柄 (占位)
    int socket_fd_ = -1;
};

}  // namespace pipeline
}  // namespace aivision

#endif  // AIVISION_PIPELINE_RTSP_PUSH_STAGE_H
