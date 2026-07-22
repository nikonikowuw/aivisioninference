package service

import (
	"net/url"
	"regexp"
)

const subStreamSuffix = "_sub"

type subStreamPathRule struct {
	pattern     *regexp.Regexp
	replacement string
}

var (
	dahuaSubStreamQueryRegexp = regexp.MustCompile(`(?i)(^|&)(subtype=)0(&|$)`)
	subStreamPathRules        = []subStreamPathRule{
		{regexp.MustCompile(`(?i)(/streaming/channels/\d*)1(/|$)`), "${1}2${2}"},
		{regexp.MustCompile(`(?i)(/unicast/c\d+/s)0(/live(?:/|$))`), "${1}1${2}"},
		{regexp.MustCompile(`(?i)(/stream)1(/|$)`), "${1}2${2}"},
		{regexp.MustCompile(`(?i)(/video)1(/|$)`), "${1}2${2}"},
		{regexp.MustCompile(`(?i)(/)main(/|$)`), "${1}sub${2}"},
	}
)

func playbackStreamID(deviceID, streamType string) string {
	if streamType == "sub" || streamType == "auxiliary" {
		return deviceID + subStreamSuffix
	}
	return deviceID
}

// DeriveSubStreamURL derives a sub-stream RTSP URL from a main RTSP URL.
// If no known vendor pattern matches, it safely falls back to the main RTSP URL.
func DeriveSubStreamURL(mainURL string) string {
	if mainURL == "" {
		return ""
	}

	parsed, err := url.Parse(mainURL)
	if err != nil {
		return mainURL
	}

	if derived := dahuaSubStreamQueryRegexp.ReplaceAllString(parsed.RawQuery, "${1}${2}1${3}"); derived != parsed.RawQuery {
		parsed.RawQuery = derived
		return parsed.String()
	}

	for _, rule := range subStreamPathRules {
		derived := rule.pattern.ReplaceAllString(parsed.Path, rule.replacement)
		if derived != parsed.Path {
			parsed.Path = derived
			parsed.RawPath = ""
			return parsed.String()
		}
	}

	return mainURL
}
