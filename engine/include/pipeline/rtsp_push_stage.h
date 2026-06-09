#ifndef AIVISION_PIPELINE_RTSP_PUSH_STAGE_H
#define AIVISION_PIPELINE_RTSP_PUSH_STAGE_H

#include "pipeline.h"
#include "encoder_stage.h"
#include <thread>
#include <atomic>
#include <string>
#include <memory>
#include <cstdint>

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
    bool SendRequest(const std::string& request);
    bool ReadResponse(int& status_code, std::string& response);
    bool SendInterleavedRtp(const uint8_t* payload, size_t payload_size, uint32_t rtp_timestamp, bool marker);
    bool SendH264Nal(const uint8_t* nal, size_t nal_size, uint32_t rtp_timestamp, bool marker);

    std::string push_url_;
    std::shared_ptr<EncoderStage> encoder_;
    StageContext ctx_;
    
    std::atomic<bool> running_{false};
    std::atomic<bool> connected_{false};
    std::unique_ptr<std::thread> thread_;
    
    int socket_fd_ = -1;
    std::string host_;
    std::string path_;
    uint16_t port_ = 554;
    uint32_t cseq_ = 1;
    std::string session_;
    uint16_t rtp_seq_ = 0;
    uint32_t rtp_ssrc_ = 0;
    uint64_t first_timestamp_ns_ = 0;
};

}  // namespace pipeline
}  // namespace aivision

#endif  // AIVISION_PIPELINE_RTSP_PUSH_STAGE_H
