#include "hal/rockchip/rkmpp_pipeline.h"

#include <algorithm>
#include <cerrno>
#include <chrono>
#include <cctype>
#include <cstring>
#include <iostream>
#include <sstream>
#include <thread>

// ====================================================================
// RKMPP/RGA 真实实现 — 仅在 AIVISION_WITH_RKMPP 编译时启用
// ====================================================================
#ifdef AIVISION_WITH_RKMPP

#include <rockchip/rk_mpi.h>
#include <rockchip/mpp_buffer.h>
#include <rockchip/rk_venc_cmd.h>
#include <rockchip/rk_mpi_cmd.h>

#include <im2d.hpp>
#include <rga.h>

#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/stat.h>
#include <fcntl.h>
#include <poll.h>
#include <netdb.h>

// ====================================================================
// MPP/RGA 版本兼容性宏
// 兼容 MPP 1.5.0 和 RGA 2.2.0
// ====================================================================

// MPP_ALIGN 对齐宏（部分 MPP 版本未导出）
#ifndef MPP_ALIGN
#define MPP_ALIGN(x, a) (((x) + (a) - 1) & ~((a) - 1))
#endif

// MPP_PACKET_FLAG_INTRA 兼容（MPP 1.5.0 未定义）
#ifndef MPP_PACKET_FLAG_INTRA
#define MPP_PACKET_FLAG_INTRA 0x00000001
#endif

#endif

namespace aivision
{
    namespace hal
    {
        namespace rockchip
        {

            using namespace pipeline;

// ====================================================================
// 日志宏
// ====================================================================
#define RK_LOG_D(msg) std::cout << "[RKMPP] " << msg << std::endl
#define RK_LOG_I(msg) std::cout << "[RKMPP] " << msg << std::endl
#define RK_LOG_W(msg) std::cerr << "[RKMPP WARN] " << msg << std::endl
#define RK_LOG_E(msg) std::cerr << "[RKMPP ERROR] " << msg << std::endl

            // ====================================================================
            // 构造函数 / 析构函数
            // ====================================================================
            RKMPPPipeline::RKMPPPipeline() = default;

            RKMPPPipeline::~RKMPPPipeline()
            {
                Stop();
                EncodeDestroy();
#ifdef AIVISION_WITH_RKMPP
                delete[] rga_dst_buf_;
                rga_dst_buf_ = nullptr;
                rga_dst_buf_size_ = 0;
#endif
            }

            // ====================================================================
            // Initialize — 解析配置 JSON
            // ====================================================================
            bool RKMPPPipeline::Initialize(const std::string &config_json)
            {
                std::lock_guard<std::mutex> lock(mutex_);
                config_json_ = config_json.empty() ? "{}" : config_json;
                last_status_ = HALStatus::Success();
                state_ = PipelineState::Idle;

#ifdef AIVISION_WITH_RKMPP
                // 解析 RGA 配置
                // 支持字段: "rga_enable", "rga_output_width", "rga_output_height"
                rga_enabled_ = false;
                rga_dst_width_ = 0;
                rga_dst_height_ = 0;

                // 简单 JSON 解析 — 查找 "rga_enable":true
                auto find_json_bool = [&](const std::string &key, bool def) -> bool
                {
                    auto pos = config_json_.find(key);
                    if (pos == std::string::npos)
                        return def;
                    auto val_pos = config_json_.find(':', pos);
                    if (val_pos == std::string::npos)
                        return def;
                    return config_json_.find("true", val_pos) < config_json_.find('}', val_pos);
                };
                auto find_json_int = [&](const std::string &key, int def) -> int
                {
                    auto pos = config_json_.find(key);
                    if (pos == std::string::npos)
                        return def;
                    auto val_pos = config_json_.find(':', pos);
                    if (val_pos == std::string::npos)
                        return def;
                    auto start = config_json_.find_first_of("0123456789-", val_pos);
                    if (start == std::string::npos)
                        return def;
                    auto end = config_json_.find_first_not_of("0123456789", start);
                    return std::stoi(config_json_.substr(start, end - start));
                };

                rga_enabled_ = find_json_bool("\"rga_enable\"", false);
                rga_dst_width_ = find_json_int("\"rga_output_width\"", 0);
                rga_dst_height_ = find_json_int("\"rga_output_height\"", 0);

                if (rga_enabled_ && (rga_dst_width_ <= 0 || rga_dst_height_ <= 0))
                {
                    RK_LOG_W("rga_enable=true but invalid rga_output_width/height, disabling RGA");
                    rga_enabled_ = false;
                }

                // 解析编码初始配置（可选，也可由 EncodeInit 传入）
                if (find_json_bool("\"encoder_enable\"", false))
                {
                    enc_width_ = find_json_int("\"enc_width\"", 1920);
                    enc_height_ = find_json_int("\"enc_height\"", 1080);
                    enc_fps_ = find_json_int("\"enc_fps\"", 25);
                    enc_bitrate_ = find_json_int("\"enc_bitrate\"", 4000000);
                    enc_gop_ = find_json_int("\"enc_gop\"", 50);
                }

                RK_LOG_I("Initialized. RGA=" << (rga_enabled_ ? "on" : "off")
                                             << (rga_enabled_ ? " dst=" + std::to_string(rga_dst_width_) + "x" + std::to_string(rga_dst_height_) : ""));
#endif

                return true;
            }

            // ====================================================================
            // Start — 启动拉流
            // ====================================================================
            bool RKMPPPipeline::Start(const std::string &url)
            {
                std::lock_guard<std::mutex> lock(mutex_);

#ifndef AIVISION_WITH_RKMPP
                (void)url;
                running_.store(false);
                state_ = PipelineState::Error;
                last_status_ = HALStatus::Error(
                    HALStatusCode::Unsupported,
                    "RKMPP SDK is not enabled at build time; rebuild with AIVISION_WITH_RKMPP and link MPP/RGA");
                RK_LOG_E(last_status_.message);
                if (state_cb_)
                    state_cb_(state_);
                return false;
#else
                if (running_.load())
                {
                    RK_LOG_W("Already running, stop first");
                    return false;
                }

                // 检查 URL 协议
                if (url.find("rtsp://") != 0)
                {
                    SetError(HALStatusCode::Unsupported,
                             "Only rtsp:// URLs are supported, got: " + url.substr(0, url.find(':')));
                    if (state_cb_)
                        state_cb_(state_);
                    return false;
                }

                auto notify_state = [&]() {
                    if (state_cb_)
                        state_cb_(state_);
                };
                auto fail_after_connect = [&](bool destroy_decoder = false) {
                    if (destroy_decoder)
                        DestroyDecoder();
                    RtspDisconnect();
                    notify_state();
                    return false;
                };

                state_ = PipelineState::Connecting;
                notify_state();

                // 1. RTSP 连接
                if (!RtspConnect(url))
                {
                    RK_LOG_E("RTSP connect failed");
                    notify_state();
                    return false;
                }

                // 2. OPTIONS
                if (!RtspOptions())
                {
                    return fail_after_connect();
                }

                // 3. DESCRIBE — 获取 SDP，解析 SPS/PPS
                std::string sdp;
                if (!RtspDescribe(sdp))
                {
                    return fail_after_connect();
                }
                RK_LOG_I("SDP: " << sdp.substr(0, 200));

                int coding_type = MPP_VIDEO_CodingAVC;
                std::string codec_sdp = sdp;
                std::transform(codec_sdp.begin(), codec_sdp.end(), codec_sdp.begin(),
                               [](unsigned char ch) { return static_cast<char>(std::tolower(ch)); });
                if (codec_sdp.find("h265") != std::string::npos || codec_sdp.find("hevc") != std::string::npos)
                {
                    coding_type = MPP_VIDEO_CodingHEVC;
                    RK_LOG_I("Detected H.265 codec from SDP");
                }

                // 解析 SPS/PPS（H.264）
                if (coding_type == MPP_VIDEO_CodingAVC)
                {
                    ParseSpsPps(sdp);
                }

                // 4. SETUP
                if (!RtspSetup())
                {
                    return fail_after_connect();
                }

                // 5. 初始化 MPP 解码器
                if (!InitDecoder(coding_type))
                {
                    return fail_after_connect();
                }

                // 6. PLAY
                if (!RtspPlay())
                {
                    return fail_after_connect(true);
                }

                running_.store(true);
                state_ = PipelineState::Streaming;
                notify_state();

                // 7. 启动拉流线程
                have_seq_ = false;
                expected_seq_ = 0;
                nal_count_ = 0;
                has_sps_ = false;
                has_pps_ = false;
                fu_buffer_.clear();

                pull_thread_ = std::make_unique<std::thread>(&RKMPPPipeline::PullLoop, this);
                RK_LOG_I("RTSP playing: " << url);
                return true;
#endif
            }

            // ====================================================================
            // Stop — 停止拉流
            // ====================================================================
            void RKMPPPipeline::Stop()
            {
#ifdef AIVISION_WITH_RKMPP
                running_.store(false);
                paused_ = false;

                // 先断开 socket，唤醒可能阻塞在 recv() 的拉流线程
                RtspDisconnect();

                // 等待拉流线程
                if (pull_thread_ && pull_thread_->joinable())
                {
                    pull_thread_->join();
                    pull_thread_.reset();
                }

                // 释放 RGA 持久化缓冲
                delete[] rga_dst_buf_;
                rga_dst_buf_ = nullptr;
                rga_dst_buf_size_ = 0;

                DestroyDecoder();
                RK_LOG_I("Stopped");
#endif

                SetState(PipelineState::Stopped);
            }

            // ====================================================================
            // Pause / Resume
            // ====================================================================
            void RKMPPPipeline::Pause()
            {
                paused_ = true;
                SetState(PipelineState::Paused);
            }

            void RKMPPPipeline::Resume()
            {
                paused_ = false;
                if (running_.load())
                {
                    SetState(PipelineState::Streaming);
                }
            }

            // ====================================================================
            // 查询状态
            // ====================================================================
            bool RKMPPPipeline::IsRunning() const
            {
                return running_.load();
            }

            void RKMPPPipeline::SetFrameCallback(FrameCallback cb)
            {
                std::lock_guard<std::mutex> lock(mutex_);
                frame_cb_ = std::move(cb);
            }

            void RKMPPPipeline::SetStateCallback(StateCallback cb)
            {
                std::lock_guard<std::mutex> lock(mutex_);
                state_cb_ = std::move(cb);
            }

            std::string RKMPPPipeline::GetPipelineType() const
            {
                return "rockchip-rkmpp";
            }

            PipelineState RKMPPPipeline::GetState() const
            {
                std::lock_guard<std::mutex> lock(mutex_);
                return state_;
            }

            int RKMPPPipeline::GetDecodeHWType() const
            {
                return 1; // 硬件解码类型
            }

            HALCapabilities RKMPPPipeline::GetCapabilities() const
            {
                HALCapabilities caps;
                caps.platform = "rockchip-rkmpp";
                caps.decode_codecs = {VideoCodec::H264, VideoCodec::H265, VideoCodec::MJPEG};
                caps.encode_codecs = {VideoCodec::H264, VideoCodec::H265};
                caps.max_streams = 16;
                caps.max_width = 3840;
                caps.max_height = 2160;
                caps.supports_zero_copy = true;
                caps.supports_hardware_encode = true;
                return caps;
            }

            HALStatus RKMPPPipeline::GetLastStatus() const
            {
                std::lock_guard<std::mutex> lock(mutex_);
                return last_status_;
            }

            // ====================================================================
            // 编码器接口
            // ====================================================================
            bool RKMPPPipeline::EncodeInit(const std::string &config_json)
            {
                std::lock_guard<std::mutex> lock(mutex_);

#ifndef AIVISION_WITH_RKMPP
                (void)config_json;
                encoder_initialized_ = false;
                last_status_ = HALStatus::Error(
                    HALStatusCode::Unsupported,
                    "RKMPP encoder is not enabled at build time");
                return false;
#else
                // 解析编码配置
                auto find_int = [&](const std::string &key, int def) -> int
                {
                    auto pos = config_json.find(key);
                    if (pos == std::string::npos)
                        return def;
                    auto val_pos = config_json.find(':', pos);
                    if (val_pos == std::string::npos)
                        return def;
                    auto start = config_json.find_first_of("0123456789-", val_pos);
                    if (start == std::string::npos)
                        return def;
                    auto end = config_json.find_first_not_of("0123456789", start);
                    return std::stoi(config_json.substr(start, end - start));
                };

                enc_width_ = find_int("\"width\"", enc_width_ > 0 ? enc_width_ : 1920);
                enc_height_ = find_int("\"height\"", enc_height_ > 0 ? enc_height_ : 1080);
                enc_fps_ = find_int("\"fps\"", 25);
                enc_bitrate_ = find_int("\"bitrate\"", 4000000);
                enc_gop_ = find_int("\"gop\"", 50);

                std::string codec_str = "h264";
                auto cpos = config_json.find("\"codec\"");
                if (cpos != std::string::npos)
                {
                    auto vpos = config_json.find(':', cpos);
                    if (vpos != std::string::npos)
                    {
                        auto q1 = config_json.find('"', vpos);
                        if (q1 != std::string::npos)
                        {
                            auto q2 = config_json.find('"', q1 + 1);
                            if (q2 != std::string::npos)
                            {
                                codec_str = config_json.substr(q1 + 1, q2 - q1 - 1);
                            }
                        }
                    }
                }

                int coding_type = MPP_VIDEO_CodingAVC; // H264
                if (codec_str == "h265" || codec_str == "hevc")
                {
                    coding_type = MPP_VIDEO_CodingHEVC;
                }

                bool ok = InitEncoder(enc_width_, enc_height_, coding_type);
                encoder_initialized_ = ok;
                return ok;
#endif
            }

            bool RKMPPPipeline::EncodeFrameEx(HwBufferPtr frame,
                                              uint8_t *data,
                                              size_t size,
                                              size_t &out_size,
                                              EncodedPacketDesc &desc)
            {
                std::lock_guard<std::mutex> lock(mutex_);
                out_size = 0;

#ifndef AIVISION_WITH_RKMPP
                (void)frame;
                (void)data;
                (void)size;
                (void)desc;
                last_status_ = HALStatus::Error(
                    HALStatusCode::Unsupported,
                    "RKMPP encoder is not enabled at build time");
                return false;
#else
                if (!encoder_initialized_ || !mpp_enc_ctx_ || !mpp_enc_mpi_)
                {
                    last_status_ = HALStatus::Error(
                        HALStatusCode::EncodeFailed, "Encoder not initialized");
                    return false;
                }

                if (!frame || !frame->IsValid())
                {
                    last_status_ = HALStatus::Error(
                        HALStatusCode::EncodeFailed, "Invalid input frame");
                    return false;
                }

                const HwBufferDesc &hw_desc = frame->Desc();
                if (hw_desc.memory_type != HwBufferMemoryType::DMABuf || hw_desc.dma_fd < 0)
                {
                    last_status_ = HALStatus::Error(
                        HALStatusCode::EncodeFailed, "Frame must be DMABuf type");
                    return false;
                }

                MPP_RET ret = MPP_OK;

                // 1. 从 DMA fd 导入到编码器的 buffer group
                // MPP 1.5.0 兼容: 使用 mpp_buffer_import(buffer, info)
                MppBuffer enc_buf = nullptr;
                MppBufferInfo buf_info = {};
                buf_info.type = MPP_BUFFER_TYPE_EXT_DMA;
                buf_info.fd = hw_desc.dma_fd;
                buf_info.size = hw_desc.size;
                ret = mpp_buffer_import(&enc_buf, &buf_info);
                if (ret != MPP_OK || !enc_buf)
                {
                    RK_LOG_W("mpp_buffer_import failed for DMA fd " << hw_desc.dma_fd);
                    last_status_ = HALStatus::Error(
                        HALStatusCode::EncodeFailed,
                        "mpp_buffer_import failed: " + std::to_string(ret));
                    return false;
                }

                // 2. 创建输入帧
                MppFrame enc_frame = nullptr;
                ret = mpp_frame_init(&enc_frame);
                if (ret != MPP_OK)
                {
                    mpp_buffer_put(enc_buf);
                    last_status_ = HALStatus::Error(
                        HALStatusCode::EncodeFailed, "mpp_frame_init failed");
                    return false;
                }

                int fenc_width = enc_width_ > 0 ? enc_width_ : static_cast<int>(hw_desc.width);
                int fenc_height = enc_height_ > 0 ? enc_height_ : static_cast<int>(hw_desc.height);

                mpp_frame_set_width(enc_frame, fenc_width);
                mpp_frame_set_height(enc_frame, fenc_height);
                mpp_frame_set_hor_stride(enc_frame, MPP_ALIGN(fenc_width, 16));
                mpp_frame_set_ver_stride(enc_frame, MPP_ALIGN(fenc_height, 16));
                mpp_frame_set_fmt(enc_frame, MPP_FMT_YUV420SP);
                mpp_frame_set_buffer(enc_frame, enc_buf);

                // 3. 创建输出包
                MppPacket enc_packet = nullptr;
                mpp_packet_init_with_buffer(&enc_packet, nullptr);
                if (!enc_packet)
                {
                    mpp_frame_deinit(&enc_frame);
                    mpp_buffer_put(enc_buf);
                    return false;
                }

                // 通过 meta 关联输出包
                MppMeta meta = mpp_frame_get_meta(enc_frame);
                mpp_packet_set_length(enc_packet, 0);
                mpp_meta_set_packet(meta, KEY_OUTPUT_PACKET, enc_packet);

                // 4. 送帧编码
                ret = static_cast<MppApi *>(mpp_enc_mpi_)->encode_put_frame(static_cast<MppCtx>(mpp_enc_ctx_), enc_frame);
                if (ret != MPP_OK)
                {
                    RK_LOG_E("encode_put_frame failed: " << ret);
                    mpp_packet_deinit(&enc_packet);
                    mpp_frame_deinit(&enc_frame);
                    mpp_buffer_put(enc_buf);
                    last_status_ = HALStatus::Error(
                        HALStatusCode::EncodeFailed, "encode_put_frame failed");
                    return false;
                }

                // 5. 获取编码输出
                ret = static_cast<MppApi *>(mpp_enc_mpi_)->encode_get_packet(static_cast<MppCtx>(mpp_enc_ctx_), &enc_packet);
                if (ret != MPP_OK)
                {
                    RK_LOG_E("encode_get_packet failed: " << ret);
                    mpp_packet_deinit(&enc_packet);
                    mpp_frame_deinit(&enc_frame);
                    mpp_buffer_put(enc_buf);
                    last_status_ = HALStatus::Error(
                        HALStatusCode::EncodeFailed, "encode_get_packet failed");
                    return false;
                }

                // 6. 复制编码数据
                if (enc_packet && mpp_packet_get_length(enc_packet) > 0)
                {
                    void *pkt_ptr = mpp_packet_get_data(enc_packet);
                    size_t pkt_len = mpp_packet_get_length(enc_packet);

                    if (pkt_len <= size)
                    {
                        std::memcpy(data, pkt_ptr, pkt_len);
                        out_size = pkt_len;
                        desc.pts_ns = mpp_packet_get_pts(enc_packet);
                        desc.dts_ns = mpp_packet_get_dts(enc_packet);
                        desc.is_key_frame = (mpp_packet_get_flag(enc_packet) & MPP_PACKET_FLAG_INTRA) != 0;
                        desc.codec = (enc_codec_type_ == MPP_VIDEO_CodingHEVC) ? VideoCodec::H265 : VideoCodec::H264;

                        // 提取 extra_data (SPS/PPS)
                        if (desc.is_key_frame)
                        {
                            // 通过 MPP_ENC_GET_HDR_SYNC 获取头信息
                            MppPacket hdr_packet = nullptr;
                            mpp_packet_init_with_buffer(&hdr_packet, nullptr);
                            if (hdr_packet)
                            {
                                ret = static_cast<MppApi *>(mpp_enc_mpi_)->control(static_cast<MppCtx>(mpp_enc_ctx_), MPP_ENC_GET_HDR_SYNC, hdr_packet);
                                if (ret == MPP_OK)
                                {
                                    size_t hdr_len = mpp_packet_get_length(hdr_packet);
                                    if (hdr_len > 0)
                                    {
                                        desc.extra_data.resize(hdr_len);
                                        std::memcpy(desc.extra_data.data(),
                                                    mpp_packet_get_data(hdr_packet), hdr_len);
                                    }
                                }
                                mpp_packet_deinit(&hdr_packet);
                            }
                        }
                    }
                    else
                    {
                        RK_LOG_W("Output buffer too small: " << pkt_len << " > " << size);
                        mpp_packet_deinit(&enc_packet);
                        mpp_frame_deinit(&enc_frame);
                        mpp_buffer_put(enc_buf);
                        last_status_ = HALStatus::Error(
                            HALStatusCode::EncodeFailed, "Output buffer too small");
                        return false;
                    }
                }

                // 7. 清理
                mpp_packet_deinit(&enc_packet);
                mpp_frame_deinit(&enc_frame);
                mpp_buffer_put(enc_buf);

                if (out_size > 0)
                {
                    last_status_ = HALStatus::Success();
                    return true;
                }

                return false;
#endif
            }

            void RKMPPPipeline::EncodeDestroy()
            {
#ifndef AIVISION_WITH_RKMPP
                encoder_initialized_ = false;
#else
                DestroyEncoder();
#endif
            }

            // ====================================================================
            // 内部辅助方法
            // ====================================================================
            void RKMPPPipeline::SetState(PipelineState state)
            {
                StateCallback cb;
                {
                    std::lock_guard<std::mutex> lock(mutex_);
                    state_ = state;
                    cb = state_cb_;
                }
                if (cb)
                    cb(state);
            }

            void RKMPPPipeline::SetError(HALStatusCode code, std::string message)
            {
                std::lock_guard<std::mutex> lock(mutex_);
                last_status_ = HALStatus::Error(code, std::move(message));
                state_ = PipelineState::Error;
            }

// ====================================================================
// AIVISION_WITH_RKMPP — 真实实现
// ====================================================================
#ifdef AIVISION_WITH_RKMPP

            // ---------- MPP 解码器初始化 ----------
            bool RKMPPPipeline::InitDecoder(int coding_type)
            {
                MPP_RET ret = MPP_OK;

                // 保存编码类型
                dec_type_ = coding_type;

                // 1. 创建 MPP 上下文
                MppCtx ctx = nullptr;
                MppApi *mpi = nullptr;
                ret = mpp_create(&ctx, &mpi);
                if (ret != MPP_OK)
                {
                    RK_LOG_E("mpp_create failed: " << ret);
                    return false;
                }
                mpp_dec_ctx_ = ctx;
                mpp_dec_mpi_ = mpi;

                // 2. 初始化解码器
                ret = mpp_init(ctx, MPP_CTX_DEC, static_cast<MppCodingType>(coding_type));
                if (ret != MPP_OK)
                {
                    RK_LOG_E("mpp_init DEC failed: " << ret);
                    mpp_destroy(ctx);
                    mpp_dec_ctx_ = nullptr;
                    mpp_dec_mpi_ = nullptr;
                    return false;
                }

                // 3. 设置解码参数 — 启用自动帧分割
                MppDecCfg cfg = nullptr;
                ret = mpp_dec_cfg_init(&cfg);
                if (ret == MPP_OK)
                {
                    mpp_dec_cfg_set_u32(cfg, "base:split_parse", 1);
                    ret = mpi->control(ctx, MPP_DEC_SET_CFG, cfg);
                    if (ret != MPP_OK)
                    {
                        RK_LOG_W("MPP_DEC_SET_CFG failed, continuing: " << ret);
                    }
                    mpp_dec_cfg_deinit(cfg);
                }

                // 4. 设置 blocking 模式
                RK_S64 timeout = MPP_POLL_BLOCK;
                ret = mpi->control(ctx, MPP_SET_INPUT_TIMEOUT, &timeout);
                if (ret != MPP_OK)
                {
                    RK_LOG_W("MPP_SET_INPUT_TIMEOUT failed: " << ret);
                }
                ret = mpi->control(ctx, MPP_SET_OUTPUT_TIMEOUT, &timeout);
                if (ret != MPP_OK)
                {
                    RK_LOG_W("MPP_SET_OUTPUT_TIMEOUT failed: " << ret);
                }

                // 5. 创建内部 buffer group
                MppBufferGroup frame_group = nullptr;
                ret = mpp_buffer_group_get_internal(&frame_group, MPP_BUFFER_TYPE_DRM);
                if (ret != MPP_OK)
                {
                    RK_LOG_E("mpp_buffer_group_get_internal failed: " << ret);
                    mpp_destroy(ctx);
                    mpp_dec_ctx_ = nullptr;
                    mpp_dec_mpi_ = nullptr;
                    return false;
                }
                dec_frame_group_ = frame_group;

                // 分配给解码器
                ret = mpi->control(ctx, MPP_DEC_SET_EXT_BUF_GROUP, frame_group);
                if (ret != MPP_OK)
                {
                    RK_LOG_W("MPP_DEC_SET_EXT_BUF_GROUP failed: " << ret);
                }

                ret = mpp_buffer_group_limit_config(frame_group, 0, 16);
                if (ret != MPP_OK)
                {
                    RK_LOG_W("mpp_buffer_group_limit_config failed: " << ret);
                }

                decoder_initialized_ = true;
                dec_width_ = 0;
                dec_height_ = 0;

                RK_LOG_I("MPP decoder initialized, coding_type=" << coding_type);
                return true;
            }

            // ---------- 销毁解码器 ----------
            void RKMPPPipeline::DestroyDecoder()
            {
                if (mpp_dec_ctx_)
                {
                    if (mpp_dec_mpi_)
                    {
                        static_cast<MppApi *>(mpp_dec_mpi_)->reset(static_cast<MppCtx>(mpp_dec_ctx_));
                    }
                    mpp_destroy(static_cast<MppCtx>(mpp_dec_ctx_));
                    mpp_dec_ctx_ = nullptr;
                    mpp_dec_mpi_ = nullptr;
                }
                if (dec_frame_group_)
                {
                    mpp_buffer_group_put(static_cast<MppBufferGroup>(dec_frame_group_));
                    dec_frame_group_ = nullptr;
                }
                decoder_initialized_ = false;
                dec_width_ = 0;
                dec_height_ = 0;
                RK_LOG_I("MPP decoder destroyed");
            }

            // ---------- 解码循环（线程函数） ----------
            // ============================================================
            // PullLoop — RTSP 拉流 + RTP 解包 + MPP 解码主循环
            // ============================================================
            void RKMPPPipeline::PullLoop()
            {
                RK_LOG_I("PullLoop started");
                RtpPacket pkt;
                int pkt_count = 0;

                while (running_.load() && !paused_)
                {
                    if (!RecvRtpPacket(pkt))
                    {
                        if (running_.load())
                        {
                            RK_LOG_W("RecvRtp failed, retrying...");
                            std::this_thread::sleep_for(std::chrono::seconds(1));
                            continue;
                        }
                        break;
                    }
                    pkt_count++;
                    HandleRtpPacket(pkt);
                }

                // 发送 EOS 到 MPP 解码器
                if (decoder_initialized_ && mpp_dec_ctx_ && mpp_dec_mpi_)
                {
                    MppApi *mpi = static_cast<MppApi *>(mpp_dec_mpi_);
                    MppCtx ctx = static_cast<MppCtx>(mpp_dec_ctx_);

                    MppPacket eos_pkt = nullptr;
                    mpp_packet_init(&eos_pkt, nullptr, 0);
                    if (eos_pkt)
                    {
                        mpp_packet_set_eos(eos_pkt);
                        mpi->decode_put_packet(ctx, eos_pkt);
                        mpp_packet_deinit(&eos_pkt);

                        int timeout = 50;
                        while (timeout-- > 0)
                        {
                            MppFrame frame = nullptr;
                            MPP_RET ret = mpi->decode_get_frame(ctx, &frame);
                            if (ret != MPP_OK || !frame)
                                break;
                            if (mpp_frame_get_eos(frame))
                            {
                                mpp_frame_deinit(&frame);
                                break;
                            }
                            HandleDecodedFrame(frame);
                            if (mpp_frame_get_info_change(frame))
                                mpp_frame_deinit(&frame);
                        }
                    }
                }

                RK_LOG_I("PullLoop ended, packets=" << pkt_count);
            }

            // ============================================================
            // DecodeNal — 将单个 NAL 单元喂给 MPP 解码器
            // ============================================================
            void RKMPPPipeline::DecodeNal(const uint8_t *data, size_t size, int64_t pts)
            {
                if (!decoder_initialized_ || !mpp_dec_ctx_ || !mpp_dec_mpi_)
                    return;
                if (!data || size == 0)
                    return;

                MppApi *mpi = static_cast<MppApi *>(mpp_dec_mpi_);
                MppCtx ctx = static_cast<MppCtx>(mpp_dec_ctx_);
                MPP_RET ret = MPP_OK;

                MppPacket packet = nullptr;
                ret = mpp_packet_init(&packet, const_cast<void *>(static_cast<const void *>(data)), size);
                if (ret != MPP_OK)
                {
                    RK_LOG_E("mpp_packet_init failed: " << ret);
                    return;
                }

                mpp_packet_set_pts(packet, pts);

                ret = mpi->decode_put_packet(ctx, packet);
                if (ret != MPP_OK)
                {
                    if (ret != MPP_ERR_BUFFER_FULL)
                        RK_LOG_W("decode_put_packet failed: " << ret);
                    mpp_packet_deinit(&packet);
                    return;
                }

                mpp_packet_deinit(&packet);

                int max_frames = 16;
                while (max_frames-- > 0)
                {
                    MppFrame frame = nullptr;
                    ret = mpi->decode_get_frame(ctx, &frame);
                    if (ret == MPP_ERR_TIMEOUT || ret != MPP_OK)
                        break;
                    if (!frame)
                        break;

                    HandleDecodedFrame(frame);

                    if (mpp_frame_get_info_change(frame) || mpp_frame_get_eos(frame))
                        mpp_frame_deinit(&frame);
                }
            }

            // ============================================================
            // RTSP 客户端方法
            // ============================================================

            bool RKMPPPipeline::RtspConnect(const std::string &url)
            {
                std::string u = url;
                if (u.substr(0, 7) == "rtsp://")
                    u = u.substr(7);
                auto slash = u.find('/');
                std::string hostport = (slash == std::string::npos) ? u : u.substr(0, slash);
                rtsp_path_ = (slash == std::string::npos) ? "/" : u.substr(slash);
                // 支持 user:password@host:port 格式
                auto at_pos = hostport.rfind('@');
                if (at_pos != std::string::npos)
                    hostport = hostport.substr(at_pos + 1);
                auto colon = hostport.rfind(':');
                if (colon != std::string::npos && colon > 0)
                {
                    rtsp_host_ = hostport.substr(0, colon);
                    try {
                        rtsp_port_ = std::stoi(hostport.substr(colon + 1));
                    } catch (...) {
                        rtsp_port_ = 554;
                        rtsp_host_ = hostport;
                    }
                }
                else
                {
                    rtsp_host_ = hostport;
                    rtsp_port_ = 554;
                }

                if (rtsp_host_.empty())
                {
                    last_status_ = HALStatus::Error(HALStatusCode::InvalidConfig, "Invalid RTSP URL");
                    return false;
                }

                addrinfo hints{};
                hints.ai_family = AF_UNSPEC;
                hints.ai_socktype = SOCK_STREAM;
                addrinfo *result = nullptr;
                std::string port_str = std::to_string(rtsp_port_);

                if (getaddrinfo(rtsp_host_.c_str(), port_str.c_str(), &hints, &result) != 0)
                {
                    last_status_ = HALStatus::Error(HALStatusCode::OpenStreamFailed,
                                                    "DNS failed: " + rtsp_host_);
                    return false;
                }

                for (addrinfo *ai = result; ai; ai = ai->ai_next)
                {
                    rtsp_socket_ = ::socket(ai->ai_family, ai->ai_socktype, ai->ai_protocol);
                    if (rtsp_socket_ < 0)
                        continue;
                    struct timeval tv{5, 0};
                    setsockopt(rtsp_socket_, SOL_SOCKET, SO_SNDTIMEO, &tv, sizeof(tv));
                    if (::connect(rtsp_socket_, ai->ai_addr, ai->ai_addrlen) == 0)
                    {
                        struct timeval rv{10, 0};
                        setsockopt(rtsp_socket_, SOL_SOCKET, SO_RCVTIMEO, &rv, sizeof(rv));
                        break;
                    }
                    ::close(rtsp_socket_);
                    rtsp_socket_ = -1;
                }
                freeaddrinfo(result);

                if (rtsp_socket_ < 0)
                {
                    last_status_ = HALStatus::Error(HALStatusCode::OpenStreamFailed,
                                                    "TCP connect failed: " + rtsp_host_);
                    return false;
                }

                RK_LOG_I("RTSP connected to " << rtsp_host_ << ":" << rtsp_port_);
                rtsp_cseq_ = 1;
                rtsp_session_.clear();
                return true;
            }

            void RKMPPPipeline::RtspDisconnect()
            {
                if (rtsp_socket_ >= 0)
                {
                    ::shutdown(rtsp_socket_, SHUT_RDWR);
                    ::close(rtsp_socket_);
                    rtsp_socket_ = -1;
                }
            }

            bool RKMPPPipeline::RtspSendRequest(const std::string &req)
            {
                ssize_t sent = ::send(rtsp_socket_, req.data(), req.size(), 0);
                if (sent != static_cast<ssize_t>(req.size()))
                {
                    RK_LOG_E("RTSP send failed: sent=" << sent << " expected=" << req.size() << " errno=" << errno);
                    return false;
                }
                RK_LOG_D(">> " << req.substr(0, req.find("\r\n")));
                return true;
            }

            bool RKMPPPipeline::RtspReadResponse(int &status_code, std::string &response)
            {
                response.clear();
                char buf[4096];
                while (response.find("\r\n\r\n") == std::string::npos)
                {
                    ssize_t n = ::recv(rtsp_socket_, buf, sizeof(buf), 0);
                    if (n <= 0)
                    {
                        RK_LOG_E("RTSP recv failed: n=" << n << " errno=" << errno);
                        return false;
                    }
                    response.append(buf, static_cast<size_t>(n));
                    if (response.size() > 64 * 1024)
                        return false;
                }
                std::istringstream stream(response);
                std::string version;
                stream >> version >> status_code;
                RK_LOG_D("<< " << version << " " << status_code);
                return status_code > 0;
            }

            bool RKMPPPipeline::RtspOptions()
            {
                std::string uri = "rtsp://" + rtsp_host_ + ":" + std::to_string(rtsp_port_) + rtsp_path_;
                std::ostringstream req;
                req << "OPTIONS " << uri << " RTSP/1.0\r\n"
                    << "CSeq: " << rtsp_cseq_++ << "\r\n"
                    << "User-Agent: aivision-engine\r\n\r\n";
                int status = 0;
                std::string resp;
                return RtspSendRequest(req.str()) && RtspReadResponse(status, resp) &&
                       status >= 200 && status < 300;
            }

            bool RKMPPPipeline::RtspDescribe(std::string &sdp)
            {
                std::string uri = "rtsp://" + rtsp_host_ + ":" + std::to_string(rtsp_port_) + rtsp_path_;
                std::ostringstream req;
                req << "DESCRIBE " << uri << " RTSP/1.0\r\n"
                    << "CSeq: " << rtsp_cseq_++ << "\r\n"
                    << "User-Agent: aivision-engine\r\n"
                    << "Accept: application/sdp\r\n\r\n";
                int status = 0;
                std::string resp;
                if (!RtspSendRequest(req.str()) || !RtspReadResponse(status, resp) ||
                    status < 200 || status >= 300)
                    return false;
                auto pos = resp.find("\r\n\r\n");
                if (pos != std::string::npos)
                    sdp = resp.substr(pos + 4);
                return !sdp.empty();
            }

            bool RKMPPPipeline::RtspSetup()
            {
                std::string uri = "rtsp://" + rtsp_host_ + ":" + std::to_string(rtsp_port_) + rtsp_path_;
                std::ostringstream req;
                req << "SETUP " << uri << "/trackID=0 RTSP/1.0\r\n"
                    << "CSeq: " << rtsp_cseq_++ << "\r\n"
                    << "User-Agent: aivision-engine\r\n"
                    << "Transport: RTP/AVP/TCP;unicast;interleaved=0-1\r\n\r\n";
                int status = 0;
                std::string resp;
                if (!RtspSendRequest(req.str()) || !RtspReadResponse(status, resp) ||
                    status < 200 || status >= 300)
                    return false;

                auto sipos = resp.find("Session:");
                if (sipos == std::string::npos)
                    sipos = resp.find("session:");
                if (sipos != std::string::npos)
                {
                    auto end = resp.find("\r\n", sipos);
                    std::string line = resp.substr(sipos + 8, end - (sipos + 8));
                    auto trimstart = line.find_first_not_of(" \t");
                    auto trimend = line.find_last_not_of(" \t");
                    if (trimstart != std::string::npos && trimend != std::string::npos)
                        line = line.substr(trimstart, trimend - trimstart + 1);
                    auto semi = line.find(';');
                    rtsp_session_ = (semi == std::string::npos) ? line : line.substr(0, semi);
                }
                return true;
            }

            bool RKMPPPipeline::RtspPlay()
            {
                std::string uri = "rtsp://" + rtsp_host_ + ":" + std::to_string(rtsp_port_) + rtsp_path_;
                std::ostringstream req;
                req << "PLAY " << uri << " RTSP/1.0\r\n"
                    << "CSeq: " << rtsp_cseq_++ << "\r\n"
                    << "User-Agent: aivision-engine\r\n";
                if (!rtsp_session_.empty())
                    req << "Session: " << rtsp_session_ << "\r\n";
                req << "\r\n";
                int status = 0;
                std::string resp;
                return RtspSendRequest(req.str()) && RtspReadResponse(status, resp) &&
                       status >= 200 && status < 300;
            }

            void RKMPPPipeline::RtspTeardown()
            {
                if (rtsp_socket_ < 0)
                    return;
                std::string uri = "rtsp://" + rtsp_host_ + ":" + std::to_string(rtsp_port_) + rtsp_path_;
                std::ostringstream req;
                req << "TEARDOWN " << uri << " RTSP/1.0\r\n"
                    << "CSeq: " << rtsp_cseq_++ << "\r\n"
                    << "User-Agent: aivision-engine\r\n";
                if (!rtsp_session_.empty())
                    req << "Session: " << rtsp_session_ << "\r\n";
                req << "\r\n";
                int status = 0;
                std::string resp;
                RtspSendRequest(req.str());
                RtspReadResponse(status, resp);
            }

            // ============================================================
            // RTP 包接收 (TCP 交织模式)
            // ============================================================
            bool RKMPPPipeline::RecvRtpPacket(RtpPacket &pkt)
            {
                if (rtsp_socket_ < 0)
                    return false;

                uint8_t header[4];
                size_t got = 0;
                while (got < 4)
                {
                    ssize_t n = ::recv(rtsp_socket_, header + got, 4 - got, 0);
                    if (n <= 0)
                        return false;
                    got += static_cast<size_t>(n);
                }

                if (header[0] != 0x24)
                {
                    RK_LOG_W("Invalid RTP marker: 0x" << std::hex << (int)header[0] << std::dec);
                    return false;
                }

                uint16_t len = (static_cast<uint16_t>(header[2]) << 8) | header[3];
                if (len < 12 || len > 65535)
                    return false;

                std::vector<uint8_t> buf(len);
                got = 0;
                while (got < len)
                {
                    ssize_t n = ::recv(rtsp_socket_, buf.data() + got, len - got, 0);
                    if (n <= 0)
                        return false;
                    got += static_cast<size_t>(n);
                }

                uint8_t version = (buf[0] >> 6) & 0x03;
                if (version != 2)
                    return false;

                uint8_t cc = buf[0] & 0x0F;
                pkt.marker = (buf[1] >> 7) & 0x01;
                pkt.payload_type = buf[1] & 0x7F;
                pkt.seq = (static_cast<uint16_t>(buf[2]) << 8) | buf[3];
                pkt.timestamp = (static_cast<uint32_t>(buf[4]) << 24) |
                                (static_cast<uint32_t>(buf[5]) << 16) |
                                (static_cast<uint32_t>(buf[6]) << 8) |
                                static_cast<uint32_t>(buf[7]);

                size_t header_len = 12 + cc * 4;
                if (header_len >= len)
                    return false;

                pkt.payload.assign(buf.begin() + static_cast<ptrdiff_t>(header_len), buf.end());
                return true;
            }

            // ============================================================
            // H.264/H.265 NAL 处理
            // ============================================================
            void RKMPPPipeline::HandleRtpPacket(const RtpPacket &pkt)
            {
                if (pkt.payload.empty())
                    return;

                if (have_seq_ && pkt.seq != expected_seq_)
                {
                    fu_buffer_.clear();
                }
                expected_seq_ = pkt.seq + 1;
                have_seq_ = true;

                const uint8_t *data = pkt.payload.data();
                size_t size = pkt.payload.size();
                uint8_t nal_unit_type = data[0] & 0x1F;

                if (dec_type_ == MPP_VIDEO_CodingAVC)
                {
                    // H.264
                    if (nal_unit_type >= 1 && nal_unit_type <= 23)
                    {
                        EmitNal(data, size, pkt.timestamp);
                    }
                    else if (nal_unit_type == 28)
                    {
                        // FU-A
                        if (size < 2)
                            return;
                        uint8_t fu_header = data[1];
                        bool start = (fu_header >> 7) & 0x01;
                        bool end = (fu_header >> 6) & 0x01;
                        uint8_t actual_type = fu_header & 0x1F;
                        if (start)
                        {
                            fu_buffer_.clear();
                            fu_buffer_.push_back((data[0] & 0x60) | actual_type);
                            fu_buffer_.insert(fu_buffer_.end(), data + 2, data + size);
                        }
                        else if (!fu_buffer_.empty())
                        {
                            fu_buffer_.insert(fu_buffer_.end(), data + 2, data + size);
                            if (end)
                            {
                                EmitNal(fu_buffer_.data(), fu_buffer_.size(), pkt.timestamp);
                                fu_buffer_.clear();
                            }
                        }
                    }
                    else if (nal_unit_type == 24)
                    {
                        // STAP-A
                        size_t offset = 1;
                        while (offset + 2 <= size)
                        {
                            uint16_t nal_size = (static_cast<uint16_t>(data[offset]) << 8) | data[offset + 1];
                            offset += 2;
                            if (offset + nal_size > size)
                                break;
                            EmitNal(data + offset, nal_size, pkt.timestamp);
                            offset += nal_size;
                        }
                    }
                }
                else
                {
                    // H.265
                    if (nal_unit_type <= 47)
                    {
                        EmitNal(data, size, pkt.timestamp);
                    }
                    else if (nal_unit_type == 49)
                    {
                        if (size < 2)
                            return;
                        uint8_t fu_header = data[1];
                        bool start = (fu_header >> 7) & 0x01;
                        bool end = (fu_header >> 6) & 0x01;
                        uint8_t actual_type = fu_header & 0x3F;
                        if (start)
                        {
                            fu_buffer_.clear();
                            fu_buffer_.push_back(((data[0] & 0x81) | (actual_type << 1)));
                            fu_buffer_.insert(fu_buffer_.end(), data + 2, data + size);
                        }
                        else if (!fu_buffer_.empty())
                        {
                            fu_buffer_.insert(fu_buffer_.end(), data + 2, data + size);
                            if (end)
                            {
                                EmitNal(fu_buffer_.data(), fu_buffer_.size(), pkt.timestamp);
                                fu_buffer_.clear();
                            }
                        }
                    }
                    else if (nal_unit_type == 48)
                    {
                        size_t offset = 2;
                        while (offset + 2 <= size)
                        {
                            uint16_t nal_size = (static_cast<uint16_t>(data[offset]) << 8) | data[offset + 1];
                            offset += 2;
                            if (offset + nal_size > size)
                                break;
                            EmitNal(data + offset, nal_size, pkt.timestamp);
                            offset += nal_size;
                        }
                    }
                }
            }

            void RKMPPPipeline::EmitNal(const uint8_t *data, size_t size, uint32_t timestamp)
            {
                if (!data || size == 0)
                    return;

                nal_count_++;

                if (dec_type_ == MPP_VIDEO_CodingAVC)
                {
                    uint8_t nalu_type = data[0] & 0x1F;
                    if (nalu_type == 7)
                    { // SPS
                        sps_.assign(data, data + size);
                        has_sps_ = true;
                    }
                    if (nalu_type == 8)
                    { // PPS
                        pps_.assign(data, data + size);
                        has_pps_ = true;
                    }
                }

                // 添加起始码后喂给 MPP 解码器
                const uint8_t start_code[] = {0x00, 0x00, 0x00, 0x01};
                std::vector<uint8_t> annex_b(4 + size);
                std::memcpy(annex_b.data(), start_code, 4);
                std::memcpy(annex_b.data() + 4, data, size);

                int64_t pts_ns = static_cast<int64_t>(timestamp) * 1000000LL / 90000LL;
                DecodeNal(annex_b.data(), annex_b.size(), pts_ns);
            }

            bool RKMPPPipeline::ParseSpsPps(const std::string &sdp)
            {
                auto pos = sdp.find("sprop-parameter-sets=");
                if (pos == std::string::npos)
                    return false;

                auto end = sdp.find_first_of(";\r\n", pos);
                std::string val = sdp.substr(pos + 21, end - (pos + 21));
                auto comma = val.find(',');
                if (comma == std::string::npos)
                    return false;

                auto b64decode = [](const std::string &in) -> std::vector<uint8_t>
                {
                    static const std::string chars =
                        "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
                    std::vector<uint8_t> out;
                    int val = 0, bits = 0;
                    for (char c : in)
                    {
                        if (c == '=')
                            break;
                        auto p = chars.find(c);
                        if (p == std::string::npos)
                            continue;
                        val = (val << 6) | static_cast<int>(p);
                        bits += 6;
                        if (bits >= 8)
                        {
                            bits -= 8;
                            out.push_back(static_cast<uint8_t>((val >> bits) & 0xFF));
                        }
                    }
                    return out;
                };

                sps_ = b64decode(val.substr(0, comma));
                pps_ = b64decode(val.substr(comma + 1));
                has_sps_ = !sps_.empty();
                has_pps_ = !pps_.empty();
                RK_LOG_I("Parsed SPS=" << sps_.size() << " bytes, PPS=" << pps_.size() << " bytes");
                return has_sps_ && has_pps_;
            }

            void RKMPPPipeline::HandleDecodedFrame(void *frame)
            {
                MppFrame mpp_frame = static_cast<MppFrame>(frame);
                if (!mpp_frame)
                    return;

                // 检查 info_change（分辨率变化）
                if (mpp_frame_get_info_change(mpp_frame))
                {
                    RK_U32 new_width = mpp_frame_get_width(mpp_frame);
                    RK_U32 new_height = mpp_frame_get_height(mpp_frame);
                    RK_U32 new_hor_stride = mpp_frame_get_hor_stride(mpp_frame);
                    RK_U32 new_ver_stride = mpp_frame_get_ver_stride(mpp_frame);
                    size_t buf_size = mpp_frame_get_buf_size(mpp_frame);

                    RK_LOG_I("Info change: " << new_width << "x" << new_height
                                             << " stride=" << new_hor_stride << "x" << new_ver_stride
                                             << " buf_size=" << buf_size);

                    dec_width_ = new_width;
                    dec_height_ = new_height;
                    dec_hor_stride_ = new_hor_stride;
                    dec_ver_stride_ = new_ver_stride;

                    // 如果已经有 buffer group，清空并重新配置
                    if (dec_frame_group_)
                    {
                        mpp_buffer_group_clear(static_cast<MppBufferGroup>(dec_frame_group_));
                    }
                    else
                    {
                        // 创建新的 buffer group
                        MppBufferGroup fg = nullptr;
                        if (mpp_buffer_group_get_internal(&fg, MPP_BUFFER_TYPE_DRM) == MPP_OK)
                        {
                            dec_frame_group_ = fg;
                            static_cast<MppApi *>(mpp_dec_mpi_)->control(static_cast<MppCtx>(mpp_dec_ctx_), MPP_DEC_SET_EXT_BUF_GROUP, fg);
                        }
                    }

                    if (dec_frame_group_)
                    {
                        mpp_buffer_group_limit_config(
                            static_cast<MppBufferGroup>(dec_frame_group_),
                            buf_size, 24);
                    }

                    // 通知解码器 info change 已处理
                    static_cast<MppApi *>(mpp_dec_mpi_)->control(static_cast<MppCtx>(mpp_dec_ctx_), MPP_DEC_SET_INFO_CHANGE_READY, nullptr);
                    return;
                }

                // 检查 EOS
                if (mpp_frame_get_eos(mpp_frame))
                {
                    RK_LOG_I("Received EOS frame");
                    return;
                }

                // 检查丢弃帧
                if (mpp_frame_get_discard(mpp_frame) || mpp_frame_get_errinfo(mpp_frame))
                {
                    RK_LOG_W("Discard/errinfo frame");
                    mpp_frame_deinit(&mpp_frame);
                    return;
                }

                // 转换为 HwBuffer 并调用回调
                auto hw_buf = MppFrameToHwBuffer(mpp_frame);
                if (!hw_buf)
                    return;

                // 如果需要 RGA Resize
                if (rga_enabled_ && rga_dst_width_ > 0 && rga_dst_height_ > 0)
                {
                    HwBufferDesc src_desc = hw_buf->Desc();
                    HwBufferDesc dst_desc;
                    dst_desc.memory_type = HwBufferMemoryType::HostMemory;
                    dst_desc.width = static_cast<uint32_t>(rga_dst_width_);
                    dst_desc.height = static_cast<uint32_t>(rga_dst_height_);

                    if (RgaResize(src_desc, dst_desc))
                    {
                        // 用缩放后的帧替代原始帧
                        hw_buf = MakeHwBuffer(dst_desc);
                    }
                    else
                    {
                        RK_LOG_W("RGA resize failed, using original frame");
                    }
                }

                // 调用回调
                FrameCallback cb;
                {
                    std::lock_guard<std::mutex> lock(mutex_);
                    cb = frame_cb_;
                }
                if (cb)
                {
                    cb(hw_buf);
                }
            }

            // ---------- MppFrame 转 HwBuffer ----------
            HwBufferPtr RKMPPPipeline::MppFrameToHwBuffer(void *frame)
            {
                MppFrame mpp_frame = static_cast<MppFrame>(frame);
                if (!mpp_frame)
                    return nullptr;

                MppBuffer buffer = mpp_frame_get_buffer(mpp_frame);
                if (!buffer)
                    return nullptr;

                // 获取 DMA fd
                int dma_fd = mpp_buffer_get_fd(buffer);
                if (dma_fd < 0)
                    return nullptr;

                size_t buf_size = mpp_frame_get_buf_size(mpp_frame);

                // 构建描述符
                HwBufferDesc desc;
                desc.memory_type = HwBufferMemoryType::DMABuf;
                desc.dma_fd = dma_fd;
                desc.dma_buf_fd = -1; // 非 RGA 导入
                desc.size = buf_size;
                desc.width = mpp_frame_get_width(mpp_frame);
                desc.height = mpp_frame_get_height(mpp_frame);
                desc.pixel_format = 0; // MPP_FMT_YUV420SP
                desc.phys_addr = 0;
                desc.native_handle = mpp_frame; // 持有 MppFrame 引用

                // 创建 HwBuffer，设置释放回调释放 MppFrame
                auto hw_buf = std::make_shared<HwBuffer>(desc);
                hw_buf->SetReleaseCallback([](const HwBufferDesc &d)
                                           {
        if (d.native_handle) {
            MppFrame frame = static_cast<MppFrame>(d.native_handle);
            mpp_frame_deinit(&frame);
        } });

                return hw_buf;
            }

            // ---------- RGA Resize ----------
            bool RKMPPPipeline::RgaResize(const HwBufferDesc &src_desc,
                                          HwBufferDesc &dst_desc)
            {
                if (!rga_enabled_)
                    return false;

                int src_w = static_cast<int>(src_desc.width);
                int src_h = static_cast<int>(src_desc.height);
                int dst_w = static_cast<int>(dst_desc.width);
                int dst_h = static_cast<int>(dst_desc.height);

                if (src_w <= 0 || src_h <= 0 || dst_w <= 0 || dst_h <= 0)
                    return false;

                // 计算目标缓冲大小 (NV12: w*h + w*h/2)
                int dst_buf_size = dst_w * dst_h * 3 / 2;

                // 管理持久化缓冲，避免频繁分配
                if (!rga_dst_buf_ || rga_dst_buf_size_ < dst_buf_size)
                {
                    delete[] rga_dst_buf_;
                    rga_dst_buf_ = new uint8_t[static_cast<size_t>(dst_buf_size)];
                    rga_dst_buf_size_ = dst_buf_size;
                }

                // 导入源 DMA buffer fd 到 RGA
                rga_buffer_handle_t src_handle = importbuffer_fd(
                    src_desc.dma_fd,
                    src_w, src_h,
                    RK_FORMAT_YCbCr_420_SP);
                if (src_handle <= 0)
                {
                    RK_LOG_W("RGA import src buffer failed");
                    return false;
                }

                // 导入目标虚拟地址到 RGA
                rga_buffer_handle_t dst_handle = importbuffer_virtualaddr(
                    rga_dst_buf_,
                    dst_w, dst_h,
                    RK_FORMAT_YCbCr_420_SP);
                if (dst_handle <= 0)
                {
                    releasebuffer_handle(src_handle);
                    RK_LOG_W("RGA import dst buffer failed");
                    return false;
                }

                rga_buffer_t src_img = wrapbuffer_handle(
                    src_handle, src_w, src_h, RK_FORMAT_YCbCr_420_SP);
                rga_buffer_t dst_img = wrapbuffer_handle(
                    dst_handle, dst_w, dst_h, RK_FORMAT_YCbCr_420_SP);

// 检查参数 - RGA 2.2.0 兼容方式
// 旧版本 imcheck 宏不支持空参数，直接跳过检查
// 实际的参数有效性由 RGA 驱动在执行时验证
#if defined(IM_CHECK)
                IM_STATUS check_ret = imcheck(src_img, dst_img, {}, {});
                if (check_ret != IM_STATUS_NOERROR && check_ret != IM_STATUS_SUCCESS)
                {
                    RK_LOG_W("RGA imcheck failed: " << imStrError(check_ret));
                    releasebuffer_handle(src_handle);
                    releasebuffer_handle(dst_handle);
                    return false;
                }
#endif

                // 执行硬件缩放（同步模式，默认 sync=1）
                IM_STATUS ret = imresize(src_img, dst_img);
                if (ret != IM_STATUS_SUCCESS)
                {
                    RK_LOG_W("RGA imresize failed: " << imStrError(ret));
                    releasebuffer_handle(src_handle);
                    releasebuffer_handle(dst_handle);
                    return false;
                }

                // RGA 同步操作完成，释放 RGA 句柄
                releasebuffer_handle(src_handle);
                releasebuffer_handle(dst_handle);

                // 更新输出描述符
                // 使用 native_handle 持有持久化缓冲区的指针
                dst_desc.memory_type = HwBufferMemoryType::HostMemory;
                dst_desc.dma_fd = -1;
                dst_desc.dma_buf_fd = -1;
                dst_desc.size = static_cast<size_t>(dst_buf_size);
                dst_desc.width = static_cast<uint32_t>(dst_w);
                dst_desc.height = static_cast<uint32_t>(dst_h);
                dst_desc.pixel_format = src_desc.pixel_format;
                dst_desc.phys_addr = 0;
                dst_desc.native_handle = rga_dst_buf_;

                return true;
            }

            // ---------- MPP 编码器初始化 ----------
            bool RKMPPPipeline::InitEncoder(int width, int height, int coding_type)
            {
                MPP_RET ret = MPP_OK;

                // 如果已有编码器，先销毁
                DestroyEncoder();

                enc_width_ = width;
                enc_height_ = height;
                enc_hor_stride_ = MPP_ALIGN(width, 16);
                enc_ver_stride_ = MPP_ALIGN(height, 16);
                enc_codec_type_ = coding_type;

                // 1. 创建编码器上下文
                MppCtx ctx = nullptr;
                MppApi *mpi = nullptr;
                ret = mpp_create(&ctx, &mpi);
                if (ret != MPP_OK)
                {
                    RK_LOG_E("mpp_create for encoder failed: " << ret);
                    return false;
                }
                mpp_enc_ctx_ = ctx;
                mpp_enc_mpi_ = mpi;

                // 2. 设置超时
                MppPollType timeout = MPP_POLL_BLOCK;
                ret = mpi->control(ctx, MPP_SET_OUTPUT_TIMEOUT, &timeout);
                if (ret != MPP_OK)
                {
                    RK_LOG_W("MPP_SET_OUTPUT_TIMEOUT failed: " << ret);
                }

                // 3. 初始化编码器
                ret = mpp_init(ctx, MPP_CTX_ENC, static_cast<MppCodingType>(coding_type));
                if (ret != MPP_OK)
                {
                    RK_LOG_E("mpp_init ENC failed: " << ret);
                    mpp_destroy(ctx);
                    mpp_enc_ctx_ = nullptr;
                    mpp_enc_mpi_ = nullptr;
                    return false;
                }

                // 4. 创建编码配置
                MppEncCfg cfg = nullptr;
                ret = mpp_enc_cfg_init(&cfg);
                if (ret != MPP_OK)
                {
                    RK_LOG_E("mpp_enc_cfg_init failed: " << ret);
                    mpp_destroy(ctx);
                    mpp_enc_ctx_ = nullptr;
                    mpp_enc_mpi_ = nullptr;
                    return false;
                }
                enc_cfg_ = cfg;

                // 5. 获取默认配置
                ret = mpi->control(ctx, MPP_ENC_GET_CFG, cfg);
                if (ret != MPP_OK)
                {
                    RK_LOG_E("MPP_ENC_GET_CFG failed: " << ret);
                    mpp_enc_cfg_deinit(cfg);
                    mpp_destroy(ctx);
                    mpp_enc_ctx_ = nullptr;
                    mpp_enc_mpi_ = nullptr;
                    enc_cfg_ = nullptr;
                    return false;
                }

                // 6. 设置编码参数
                // — 输入图像参数
                mpp_enc_cfg_set_s32(cfg, "prep:width", enc_width_);
                mpp_enc_cfg_set_s32(cfg, "prep:height", enc_height_);
                mpp_enc_cfg_set_s32(cfg, "prep:hor_stride", enc_hor_stride_);
                mpp_enc_cfg_set_s32(cfg, "prep:ver_stride", enc_ver_stride_);
                mpp_enc_cfg_set_s32(cfg, "prep:format", MPP_FMT_YUV420SP);

                // — 码控参数
                int bps = enc_bitrate_;
                mpp_enc_cfg_set_s32(cfg, "rc:mode", MPP_ENC_RC_MODE_CBR);
                mpp_enc_cfg_set_s32(cfg, "rc:bps_target", bps);
                mpp_enc_cfg_set_s32(cfg, "rc:bps_max", bps * 12 / 10);
                mpp_enc_cfg_set_s32(cfg, "rc:bps_min", bps * 8 / 10);
                mpp_enc_cfg_set_s32(cfg, "rc:quality", 0); // 最佳质量

                // — 编码帧率
                mpp_enc_cfg_set_s32(cfg, "rc:fps_in_flex", 0);
                mpp_enc_cfg_set_s32(cfg, "rc:fps_in_num", enc_fps_);
                mpp_enc_cfg_set_s32(cfg, "rc:fps_in_den", 1);
                mpp_enc_cfg_set_s32(cfg, "rc:fps_out_flex", 0);
                mpp_enc_cfg_set_s32(cfg, "rc:fps_out_num", enc_fps_);
                mpp_enc_cfg_set_s32(cfg, "rc:fps_out_den", 1);

                // — GOP 参数
                mpp_enc_cfg_set_s32(cfg, "rc:gop", enc_gop_);
                mpp_enc_cfg_set_s32(cfg, "rc:gop_mode", 0); // normal P mode

                // — 编码器类型参数
                mpp_enc_cfg_set_s32(cfg, "codec:type", coding_type);
                if (coding_type == MPP_VIDEO_CodingAVC)
                {
                    // H.264 参数
                    mpp_enc_cfg_set_s32(cfg, "codec:h264:profile", 100); // High profile
                    mpp_enc_cfg_set_s32(cfg, "codec:h264:level", 40);    // Level 4.0
                    mpp_enc_cfg_set_s32(cfg, "codec:h264:trans_8x8", 1); // Enable 8x8 transform
                }
                else if (coding_type == MPP_VIDEO_CodingHEVC)
                {
                    // H.265 参数
                    mpp_enc_cfg_set_s32(cfg, "codec:hevc:profile", 0); // Main
                    mpp_enc_cfg_set_s32(cfg, "codec:hevc:tier", 0);    // Main tier
                    mpp_enc_cfg_set_s32(cfg, "codec:hevc:level", 120); // Level 4.0
                }

                // 7. 应用配置
                ret = mpi->control(ctx, MPP_ENC_SET_CFG, cfg);
                if (ret != MPP_OK)
                {
                    RK_LOG_E("MPP_ENC_SET_CFG failed: " << ret);
                    mpp_enc_cfg_deinit(cfg);
                    mpp_destroy(ctx);
                    mpp_enc_ctx_ = nullptr;
                    mpp_enc_mpi_ = nullptr;
                    enc_cfg_ = nullptr;
                    return false;
                }

                // 8. 创建编码器 buffer group
                ret = mpp_buffer_group_get_internal(
                    reinterpret_cast<MppBufferGroup *>(&enc_buf_grp_),
                    MPP_BUFFER_TYPE_DRM);
                if (ret != MPP_OK)
                {
                    RK_LOG_W("mpp_buffer_group_get_internal for encoder failed: " << ret);
                    enc_buf_grp_ = nullptr;
                }

                // 9. 获取 SPS/PPS 头，传递给调用方
                MppPacket hdr_packet = nullptr;
                mpp_packet_init_with_buffer(&hdr_packet, nullptr);
                if (hdr_packet)
                {
                    ret = mpi->control(ctx, MPP_ENC_GET_HDR_SYNC, hdr_packet);
                    if (ret == MPP_OK)
                    {
                        size_t hdr_len = mpp_packet_get_length(hdr_packet);
                        if (hdr_len > 0)
                        {
                            RK_LOG_I("Encoder header size=" << hdr_len << " bytes");
                        }
                    }
                    mpp_packet_deinit(&hdr_packet);
                }

                RK_LOG_I("MPP encoder initialized: "
                         << (coding_type == MPP_VIDEO_CodingAVC ? "H.264" : "H.265")
                         << " " << width << "x" << height
                         << " @" << enc_fps_ << "fps"
                         << " " << bps / 1000 << "kbps"
                         << " GOP=" << enc_gop_);
                return true;
            }

            // ---------- 销毁编码器 ----------
            void RKMPPPipeline::DestroyEncoder()
            {
                if (mpp_enc_ctx_)
                {
                    if (mpp_enc_mpi_)
                    {
                        static_cast<MppApi *>(mpp_enc_mpi_)->reset(static_cast<MppCtx>(mpp_enc_ctx_));
                    }
                    mpp_destroy(static_cast<MppCtx>(mpp_enc_ctx_));
                    mpp_enc_ctx_ = nullptr;
                    mpp_enc_mpi_ = nullptr;
                }
                if (enc_cfg_)
                {
                    mpp_enc_cfg_deinit(static_cast<MppEncCfg>(enc_cfg_));
                    enc_cfg_ = nullptr;
                }
                if (enc_buf_grp_)
                {
                    mpp_buffer_group_put(static_cast<MppBufferGroup>(enc_buf_grp_));
                    enc_buf_grp_ = nullptr;
                }
                encoder_initialized_ = false;
                RK_LOG_I("MPP encoder destroyed");
            }

#endif // AIVISION_WITH_RKMPP

        } // namespace rockchip
    } // namespace hal
} // namespace aivision
