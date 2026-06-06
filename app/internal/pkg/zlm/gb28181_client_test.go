package zlm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestClient 创建一个指向 mock HTTP 服务器的 ZLM Client
func newTestClient(handler http.HandlerFunc) (*Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	return NewClient(server.URL, "test-secret", nil), server
}

func TestOpenRtpServer_Success(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		// 验证路径
		if !strings.Contains(r.URL.Path, "openRtpServer") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// 验证 secret
		if r.URL.Query().Get("secret") != "test-secret" {
			t.Error("missing or wrong secret")
		}
		resp := map[string]interface{}{
			"code": 0,
			"port": 55463,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	port, err := client.OpenRtpServer(context.Background(), OpenRtpServerRequest{
		Port:     0,
		TCPMode:  0,
		StreamID: "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if port != 55463 {
		t.Errorf("expected port 55463, got %d", port)
	}
}

func TestStartSendRtp_Success(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "startSendRtp") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// 验证请求体
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["dst_url"] != "192.168.1.100" {
			t.Errorf("wrong dst_url: %v", body["dst_url"])
		}
		if body["is_udp"] != float64(1) {
			t.Errorf("wrong is_udp: %v", body["is_udp"])
		}
		resp := map[string]interface{}{
			"code":       0,
			"local_port": 57152,
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	localPort, err := client.StartSendRtp(context.Background(), StartSendRtpRequest{
		Vhost:   "__defaultVhost__",
		App:     "live",
		Stream:  "test",
		SSRC:    "1",
		DstURL:  "192.168.1.100",
		DstPort: 10000,
		IsUDP:   1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if localPort != 57152 {
		t.Errorf("expected local_port 57152, got %d", localPort)
	}
}

func TestStopSendRtp_Success(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "stopSendRtp") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"code":0}`))
	})
	defer server.Close()

	err := client.StopSendRtp(context.Background(), StopSendRtpRequest{
		Vhost:  "__defaultVhost__",
		App:    "live",
		Stream: "test",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestListRtpServer_Success(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"code": 0,
			"data": []map[string]interface{}{
				{"port": 52183, "stream_id": "test"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	servers, err := client.ListRtpServer(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(servers) != 1 || servers[0].Port != 52183 {
		t.Errorf("unexpected result: %+v", servers)
	}
}

func TestCloseRtpServer_Success(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "closeRtpServer") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"code":0,"hit":1}`))
	})
	defer server.Close()

	if err := client.CloseRtpServer(context.Background(), "test"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSetRecordSpeed_Success(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "setRecordSpeed") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"code":0}`))
	})
	defer server.Close()

	err := client.SetRecordSpeed(context.Background(), SetRecordSpeedRequest{
		Vhost:  "__defaultVhost__",
		App:    "live",
		Stream: "obs",
		Speed:  2.0,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestZLMError_Returned(t *testing.T) {
	client, server := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":-1,"msg":"stream not found"}`))
	})
	defer server.Close()

	_, err := client.OpenRtpServer(context.Background(), OpenRtpServerRequest{
		Port:     0,
		TCPMode:  0,
		StreamID: "missing",
	})
	if err == nil {
		t.Error("expected error, got nil")
	}
}
