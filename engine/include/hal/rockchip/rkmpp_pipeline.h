#ifndef AIVISION_HAL_ROCKCHIP_RKMPP_PIPELINE_H
#define AIVISION_HAL_ROCKCHIP_RKMPP_PIPELINE_H

#include "pipeline/hal.h"

#include <atomic>
#include <mutex>
#include <string>

namespace aivision {
namespace hal {
namespace rockchip {

/// 瑞芯微 RKMPP/RGA 硬件媒体流水线适配器。
/// 该类只暴露 IMediaPipeline 统一接口，具体 MPP/RGA/V4L2 细节限制在插件内部。
class RKMPPPipeline final : public pipeline::IMediaPipeline {
public:
    RKMPPPipeline() = default;
    ~RKMPPPipeline() override;

    bool Initialize(const std::string& config_json) override;
    bool Start(const std::string& url) override;
    void Stop() override;
    void Pause() override;
    void Resume() override;
    bool IsRunning() const override;
    void SetFrameCallback(pipeline::FrameCallback cb) override;
    void SetStateCallback(pipeline::StateCallback cb) override;
    std::string GetPipelineType() const override;
    pipeline::PipelineState GetState() const override;
    bool EncodeInit(const std::string& config_json) override;
    bool EncodeFrameEx(pipeline::HwBufferPtr frame,
                       uint8_t* data,
                       size_t size,
                       size_t& out_size,
                       pipeline::EncodedPacketDesc& desc) override;
    void EncodeDestroy() override;
    int GetDecodeHWType() const override;
    pipeline::HALCapabilities GetCapabilities() const override;
    pipeline::HALStatus GetLastStatus() const override;

private:
    void SetState(pipeline::PipelineState state);
    void SetError(pipeline::HALStatusCode code, std::string message);

    mutable std::mutex mutex_;
    std::string config_json_;
    std::string stream_url_;
    pipeline::FrameCallback frame_cb_;
    pipeline::StateCallback state_cb_;
    pipeline::PipelineState state_ = pipeline::PipelineState::Idle;
    pipeline::HALStatus last_status_ = pipeline::HALStatus::Success();
    std::atomic<bool> running_{false};
    bool encoder_initialized_ = false;
};

}  // namespace rockchip
}  // namespace hal
}  // namespace aivision

extern "C" aivision::pipeline::IMediaPipeline* CreatePipeline();
extern "C" void DestroyPipeline(aivision::pipeline::IMediaPipeline* pipeline);

#endif  // AIVISION_HAL_ROCKCHIP_RKMPP_PIPELINE_H
