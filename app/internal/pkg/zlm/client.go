package zlm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Client is a ZLMediaKit API client.
type Client struct {
	apiURL     string
	secret     string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewClient creates a new ZLM API client.
func NewClient(apiURL, secret string, logger *zap.Logger) *Client {
	if !strings.HasSuffix(apiURL, "/") {
		apiURL += "/"
	}
	return &Client{
		apiURL: apiURL,
		secret: secret,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		logger: logger,
	}
}

// buildURL appends secret and params to the API endpoint.
func (c *Client) buildURL(api string, params url.Values) string {
	if params == nil {
		params = url.Values{}
	}
	params.Set("secret", c.secret)
	return fmt.Sprintf("%sindex/api/%s?%s", c.apiURL, api, params.Encode())
}

// doRequest performs the HTTP request and unmarshals the response.
func (c *Client) doRequest(ctx context.Context, method, api string, queryParams url.Values, body interface{}, result interface{}) error {
	fullURL := c.buildURL(api, queryParams)

	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http execute: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status error: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var baseRsp ZLMRsp
	if err := json.Unmarshal(respBody, &baseRsp); err != nil {
		return fmt.Errorf("unmarshal base response: %w", err)
	}

	if baseRsp.Code != 0 {
		return &ZLMError{Code: baseRsp.Code, Msg: baseRsp.Msg}
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal result: %w", err)
		}
	}

	return nil
}

// AddStreamProxy adds a stream proxy to ZLM.
func (c *Client) AddStreamProxy(ctx context.Context, req AddStreamProxyRequest) (string, error) {
	var resp AddStreamProxyResponse
	err := c.doRequest(ctx, http.MethodPost, "addStreamProxy", nil, req, &resp)
	if err != nil {
		return "", err
	}
	return resp.Data.Key, nil
}

// CloseStream closes a specific stream.
func (c *Client) CloseStream(ctx context.Context, req CloseStreamRequest) error {
	return c.doRequest(ctx, http.MethodPost, "close_stream", nil, req, nil)
}

// GetMediaList gets the list of current media streams.
func (c *Client) GetMediaList(ctx context.Context, req GetMediaListRequest) ([]MediaItem, error) {
	params := url.Values{}
	if req.Vhost != "" {
		params.Set("vhost", req.Vhost)
	}
	if req.App != "" {
		params.Set("app", req.App)
	}
	if req.Stream != "" {
		params.Set("schema", req.Schema)
	}
	if req.Stream != "" {
		params.Set("stream", req.Stream)
	}

	var resp GetMediaListResponse
	err := c.doRequest(ctx, http.MethodGet, "getMediaList", params, nil, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

// GetSnap captures a snapshot from a stream and returns the image bytes.
func (c *Client) GetSnap(ctx context.Context, req GetSnapRequest) ([]byte, error) {
	params := url.Values{}
	params.Set("url", req.URL)
	if req.TimeoutSec > 0 {
		params.Set("timeout_sec", fmt.Sprintf("%d", req.TimeoutSec))
	}
	if req.ExpireSec > 0 {
		params.Set("expire_sec", fmt.Sprintf("%d", req.ExpireSec))
	}

	fullURL := c.buildURL("getSnap", params)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status error: %d, body: %s", resp.StatusCode, string(respBody))
	}

	// If response is short and looks like JSON, check for ZLM error
	if len(respBody) < 500 && json.Valid(respBody) {
		var baseRsp ZLMRsp
		if err := json.Unmarshal(respBody, &baseRsp); err == nil && baseRsp.Code != 0 {
			return nil, &ZLMError{Code: baseRsp.Code, Msg: baseRsp.Msg}
		}
	}

	return respBody, nil
}

// StartRecord starts recording a stream.
func (c *Client) StartRecord(ctx context.Context, req RecordRequest) error {
	return c.doRequest(ctx, http.MethodPost, "startRecord", nil, req, nil)
}

// StopRecord stops recording a stream.
func (c *Client) StopRecord(ctx context.Context, req RecordRequest) error {
	return c.doRequest(ctx, http.MethodPost, "stopRecord", nil, req, nil)
}

// IsRecordingResponse represents the response from isRecording API.
type IsRecordingResponse struct {
	ZLMRsp
	Status bool `json:"status"`
}

// IsRecording checks if a stream is being recorded.
func (c *Client) IsRecording(ctx context.Context, req RecordRequest) (bool, error) {
	params := url.Values{}
	params.Set("type", fmt.Sprintf("%d", req.Type))
	params.Set("vhost", req.Vhost)
	params.Set("app", req.App)
	params.Set("stream", req.Stream)

	var resp IsRecordingResponse
	err := c.doRequest(ctx, http.MethodGet, "isRecording", params, nil, &resp)
	if err != nil {
		return false, err
	}
	return resp.Status, nil
}

// IsMediaOnlineResponse represents the response from isMediaOnline API.
type IsMediaOnlineResponse struct {
	ZLMRsp
	Online bool `json:"online"`
}

// IsMediaOnline checks if a media stream is online.
func (c *Client) IsMediaOnline(ctx context.Context, schema, vhost, app, stream string) (bool, error) {
	params := url.Values{}
	params.Set("schema", schema)
	params.Set("vhost", vhost)
	params.Set("app", app)
	params.Set("stream", stream)

	var resp IsMediaOnlineResponse
	err := c.doRequest(ctx, http.MethodGet, "isMediaOnline", params, nil, &resp)
	if err != nil {
		return false, err
	}
	return resp.Online, nil
}
