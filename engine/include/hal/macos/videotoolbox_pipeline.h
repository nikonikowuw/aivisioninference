// macOS VideoToolbox 原生硬件解码 HAL 插件
// 零 FFmpeg 依赖：原生 socket RTSP/RTP + VTDecompressionSession + VTCompressionSession
#pragma once

#include "pipeline/hal.h"

#include <CoreMedia/CMSampleBuffer.h>
#include <CoreVideo/CVPixelBuffer.h>
#include <VideoToolbox/VTDecompressionSession.h>
#include <VideoToolbox/VTCompressionSession.h>

#include <atomic>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <vector>

namespace aivision { namespace pipeline {

class VideoToolboxPipeline : public IMediaPipeline {
public:
    VideoToolboxPipeline();
    ~VideoToolboxPipeline() override;

    // IMediaPipeline 接口
    bool Initialize(const std::string& config_json) override;
    bool Start(const std::string& url) override;
    void Stop() override;
    void Pause() override;
    void Resume() override;
    bool IsRunning() const override { return running_.load(); }
    PipelineState GetState() const override { return state_; }
    void SetFrameCallback(FrameCallback cb) override { frame_callback_ = std::move(cb); }
    void SetStateCallback(StateCallback cb) override { state_callback_ = std::move(cb); }
    std::string GetPipelineType() const override { return "videotoolbox-native"; }
    int GetDecodeHWType() const override { return 4; }  // VideoToolbox
    HALCapabilities GetCapabilities() const override;
    HALStatus GetLastStatus() const override { return last_status_; }

    bool EncodeInit(const std::string& config_json) override;
    bool EncodeFrame(HwBufferPtr frame, uint8_t* data, size_t size, size_t& out_size) override;
    bool EncodeFrameEx(HwBufferPtr frame, uint8_t* data, size_t size, size_t& out_size, EncodedPacketDesc& desc) override;
    void EncodeDestroy() override;

private:
    // ---- RTSP/RTP 原生客户端 ----
    struct RtpPacket {
        uint8_t  payload_type = 0;
        bool     marker = false;
        uint16_t seq = 0;
        uint32_t timestamp = 0;
        std::vector<uint8_t> payload;
    };

    bool RtspConnect(const std::string& url);
    void RtspDisconnect();
    bool RtspSendRequest(const std::string& req);
    bool RtspReadResponse(int& status_code, std::string& response);
    std::string RtspDigestHeader(const std::string& method, const std::string& uri) const;
    bool RtspParseAuthChallenge(const std::string& resp);
    bool RtspSendCommand(const std::string& method, std::string& resp,
                          const std::string& track = "",
                          bool include_transport = true);
    bool RtspOptions();
    bool RtspDescribe(std::string& sdp);
    bool RtspSetup();
    bool RtspPlay();
    bool RtspTeardown();

    bool RecvRtpPacket(RtpPacket& pkt);
    void HandleRtpPacket(const RtpPacket& pkt);
    void EmitNal(const uint8_t* data, size_t size, uint32_t timestamp);

    // ---- H264/H265 ----
    bool ParseCodecParams(const std::string& sdp);
    bool CreateFormatDescription();

    // ---- VideoToolbox 解码 ----
    bool InitDecoder();
    void DestroyDecoder();
    bool FeedNalToDecoder(const uint8_t* data, size_t size);
    static void DecodeCallback(void* refcon, void* source, OSStatus status,
                               VTDecodeInfoFlags flags, CVImageBufferRef img,
                               CMTime pts, CMTime duration);

    // ---- VideoToolbox 编码 ----
    bool InitEncoder(int width, int height);
    void DestroyEncoder();

    // ---- 拉流循环 ----
    void PullLoop();

    // 状态
    std::atomic<bool> running_{false};
    std::atomic<bool> paused_{false};
    PipelineState state_ = PipelineState::Idle;
    HALStatus last_status_;

    // RTSP/RTP
    int rtsp_socket_ = -1;
    std::string username_;
    std::string password_;
    std::string host_;
    int port_ = 0;
    std::string path_;
    int cseq_ = 0;
    std::string session_;
    std::string digest_realm_;
    std::string digest_nonce_;
    std::vector<std::pair<std::string, std::string>> sdp_tracks_;
    int interleaved_start_ = 0;
    uint16_t expected_seq_ = 0;
    bool have_seq_ = false;

    // H264/H265
    bool is_hevc_ = false;
    std::vector<uint8_t> sps_;
    std::vector<uint8_t> pps_;
    std::vector<uint8_t> vps_;
    bool have_vps_ = false;
    bool have_sps_ = false;
    bool have_pps_ = false;
    std::vector<uint8_t> fu_buffer_;

    // VideoToolbox 解码
    VTDecompressionSessionRef decode_session_ = nullptr;
    CMVideoFormatDescriptionRef format_desc_ = nullptr;
    bool decoder_initialized_ = false;

    // VideoToolbox 编码
    VTCompressionSessionRef encode_session_ = nullptr;
    bool encoder_initialized_ = false;
    std::mutex enc_mu_;
    std::vector<uint8_t>* encode_output_ = nullptr;  // 编码输出临时指针

    // 线程
    std::unique_ptr<std::thread> pull_thread_;

    // 回调
    FrameCallback frame_callback_;
    StateCallback state_callback_;
};

}} // namespace aivision::pipeline
