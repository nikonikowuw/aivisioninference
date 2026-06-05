package zlm

import "fmt"

// ZLMRsp is the standard response from ZLMediaKit API.
type ZLMRsp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg,omitempty"`
}

// ZLMError implements error interface for ZLM API errors.
type ZLMError struct {
	Code int
	Msg  string
}

func (e *ZLMError) Error() string {
	return fmt.Sprintf("zlm error: code=%d, msg=%s", e.Code, e.Msg)
}

// AddStreamProxyRequest represents the parameters for adding a stream proxy.
type AddStreamProxyRequest struct {
	Vhost        string `json:"vhost"`
	App          string `json:"app"`
	Stream       string `json:"stream"`
	URL          string `json:"url"`
	RetryCount   int    `json:"retry_count"`
	RtpType      int    `json:"rtp_type"` // 0:tcp, 1:udp, 2:multicast
	TimeoutSec   int    `json:"timeout_sec"`
	EnableHls    int    `json:"enable_hls"`
	EnableMp4    int    `json:"enable_mp4"`
	EnableRtsp   int    `json:"enable_rtsp"`
	EnableRtmp   int    `json:"enable_rtmp"`
	EnableTs     int    `json:"enable_ts"`
	EnableFmp4   int    `json:"enable_fmp4"`
	HlsSavePath  string `json:"hls_save_path,omitempty"`
	Mp4SavePath  string `json:"mp4_save_path,omitempty"`
}

// AddStreamProxyResponse represents the data returned by addStreamProxy API.
type AddStreamProxyResponse struct {
	ZLMRsp
	Data struct {
		Key string `json:"key"`
	} `json:"data"`
}

// CloseStreamRequest represents the parameters for closing a stream.
type CloseStreamRequest struct {
	Vhost  string `json:"vhost"`
	App    string `json:"app"`
	Stream string `json:"stream"`
	Force  int    `json:"force"`
}

// GetSnapRequest represents the parameters for getting a snapshot.
type GetSnapRequest struct {
	URL        string `json:"url"`
	TimeoutSec int    `json:"timeout_sec"`
	ExpireSec  int    `json:"expire_sec"`
}

// GetMediaListRequest represents the parameters for getting media list.
type GetMediaListRequest struct {
	Vhost  string `json:"vhost,omitempty"`
	App    string `json:"app,omitempty"`
	Stream string `json:"stream,omitempty"`
	Schema string `json:"schema,omitempty"`
}

// MediaItem represents a single media item in the media list.
type MediaItem struct {
	App            string `json:"app"`
	Stream         string `json:"stream"`
	Vhost          string `json:"vhost"`
	Schema         string `json:"schema"`
	ReaderCount    int    `json:"readerCount"`
	TotalReaderCount int  `json:"totalReaderCount"`
	OriginType     int    `json:"originType"`
	OriginTypeStr  string `json:"originTypeStr"`
	CreateStamp    int64  `json:"createStamp"`
	AliveSecond    int    `json:"aliveSecond"`
	BytesSpeed     int    `json:"bytesSpeed"`
}

// GetMediaListResponse represents the response from getMediaList API.
type GetMediaListResponse struct {
	ZLMRsp
	Data []MediaItem `json:"data"`
}

// RecordRequest represents parameters for recording control.
type RecordRequest struct {
	Type          int    `json:"type"` // 0:hls, 1:mp4
	Vhost         string `json:"vhost"`
	App           string `json:"app"`
	Stream        string `json:"stream"`
	CustomizedPath string `json:"customized_path,omitempty"`
	MaxSecond     int    `json:"max_second,omitempty"`
}
