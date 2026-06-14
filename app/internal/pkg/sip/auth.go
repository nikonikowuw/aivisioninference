package sip

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	realmRegexp    = regexp.MustCompile(`realm="([^"]+)"`)
	nonceRegexp    = regexp.MustCompile(`nonce="([^"]+)"`)
	usernameRegexp = regexp.MustCompile(`username="([^"]+)"`)
	uriRegexp      = regexp.MustCompile(`uri="([^"]+)"`)
	responseRegexp = regexp.MustCompile(`response="([^"]+)"`)
	qopRegexp      = regexp.MustCompile(`qop="?([^",]+)"?`)
	ncRegexp       = regexp.MustCompile(`nc=([0-9a-fA-F]+)`)
	cnonceRegexp   = regexp.MustCompile(`cnonce="([^"]+)"`)
)

func GenerateNonce() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x_%d", hex.EncodeToString(b), time.Now().Unix())
}

func parseAuthParams(authHeader string) map[string]string {
	params := make(map[string]string)
	matches := func(re *regexp.Regexp, key string) {
		if m := re.FindStringSubmatch(authHeader); len(m) > 1 {
			params[key] = m[1]
		}
	}
	matches(usernameRegexp, "username")
	matches(realmRegexp, "realm")
	matches(nonceRegexp, "nonce")
	matches(uriRegexp, "uri")
	matches(responseRegexp, "response")
	matches(qopRegexp, "qop")
	matches(ncRegexp, "nc")
	matches(cnonceRegexp, "cnonce")
	return params
}

func VerifyDigest(authHeader, method, password string) bool {
	params := parseAuthParams(authHeader)
	username := params["username"]
	realm := params["realm"]
	nonce := params["nonce"]
	uri := params["uri"]
	response := params["response"]

	if username == "" || realm == "" || nonce == "" || uri == "" || response == "" {
		return false
	}

	// Verify nonce timestamp to prevent replay attacks (e.g. within 5 minutes)
	parts := strings.Split(nonce, "_")
	if len(parts) > 1 {
		var ts int64
		_, err := fmt.Sscanf(parts[1], "%d", &ts)
		if err == nil {
			if time.Now().Unix()-ts > 300 { // 5 minutes expiry
				return false
			}
		}
	}

	h := md5.New()
	// HA1 = MD5(username:realm:password)
	h.Write([]byte(fmt.Sprintf("%s:%s:%s", username, realm, password)))
	ha1 := hex.EncodeToString(h.Sum(nil))

	h.Reset()
	// HA2 = MD5(method:uri)
	h.Write([]byte(fmt.Sprintf("%s:%s", method, uri)))
	ha2 := hex.EncodeToString(h.Sum(nil))

	h.Reset()
	var expectedResponse string
	qop := params["qop"]
	if qop == "auth" {
		nc := params["nc"]
		cnonce := params["cnonce"]
		// Response = MD5(HA1:nonce:nc:cnonce:qop:HA2)
		h.Write([]byte(fmt.Sprintf("%s:%s:%s:%s:%s:%s", ha1, nonce, nc, cnonce, qop, ha2)))
		expectedResponse = hex.EncodeToString(h.Sum(nil))
	} else {
		// Response = MD5(HA1:nonce:HA2)
		h.Write([]byte(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2)))
		expectedResponse = hex.EncodeToString(h.Sum(nil))
	}

	return strings.EqualFold(response, expectedResponse)
}
