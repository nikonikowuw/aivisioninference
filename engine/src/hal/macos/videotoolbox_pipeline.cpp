// macOS VideoToolbox 原生硬件解码 HAL 插件
// 零 FFmpeg 依赖：原生 socket RTSP/RTP + VTDecompressionSession + VTCompressionSession
#include "hal/macos/videotoolbox_pipeline.h"

#include <CoreMedia/CMFormatDescription.h>
#include <CoreMedia/CMSampleBuffer.h>
#include <CoreVideo/CVPixelBuffer.h>
#include <VideoToolbox/VTDecompressionSession.h>
#include <VideoToolbox/VTCompressionSession.h>

#include <sys/socket.h>
#include <netdb.h>
#include <unistd.h>

#include <chrono>
#include <cstring>
#include <iostream>
#include <sstream>

namespace aivision { namespace pipeline {

// ============================================================
// 工具函数
// ============================================================

static std::string Trim(const std::string& s) {
    size_t a = s.find_first_not_of(" \t\r\n");
    if (a == std::string::npos) return "";
    size_t b = s.find_last_not_of(" \t\r\n");
    return s.substr(a, b - a + 1);
}

static bool ParseRtspUrl(const std::string& url, std::string& host, int& port, std::string& path) {
    std::string u = url;
    if (u.substr(0, 7) == "rtsp://") u = u.substr(7);
    auto slash = u.find('/');
    std::string hostport = (slash == std::string::npos) ? u : u.substr(0, slash);
    path = (slash == std::string::npos) ? "/" : u.substr(slash);
    auto colon = hostport.find(':');
    if (colon != std::string::npos) {
        host = hostport.substr(0, colon);
        port = std::stoi(hostport.substr(colon + 1));
    } else {
        host = hostport;
        port = 554;
    }
    return !host.empty();
}

// ============================================================
// 构造/析构
// ============================================================

VideoToolboxPipeline::VideoToolboxPipeline() = default;
VideoToolboxPipeline::~VideoToolboxPipeline() { Stop(); }

HALCapabilities VideoToolboxPipeline::GetCapabilities() const {
    HALCapabilities caps{};
    caps.platform = "macos-videotoolbox-native";
    caps.decode_codecs = {VideoCodec::H264};
    caps.encode_codecs = {VideoCodec::H264};
    caps.max_streams = 16;
    caps.max_width = 4096;
    caps.max_height = 4096;
    caps.supports_zero_copy = true;
    caps.supports_hardware_encode = true;
    return caps;
}

bool VideoToolboxPipeline::Initialize(const std::string& /*config_json*/) {
    state_ = PipelineState::Idle;
    last_status_ = HALStatus::Success();
    return true;
}

// ============================================================
// RTSP 客户端 (原生 socket)
// ============================================================

bool VideoToolboxPipeline::RtspConnect(const std::string& url) {
    if (!ParseRtspUrl(url, host_, port_, path_)) {
        last_status_ = HALStatus::Error(HALStatusCode::InvalidConfig, "Invalid RTSP URL");
        return false;
    }
    addrinfo hints{};
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    addrinfo* result = nullptr;
    std::string port_str = std::to_string(port_);
    if (getaddrinfo(host_.c_str(), port_str.c_str(), &hints, &result) != 0) {
        last_status_ = HALStatus::Error(HALStatusCode::OpenStreamFailed, "DNS failed: " + host_);
        return false;
    }
    for (addrinfo* ai = result; ai; ai = ai->ai_next) {
        rtsp_socket_ = ::socket(ai->ai_family, ai->ai_socktype, ai->ai_protocol);
        if (rtsp_socket_ < 0) continue;
        struct timeval tv{5, 0};
        setsockopt(rtsp_socket_, SOL_SOCKET, SO_SNDTIMEO, &tv, sizeof(tv));
        if (::connect(rtsp_socket_, ai->ai_addr, ai->ai_addrlen) == 0) {
            struct timeval rv{10, 0};
            setsockopt(rtsp_socket_, SOL_SOCKET, SO_RCVTIMEO, &rv, sizeof(rv));
            break;
        }
        ::close(rtsp_socket_);
        rtsp_socket_ = -1;
    }
    freeaddrinfo(result);
    if (rtsp_socket_ < 0) {
        std::cerr << "[VideoToolbox] TCP connect failed: " << host_ << ":" << port_str << std::endl;
        last_status_ = HALStatus::Error(HALStatusCode::OpenStreamFailed, "TCP connect failed");
        return false;
    }
    std::cout << "[VideoToolbox] RTSP connected to " << host_ << ":" << port_ << std::endl;
    cseq_ = 1;
    session_.clear();
    return true;
}

void VideoToolboxPipeline::RtspDisconnect() {
    if (rtsp_socket_ >= 0) { ::close(rtsp_socket_); rtsp_socket_ = -1; }
}

bool VideoToolboxPipeline::RtspSendRequest(const std::string& req) {
    ssize_t sent = ::send(rtsp_socket_, req.data(), req.size(), 0);
    if (sent != static_cast<ssize_t>(req.size())) {
        std::cerr << "[VideoToolbox] RTSP send failed: sent=" << sent << " expected=" << req.size() << " errno=" << errno << std::endl;
        return false;
    }
    std::cout << "[VideoToolbox] >> " << req.substr(0, req.find("\r\n")) << std::endl;
    return true;
}

bool VideoToolboxPipeline::RtspReadResponse(int& status_code, std::string& response) {
    response.clear();
    char buf[4096];
    while (response.find("\r\n\r\n") == std::string::npos) {
        ssize_t n = ::recv(rtsp_socket_, buf, sizeof(buf), 0);
        if (n <= 0) {
            std::cerr << "[VideoToolbox] RTSP recv failed: n=" << n << " errno=" << errno << std::endl;
            return false;
        }
        response.append(buf, static_cast<size_t>(n));
        if (response.size() > 64 * 1024) return false;
    }
    std::istringstream stream(response);
    std::string version;
    stream >> version >> status_code;
    std::cout << "[VideoToolbox] << " << version << " " << status_code << std::endl;
    return status_code > 0;
}

bool VideoToolboxPipeline::RtspOptions() {
    std::string uri = "rtsp://" + host_ + ":" + std::to_string(port_) + path_;
    std::ostringstream req;
    req << "OPTIONS " << uri << " RTSP/1.0\r\n"
        << "CSeq: " << cseq_++ << "\r\nUser-Agent: aivision-engine\r\n\r\n";
    int status = 0; std::string resp;
    return RtspSendRequest(req.str()) && RtspReadResponse(status, resp) && status >= 200 && status < 300;
}

bool VideoToolboxPipeline::RtspDescribe(std::string& sdp) {
    std::string uri = "rtsp://" + host_ + ":" + std::to_string(port_) + path_;
    std::ostringstream req;
    req << "DESCRIBE " << uri << " RTSP/1.0\r\n"
        << "CSeq: " << cseq_++ << "\r\nUser-Agent: aivision-engine\r\nAccept: application/sdp\r\n\r\n";
    int status = 0; std::string resp;
    if (!RtspSendRequest(req.str()) || !RtspReadResponse(status, resp) || status < 200 || status >= 300)
        return false;
    auto pos = resp.find("\r\n\r\n");
    if (pos != std::string::npos) sdp = resp.substr(pos + 4);
    return !sdp.empty();
}

bool VideoToolboxPipeline::RtspSetup() {
    std::string uri = "rtsp://" + host_ + ":" + std::to_string(port_) + path_;
    std::ostringstream req;
    req << "SETUP " << uri << "/trackID=0 RTSP/1.0\r\n"
        << "CSeq: " << cseq_++ << "\r\nUser-Agent: aivision-engine\r\n"
        << "Transport: RTP/AVP/TCP;unicast;interleaved=0-1\r\n\r\n";
    int status = 0; std::string resp;
    if (!RtspSendRequest(req.str()) || !RtspReadResponse(status, resp) || status < 200 || status >= 300)
        return false;
    auto pos = resp.find("Session:");
    if (pos == std::string::npos) pos = resp.find("session:");
    if (pos != std::string::npos) {
        auto end = resp.find("\r\n", pos);
        std::string line = resp.substr(pos + 8, end - (pos + 8));
        auto semi = line.find(';');
        session_ = Trim(semi == std::string::npos ? line : line.substr(0, semi));
    }
    return true;
}

bool VideoToolboxPipeline::RtspPlay() {
    std::string uri = "rtsp://" + host_ + ":" + std::to_string(port_) + path_;
    std::ostringstream req;
    req << "PLAY " << uri << " RTSP/1.0\r\n"
        << "CSeq: " << cseq_++ << "\r\nUser-Agent: aivision-engine\r\n";
    if (!session_.empty()) req << "Session: " << session_ << "\r\n";
    req << "\r\n";
    int status = 0; std::string resp;
    return RtspSendRequest(req.str()) && RtspReadResponse(status, resp) && status >= 200 && status < 300;
}

bool VideoToolboxPipeline::RtspTeardown() {
    if (rtsp_socket_ < 0) return false;
    std::string uri = "rtsp://" + host_ + ":" + std::to_string(port_) + path_;
    std::ostringstream req;
    req << "TEARDOWN " << uri << " RTSP/1.0\r\n"
        << "CSeq: " << cseq_++ << "\r\nUser-Agent: aivision-engine\r\n";
    if (!session_.empty()) req << "Session: " << session_ << "\r\n";
    req << "\r\n";
    int status = 0; std::string resp;
    RtspSendRequest(req.str());
    RtspReadResponse(status, resp);
    return true;
}

// ============================================================
// RTP 解包
// ============================================================

bool VideoToolboxPipeline::RecvRtpPacket(RtpPacket& pkt) {
    uint8_t header[4];
    size_t got = 0;
    while (got < 4) {
        ssize_t n = ::recv(rtsp_socket_, header + got, 4 - got, 0);
        if (n <= 0) return false;
        got += static_cast<size_t>(n);
    }
    if (header[0] != 0x24) return false;
    uint16_t len = (static_cast<uint16_t>(header[2]) << 8) | header[3];
    if (len < 12 || len > 65535) return false;

    std::vector<uint8_t> buf(len);
    got = 0;
    while (got < len) {
        ssize_t n = ::recv(rtsp_socket_, buf.data() + got, len - got, 0);
        if (n <= 0) return false;
        got += static_cast<size_t>(n);
    }

    uint8_t version = (buf[0] >> 6) & 0x03;
    if (version != 2) return false;
    uint8_t cc = buf[0] & 0x0F;
    pkt.marker = (buf[1] >> 7) & 0x01;
    pkt.payload_type = buf[1] & 0x7F;
    pkt.seq = (static_cast<uint16_t>(buf[2]) << 8) | buf[3];
    pkt.timestamp = (static_cast<uint32_t>(buf[4]) << 24) | (static_cast<uint32_t>(buf[5]) << 16) |
                    (static_cast<uint32_t>(buf[6]) << 8) | static_cast<uint32_t>(buf[7]);
    size_t header_len = 12 + cc * 4;
    if (header_len >= len) return false;
    pkt.payload.assign(buf.begin() + static_cast<ptrdiff_t>(header_len), buf.end());
    return true;
}

// ============================================================
// H264 NAL 处理
// ============================================================

void VideoToolboxPipeline::EmitNal(const uint8_t* data, size_t size, uint32_t /*timestamp*/) {
    if (!data || size == 0) return;
    uint8_t nalu_type = data[0] & 0x1F;
    static int nal_count = 0;
    if (++nal_count <= 5 || nal_count % 100 == 0)
        std::cout << "[VideoToolbox] NAL type=" << (int)nalu_type << " size=" << size << " total=" << nal_count << std::endl;

    if (nalu_type == 7) {  // SPS
        sps_.assign(data, data + size);
        have_sps_ = true;
        if (have_sps_ && have_pps_ && !decoder_initialized_) {
            CreateFormatDescription();
            InitDecoder();
        }
        return;
    }
    if (nalu_type == 8) {  // PPS
        pps_.assign(data, data + size);
        have_pps_ = true;
        if (have_sps_ && have_pps_ && !decoder_initialized_) {
            CreateFormatDescription();
            InitDecoder();
        }
        return;
    }
    if (decoder_initialized_) FeedNalToDecoder(data, size);
}

void VideoToolboxPipeline::HandleRtpPacket(const RtpPacket& pkt) {
    if (pkt.payload.empty()) return;
    if (have_seq_ && pkt.seq != expected_seq_) {
        fu_buffer_.clear();  // 丢包，清空 FU-A 缓冲
    }
    expected_seq_ = pkt.seq + 1;
    have_seq_ = true;

    const uint8_t* data = pkt.payload.data();
    size_t size = pkt.payload.size();
    uint8_t nalu_type = data[0] & 0x1F;

    if (nalu_type >= 1 && nalu_type <= 23) {
        EmitNal(data, size, pkt.timestamp);
    } else if (nalu_type == 28) {  // FU-A
        if (size < 2) return;
        uint8_t fu_header = data[1];
        bool start = (fu_header >> 7) & 0x01;
        bool end = (fu_header >> 6) & 0x01;
        uint8_t actual_type = fu_header & 0x1F;
        if (start) {
            fu_buffer_.clear();
            fu_buffer_.push_back((data[0] & 0x60) | actual_type);
            fu_buffer_.insert(fu_buffer_.end(), data + 2, data + size);
        } else if (!fu_buffer_.empty()) {
            fu_buffer_.insert(fu_buffer_.end(), data + 2, data + size);
            if (end) {
                EmitNal(fu_buffer_.data(), fu_buffer_.size(), pkt.timestamp);
                fu_buffer_.clear();
            }
        }
    } else if (nalu_type == 24) {  // STAP-A
        size_t offset = 1;
        while (offset + 2 <= size) {
            uint16_t nal_size = (static_cast<uint16_t>(data[offset]) << 8) | data[offset + 1];
            offset += 2;
            if (offset + nal_size > size) break;
            EmitNal(data + offset, nal_size, pkt.timestamp);
            offset += nal_size;
        }
    }
}

// ============================================================
// SPS/PPS + FormatDescription
// ============================================================

bool VideoToolboxPipeline::ParseSpsPps(const std::string& sdp) {
    auto pos = sdp.find("sprop-parameter-sets=");
    if (pos == std::string::npos) return false;
    auto end = sdp.find_first_of(";\r\n", pos);
    std::string val = sdp.substr(pos + 21, end - (pos + 21));
    auto comma = val.find(',');
    if (comma == std::string::npos) return false;

    auto b64decode = [](const std::string& in) -> std::vector<uint8_t> {
        static const std::string chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
        std::vector<uint8_t> out;
        int val = 0, bits = 0;
        for (char c : in) {
            if (c == '=') break;
            auto p = chars.find(c);
            if (p == std::string::npos) continue;
            val = (val << 6) | static_cast<int>(p);
            bits += 6;
            if (bits >= 8) { bits -= 8; out.push_back(static_cast<uint8_t>((val >> bits) & 0xFF)); }
        }
        return out;
    };

    sps_ = b64decode(val.substr(0, comma));
    pps_ = b64decode(val.substr(comma + 1));
    have_sps_ = !sps_.empty();
    have_pps_ = !pps_.empty();
    return have_sps_ && have_pps_;
}

bool VideoToolboxPipeline::CreateFormatDescription() {
    if (!have_sps_ || !have_pps_) return false;
    if (format_desc_) { CFRelease(format_desc_); format_desc_ = nullptr; }
    const uint8_t* params[2] = {sps_.data(), pps_.data()};
    const size_t sizes[2] = {sps_.size(), pps_.size()};
    OSStatus status = CMVideoFormatDescriptionCreateFromH264ParameterSets(
        kCFAllocatorDefault, 2, params, sizes, 4, &format_desc_);
    if (status != noErr) {
        std::cerr << "[VideoToolbox] Format description failed: " << status << std::endl;
        return false;
    }
    return true;
}

// ============================================================
// VTDecompressionSession
// ============================================================

bool VideoToolboxPipeline::InitDecoder() {
    if (!format_desc_) return false;

    CFMutableDictionaryRef outDict = CFDictionaryCreateMutable(kCFAllocatorDefault, 2,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    int32_t pixelFormat = kCVPixelFormatType_420YpCbCr8BiPlanarVideoRange;
    CFNumberRef pfNum = CFNumberCreate(kCFAllocatorDefault, kCFNumberSInt32Type, &pixelFormat);
    CFDictionarySetValue(outDict, kCVPixelBufferPixelFormatTypeKey, pfNum);
    CFDictionaryRef emptyDict = CFDictionaryCreate(kCFAllocatorDefault, nullptr, nullptr, 0, nullptr, nullptr);
    CFDictionarySetValue(outDict, kCVPixelBufferIOSurfacePropertiesKey, emptyDict);
    CFRelease(pfNum);
    CFRelease(emptyDict);

    VTDecompressionOutputCallbackRecord cb;
    cb.decompressionOutputCallback = DecodeCallback;
    cb.decompressionOutputRefCon = this;

    OSStatus status = VTDecompressionSessionCreate(
        kCFAllocatorDefault, format_desc_, nullptr, outDict, &cb, &decode_session_);
    CFRelease(outDict);
    if (status != noErr) {
        std::cerr << "[VideoToolbox] Decompression session failed: " << status << std::endl;
        last_status_ = HALStatus::Error(HALStatusCode::DecodeFailed, "VTDecompressionSessionCreate failed");
        return false;
    }
    decoder_initialized_ = true;
    std::cout << "[VideoToolbox] Decompression session created" << std::endl;
    return true;
}

void VideoToolboxPipeline::DestroyDecoder() {
    if (decode_session_) { VTDecompressionSessionInvalidate(decode_session_); CFRelease(decode_session_); decode_session_ = nullptr; }
    if (format_desc_) { CFRelease(format_desc_); format_desc_ = nullptr; }
    decoder_initialized_ = false;
}

void VideoToolboxPipeline::DecodeCallback(void* refcon, void* /*source*/, OSStatus status,
                                          VTDecodeInfoFlags /*flags*/, CVImageBufferRef img,
                                          CMTime pts, CMTime /*duration*/) {
    if (status != noErr || !img) return;
    auto* self = static_cast<VideoToolboxPipeline*>(refcon);
    if (!self->frame_callback_) return;

    CVPixelBufferRetain(img);
    HwBufferDesc desc{};
    desc.memory_type = HwBufferMemoryType::CVPixelBuffer;
    desc.native_handle = img;
    desc.width = static_cast<uint32_t>(CVPixelBufferGetWidth(img));
    desc.height = static_cast<uint32_t>(CVPixelBufferGetHeight(img));
    auto buf = std::make_shared<HwBuffer>(desc);
    buf->SetReleaseCallback([](const HwBufferDesc& d) {
        if (d.native_handle) CVPixelBufferRelease(static_cast<CVPixelBufferRef>(d.native_handle));
    });
    self->frame_callback_(buf);
}

bool VideoToolboxPipeline::FeedNalToDecoder(const uint8_t* data, size_t size) {
    if (!decoder_initialized_ || !data || size == 0) return false;
    const uint8_t start_code[] = {0x00, 0x00, 0x00, 0x01};

    CMBlockBufferRef block = nullptr;
    OSStatus s = CMBlockBufferCreateWithMemoryBlock(kCFAllocatorDefault, nullptr, size + 4,
        kCFAllocatorDefault, nullptr, 0, size + 4, 0, &block);
    if (s != noErr) return false;
    CMBlockBufferReplaceDataBytes(start_code, block, 0, 4);
    CMBlockBufferReplaceDataBytes(data, block, 4, size);

    CMSampleBufferRef sample = nullptr;
    const size_t sample_size = size + 4;
    s = CMSampleBufferCreateReady(kCFAllocatorDefault, block, format_desc_, 1, 0, nullptr, 1, &sample_size, &sample);
    CFRelease(block);
    if (s != noErr) return false;

    VTDecodeFrameFlags flags = kVTDecodeFrame_EnableAsynchronousDecompression;
    VTDecodeInfoFlags info = 0;
    s = VTDecompressionSessionDecodeFrame(decode_session_, sample, flags, nullptr, &info);
    CFRelease(sample);
    return s == noErr;
}

// ============================================================
// VTCompressionSession
// ============================================================

static void EncodeOutputCallback(void* refcon, void* /*src*/, OSStatus status,
                                  VTEncodeInfoFlags /*flags*/, CMSampleBufferRef sb) {
    if (status != noErr || !sb) return;
    auto* output = static_cast<std::vector<uint8_t>*>(refcon);
    CMBlockBufferRef block = CMSampleBufferGetDataBuffer(sb);
    if (!block) return;
    char* ptr = nullptr;
    size_t total = 0, at_offset = 0;
    if (CMBlockBufferGetDataPointer(block, 0, &at_offset, &total, &ptr) != noErr) return;
    if (ptr && total > 0) output->insert(output->end(), reinterpret_cast<uint8_t*>(ptr), reinterpret_cast<uint8_t*>(ptr) + total);
}

bool VideoToolboxPipeline::EncodeInit(const std::string& /*config_json*/) { return true; }

bool VideoToolboxPipeline::InitEncoder(int width, int height) {
    std::lock_guard<std::mutex> lock(enc_mu_);
    if (encoder_initialized_) return true;
    OSStatus s = VTCompressionSessionCreate(kCFAllocatorDefault, width, height,
        kCMVideoCodecType_H264, nullptr, nullptr, nullptr, EncodeOutputCallback, this, &encode_session_);
    if (s != noErr) return false;
    VTSessionSetProperty(encode_session_, kVTCompressionPropertyKey_RealTime, kCFBooleanTrue);
    int32_t br = width * height * 3;
    CFNumberRef brNum = CFNumberCreate(kCFAllocatorDefault, kCFNumberSInt32Type, &br);
    VTSessionSetProperty(encode_session_, kVTCompressionPropertyKey_AverageBitRate, brNum);
    CFRelease(brNum);
    VTSessionSetProperty(encode_session_, kVTCompressionPropertyKey_ProfileLevel, kVTProfileLevel_H264_Main_AutoLevel);
    VTCompressionSessionPrepareToEncodeFrames(encode_session_);
    encoder_initialized_ = true;
    std::cout << "[VideoToolbox] Encoder: " << width << "x" << height << std::endl;
    return true;
}

void VideoToolboxPipeline::DestroyEncoder() {
    std::lock_guard<std::mutex> lock(enc_mu_);
    if (encode_session_) { VTCompressionSessionInvalidate(encode_session_); CFRelease(encode_session_); encode_session_ = nullptr; }
    encoder_initialized_ = false;
}

bool VideoToolboxPipeline::EncodeFrame(HwBufferPtr frame, uint8_t* data, size_t size, size_t& out_size) {
    EncodedPacketDesc desc;
    return EncodeFrameEx(std::move(frame), data, size, out_size, desc);
}

bool VideoToolboxPipeline::EncodeFrameEx(HwBufferPtr frame, uint8_t* data, size_t size, size_t& out_size, EncodedPacketDesc& desc) {
    if (!frame || frame->Desc().memory_type != HwBufferMemoryType::CVPixelBuffer || !frame->Desc().native_handle)
        return false;

    CVPixelBufferRef cvpb = static_cast<CVPixelBufferRef>(const_cast<void*>(frame->Desc().native_handle));
    int w = static_cast<int>(CVPixelBufferGetWidth(cvpb));
    int h = static_cast<int>(CVPixelBufferGetHeight(cvpb));
    if (!encoder_initialized_ && !InitEncoder(w, h)) return false;

    std::vector<uint8_t> output;
    encode_output_ = &output;

    CMTime pts = CMTimeMake(
        std::chrono::duration_cast<std::chrono::microseconds>(std::chrono::steady_clock::now().time_since_epoch()).count(), 1000000);
    VTEncodeInfoFlags flags = 0;
    OSStatus s = VTCompressionSessionEncodeFrame(encode_session_, cvpb, pts, kCMTimeInvalid, nullptr, nullptr, &flags);
    if (s == noErr) s = VTCompressionSessionCompleteFrames(encode_session_, kCMTimeInvalid);
    encode_output_ = nullptr;

    if (s != noErr || output.empty()) return false;

    out_size = std::min(size, output.size());
    std::memcpy(data, output.data(), out_size);
    desc.codec = VideoCodec::H264;
    desc.is_key_frame = true;  // VTCompressionSession 不方便判断，保守标记
    return true;
}

void VideoToolboxPipeline::EncodeDestroy() { DestroyEncoder(); }

// ============================================================
// Start / Stop / PullLoop
// ============================================================

bool VideoToolboxPipeline::Start(const std::string& url) {
    if (running_.exchange(true)) return true;
    state_ = PipelineState::Connecting;
    if (state_callback_) state_callback_(state_);

    if (!RtspConnect(url)) { std::cerr << "[VideoToolbox] RtspConnect failed" << std::endl; running_ = false; state_ = PipelineState::Error; return false; }
    if (!RtspOptions()) { std::cerr << "[VideoToolbox] RtspOptions failed" << std::endl; RtspDisconnect(); running_ = false; last_status_ = HALStatus::Error(HALStatusCode::OpenStreamFailed, "OPTIONS failed"); return false; }

    std::string sdp;
    if (!RtspDescribe(sdp)) { std::cerr << "[VideoToolbox] RtspDescribe failed" << std::endl; RtspDisconnect(); running_ = false; last_status_ = HALStatus::Error(HALStatusCode::OpenStreamFailed, "DESCRIBE failed"); return false; }
    std::cout << "[VideoToolbox] SDP: " << sdp.substr(0, 200) << std::endl;
    ParseSpsPps(sdp);

    if (!RtspSetup()) { std::cerr << "[VideoToolbox] RtspSetup failed" << std::endl; RtspDisconnect(); running_ = false; last_status_ = HALStatus::Error(HALStatusCode::OpenStreamFailed, "SETUP failed"); return false; }
    if (!RtspPlay()) { std::cerr << "[VideoToolbox] RtspPlay failed" << std::endl; RtspDisconnect(); running_ = false; last_status_ = HALStatus::Error(HALStatusCode::OpenStreamFailed, "PLAY failed"); return false; }

    state_ = PipelineState::Streaming;
    if (state_callback_) state_callback_(state_);
    std::cout << "[VideoToolbox] RTSP playing: " << url << std::endl;

    pull_thread_ = std::make_unique<std::thread>(&VideoToolboxPipeline::PullLoop, this);
    return true;
}

void VideoToolboxPipeline::PullLoop() {
    RtpPacket pkt;
    int pkt_count = 0;
    while (running_ && !paused_) {
        if (!RecvRtpPacket(pkt)) {
            if (running_) { std::cerr << "[VideoToolbox] RecvRtp failed" << std::endl; std::this_thread::sleep_for(std::chrono::seconds(1)); continue; }
            break;
        }
        if (++pkt_count <= 3 || pkt_count % 200 == 0)
            std::cout << "[VideoToolbox] RTP pkt seq=" << pkt.seq << " type=" << (int)pkt.payload_type << " size=" << pkt.payload.size() << " count=" << pkt_count << std::endl;
        HandleRtpPacket(pkt);
    }
}

void VideoToolboxPipeline::Stop() {
    running_ = false;
    if (pull_thread_ && pull_thread_->joinable()) pull_thread_->join();
    pull_thread_.reset();
    RtspTeardown();
    RtspDisconnect();
    DestroyDecoder();
    DestroyEncoder();
    state_ = PipelineState::Stopped;
}

void VideoToolboxPipeline::Pause() { paused_ = true; }
void VideoToolboxPipeline::Resume() { paused_ = false; }

}} // namespace aivision::pipeline

// HAL 插件导出
extern "C" {
aivision::pipeline::IMediaPipeline* CreatePipeline() {
    return new aivision::pipeline::VideoToolboxPipeline();
}
void DestroyPipeline(aivision::pipeline::IMediaPipeline* p) { delete p; }
}
