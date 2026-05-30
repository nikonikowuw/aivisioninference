// Package httpx 提供 HTTP 请求工具函数，包括安全连接检测和可信代理校验。
package httpx

import (
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// TrustedProxies is a list of trusted proxy IPs or CIDR notations.
// When non-empty, forwarded headers (X-Forwarded-Proto, X-Forwarded-Ssl) are
// only trusted if the request's immediate peer is in this list.
// When empty (default), forwarded headers from any IP are accepted —
// set this in production to prevent header spoofing.
var TrustedProxies []string

// IsSecureRequest reports whether the incoming request should be treated as HTTPS.
// It always honors direct TLS connections. For proxy-terminated TLS, it checks
// X-Forwarded-Proto and X-Forwarded-Ssl headers with optional trusted proxy
// validation when TrustedProxies is configured.
func IsSecureRequest(c *gin.Context) bool {
	if c.Request.TLS != nil {
		return true
	}

	secure := checkForwardedProto(c.Request) || checkForwardedSSL(c.Request)
	if !secure {
		return false
	}

	if len(TrustedProxies) > 0 && !isTrustedProxy(c.Request) {
		return false
	}

	return true
}

// checkForwardedProto checks the X-Forwarded-Proto header.
// Handles comma-separated values from chained proxies by using the outermost
// (leftmost) value.
func checkForwardedProto(r *http.Request) bool {
	proto := r.Header.Get("X-Forwarded-Proto")
	if proto == "" {
		return false
	}
	if idx := strings.IndexByte(proto, ','); idx > 0 {
		proto = proto[:idx]
	}
	proto = strings.TrimSpace(proto)
	return strings.EqualFold(proto, "https")
}

// checkForwardedSSL checks the X-Forwarded-Ssl header (value: "on").
func checkForwardedSSL(r *http.Request) bool {
	return strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Ssl")), "on")
}

// isTrustedProxy checks if the immediate peer IP is in the trusted proxy list.
func isTrustedProxy(r *http.Request) bool {
	peerIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(peerIP); err == nil {
		peerIP = host
	}
	peer := net.ParseIP(peerIP)
	if peer == nil {
		return false
	}
	for _, cidr := range TrustedProxies {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			if ip := net.ParseIP(cidr); ip != nil && ip.Equal(peer) {
				return true
			}
			continue
		}
		if network.Contains(peer) {
			return true
		}
	}
	return false
}
