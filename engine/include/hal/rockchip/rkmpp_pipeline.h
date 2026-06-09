#ifndef AIVISION_HAL_ROCKCHIP_RKMPP_PIPELINE_H
#define AIVISION_HAL_ROCKCHIP_RKMPP_PIPELINE_H

#include "pipeline/hal.h"

#include <atomic>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <vector>

namespace aivision {
namespace hal {
namespace rockchip {

/// RTP 包结构
struct RtpPacket {
    uint8_t marker = 0;
    uint8_t payload_type = 0;
    uint16_t seq = 0;
    uint32_t timestamp = 0;
    std::vector<uint8_t> payload;
};

/// 瑞芯微 RKMPP/RGA 硬件媒体流水线适配器。
/// 统一封装 MPP 解码/编码、RGA 硬件缩放，对外暴露 IMediaPipeline 接口。
/// 零 FFmpeg 依赖：原生 socket RTSP/RTP + MPP 解码 + RGA 缩放 + MPP 编码。
class RKMPPPipeline final : public pipeline::IMediaPipeline {
public:
    RKMPPPipeline();
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

#ifdef AIVISION_WITH_RKMPP
    // ================ RTSP 客户端 ================
    int rtsp_socket_ = -1;
    std::string rtsp_host_;
    int rtsp_port_ = 554;
    std::string rtsp_path_;
    std::string rtsp_session_;
    int rtsp_cseq_ = 1;

    bool RtspConnect(const std::string& url);
    void RtspDisconnect();
    bool RtspSendRequest(const std::string& req);
    bool RtspReadResponse(int& status_code, std::string& response);
    bool RtspOptions();
    bool RtspDescribe(std::string& sdp);
    bool RtspSetup();
    bool RtspPlay();
    void RtspTeardown();

    // ================ RTP 接收 ================
    std::unique_ptr<std::thread> pull_thread_;
    bool RecvRtpPacket(RtpPacket& pkt);

    // ================ H.264/H.265 NAL 处理 ================
    uint16_t expected_seq_ = 0;
    bool have_seq_ = false;
    std::vector<uint8_t> fu_buffer_;
    int nal_count_ = 0;
    bool has_sps_ = false;
    bool has_pps_ = false;
    std::vector<uint8_t> sps_;
    std::vector<uint8_t> pps_;

    void HandleRtpPacket(const RtpPacket& pkt);
    void EmitNal(const uint8_t* data, size_t size, uint32_t timestamp);
    bool ParseSpsPps(const std::string& sdp);

    // ================ MPP 解码器 ================
    void* mpp_dec_ctx_ = nullptr;       // MppCtx
    void* mpp_dec_mpi_ = nullptr;       // MppApi*
    void* dec_frame_group_ = nullptr;   // MppBufferGroup
    int dec_type_ = 0;                  // MppCodingType
    bool decoder_initialized_ = false;
    int dec_width_ = 0;
    int dec_height_ = 0;
    int dec_hor_stride_ = 0;
    int dec_ver_stride_ = 0;

    bool InitDecoder(int coding_type);
    void DestroyDecoder();
    void PullLoop();
    void DecodeNal(const uint8_t* data, size_t size, int64_t pts);
    void HandleDecodedFrame(void* frame);  // MppFrame
    pipeline::HwBufferPtr MppFrameToHwBuffer(void* frame); // MppFrame

    // ================ RGA Resize ================
    bool rga_enabled_ = false;
    int rga_dst_width_ = 0;
    int rga_dst_height_ = 0;
    uint8_t* rga_dst_buf_ = nullptr;
    int rga_dst_buf_size_ = 0;

    bool RgaResize(const pipeline::HwBufferDesc& src_desc,
                   pipeline::HwBufferDesc& dst_desc);

    // ================ MPP 编码器 ================
    void* mpp_enc_ctx_ = nullptr;       // MppCtx
    void* mpp_enc_mpi_ = nullptr;       // MppApi*
    void* enc_cfg_ = nullptr;           // MppEncCfg
    void* enc_buf_grp_ = nullptr;       // MppBufferGroup
    int enc_width_ = 0;
    int enc_height_ = 0;
    int enc_hor_stride_ = 0;
    int enc_ver_stride_ = 0;
    int enc_fps_ = 25;
    int enc_bitrate_ = 4000000;
    int enc_gop_ = 50;
    int enc_codec_type_ = 0;            // MppCodingType

    bool InitEncoder(int width, int height, int coding_type);
    void DestroyEncoder();
#endif

    // ---------- 基础状态 ----------
    mutable std::mutex mutex_;
    std::string config_json_;
    pipeline::FrameCallback frame_cb_;
    pipeline::StateCallback state_cb_;
    pipeline::PipelineState state_ = pipeline::PipelineState::Idle;
    pipeline::HALStatus last_status_ = pipeline::HALStatus::Success();
    std::atomic<bool> running_{false};
    bool encoder_initialized_ = false;
    bool paused_ = false;
};

}  // namespace rockchip
}  // namespace hal
}  // namespace aivision

extern "C" aivision::pipeline::IMediaPipeline* CreatePipeline();
extern "C" void DestroyPipeline(aivision::pipeline::IMediaPipeline* pipeline);

#endif  // AIVISION_HAL_ROCKCHIP_RKMPP_PIPELINE_H
