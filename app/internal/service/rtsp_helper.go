package service

import (
	"regexp"
	"strings"
)

var (
	// 大华: subtype=0 -> subtype=1
	dahuaRegexp = regexp.MustCompile(`(?i)(subtype=)0(\b|&)`)
	// 宇视: /unicast/c1/s0/live -> /unicast/c1/s1/live
	univiewRegexp = regexp.MustCompile(`(?i)(/unicast/c\d+/s)0(/live)`)
	// TP-Link / 通用: /stream1 -> /stream2
	streamNumRegexp = regexp.MustCompile(`(?i)(/stream)1(\b|[\?\/])`)
	// 通用: /main -> /sub
	mainSubRegexp = regexp.MustCompile(`(?i)(/)main(/|\b)`)
	// 通用: /video1 -> /video2
	videoNumRegexp = regexp.MustCompile(`(?i)(/video)1(\b|[\?\/])`)
)

// DeriveSubStreamURL derives a sub-stream RTSP URL from a main RTSP URL.
// If no known vendor pattern matches, it safely falls back to the main RTSP URL.
func DeriveSubStreamURL(mainURL string) string {
	if mainURL == "" {
		return ""
	}

	// 1. 海康威视 /Streaming/Channels/101 -> 102
	if strings.Contains(strings.ToLower(mainURL), "/streaming/channels/") {
		re := regexp.MustCompile(`(?i)(/Streaming/Channels/\d*)1(\b|[\?\/])`)
		if re.MatchString(mainURL) {
			return re.ReplaceAllString(mainURL, "${1}2${2}")
		}
	}

	// 2. 大华 subtype=0 -> subtype=1
	if dahuaRegexp.MatchString(mainURL) {
		return dahuaRegexp.ReplaceAllString(mainURL, "${1}1${2}")
	}

	// 3. 宇视 /unicast/c1/s0/live -> /unicast/c1/s1/live
	if univiewRegexp.MatchString(mainURL) {
		return univiewRegexp.ReplaceAllString(mainURL, "${1}1${2}")
	}

	// 4. TP-Link / 通用 /stream1 -> /stream2
	if streamNumRegexp.MatchString(mainURL) {
		return streamNumRegexp.ReplaceAllString(mainURL, "${1}2${2}")
	}

	// 5. 通用 /video1 -> /video2
	if videoNumRegexp.MatchString(mainURL) {
		return videoNumRegexp.ReplaceAllString(mainURL, "${1}2${2}")
	}

	// 6. 通用 /main/ -> /sub/
	if mainSubRegexp.MatchString(mainURL) {
		return mainSubRegexp.ReplaceAllString(mainURL, "${1}sub${2}")
	}

	return mainURL
}
