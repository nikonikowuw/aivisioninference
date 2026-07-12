#include "pipeline/rtsp_push_stage.h"

#include <algorithm>
#include <arpa/inet.h>
#include <chrono>
#include <cstring>
#include <iostream>
#include <netdb.h>
#include <random>
#include <sstream>
#include <sys/socket.h>
#include <unistd.h>

namespace aivision {
namespace pipeline {
namespace {
constexpr size_t kMaxRtpPayload = 1200;
constexpr uint8_t kRtpPayloadTypeH264 = 96;
constexpr uint32_t kVideoClockRate = 90000;

bool SendAll(int fd, const void* data, size_t size) {
    const auto* ptr = static_cast<const uint8_t*>(data);
    while (size > 0) {
        ssize_t n = ::send(fd, ptr, size, 0);
        if (n <= 0) return false;
        ptr += n;
        size -= static_cast<size_t>(n);
    }
    return true;
}

std::string Trim(const std::string& value) {
    size_t start = value.find_first_not_of(" \t\r\n");
    if (start == std::string::npos) return "";
    size_t end = value.find_last_not_of(" \t\r\n");
    return value.substr(start, end - start + 1);
}

bool ParseRtspUrl(const std::string& url, std::string& host, uint16_t& port) {
    constexpr const char* prefix = "rtsp://";
    if (url.rfind(prefix, 0) != 0) return false;

    std::string rest = url.substr(std::strlen(prefix));
    size_t slash = rest.find('/');
    std::string authority = slash == std::string::npos ? rest : rest.substr(0, slash);
    if (authority.empty()) return false;

    size_t colon = authority.rfind(':');
    if (colon != std::string::npos) {
        host = authority.substr(0, colon);
        int parsed_port = std::stoi(authority.substr(colon + 1));
        if (parsed_port <= 0 || parsed_port > 65535) return false;
        port = static_cast<uint16_t>(parsed_port);
    } else {
        host = authority;
        port = 554;
    }
    return !host.empty();
}

std::vector<std::pair<const uint8_t*, size_t>> SplitAnnexBNals(const std::vector<uint8_t>& data) {
    std::vector<std::pair<const uint8_t*, size_t>> nals;
    auto find_start = [&](size_t from, size_t& pos, size_t& prefix_len) -> bool {
        for (size_t i = from; i + 3 <= data.size(); ++i) {
            if (i + 4 <= data.size() && data[i] == 0 && data[i + 1] == 0 && data[i + 2] == 0 && data[i + 3] == 1) {
                pos = i;
                prefix_len = 4;
                return true;
            }
            if (data[i] == 0 && data[i + 1] == 0 && data[i + 2] == 1) {
                pos = i;
                prefix_len = 3;
                return true;
            }
        }
        return false;
    };

    size_t pos = 0;
    size_t prefix_len = 0;
    if (!find_start(0, pos, prefix_len)) {
        if (!data.empty()) nals.emplace_back(data.data(), data.size());
        return nals;
    }

    while (true) {
        size_t nal_start = pos + prefix_len;
        size_t next_pos = 0;
        size_t next_prefix = 0;
        bool has_next = find_start(nal_start, next_pos, next_prefix);
        size_t nal_end = has_next ? next_pos : data.size();
        while (nal_end > nal_start && data[nal_end - 1] == 0) --nal_end;
        if (nal_end > nal_start) nals.emplace_back(data.data() + nal_start, nal_end - nal_start);
        if (!has_next) break;
        pos = next_pos;
        prefix_len = next_prefix;
    }
    return nals;
}

uint32_t Timestamp90k(uint64_t first_timestamp_ns, uint64_t timestamp_ns) {
    if (timestamp_ns <= first_timestamp_ns) return 0;
    return static_cast<uint32_t>(((timestamp_ns - first_timestamp_ns) * kVideoClockRate) / 1000000000ULL);
}
}  // namespace

RtspPushStage::RtspPushStage(const std::string& push_url, std::shared_ptr<EncoderStage> encoder)
    : push_url_(push_url), encoder_(std::move(encoder)) {
    std::random_device rd;
    rtp_seq_ = static_cast<uint16_t>(rd());
    rtp_ssrc_ = rd();
}

RtspPushStage::~RtspPushStage() {
    Stop();
}

bool RtspPushStage::Init(const StageContext& ctx) {
    (void)ctx;
    return encoder_ != nullptr;
}

bool RtspPushStage::Run() {
    if (running_.load()) return true;
    running_.store(true);
    thread_ = std::make_unique<std::thread>(&RtspPushStage::Loop, this);
    return true;
}

void RtspPushStage::Stop() {
    if (!running_.load()) return;
    running_.store(false);
    Disconnect(false);
    if (thread_ && thread_->joinable()) {
        thread_->join();
    }
}

void RtspPushStage::PushFrame(const FrameContext& frame) {
    (void)frame;
}

void RtspPushStage::Loop() {
    uint32_t reconnect_delay_ms = 5000;

    while (running_.load()) {
        if (!connected_.load()) {
            if (Connect()) {
                connected_.store(true);
                reconnect_delay_ms = 5000;
                std::cout << "[RTSP] Publishing to " << push_url_ << std::endl;
            } else {
                std::this_thread::sleep_for(std::chrono::milliseconds(reconnect_delay_ms));
                reconnect_delay_ms = std::min(reconnect_delay_ms * 2, 120000u);
                continue;
            }
        }

        auto [pkt, ok] = encoder_->PopPacket(100);
        if (!ok) continue;

        if (!SendPacket(pkt)) {
            std::cerr << "[RTSP] Send failed, reconnecting..." << std::endl;
            connected_.store(false);
            Disconnect();
        }
    }
}

bool RtspPushStage::Connect() {
    Disconnect();
    if (!ParseRtspUrl(push_url_, host_, port_)) {
        std::cerr << "[RTSP] Invalid push URL: " << push_url_ << std::endl;
        return false;
    }

    addrinfo hints{};
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    addrinfo* result = nullptr;
    std::string port_str = std::to_string(port_);
    if (getaddrinfo(host_.c_str(), port_str.c_str(), &hints, &result) != 0) {
        std::cerr << "[RTSP] Failed to resolve host: " << host_ << std::endl;
        return false;
    }

    for (addrinfo* ai = result; ai != nullptr; ai = ai->ai_next) {
        socket_fd_ = ::socket(ai->ai_family, ai->ai_socktype, ai->ai_protocol);
        if (socket_fd_ < 0) continue;
        if (::connect(socket_fd_, ai->ai_addr, ai->ai_addrlen) == 0) break;
        ::close(socket_fd_);
        socket_fd_ = -1;
    }
    freeaddrinfo(result);
    if (socket_fd_ < 0) {
        std::cerr << "[RTSP] Failed to connect: " << host_ << ':' << port_ << std::endl;
        return false;
    }

    cseq_ = 1;
    session_.clear();
    first_timestamp_ns_ = 0;

    int status = 0;
    std::string response;
    std::ostringstream options;
    options << "OPTIONS " << push_url_ << " RTSP/1.0\r\n"
            << "CSeq: " << cseq_++ << "\r\n"
            << "User-Agent: aivision-engine\r\n\r\n";
    if (!SendRequest(options.str()) || !ReadResponse(status, response) || status < 200 || status >= 300) return false;

    std::ostringstream sdp;
    sdp << "v=0\r\n"
        << "o=- 0 0 IN IP4 127.0.0.1\r\n"
        << "s=AIVisionInference\r\n"
        << "c=IN IP4 0.0.0.0\r\n"
        << "t=0 0\r\n"
        << "a=control:*\r\n"
        << "m=video 0 RTP/AVP " << static_cast<int>(kRtpPayloadTypeH264) << "\r\n"
        << "a=rtpmap:" << static_cast<int>(kRtpPayloadTypeH264) << " H264/90000\r\n"
        << "a=fmtp:" << static_cast<int>(kRtpPayloadTypeH264) << " packetization-mode=1\r\n"
        << "a=control:trackID=0\r\n";
    std::string sdp_body = sdp.str();

    std::ostringstream announce;
    announce << "ANNOUNCE " << push_url_ << " RTSP/1.0\r\n"
             << "CSeq: " << cseq_++ << "\r\n"
             << "User-Agent: aivision-engine\r\n"
             << "Content-Type: application/sdp\r\n"
             << "Content-Length: " << sdp_body.size() << "\r\n\r\n"
             << sdp_body;
    if (!SendRequest(announce.str()) || !ReadResponse(status, response) || status < 200 || status >= 300) return false;

    std::ostringstream setup;
    setup << "SETUP " << push_url_ << "/trackID=0 RTSP/1.0\r\n"
          << "CSeq: " << cseq_++ << "\r\n"
          << "User-Agent: aivision-engine\r\n"
          << "Transport: RTP/AVP/TCP;unicast;interleaved=0-1\r\n\r\n";
    if (!SendRequest(setup.str()) || !ReadResponse(status, response) || status < 200 || status >= 300) return false;

    size_t session_pos = response.find("Session:");
    if (session_pos == std::string::npos) session_pos = response.find("session:");
    if (session_pos != std::string::npos) {
        size_t line_end = response.find("\r\n", session_pos);
        std::string line = response.substr(session_pos + 8, line_end - (session_pos + 8));
        size_t semicolon = line.find(';');
        session_ = Trim(semicolon == std::string::npos ? line : line.substr(0, semicolon));
    }
    if (session_.empty()) {
        std::cerr << "[RTSP] SETUP response missing Session header" << std::endl;
        return false;
    }

    std::ostringstream record;
    record << "RECORD " << push_url_ << " RTSP/1.0\r\n"
           << "CSeq: " << cseq_++ << "\r\n"
           << "User-Agent: aivision-engine\r\n"
           << "Session: " << session_ << "\r\n"
           << "Range: npt=0.000-\r\n\r\n";
    if (!SendRequest(record.str()) || !ReadResponse(status, response) || status < 200 || status >= 300) return false;

    return true;
}

void RtspPushStage::Disconnect(bool send_teardown) {
    if (socket_fd_ >= 0) {
        if (send_teardown && !session_.empty()) {
            std::ostringstream teardown;
            teardown << "TEARDOWN " << push_url_ << " RTSP/1.0\r\n"
                     << "CSeq: " << cseq_++ << "\r\n"
                     << "Session: " << session_ << "\r\n\r\n";
            SendRequest(teardown.str());
        }
        ::shutdown(socket_fd_, SHUT_RDWR);
        ::close(socket_fd_);
        socket_fd_ = -1;
    }
    session_.clear();
    connected_.store(false);
}

bool RtspPushStage::SendRequest(const std::string& request) {
    return socket_fd_ >= 0 && SendAll(socket_fd_, request.data(), request.size());
}

bool RtspPushStage::ReadResponse(int& status_code, std::string& response) {
    status_code = 0;
    response.clear();
    char buffer[4096];
    while (response.find("\r\n\r\n") == std::string::npos) {
        ssize_t n = ::recv(socket_fd_, buffer, sizeof(buffer), 0);
        if (n <= 0) return false;
        response.append(buffer, static_cast<size_t>(n));
        if (response.size() > 64 * 1024) return false;
    }

    std::istringstream stream(response);
    std::string version;
    stream >> version >> status_code;
    if (status_code <= 0) {
        std::cerr << "[RTSP] Invalid response: " << response << std::endl;
        return false;
    }
    if (status_code < 200 || status_code >= 300) {
        std::cerr << "[RTSP] Request failed, status=" << status_code << ", response=" << response << std::endl;
    }
    return true;
}

bool RtspPushStage::SendPacket(const EncodedPacket& pkt) {
    if (pkt.data.empty()) return true;
    if (first_timestamp_ns_ == 0) first_timestamp_ns_ = pkt.timestamp_ns;
    uint32_t rtp_timestamp = Timestamp90k(first_timestamp_ns_, pkt.timestamp_ns);

    auto nals = SplitAnnexBNals(pkt.data);
    if (nals.empty()) return true;

    for (size_t i = 0; i < nals.size(); ++i) {
        bool marker = i + 1 == nals.size();
        if (!SendH264Nal(nals[i].first, nals[i].second, rtp_timestamp, marker)) return false;
    }
    return true;
}

bool RtspPushStage::SendH264Nal(const uint8_t* nal, size_t nal_size, uint32_t rtp_timestamp, bool marker) {
    if (!nal || nal_size == 0) return true;

    if (nal_size <= kMaxRtpPayload) {
        return SendInterleavedRtp(nal, nal_size, rtp_timestamp, marker);
    }

    uint8_t nal_header = nal[0];
    uint8_t nri = nal_header & 0x60;
    uint8_t nal_type = nal_header & 0x1F;
    uint8_t fu_indicator = nri | 28;
    size_t offset = 1;
    bool first = true;
    while (offset < nal_size) {
        size_t chunk = std::min(kMaxRtpPayload - 2, nal_size - offset);
        bool last = offset + chunk >= nal_size;
        std::vector<uint8_t> fu(2 + chunk);
        fu[0] = fu_indicator;
        fu[1] = nal_type;
        if (first) fu[1] |= 0x80;
        if (last) fu[1] |= 0x40;
        std::memcpy(fu.data() + 2, nal + offset, chunk);
        if (!SendInterleavedRtp(fu.data(), fu.size(), rtp_timestamp, marker && last)) return false;
        offset += chunk;
        first = false;
    }
    return true;
}

bool RtspPushStage::SendInterleavedRtp(const uint8_t* payload, size_t payload_size, uint32_t rtp_timestamp, bool marker) {
    if (socket_fd_ < 0 || payload_size > 0xFFFF - 12) return false;

    uint16_t rtp_len = static_cast<uint16_t>(12 + payload_size);
    std::vector<uint8_t> packet(4 + rtp_len);
    packet[0] = '$';
    packet[1] = 0;
    packet[2] = static_cast<uint8_t>(rtp_len >> 8);
    packet[3] = static_cast<uint8_t>(rtp_len & 0xFF);

    uint8_t* rtp = packet.data() + 4;
    rtp[0] = 0x80;
    rtp[1] = static_cast<uint8_t>(kRtpPayloadTypeH264 | (marker ? 0x80 : 0));
    uint16_t seq = rtp_seq_++;
    rtp[2] = static_cast<uint8_t>(seq >> 8);
    rtp[3] = static_cast<uint8_t>(seq & 0xFF);
    rtp[4] = static_cast<uint8_t>(rtp_timestamp >> 24);
    rtp[5] = static_cast<uint8_t>((rtp_timestamp >> 16) & 0xFF);
    rtp[6] = static_cast<uint8_t>((rtp_timestamp >> 8) & 0xFF);
    rtp[7] = static_cast<uint8_t>(rtp_timestamp & 0xFF);
    rtp[8] = static_cast<uint8_t>(rtp_ssrc_ >> 24);
    rtp[9] = static_cast<uint8_t>((rtp_ssrc_ >> 16) & 0xFF);
    rtp[10] = static_cast<uint8_t>((rtp_ssrc_ >> 8) & 0xFF);
    rtp[11] = static_cast<uint8_t>(rtp_ssrc_ & 0xFF);
    std::memcpy(rtp + 12, payload, payload_size);

    if (!SendAll(socket_fd_, packet.data(), packet.size())) return false;
    total_bytes_sent_.fetch_add(packet.size());
    return true;
}

}  // namespace pipeline
}  // namespace aivision
