// macOS VideoToolbox 原生硬件解码 HAL 插件
// 零 FFmpeg 依赖：原生 socket RTSP/RTP + VTDecompressionSession + VTCompressionSession
#include "hal/macos/videotoolbox_pipeline.h"

#include <nlohmann/json.hpp>

#include <CoreMedia/CMFormatDescription.h>
#include <CoreMedia/CMSampleBuffer.h>
#include <CoreVideo/CVPixelBuffer.h>
#include <VideoToolbox/VTDecompressionSession.h>
#include <VideoToolbox/VTCompressionSession.h>

#include <openssl/md5.h>

#include <sys/socket.h>
#include <netdb.h>
#include <unistd.h>

#include "string_utils.h"
#include "logger/logger.h"
#include <algorithm>
#include <chrono>
#include <cstring>
#include <sstream>

namespace aivision { namespace pipeline {



static std::string Base64Encode(const std::string& in) {
    static const char* table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=";
    std::string out;
    out.reserve(((in.size() + 2) / 3) * 4);
    uint8_t buf[3];
    for (size_t i = 0; i < in.size(); i += 3) {
        size_t n = std::min(size_t{3}, in.size() - i);
        for (size_t j = 0; j < n; j++) buf[j] = static_cast<uint8_t>(in[i + j]);
        out += table[buf[0] >> 2];
        out += table[((buf[0] & 0x03) << 4) | (buf[1] >> 4)];
        out += (n > 1) ? table[((buf[1] & 0x0F) << 2) | (buf[2] >> 6)] : '=';
        out += (n > 2) ? table[buf[2] & 0x3F] : '=';
    }
    return out;
}

static bool ParseRtspUrl(const std::string& url, std::string& host, int& port,
                          std::string& path, std::string& username, std::string& password) {
    std::string u = url;
    if (u.substr(0, 7) == "rtsp://") u = u.substr(7);
    auto slash = u.find('/');
    std::string userhost = (slash == std::string::npos) ? u : u.substr(0, slash);
    path = (slash == std::string::npos) ? "/" : u.substr(slash);
    // 支持 user:password@host:port 格式
    auto at_pos = userhost.rfind('@');
    if (at_pos != std::string::npos) {
        username = userhost.substr(0, at_pos);
        auto colon = username.find(':');
        if (colon != std::string::npos) {
            password = username.substr(colon + 1);
            username = username.substr(0, colon);
        }
        userhost = userhost.substr(at_pos + 1);
    }
    auto colon = userhost.rfind(':');
    if (colon != std::string::npos && colon > 0) {
        host = userhost.substr(0, colon);
        try {
            port = std::stoi(userhost.substr(colon + 1));
        } catch (...) {
            port = 554;
            host = userhost;
        }
    } else {
        host = userhost;
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
    caps.decode_codecs = {VideoCodec::H264, VideoCodec::H265};
    caps.encode_codecs = {VideoCodec::H264, VideoCodec::H265};
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
    username_.clear();
    password_.clear();
    if (!ParseRtspUrl(url, host_, port_, path_, username_, password_)) {
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
        LOG_ERROR("[VideoToolbox] TCP connect failed: {}:{}", host_, port_str);
        last_status_ = HALStatus::Error(HALStatusCode::OpenStreamFailed, "TCP connect failed");
        return false;
    }
    LOG_INFO("[VideoToolbox] RTSP connected to {}:{}", host_, port_);
    cseq_ = 1;
    session_.clear();
    return true;
}

void VideoToolboxPipeline::RtspDisconnect() {
    rx_buffer_.clear();
    if (rtsp_socket_ >= 0) { ::close(rtsp_socket_); rtsp_socket_ = -1; }
}

bool VideoToolboxPipeline::RtspSendRequest(const std::string& req) {
    ssize_t sent = ::send(rtsp_socket_, req.data(), req.size(), 0);
    if (sent != static_cast<ssize_t>(req.size())) {
        LOG_ERROR("[VideoToolbox] RTSP send failed: sent={} expected={} errno={}", sent, req.size(), errno);
        return false;
    }
    // 打印请求行 + Authorization 状态方便调试
    auto first_crlf = req.find("\r\n");
    if (first_crlf != std::string::npos) {
        LOG_INFO("[VideoToolbox] >> {}", req.substr(0, first_crlf));
        auto sess_pos = req.find("Session:");
        if (sess_pos != std::string::npos) {
            auto sess_end = req.find("\r\n", sess_pos);
            LOG_INFO("[VideoToolbox] >> {}", req.substr(sess_pos, sess_end - sess_pos));
        }
        auto auth_pos = req.find("Authorization:");
        if (auth_pos != std::string::npos) {
            auto auth_end = req.find("\r\n", auth_pos);
            std::string auth_line = req.substr(auth_pos, auth_end - auth_pos);
            if (auth_line.find("Digest") != std::string::npos)
                LOG_INFO("[VideoToolbox] >> Authorization: Digest ****");
            else
                LOG_INFO("[VideoToolbox] >> Authorization: Basic ****");
        }
    }
    return true;
}

bool VideoToolboxPipeline::RtspReadResponse(int& status_code, std::string& response) {
    response.clear();
    char buf[4096];
    while (true) {
        auto pos = response.find("\r\n\r\n");
        if (pos != std::string::npos) {
            size_t header_end = pos + 4;
            if (header_end < response.size()) {
                const char* leftover = response.data() + header_end;
                size_t leftover_len = response.size() - header_end;
                rx_buffer_.insert(rx_buffer_.end(), leftover, leftover + leftover_len);
                response.resize(header_end);
            }
            break;
        }

        if (!rx_buffer_.empty()) {
            response.append(reinterpret_cast<const char*>(rx_buffer_.data()), rx_buffer_.size());
            rx_buffer_.clear();
            continue;
        }

        ssize_t n = ::recv(rtsp_socket_, buf, sizeof(buf), 0);
        if (n <= 0) {
            LOG_ERROR("[VideoToolbox] RTSP recv failed: n={} errno={}", n, errno);
            return false;
        }
        response.append(buf, static_cast<size_t>(n));
        if (response.size() > 64 * 1024) return false;
    }
    std::istringstream stream(response);
    std::string version;
    stream >> version >> status_code;
    LOG_INFO("[VideoToolbox] << {} {}", version, status_code);
    return status_code > 0;
}

static std::string ExtractQuoted(const std::string& haystack, const std::string& key) {
    auto pos = haystack.find(key + "=\"");
    if (pos == std::string::npos) return "";
    pos += key.size() + 2;
    auto end = haystack.find('"', pos);
    if (end == std::string::npos) return "";
    return haystack.substr(pos, end - pos);
}

static std::string MD5Hex(const std::string& in) {
    unsigned char digest[MD5_DIGEST_LENGTH];
    MD5(reinterpret_cast<const unsigned char*>(in.data()), in.size(), digest);
    char hex[MD5_DIGEST_LENGTH * 2 + 1];
    for (int i = 0; i < MD5_DIGEST_LENGTH; i++)
        snprintf(hex + i * 2, 3, "%02x", digest[i]);
    return hex;
}

// 根据 401 响应生成 Digest auth header
std::string VideoToolboxPipeline::RtspDigestHeader(const std::string& method,
                                                     const std::string& uri) const {
    // HA1 = MD5(user:realm:pass)
    std::string ha1 = MD5Hex(username_ + ":" + digest_realm_ + ":" + password_);
    // HA2 = MD5(method:uri)
    std::string ha2 = MD5Hex(method + ":" + uri);
    // response = MD5(HA1:nonce:HA2)
    std::string resp_hash = MD5Hex(ha1 + ":" + digest_nonce_ + ":" + ha2);

    std::ostringstream hdr;
    hdr << "Authorization: Digest username=\"" << username_
        << "\", realm=\"" << digest_realm_
        << "\", nonce=\"" << digest_nonce_
        << "\", uri=\"" << uri
        << "\", algorithm=MD5, response=\"" << resp_hash << "\"\r\n";
    return hdr.str();
}

// 从 WWW-Authenticate 响应头提取 Digest 参数
bool VideoToolboxPipeline::RtspParseAuthChallenge(const std::string& resp) {
    auto www_pos = resp.find("WWW-Authenticate:");
    if (www_pos == std::string::npos) www_pos = resp.find("www-authenticate:");
    if (www_pos == std::string::npos) return false;
    auto eol = resp.find("\r\n", www_pos);
    std::string www_line = resp.substr(www_pos, eol - www_pos);

    if (www_line.find("Digest") != std::string::npos ||
        www_line.find("digest") != std::string::npos) {
        digest_realm_ = ExtractQuoted(www_line, "realm");
        digest_nonce_ = ExtractQuoted(www_line, "nonce");
        return !digest_realm_.empty() && !digest_nonce_.empty();
    }
    // Basic 不需要记录参数
    return www_line.find("Basic") != std::string::npos ||
           www_line.find("basic") != std::string::npos;
}

// 发送 RTSP 命令，若收到 401 则自动用 Digest 或 Basic auth 重试一次
bool VideoToolboxPipeline::RtspSendCommand(const std::string& method, std::string& resp,
                                            const std::string& track,
                                            bool include_transport) {
    std::string uri = "rtsp://" + host_ + ":" + std::to_string(port_) + path_;
    std::string full_uri = uri + (track.empty() ? "" : "/" + track);

    auto build_request = [&]() {
        std::ostringstream req;
        req << method << " " << full_uri << " RTSP/1.0\r\n"
            << "CSeq: " << cseq_++ << "\r\nUser-Agent: aivision-engine\r\n";
        if (!username_.empty()) {
            if (!digest_realm_.empty()) {
                req << RtspDigestHeader(method, full_uri);
            } else {
                req << "Authorization: Basic " << Base64Encode(username_ + ":" + password_) << "\r\n";
            }
        }
        if (method == "DESCRIBE")
            req << "Accept: application/sdp\r\n";
        if (method == "SETUP" && include_transport) {
            req << "Transport: RTP/AVP/TCP;unicast;interleaved="
                << interleaved_start_ << "-" << (interleaved_start_ + 1) << "\r\n";
        }
        if (!session_.empty() && (method == "SETUP" || method == "PLAY" || method == "TEARDOWN"))
            req << "Session: " << session_ << "\r\n";
        if (method == "PLAY") {
            // 部分 RTSP 服务器要求 Range 头才能进入播放状态
            req << "Range: npt=0.000-\r\n";
        }
        req << "\r\n";
        return req.str();
    };

    int status = 0;
    if (!RtspSendRequest(build_request()) || !RtspReadResponse(status, resp))
        return false;

    // 收到 401 且有凭证时，尝试 Digest 或 Basic auth
    if (status == 401 && !username_.empty()) {
        RtspParseAuthChallenge(resp);
        LOG_INFO("[VideoToolbox] Server auth: realm=\"{}\" nonce=\"{}\"", digest_realm_, digest_nonce_);

        bool ok = RtspSendRequest(build_request()) && RtspReadResponse(status, resp);
        return ok && status >= 200 && status < 300;
    }

    return status >= 200 && status < 300;
}

bool VideoToolboxPipeline::RtspOptions() {
    std::string resp;
    return RtspSendCommand("OPTIONS", resp);
}

bool VideoToolboxPipeline::RtspDescribe(std::string& sdp) {
    std::string resp;
    if (!RtspSendCommand("DESCRIBE", resp))
        return false;
    // RtspReadResponse 会把 \r\n\r\n 之后的正文（即 SDP）剥离到 rx_buffer_ 中，
    // 为后续 PullLoop 的 RTP interleaved 帧做准备。但 DESCRIBE 的响应正文必须解析，
    // 因此优先从 rx_buffer_ 中读取 SDP。
    if (!rx_buffer_.empty()) {
        sdp.assign(reinterpret_cast<const char*>(rx_buffer_.data()), rx_buffer_.size());
        rx_buffer_.clear();
    }
    if (sdp.empty()) {
        auto pos = resp.find("\r\n\r\n");
        if (pos != std::string::npos) sdp = resp.substr(pos + 4);
    }
    return !sdp.empty();
}

bool VideoToolboxPipeline::RtspSetup() {
    if (sdp_tracks_.empty()) {
        LOG_ERROR("[VideoToolbox] No SDP tracks to SETUP");
        return false;
    }
    for (size_t i = 0; i < sdp_tracks_.size(); i++) {
        const auto& track = sdp_tracks_[i];
        // 只 SETUP video 类型的 track（部分相机不支持同时 SETUP audio+video）
        if (track.first != "video") {
            LOG_INFO("[VideoToolbox] Skipping non-video track: {}={}", track.first, track.second);
            interleaved_start_ += 2;
            continue;
        }
        std::string resp;
        if (!RtspSendCommand("SETUP", resp, track.second, true)) {
            LOG_ERROR("[VideoToolbox] SETUP failed for track {} ({})", i, track.second);
            return false;
        }
        auto pos = resp.find("Session:");
        if (pos == std::string::npos) pos = resp.find("session:");
        if (pos != std::string::npos) {
            auto end = resp.find("\r\n", pos);
            std::string line = resp.substr(pos + 8, end - (pos + 8));
            auto semi = line.find(';');
            session_ = Trim(semi == std::string::npos ? line : line.substr(0, semi));
        }
        // 解析 interleaved 通道
        auto intr = resp.find("interleaved=");
        if (intr != std::string::npos) {
            auto val_start = resp.find_first_of("0123456789", intr);
            if (val_start != std::string::npos) {
                try {
                    interleaved_start_ = std::stoi(resp.substr(val_start)) + 2;
                } catch (...) {
                    interleaved_start_ += 2;
                }
            }
        } else {
            interleaved_start_ += 2;
        }
    }
    return !session_.empty();
}

bool VideoToolboxPipeline::RtspPlay() {
    std::string resp;
    // PLAY 使用 video track 的 URI（部分相机要求与 SETUP 的 URI 一致）
    std::string video_track;
    for (const auto& t : sdp_tracks_) {
        if (t.first == "video") { video_track = t.second; break; }
    }
    return RtspSendCommand("PLAY", resp, video_track);
}

bool VideoToolboxPipeline::RtspTeardown() {
    if (rtsp_socket_ < 0) return false;
    std::string resp;
    RtspSendCommand("TEARDOWN", resp);
    return true;
}

// ============================================================
// RTP 解包
// ============================================================

bool VideoToolboxPipeline::ReadExact(uint8_t* dst, size_t count) {
    size_t got = 0;
    if (!rx_buffer_.empty()) {
        size_t available = std::min(count, rx_buffer_.size());
        std::memcpy(dst, rx_buffer_.data(), available);
        rx_buffer_.erase(rx_buffer_.begin(), rx_buffer_.begin() + static_cast<ptrdiff_t>(available));
        got += available;
    }
    while (got < count) {
        ssize_t n = ::recv(rtsp_socket_, dst + got, count - got, 0);
        if (n <= 0) return false;
        got += static_cast<size_t>(n);
    }
    return true;
}

bool VideoToolboxPipeline::RecvRtpPacket(RtpPacket& pkt) {
    uint8_t header[4];
    if (!ReadExact(header, 4)) return false;

    // 若首字节不是 0x24 ($)，在字节流中同步搜索 0x24
    if (header[0] != 0x24) {
        LOG_WARN("[VideoToolbox] Misaligned RTP marker: 0x{:02x}, searching for 0x24 ($)...", header[0]);
        uint8_t byte = 0;
        int max_search = 65536;
        bool found = false;
        std::vector<uint8_t> search_buf(header + 1, header + 4);
        while (max_search-- > 0) {
            if (!search_buf.empty()) {
                byte = search_buf.front();
                search_buf.erase(search_buf.begin());
            } else {
                if (!ReadExact(&byte, 1)) return false;
            }
            if (byte == 0x24) {
                found = true;
                break;
            }
        }
        if (!found) {
            LOG_ERROR("[VideoToolbox] Failed to resynchronize RTP stream ($ marker not found)");
            return false;
        }
        header[0] = 0x24;
        if (!ReadExact(header + 1, 3)) return false;
    }

    uint16_t len = (static_cast<uint16_t>(header[2]) << 8) | header[3];
    if (len < 12 || len > 65535) return false;

    std::vector<uint8_t> buf(len);
    if (!ReadExact(buf.data(), len)) return false;

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
// H264/H265 NAL 处理
// ============================================================

void VideoToolboxPipeline::EmitNal(const uint8_t* data, size_t size, uint32_t /*timestamp*/) {
    if (!data || size == 0) return;
    if (++nal_count_ <= 5 || nal_count_ % 100 == 0)
        LOG_INFO("[VideoToolbox] NAL size={} total={}", size, nal_count_);

    if (is_hevc_) {
        // HEVC: 2-byte NAL header, type in bits 1-6
        uint8_t nalu_type = (data[0] >> 1) & 0x3F;
        if (nalu_type == 32) {  // VPS
            vps_.assign(data, data + size);
            have_vps_ = true;
        } else if (nalu_type == 33) {  // SPS
            sps_.assign(data, data + size);
            have_sps_ = true;
        } else if (nalu_type == 34) {  // PPS
            pps_.assign(data, data + size);
            have_pps_ = true;
        }
        if (have_vps_ && have_sps_ && have_pps_ && !decoder_initialized_) {
            CreateFormatDescription();
            InitDecoder();
        }
        if (decoder_initialized_ && (nalu_type < 32 || nalu_type > 34))
            FeedNalToDecoder(data, size);
    } else {
        // H264: 1-byte NAL header
        uint8_t nalu_type = data[0] & 0x1F;
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

    if (is_hevc_) {
        // HEVC RTP 载荷格式 (RFC 7798)
        if (size < 2) return;
        uint8_t nal_type = (data[0] >> 1) & 0x3F;

        if (nal_type >= 0 && nal_type <= 47) {
            // 单 NAL 单元
            EmitNal(data, size, pkt.timestamp);
        } else if (nal_type == 49) {  // FU (Fragmentation Unit)
            if (size < 3) return;
            bool start = (data[2] >> 7) & 0x01;
            bool end = (data[2] >> 6) & 0x01;
            uint8_t fu_type = data[2] & 0x3F;
            if (start) {
                fu_buffer_.clear();
                // 2-byte HEVC NAL header with correct type
                fu_buffer_.push_back((data[0] & 0x81) | (fu_type << 1));
                fu_buffer_.push_back(data[1]);
                fu_buffer_.insert(fu_buffer_.end(), data + 3, data + size);
            } else if (!fu_buffer_.empty()) {
                fu_buffer_.insert(fu_buffer_.end(), data + 3, data + size);
                if (end) {
                    EmitNal(fu_buffer_.data(), fu_buffer_.size(), pkt.timestamp);
                    fu_buffer_.clear();
                }
            }
        } else if (nal_type == 50) {  // AP (Aggregation Packet)
            size_t offset = 2;
            while (offset + 2 < size) {
                uint16_t nal_size = (static_cast<uint16_t>(data[offset]) << 8) | data[offset + 1];
                offset += 2;
                if (offset + nal_size > size) break;
                EmitNal(data + offset, nal_size, pkt.timestamp);
                offset += nal_size;
            }
        }
    } else {
        // H264 RTP 载荷格式
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
}

// ============================================================
// SPS/PPS/VPS + FormatDescription
// ============================================================

auto B64Decode = [](const std::string& in) -> std::vector<uint8_t> {
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

bool VideoToolboxPipeline::ParseCodecParams(const std::string& sdp) {
    // 检测是否为 HEVC
    is_hevc_ = sdp.find("H265") != std::string::npos ||
               sdp.find("HEVC") != std::string::npos ||
               sdp.find("h265") != std::string::npos ||
               sdp.find("hevc") != std::string::npos;

    if (is_hevc_) {
        LOG_INFO("[VideoToolbox] Detected HEVC/H265 stream");
        auto extract_param = [&](const std::string& key, std::vector<uint8_t>& out, bool& flag) {
            auto p = sdp.find(key + "=");
            if (p == std::string::npos) return;
            auto end = sdp.find_first_of(";\r\n", p);
            std::string val = sdp.substr(p + key.size() + 1, end - (p + key.size() + 1));
            out = B64Decode(val);
            flag = !out.empty();
        };
        extract_param("sprop-vps", vps_, have_vps_);
        extract_param("sprop-sps", sps_, have_sps_);
        extract_param("sprop-pps", pps_, have_pps_);
        return have_vps_ || have_sps_ || have_pps_;
    }

    // H264
    auto pos = sdp.find("sprop-parameter-sets=");
    if (pos == std::string::npos) return false;
    auto end = sdp.find_first_of(";\r\n", pos);
    std::string val = sdp.substr(pos + 21, end - (pos + 21));
    auto comma = val.find(',');
    if (comma == std::string::npos) return false;

    sps_ = B64Decode(val.substr(0, comma));
    pps_ = B64Decode(val.substr(comma + 1));
    have_sps_ = !sps_.empty();
    have_pps_ = !pps_.empty();
    return have_sps_ && have_pps_;
}

bool VideoToolboxPipeline::CreateFormatDescription() {
    if (!have_sps_ || !have_pps_) return false;
    if (format_desc_) { CFRelease(format_desc_); format_desc_ = nullptr; }

    OSStatus status;
    if (is_hevc_) {
        if (!have_vps_) {
            LOG_ERROR("[VideoToolbox] Missing VPS for HEVC format description");
            return false;
        }
        const uint8_t* params[3] = {vps_.data(), sps_.data(), pps_.data()};
        const size_t sizes[3] = {vps_.size(), sps_.size(), pps_.size()};
        status = CMVideoFormatDescriptionCreateFromHEVCParameterSets(
            kCFAllocatorDefault, 3, params, sizes, 4, nullptr, &format_desc_);
    } else {
        const uint8_t* params[2] = {sps_.data(), pps_.data()};
        const size_t sizes[2] = {sps_.size(), pps_.size()};
        status = CMVideoFormatDescriptionCreateFromH264ParameterSets(
            kCFAllocatorDefault, 2, params, sizes, 4, &format_desc_);
    }

    if (status != noErr) {
        LOG_ERROR("[VideoToolbox] Format description failed: {}", status);
        return false;
    }
    return true;
}

// ============================================================
// VTDecompressionSession
// ============================================================

bool VideoToolboxPipeline::InitDecoder() {
    if (!format_desc_) return false;

    CFMutableDictionaryRef decoderSpec = CFDictionaryCreateMutable(kCFAllocatorDefault, 1,
        &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
    CFDictionarySetValue(decoderSpec, kVTVideoDecoderSpecification_RequireHardwareAcceleratedVideoDecoder, kCFBooleanTrue);

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
        kCFAllocatorDefault, format_desc_, decoderSpec, outDict, &cb, &decode_session_);
    CFRelease(decoderSpec);
    CFRelease(outDict);
    if (status != noErr) {
        LOG_ERROR("[VideoToolbox] Decompression session failed: {}", status);
        last_status_ = HALStatus::Error(HALStatusCode::DecodeFailed, "VideoToolbox hardware decoder unavailable");
        return false;
    }
    decoder_initialized_ = true;
    LOG_INFO("[VideoToolbox] Hardware decompression session created");
    return true;
}

void VideoToolboxPipeline::DestroyDecoder() {
    if (decode_session_) { VTDecompressionSessionInvalidate(decode_session_); CFRelease(decode_session_); decode_session_ = nullptr; }
    if (format_desc_) { CFRelease(format_desc_); format_desc_ = nullptr; }
    decoder_initialized_ = false;
}

void VideoToolboxPipeline::DecodeCallback(void* refcon, void* /*source*/, OSStatus status,
                                          VTDecodeInfoFlags /*flags*/, CVImageBufferRef img,
                                          CMTime /*pts*/, CMTime /*duration*/) {
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
    if (size > 0xFFFFFFFFu) return false;

    uint32_t nal_length = static_cast<uint32_t>(size);
    const uint8_t length_prefix[] = {
        static_cast<uint8_t>((nal_length >> 24) & 0xFF),
        static_cast<uint8_t>((nal_length >> 16) & 0xFF),
        static_cast<uint8_t>((nal_length >> 8) & 0xFF),
        static_cast<uint8_t>(nal_length & 0xFF),
    };

    CMBlockBufferRef block = nullptr;
    OSStatus s = CMBlockBufferCreateWithMemoryBlock(kCFAllocatorDefault, nullptr, size + 4,
        kCFAllocatorDefault, nullptr, 0, size + 4, 0, &block);
    if (s != noErr) return false;
    s = CMBlockBufferReplaceDataBytes(length_prefix, block, 0, 4);
    if (s == noErr) s = CMBlockBufferReplaceDataBytes(data, block, 4, size);
    if (s != noErr) {
        CFRelease(block);
        return false;
    }

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

namespace {
struct EncodeOutputContext {
    std::vector<uint8_t>* output;
    bool* sps_pps_sent;
    bool is_key_frame = false;
};
}  // namespace

static void EncodeOutputCallback(void* /*refcon*/, void* src, OSStatus status,
                                  VTEncodeInfoFlags /*flags*/, CMSampleBufferRef sb) {
    if (status != noErr || !sb || !src) return;
    auto* context = static_cast<EncodeOutputContext*>(src);
    auto* output = context->output;

    // 每个 NAL 前插入 Annex B 起始码 (00 00 00 01)，
    // 将 VT 默认的 AVCC 格式转为 Annex B，使下游 RtspPushStage 能正确解析。
    static const uint8_t kAnnexBStartCode[4] = {0x00, 0x00, 0x00, 0x01};

    // VT 只通过 CMFormatDescription 暴露参数集，不内联在码流中。每个 IDR
    // 重复携带 SPS/PPS，确保 RTSP 重连后的新解码器能从当前关键帧起播。
    context->is_key_frame = true;
    CFArrayRef attachments = CMSampleBufferGetSampleAttachmentsArray(sb, false);
    if (attachments && CFArrayGetCount(attachments) > 0) {
        auto attachment = static_cast<CFDictionaryRef>(CFArrayGetValueAtIndex(attachments, 0));
        auto not_sync = static_cast<CFBooleanRef>(CFDictionaryGetValue(attachment, kCMSampleAttachmentKey_NotSync));
        context->is_key_frame = not_sync != kCFBooleanTrue;
    }

    CMBlockBufferRef block = CMSampleBufferGetDataBuffer(sb);
    if (!block) return;
    size_t total_len = 0, at_offset = 0;
    char* raw = nullptr;
    if (CMBlockBufferGetDataPointer(block, 0, &at_offset, &total_len, &raw) != noErr || !raw || total_len < 4)
        return;
    output->reserve(total_len + 256);

    if (!*context->sps_pps_sent || context->is_key_frame) {
        CMFormatDescriptionRef fmt = CMSampleBufferGetFormatDescription(sb);
        bool header_added = false;
        if (fmt) {
            size_t sps_count = 0, pps_count = 0;
            const uint8_t* sps_ptr = nullptr; size_t sps_size = 0;
            const uint8_t* pps_ptr = nullptr; size_t pps_size = 0;
            CMVideoFormatDescriptionGetH264ParameterSetAtIndex(fmt, 0, &sps_ptr, &sps_size, &sps_count, nullptr);
            CMVideoFormatDescriptionGetH264ParameterSetAtIndex(fmt, 1, &pps_ptr, &pps_size, &pps_count, nullptr);
            if (sps_ptr && sps_size > 0) {
                output->insert(output->end(), kAnnexBStartCode, kAnnexBStartCode + 4);
                output->insert(output->end(), sps_ptr, sps_ptr + sps_size);
                header_added = true;
            }
            if (pps_ptr && pps_size > 0) {
                output->insert(output->end(), kAnnexBStartCode, kAnnexBStartCode + 4);
                output->insert(output->end(), pps_ptr, pps_ptr + pps_size);
                header_added = true;
            }
        }
        *context->sps_pps_sent = *context->sps_pps_sent || header_added;
    }

    // 遍历 CMBlockBuffer 中的 NAL 单元（AVCC 格式：4 字节长度 + NAL 数据），
    // 将每个 NAL 转为 Annex B 格式。
    const uint8_t* ptr = reinterpret_cast<const uint8_t*>(raw);
    size_t offset = 0;
    while (offset + 4 <= total_len) {
        uint32_t nal_len = (static_cast<uint32_t>(ptr[offset]) << 24) |
                           (static_cast<uint32_t>(ptr[offset + 1]) << 16) |
                           (static_cast<uint32_t>(ptr[offset + 2]) << 8) |
                            static_cast<uint32_t>(ptr[offset + 3]);
        offset += 4;
        if (nal_len == 0 || offset + nal_len > total_len) break;
        output->insert(output->end(), kAnnexBStartCode, kAnnexBStartCode + 4);
        output->insert(output->end(), ptr + offset, ptr + offset + nal_len);
        offset += nal_len;
    }
}

bool VideoToolboxPipeline::EncodeInit(const std::string& config_json) {
    // 解析编码器配置 JSON，默认值对齐 EncoderStage
    enc_bitrate_ = 4'000'000;
    enc_fps_ = 25;
    enc_gop_ = 50;
    auto j = nlohmann::json::parse(config_json, nullptr, false);
    if (j.is_object()) {
        if (j.contains("bitrate") && j["bitrate"].is_number()) enc_bitrate_ = j["bitrate"].get<int>();
        if (j.contains("fps") && j["fps"].is_number()) enc_fps_ = j["fps"].get<int>();
        if (j.contains("gop") && j["gop"].is_number()) enc_gop_ = j["gop"].get<int>();
    }
    LOG_INFO("[VideoToolbox] Encoder config: bitrate={}, fps={}, gop={}", enc_bitrate_, enc_fps_, enc_gop_);
    return true;
}

bool VideoToolboxPipeline::InitEncoder(int width, int height) {
    std::lock_guard<std::mutex> lock(enc_mu_);
    if (encoder_initialized_) return true;
    OSStatus s = VTCompressionSessionCreate(kCFAllocatorDefault, width, height,
        kCMVideoCodecType_H264, nullptr, nullptr, nullptr, EncodeOutputCallback, nullptr, &encode_session_);
    if (s != noErr) return false;

    VTSessionSetProperty(encode_session_, kVTCompressionPropertyKey_RealTime, kCFBooleanTrue);
    VTSessionSetProperty(encode_session_, kVTCompressionPropertyKey_ProfileLevel, kVTProfileLevel_H264_Main_AutoLevel);
    VTSessionSetProperty(encode_session_, kVTCompressionPropertyKey_AllowFrameReordering, kCFBooleanFalse);
    // 不允许 B 帧，降低端到端延迟

    int bitrate = enc_bitrate_ > 0 ? enc_bitrate_ : 4'000'000;
    CFNumberRef brNum = CFNumberCreate(kCFAllocatorDefault, kCFNumberSInt32Type, &bitrate);
    VTSessionSetProperty(encode_session_, kVTCompressionPropertyKey_AverageBitRate, brNum);
    CFRelease(brNum);

    int fps = enc_fps_ > 0 ? enc_fps_ : 25;
    CFNumberRef fpsNum = CFNumberCreate(kCFAllocatorDefault, kCFNumberSInt32Type, &fps);
    VTSessionSetProperty(encode_session_, kVTCompressionPropertyKey_ExpectedFrameRate, fpsNum);
    CFRelease(fpsNum);

    int gop = enc_gop_ > 0 ? enc_gop_ : 50;
    CFNumberRef gopNum = CFNumberCreate(kCFAllocatorDefault, kCFNumberSInt32Type, &gop);
    VTSessionSetProperty(encode_session_, kVTCompressionPropertyKey_MaxKeyFrameInterval, gopNum);
    CFRelease(gopNum);

    VTCompressionSessionPrepareToEncodeFrames(encode_session_);
    encoder_initialized_ = true;
    LOG_INFO("[VideoToolbox] Encoder: {}x{} bitrate={} fps={} gop={}", width, height, bitrate, fps, gop);
    return true;
}

void VideoToolboxPipeline::DestroyEncoder() {
    std::lock_guard<std::mutex> lock(enc_mu_);
    if (encode_session_) { VTCompressionSessionInvalidate(encode_session_); CFRelease(encode_session_); encode_session_ = nullptr; }
    encoder_initialized_ = false;
    sps_pps_sent_ = false;
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
    EncodeOutputContext output_context{&output, &sps_pps_sent_};

    CMTime pts = CMTimeMake(
        std::chrono::duration_cast<std::chrono::microseconds>(std::chrono::steady_clock::now().time_since_epoch()).count(), 1000000);
    VTEncodeInfoFlags flags = 0;
    OSStatus s = VTCompressionSessionEncodeFrame(encode_session_, cvpb, pts, kCMTimeInvalid, nullptr, &output_context, &flags);
    if (s == noErr) s = VTCompressionSessionCompleteFrames(encode_session_, kCMTimeInvalid);

    if (s != noErr || output.empty()) return false;

    out_size = std::min(size, output.size());
    std::memcpy(data, output.data(), out_size);
    desc.codec = VideoCodec::H264;
    desc.is_key_frame = output_context.is_key_frame;
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

    if (!RtspConnect(url)) { LOG_ERROR("[VideoToolbox] RtspConnect failed"); running_ = false; state_ = PipelineState::Error; return false; }
    if (!RtspOptions()) { LOG_ERROR("[VideoToolbox] RtspOptions failed"); RtspDisconnect(); running_ = false; last_status_ = HALStatus::Error(HALStatusCode::OpenStreamFailed, "OPTIONS failed"); return false; }

    std::string sdp;
    if (!RtspDescribe(sdp)) { LOG_ERROR("[VideoToolbox] RtspDescribe failed"); RtspDisconnect(); running_ = false; last_status_ = HALStatus::Error(HALStatusCode::OpenStreamFailed, "DESCRIBE failed"); return false; }
    LOG_INFO("[VideoToolbox] Full SDP:\n{}", sdp);
    ParseCodecParams(sdp);
    // 解析 SDP 中的 media tracks (m= 行)
    sdp_tracks_.clear();
    size_t m_pos = 0;
    while ((m_pos = sdp.find("\nm=", m_pos)) != std::string::npos ||
           (m_pos == 0 && sdp.substr(0, 2) == "m=")) {
        if (m_pos == 0 && sdp.substr(0, 2) == "m=")
            m_pos = 0;
        else
            m_pos++;
        auto eol = sdp.find("\n", m_pos);
        if (eol == std::string::npos) eol = sdp.size() - 1;
        std::string mline = sdp.substr(m_pos, eol - m_pos);
        // m=video 0 RTP/AVP 96 或 m=audio 0 RTP/AVP 0
        auto track_type = mline.substr(2, mline.find(' ') - 2);
        auto track_pos = sdp_tracks_.size();
        // 查找对应的 a=control:... 行
        std::string track_id = "trackID=" + std::to_string(track_pos);
        size_t ctrl_pos = sdp.find("a=control:", eol);
        if (ctrl_pos != std::string::npos && ctrl_pos < sdp.find("\nm=", eol)) {
            auto ctrl_eol = sdp.find("\n", ctrl_pos);
            std::string ctrl = sdp.substr(ctrl_pos + 10, ctrl_eol - ctrl_pos - 10);
            ctrl = Trim(ctrl);
            if (ctrl != "*" && !ctrl.empty()) track_id = ctrl;
        }
        sdp_tracks_.push_back({track_type, track_id});
        m_pos = eol;
    }
    if (sdp_tracks_.empty()) {
        // 没有 m= 行（a=control:*），使用基础 URL
        sdp_tracks_.push_back({"video", ""});
    }
    {
        std::string track_str;
        for (auto& t : sdp_tracks_) track_str += " " + t.first + "=" + t.second;
        LOG_INFO("[VideoToolbox] SDP tracks:{}", track_str);
    }
    if (!RtspSetup()) { LOG_ERROR("[VideoToolbox] RtspSetup failed"); RtspDisconnect(); running_ = false; last_status_ = HALStatus::Error(HALStatusCode::OpenStreamFailed, "SETUP failed"); return false; }
    if (!RtspPlay()) { LOG_ERROR("[VideoToolbox] RtspPlay failed"); RtspDisconnect(); running_ = false; last_status_ = HALStatus::Error(HALStatusCode::OpenStreamFailed, "PLAY failed"); return false; }

    state_ = PipelineState::Streaming;
    if (state_callback_) state_callback_(state_);
    LOG_INFO("[VideoToolbox] RTSP playing: {}", url);

    pull_thread_ = std::make_unique<std::thread>(&VideoToolboxPipeline::PullLoop, this);
    return true;
}

void VideoToolboxPipeline::PullLoop() {
    RtpPacket pkt;
    int pkt_count = 0;
    while (running_ && !paused_) {
        if (!RecvRtpPacket(pkt)) {
            if (running_) { LOG_ERROR("[VideoToolbox] RecvRtp failed"); std::this_thread::sleep_for(std::chrono::seconds(1)); continue; }
            break;
        }
        if (++pkt_count <= 3 || pkt_count % 200 == 0)
            LOG_INFO("[VideoToolbox] RTP pkt seq={} type={} size={} count={}", pkt.seq, static_cast<int>(pkt.payload_type), pkt.payload.size(), pkt_count);
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
