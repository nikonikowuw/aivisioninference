#ifndef AIVISION_ZLM_URL_H
#define AIVISION_ZLM_URL_H

#include <string>

namespace aivision
{

/// 解析后的 ZLM API URL 信息，配置加载时一次性解析并缓存。
struct ZLMUrlInfo
{
    std::string host;
    int http_port = 8000;
};

/// 从 ZLM API URL 字符串解析出 host 和 http_port。
/// URL 格式: "http://host:port" 或 "http://host" (默认端口 8000)。
inline ZLMUrlInfo ParseZLMUrl(const std::string &zlm_api_url)
{
    ZLMUrlInfo info;
    std::string url = zlm_api_url;

    // 移除协议前缀
    size_t proto_end = url.find("://");
    if (proto_end != std::string::npos)
        url = url.substr(proto_end + 3);

    // 解析 host:port/path
    size_t colon_pos = url.find(":");
    if (colon_pos != std::string::npos) {
        info.host = url.substr(0, colon_pos);
        std::string port_str = url.substr(colon_pos + 1);
        size_t slash_pos = port_str.find("/");
        if (slash_pos != std::string::npos)
            port_str = port_str.substr(0, slash_pos);
        try {
            info.http_port = std::stoi(port_str);
        } catch (...) {
        }
    } else {
        size_t slash_pos = url.find("/");
        if (slash_pos != std::string::npos)
            info.host = url.substr(0, slash_pos);
        else
            info.host = url;
    }
    return info;
}

/// 从 ZLM API URL 提取纯 host（不含端口和路径）。
inline std::string ExtractZLMHost(const std::string &zlm_api_url)
{
    return ParseZLMUrl(zlm_api_url).host;
}

} // namespace aivision

#endif // AIVISION_ZLM_URL_H
