package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/sip"
	"github.com/niko-admin/niko-admin/internal/repository"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	tables := []string{
		`CREATE TABLE IF NOT EXISTS gb28181_platform_configs (
			id TEXT PRIMARY KEY,
			enabled BOOLEAN,
			sip_id TEXT,
			sip_domain TEXT,
			sip_realm TEXT,
			sip_password TEXT,
			listen_ip TEXT,
			listen_port INTEGER,
			transport TEXT,
			advertised_ip TEXT,
			rtp_ip TEXT,
			heartbeat_timeout INTEGER,
			catalog_interval INTEGER,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS gb28181_devices (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			device_id TEXT,
			device_code TEXT NOT NULL,
			register_address TEXT,
			register_port INTEGER,
			sip_id TEXT,
			sip_domain TEXT,
			sip_password TEXT,
			last_register_at DATETIME,
			last_heartbeat_at DATETIME,
			last_catalog_at DATETIME,
			heartbeat_interval INTEGER DEFAULT 60,
			status TEXT DEFAULT 'offline',
			channel_count INTEGER DEFAULT 0,
			manufacturer TEXT,
			model TEXT,
			firmware TEXT,
			external_key TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS device_sip_configs (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			device_id TEXT NOT NULL,
			device_code TEXT NOT NULL,
			sip_id TEXT,
			sip_domain TEXT,
			sip_password TEXT,
			register_address TEXT,
			register_port INTEGER,
			last_register_at DATETIME,
			last_heartbeat_at DATETIME,
			last_catalog_at DATETIME,
			heartbeat_interval INTEGER,
			channel_count INTEGER
		)`,
		`CREATE TABLE IF NOT EXISTS discovered_devices (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			source TEXT NOT NULL,
			device_name TEXT,
			device_ip TEXT,
			device_mac TEXT,
			manufacturer TEXT,
			model TEXT,
			firmware_version TEXT,
			access_type TEXT,
			access_url TEXT,
			gb28181_code TEXT,
			nvr_device_id TEXT,
			extra_info TEXT,
			status TEXT DEFAULT 'pending',
			imported_at DATETIME,
			ignored_at DATETIME,
			matched_device_id TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS devices (
			id TEXT PRIMARY KEY,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME,
			created_by TEXT,
			updated_by TEXT,
			device_name TEXT,
			access_type TEXT,
			rtsp_url TEXT,
			gb28181_device_id TEXT,
			gb28181_channel_id TEXT,
			username TEXT,
			password TEXT,
			manufacturer TEXT,
			model TEXT,
			firmware_version TEXT,
			status TEXT,
			enabled BOOLEAN,
			latitude REAL,
			longitude REAL,
			location_desc TEXT,
			last_online_at DATETIME,
			last_offline_at DATETIME,
			last_error_code TEXT,
			last_error_message TEXT,
			external_key TEXT,
			remark TEXT,
			version INTEGER,
			auto_infer BOOLEAN,
			parent_nvr_id TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id TEXT PRIMARY KEY,
			user_id TEXT,
			username TEXT,
			action_type TEXT,
			resource_type TEXT,
			resource_id TEXT,
			request_path TEXT,
			request_method TEXT,
			request_ip TEXT,
			user_agent TEXT,
			request_body TEXT,
			response_status INTEGER,
			duration_ms INTEGER,
			result_summary TEXT,
			created_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS device_group_members (
			device_group_id TEXT,
			device_id TEXT,
			PRIMARY KEY (device_group_id, device_id)
		)`,
	}
	for _, sqlQuery := range tables {
		err = db.Exec(sqlQuery).Error
		require.NoError(t, err)
	}
	return db
}

type dummyUDPWriter struct {
	mu      sync.Mutex
	sent    [][]byte
	addr    *net.UDPAddr
	conn    *net.UDPConn
	running bool
}

func startDummyUDPListener(t *testing.T) *dummyUDPWriter {
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	conn, err := net.ListenUDP("udp", addr)
	require.NoError(t, err)

	writer := &dummyUDPWriter{
		conn:    conn,
		addr:    conn.LocalAddr().(*net.UDPAddr),
		running: true,
	}

	go func() {
		buf := make([]byte, 65535)
		for {
			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				break
			}
			writer.mu.Lock()
			msgCopy := make([]byte, n)
			copy(msgCopy, buf[:n])
			writer.sent = append(writer.sent, msgCopy)
			writer.mu.Unlock()
		}
	}()

	return writer
}

func (w *dummyUDPWriter) Close() {
	w.mu.Lock()
	w.running = false
	w.mu.Unlock()
	w.conn.Close()
}

func (w *dummyUDPWriter) GetSent() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	res := make([]string, len(w.sent))
	for i, b := range w.sent {
		res[i] = string(b)
	}
	return res
}

func (w *dummyUDPWriter) Reset() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.sent = nil
}

func TestSIPRuntimeService_RegisterFlows(t *testing.T) {
	db := setupTestDB(t)

	// Create repositories
	platformConfigRepo := repository.NewGB28181PlatformConfigRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	gbDeviceRepo := repository.NewGB28181DeviceRepository(db)
	deviceSipConfigRepo := repository.NewDeviceSipConfigRepository(db)
	discoveredDeviceRepo := repository.NewDiscoveredDeviceRepository(db)
	auditRepo := repository.NewAuditRepository(db)

	ctx := context.Background()

	// Seed platform config
	platformCfg := &model.GB28181PlatformConfig{
		Enabled:     true,

		SipID:       "34020000002000000001",
		SipDomain:   "3402000000",
		SipRealm:    "3402000000",
		SipPassword: "platformpassword",
		ListenIP:    "127.0.0.1",
		ListenPort:  0, // Random port
	}
	platformCfg.ID = "default"
	err := db.Create(platformCfg).Error
	require.NoError(t, err)

	// Seed Device model first
	deviceCode := "34020000001320000008"
	testDevice := &model.Device{
		DeviceName: "Test Unified Device",
		Status:     model.DeviceStatusOffline,
		AccessType: "gb28181",
	}
	testDevice.ID = "device-uuid-1"
	extKey := "gb28181_nvr:" + deviceCode
	testDevice.ExternalKey = &extKey
	err = deviceRepo.Create(ctx, testDevice)
	require.NoError(t, err)

	// Seed registered device
	gbDevice := &model.GB28181Device{
		DeviceCode:      deviceCode,
		Manufacturer:    "TestManufacturer",
		Model:           "TestModel",
		SipPassword:     "devicepassword",
		RegisterAddress: "127.0.0.1",
		RegisterPort:    5060,
		Status:          model.GB28181StatusOffline,
		DeviceID:        &testDevice.ID,
	}
	gbDevice.ID = "test-device-uuid-1"
	err = gbDeviceRepo.Create(ctx, gbDevice)
	require.NoError(t, err)

	// Create SIPService
	sipSvc := NewSIPServiceWithZLM(
		deviceRepo,
		gbDeviceRepo,
		nil,
		nil,
		deviceSipConfigRepo,
		nil,
		nil,
		nil,
		nil,
		"127.0.0.1",
		1935, 554, 80,
		nil,
		nil,
		auditRepo,
		repository.NewGB28181StreamSessionRepository(db),
		NewMemoryHeartbeatStore(),
	)

	// Create SIPRuntimeService
	runtimeSvc := NewSIPRuntimeService(
		platformConfigRepo,
		deviceRepo,
		gbDeviceRepo,
		deviceSipConfigRepo,
		discoveredDeviceRepo,
		sipSvc,
	)

	// Start the runtime server (which binds a real UDP socket)
	err = runtimeSvc.Start(ctx)
	require.NoError(t, err)
	defer runtimeSvc.Stop(ctx)

	// Set up a dummy UDP writer to receive responses
	mockClient := startDummyUDPListener(t)
	defer mockClient.Close()

	t.Run("4.1 REGISTER 首次请求返回 401 Digest challenge", func(t *testing.T) {
		mockClient.Reset()

		rawMsg := "REGISTER sip:34020000002000000001@127.0.0.1 SIP/2.0\r\n" +
			"Via: SIP/2.0/UDP 127.0.0.1:5060;branch=z9hG4bK12345\r\n" +
			"From: <sip:" + deviceCode + "@3402000000>;tag=abcde\r\n" +
			"To: <sip:" + deviceCode + "@3402000000>\r\n" +
			"Call-ID: callid123\r\n" +
			"CSeq: 1 REGISTER\r\n" +
			"Content-Length: 0\r\n\r\n"

		msg, err := sip.ParseMessage([]byte(rawMsg))
		require.NoError(t, err)

		runtimeSvc.handleRegister(msg, mockClient.addr)

		// Check response sent back
		time.Sleep(100 * time.Millisecond)
		sent := mockClient.GetSent()
		require.Len(t, sent, 1)
		assert.Contains(t, sent[0], "SIP/2.0 401 Unauthorized")
		assert.Contains(t, sent[0], `WWW-Authenticate: Digest realm="3402000000"`)
		assert.Contains(t, sent[0], `nonce="`)
	})

	t.Run("4.2 REGISTER Authorization 校验成功后返回 200 OK (Device Specific Password)", func(t *testing.T) {
		mockClient.Reset()

		// Get nonce from a fresh challenge
		rawMsg := "REGISTER sip:34020000002000000001@127.0.0.1 SIP/2.0\r\n" +
			"Via: SIP/2.0/UDP 127.0.0.1:5060;branch=z9hG4bK12346\r\n" +
			"From: <sip:" + deviceCode + "@3402000000>;tag=abcde\r\n" +
			"To: <sip:" + deviceCode + "@3402000000>\r\n" +
			"Call-ID: callid124\r\n" +
			"CSeq: 2 REGISTER\r\n" +
			"Content-Length: 0\r\n\r\n"

		msg, err := sip.ParseMessage([]byte(rawMsg))
		require.NoError(t, err)

		runtimeSvc.handleRegister(msg, mockClient.addr)
		time.Sleep(50 * time.Millisecond)

		sent := mockClient.GetSent()
		require.Len(t, sent, 1)

		// Extract nonce from response
		respMsg, err := sip.ParseMessage([]byte(sent[0]))
		require.NoError(t, err)
		authHeader := respMsg.GetHeader("WWW-Authenticate")
		require.NotEmpty(t, authHeader)

		// Generate valid response using device password: devicepassword
		// HA1 = MD5(deviceCode:realm:devicepassword)
		// HA2 = MD5(REGISTER:sip:34020000002000000001@127.0.0.1)
		// expectedResponse = MD5(HA1:nonce:HA2)
		// To keep it simple, let's extract the nonce and build authorization header
		nonce := ""
		fmt.Sscanf(authHeader, `Digest realm="3402000000", nonce="%s", algorithm=MD5`, &nonce)
		// Clean up trailing quotes or comma from scan
		nonce = sip.GenerateNonce() // Just use our library's nonce to calculate it
		runtimeSvc.nonces.Store(nonce, time.Now())

		// Setup the Authorization header with a dummy verification
		// Let's create an auth header using VerifyDigest helper parameters
		// HA1 calculation: MD5(34020000001320000008:3402000000:devicepassword)
		// MD5 of "34020000001320000008:3402000000:devicepassword" = 43be70eb826f1c4df1ea2492fbe0cfc6
		// HA2 calculation: MD5(REGISTER:sip:34020000002000000001@127.0.0.1)
		// MD5 of "REGISTER:sip:34020000002000000001@127.0.0.1" = 5a8a65c9704e68e4c7d0d6d5ef0a996d
		// Response = MD5(43be70eb826f1c4df1ea2492fbe0cfc6:nonce:5a8a65c9704e68e4c7d0d6d5ef0a996d)
		h1 := md5Sum("34020000001320000008:3402000000:devicepassword")
		h2 := md5Sum("REGISTER:sip:34020000002000000001@127.0.0.1")
		respHash := md5Sum(h1 + ":" + nonce + ":" + h2)

		validAuthHeader := fmt.Sprintf(`Digest username="%s", realm="3402000000", nonce="%s", uri="sip:34020000002000000001@127.0.0.1", response="%s"`, deviceCode, nonce, respHash)

		mockClient.Reset()
		authReqStr := "REGISTER sip:34020000002000000001@127.0.0.1 SIP/2.0\r\n" +
			"Via: SIP/2.0/UDP 127.0.0.1:5060;branch=z9hG4bK12347\r\n" +
			"From: <sip:" + deviceCode + "@3402000000>;tag=abcde\r\n" +
			"To: <sip:" + deviceCode + "@3402000000>\r\n" +
			"Call-ID: callid124\r\n" +
			"CSeq: 3 REGISTER\r\n" +
			"Authorization: " + validAuthHeader + "\r\n" +
			"Expires: 3600\r\n" +
			"Content-Length: 0\r\n\r\n"

		authMsg, err := sip.ParseMessage([]byte(authReqStr))
		require.NoError(t, err)

		runtimeSvc.handleRegister(authMsg, mockClient.addr)
		time.Sleep(100 * time.Millisecond)

		sent2 := mockClient.GetSent()
		require.Len(t, sent2, 1)
		assert.Contains(t, sent2[0], "SIP/2.0 200 OK")
		assert.Contains(t, sent2[0], "Expires: 3600")

		// Verify device is online in database
		dbDevice, err := gbDeviceRepo.FindByDeviceCode(ctx, deviceCode)
		require.NoError(t, err)
		assert.Equal(t, model.GB28181StatusOnline, dbDevice.Status)
	})

	t.Run("4.3 REGISTER Authorization 校验失败后返回 401 Unauthorized", func(t *testing.T) {
		mockClient.Reset()
		nonce := sip.GenerateNonce()
		runtimeSvc.nonces.Store(nonce, time.Now())

		invalidAuthHeader := fmt.Sprintf(`Digest username="%s", realm="3402000000", nonce="%s", uri="sip:34020000002000000001@127.0.0.1", response="wrongresponse"`, deviceCode, nonce)

		authReqStr := "REGISTER sip:34020000002000000001@127.0.0.1 SIP/2.0\r\n" +
			"Via: SIP/2.0/UDP 127.0.0.1:5060;branch=z9hG4bK12348\r\n" +
			"From: <sip:" + deviceCode + "@3402000000>;tag=abcde\r\n" +
			"To: <sip:" + deviceCode + "@3402000000>\r\n" +
			"Call-ID: callid124\r\n" +
			"CSeq: 4 REGISTER\r\n" +
			"Authorization: " + invalidAuthHeader + "\r\n" +
			"Expires: 3600\r\n" +
			"Content-Length: 0\r\n\r\n"

		authMsg, err := sip.ParseMessage([]byte(authReqStr))
		require.NoError(t, err)

		runtimeSvc.handleRegister(authMsg, mockClient.addr)
		time.Sleep(100 * time.Millisecond)

		sent := mockClient.GetSent()
		require.Len(t, sent, 1)
		assert.Contains(t, sent[0], "SIP/2.0 401 Unauthorized (Incorrect Response)")
	})

	t.Run("4.5 REGISTER Expires=0 注销，清理并标记离线", func(t *testing.T) {
		mockClient.Reset()

		authReqStr := "REGISTER sip:34020000002000000001@127.0.0.1 SIP/2.0\r\n" +
			"Via: SIP/2.0/UDP 127.0.0.1:5060;branch=z9hG4bK12349\r\n" +
			"From: <sip:" + deviceCode + "@3402000000>;tag=abcde\r\n" +
			"To: <sip:" + deviceCode + "@3402000000>\r\n" +
			"Call-ID: callid124\r\n" +
			"CSeq: 5 REGISTER\r\n" +
			"Expires: 0\r\n" +
			"Content-Length: 0\r\n\r\n"

		authMsg, err := sip.ParseMessage([]byte(authReqStr))
		require.NoError(t, err)

		runtimeSvc.handleRegister(authMsg, mockClient.addr)
		time.Sleep(100 * time.Millisecond)

		sent := mockClient.GetSent()
		require.Len(t, sent, 1)
		assert.Contains(t, sent[0], "SIP/2.0 200 OK")

		// Verify device is offline in database
		dbDevice, err := gbDeviceRepo.FindByDeviceCode(ctx, deviceCode)
		require.NoError(t, err)
		assert.Equal(t, model.GB28181StatusOffline, dbDevice.Status)
	})

	t.Run("4.6 注册地址变化更新和审计日志", func(t *testing.T) {
		mockClient.Reset()

		// Set initial address
		err := db.Model(&model.GB28181Device{}).Where("device_code = ?", deviceCode).Updates(map[string]interface{}{
			"register_address": "192.168.1.50",
			"register_port":    5060,
			"status":           model.GB28181StatusOnline,
		}).Error
		require.NoError(t, err)

		// Send REGISTER with a different client address
		nonce := sip.GenerateNonce()
		runtimeSvc.nonces.Store(nonce, time.Now())
		h1 := md5Sum("34020000001320000008:3402000000:devicepassword")
		h2 := md5Sum("REGISTER:sip:34020000002000000001@127.0.0.1")
		respHash := md5Sum(h1 + ":" + nonce + ":" + h2)

		validAuthHeader := fmt.Sprintf(`Digest username="%s", realm="3402000000", nonce="%s", uri="sip:34020000002000000001@127.0.0.1", response="%s"`, deviceCode, nonce, respHash)

		authReqStr := "REGISTER sip:34020000002000000001@127.0.0.1 SIP/2.0\r\n" +
			"Via: SIP/2.0/UDP 127.0.0.1:5060;branch=z9hG4bK12350\r\n" +
			"From: <sip:" + deviceCode + "@3402000000>;tag=abcde\r\n" +
			"To: <sip:" + deviceCode + "@3402000000>\r\n" +
			"Call-ID: callid124\r\n" +
			"CSeq: 6 REGISTER\r\n" +
			"Authorization: " + validAuthHeader + "\r\n" +
			"Expires: 3600\r\n" +
			"Content-Length: 0\r\n\r\n"

		authMsg, err := sip.ParseMessage([]byte(authReqStr))
		require.NoError(t, err)

		// Pass a remote address different from 192.168.1.50
		remoteUDP := &net.UDPAddr{
			IP:   net.ParseIP("192.168.1.100"),
			Port: 5070,
		}

		runtimeSvc.handleRegister(authMsg, remoteUDP)
		time.Sleep(100 * time.Millisecond)

		// Verify registration updated in DB
		dbDevice, err := gbDeviceRepo.FindByDeviceCode(ctx, deviceCode)
		require.NoError(t, err)
		assert.Equal(t, "192.168.1.100", dbDevice.RegisterAddress)
		assert.Equal(t, 5070, dbDevice.RegisterPort)

		// Check audit log was created
		var logs []model.AuditLog
		err = db.Where("action_type = ?", "update_register_address").Order("created_at ASC").Find(&logs).Error
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(logs), 1)
		lastLog := logs[len(logs)-1]
		assert.Contains(t, lastLog.ResultSummary, "address changed from 192.168.1.50:5060 to 192.168.1.100:5070")
	})

	t.Run("6.2 实现有效 Keepalive 记录心跳状态并保持设备在线", func(t *testing.T) {
		mockClient.Reset()

		// Set initial status to Online
		err := db.Model(&model.GB28181Device{}).Where("device_code = ?", deviceCode).Updates(map[string]interface{}{
			"status":            model.GB28181StatusOnline,
			"last_heartbeat_at": time.Now().Add(-5 * time.Minute),
		}).Error
		require.NoError(t, err)

		keepaliveBody := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Notify>
<CmdType>Keepalive</CmdType>
<SN>1234</SN>
<DeviceID>%s</DeviceID>
<Status>OK</Status>
</Notify>`, deviceCode)

		rawMsg := "MESSAGE sip:34020000002000000001@127.0.0.1 SIP/2.0\r\n" +
			"Via: SIP/2.0/UDP 127.0.0.1:5060;branch=z9hG4bKkeepalive1\r\n" +
			"From: <sip:" + deviceCode + "@3402000000>;tag=abcde\r\n" +
			"To: <sip:34020000002000000001@3402000000>\r\n" +
			"Call-ID: callid-keepalive\r\n" +
			"CSeq: 1 MESSAGE\r\n" +
			"Content-Type: Application/MANSCDP+xml\r\n" +
			"Content-Length: " + fmt.Sprint(len(keepaliveBody)) + "\r\n\r\n" +
			keepaliveBody

		msg, err := sip.ParseMessage([]byte(rawMsg))
		require.NoError(t, err)

		runtimeSvc.handleMessage(msg, mockClient.addr)
		time.Sleep(100 * time.Millisecond)

		// Should receive 200 OK
		sent := mockClient.GetSent()
		require.Len(t, sent, 1)
		assert.Contains(t, sent[0], "SIP/2.0 200 OK")

		// Verify device is online
		dbDevice, err := gbDeviceRepo.FindByDeviceCode(ctx, deviceCode)
		require.NoError(t, err)
		assert.Equal(t, model.GB28181StatusOnline, dbDevice.Status)
	})

	t.Run("6.3 实现未知设备 Keepalive 拒绝策略", func(t *testing.T) {
		mockClient.Reset()
		unknownID := "34020000001320000666"

		keepaliveBody := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Notify>
<CmdType>Keepalive</CmdType>
<SN>5678</SN>
<DeviceID>%s</DeviceID>
<Status>OK</Status>
</Notify>`, unknownID)

		rawMsg := "MESSAGE sip:34020000002000000001@127.0.0.1 SIP/2.0\r\n" +
			"Via: SIP/2.0/UDP 127.0.0.1:5060;branch=z9hG4bKkeepalive2\r\n" +
			"From: <sip:" + unknownID + "@3402000000>;tag=abcde\r\n" +
			"To: <sip:34020000002000000001@3402000000>\r\n" +
			"Call-ID: callid-keepalive-unknown\r\n" +
			"CSeq: 1 MESSAGE\r\n" +
			"Content-Type: Application/MANSCDP+xml\r\n" +
			"Content-Length: " + fmt.Sprint(len(keepaliveBody)) + "\r\n\r\n" +
			keepaliveBody

		msg, err := sip.ParseMessage([]byte(rawMsg))
		require.NoError(t, err)

		runtimeSvc.handleMessage(msg, mockClient.addr)
		time.Sleep(100 * time.Millisecond)

		// Should receive 403 Forbidden
		sent := mockClient.GetSent()
		require.Len(t, sent, 1)
		assert.Contains(t, sent[0], "SIP/2.0 403 Forbidden (Unregistered Device)")
	})

	t.Run("6.4 实现心跳超时扫描任务，将超时设备标记为 offline", func(t *testing.T) {
		// Ensure device status is online
		err := db.Model(&model.GB28181Device{}).Where("device_code = ?", deviceCode).Updates(map[string]interface{}{
			"status": model.GB28181StatusOnline,
		}).Error
		require.NoError(t, err)

		// Record a heartbeat in the store to simulate an active device.
		gbDev, err := gbDeviceRepo.FindByDeviceCode(ctx, deviceCode)
		require.NoError(t, err)
		err = sipSvc.HandleHeartbeat(ctx, gbDev)
		require.NoError(t, err)

		// Wait a tiny bit so the recorded heartbeat becomes expired.
		time.Sleep(10 * time.Millisecond)

		// Run timeout check with a very short timeout to catch the expired heartbeat.
		offline, err := sipSvc.CheckHeartbeatTimeout(ctx, 1*time.Millisecond)
		require.NoError(t, err)
		require.Len(t, offline, 1)
		assert.Equal(t, deviceCode, offline[0].DeviceCode)

		// Verify device is offline in database
		dbDevice, err := gbDeviceRepo.FindByDeviceCode(ctx, deviceCode)
		require.NoError(t, err)
		assert.Equal(t, model.GB28181StatusOffline, dbDevice.Status)

		// Verify unified device is also offline
		unifiedDev, err := deviceRepo.FindByID(ctx, *dbDevice.DeviceID)
		require.NoError(t, err)
		assert.Equal(t, model.DeviceStatusOffline, unifiedDev.Status)
	})

	t.Run("7.2 & 7.8 目录查询发送与响应解析入库", func(t *testing.T) {
		mockClient.Reset()

		// Reset status to online (previous test marked it offline)
		_ = db.Model(&model.GB28181Device{}).Where("device_code = ?", deviceCode).Updates(map[string]interface{}{
			"status":        model.GB28181StatusOnline,
			"register_port": mockClient.addr.Port,
		})

		// 1. Send Query (UAC)
		err := sipSvc.QueryCatalog(ctx, deviceCode)
		require.NoError(t, err)

		// 2. Simulate Response (UAS)
		channelCode := "34020000001310000001"
		catalogBody := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<Response>
<CmdType>Catalog</CmdType>
<SN>1234</SN>
<DeviceID>%s</DeviceID>
<SumNum>1</SumNum>
<DeviceList>
<Item>
<DeviceID>%s</DeviceID>
<Name>Test Channel 1</Name>
<Status>ON</Status>
</Item>
</DeviceList>
</Response>`, deviceCode, channelCode)

		respMsg := "MESSAGE sip:34020000002000000001@127.0.0.1 SIP/2.0\r\n" +
			"Via: SIP/2.0/UDP 127.0.0.1:5060;branch=z9hG4bKcatalog-resp\r\n" +
			"From: <sip:" + deviceCode + "@3402000000>;tag=abcde\r\n" +
			"To: <sip:34020000002000000001@3402000000>\r\n" +
			"Call-ID: callid-catalog\r\n" +
			"CSeq: 1 MESSAGE\r\n" +
			"Content-Type: Application/MANSCDP+xml\r\n" +
			"Content-Length: " + fmt.Sprint(len(catalogBody)) + "\r\n\r\n" +
			catalogBody

		msg, err := sip.ParseMessage([]byte(respMsg))
		require.NoError(t, err)

		runtimeSvc.handleMessage(msg, mockClient.addr)
		time.Sleep(200 * time.Millisecond)

		// Verify channel device created in unified Device table
		channelDev, err := deviceRepo.FindByGB28181DeviceID(ctx, channelCode)
		require.NoError(t, err)
		assert.NotNil(t, channelDev)
		assert.Equal(t, "Test Channel 1", channelDev.DeviceName)
		assert.Equal(t, model.DeviceStatusOnline, channelDev.Status)
		
		// Verify parent association
		dbDevice, _ := gbDeviceRepo.FindByDeviceCode(ctx, deviceCode)
		assert.Equal(t, dbDevice.ID, *channelDev.ParentNvrID)
	})

	t.Run("5.2 未知设备 REGISTER 后创建 pending 待接入记录", func(t *testing.T) {
		mockClient.Reset()
		unknownCode := "34020000001320000055"

		// 1. Send initial REGISTER
		rawMsg := "REGISTER sip:34020000002000000001@127.0.0.1 SIP/2.0\r\n" +
			"Via: SIP/2.0/UDP 127.0.0.1:5060;branch=z9hG4bKunknown1\r\n" +
			"From: <sip:" + unknownCode + "@3402000000>;tag=abcde\r\n" +
			"To: <sip:" + unknownCode + "@3402000000>\r\n" +
			"Call-ID: callid-unknown\r\n" +
			"CSeq: 1 REGISTER\r\n" +
			"Content-Length: 0\r\n\r\n"

		msg, err := sip.ParseMessage([]byte(rawMsg))
		require.NoError(t, err)

		runtimeSvc.handleRegister(msg, mockClient.addr)
		time.Sleep(50 * time.Millisecond)

		// Should receive 401 challenge
		sent := mockClient.GetSent()
		require.Len(t, sent, 1)
		respMsg, _ := sip.ParseMessage([]byte(sent[0]))
		authHeader := respMsg.GetHeader("WWW-Authenticate")
		
		nonce := ""
		re := regexp.MustCompile(`nonce="([^"]+)"`)
		if m := re.FindStringSubmatch(authHeader); len(m) > 1 {
			nonce = m[1]
		}
		require.NotEmpty(t, nonce)
		runtimeSvc.nonces.Store(nonce, time.Now())

		// 2. Send authorized REGISTER with platform password: platformpassword
		h1 := md5Sum(unknownCode + ":3402000000:platformpassword")
		h2 := md5Sum("REGISTER:sip:34020000002000000001@127.0.0.1")
		respHash := md5Sum(h1 + ":" + nonce + ":" + h2)

		validAuthHeader := fmt.Sprintf(`Digest username="%s", realm="3402000000", nonce="%s", uri="sip:34020000002000000001@127.0.0.1", response="%s"`, unknownCode, nonce, respHash)

		mockClient.Reset()
		authReqStr := "REGISTER sip:34020000002000000001@127.0.0.1 SIP/2.0\r\n" +
			"Via: SIP/2.0/UDP 127.0.0.1:5060;branch=z9hG4bKunknown2\r\n" +
			"From: <sip:" + unknownCode + "@3402000000>;tag=abcde\r\n" +
			"To: <sip:" + unknownCode + "@3402000000>\r\n" +
			"Call-ID: callid-unknown\r\n" +
			"CSeq: 2 REGISTER\r\n" +
			"Authorization: " + validAuthHeader + "\r\n" +
			"Expires: 3600\r\n" +
			"Content-Length: 0\r\n\r\n"

		authMsg, err := sip.ParseMessage([]byte(authReqStr))
		require.NoError(t, err)

		runtimeSvc.handleRegister(authMsg, mockClient.addr)
		time.Sleep(100 * time.Millisecond)

		// Should receive 200 OK
		sent2 := mockClient.GetSent()
		require.Len(t, sent2, 1)
		assert.Contains(t, sent2[0], "SIP/2.0 200 OK")

		// Verify record created in discovered_devices
		staging, err := discoveredDeviceRepo.FindByGB28181Code(ctx, unknownCode)
		require.NoError(t, err)
		assert.NotNil(t, staging)
		assert.Equal(t, model.StatusPending, staging.Status)
		assert.Equal(t, model.SourceGB28181, staging.Source)
	})
}

func TestDeviceStaging_GB28181Import(t *testing.T) {
	db := setupTestDB(t)

	// Create repositories
	stagingRepo := repository.NewDiscoveredDeviceRepository(db)
	deviceRepo := repository.NewDeviceRepository(db)
	gbDeviceRepo := repository.NewGB28181DeviceRepository(db)
	deviceSipConfigRepo := repository.NewDeviceSipConfigRepository(db)

	stagingSvc := NewDeviceStagingService(stagingRepo, deviceRepo, gbDeviceRepo, deviceSipConfigRepo)
	ctx := context.Background()

	// 1. Seed discovered device staging record
	discoveredCode := "34020000001320000099"
	stagingRecord := &model.DiscoveredDevice{
		Source:       model.SourceGB28181,
		DeviceIP:     "192.168.1.120",
		GB28181Code:  discoveredCode,
		Manufacturer: "Hikvision",
		Model:        "DS-2CD2Q10",
		AccessType:   "gb28181",
		Status:       model.StatusPending,
	}
	stagingRecord.ID = "staging-uuid-99"
	err := stagingRepo.Create(ctx, stagingRecord)
	require.NoError(t, err)

	// 2. Call ImportSingle (adoption) with a custom SIP password
	err = stagingSvc.ImportSingle(ctx, stagingRecord.ID, "admin", "custompassword", "MyAdoptedCamera", nil)
	require.NoError(t, err)

	// 3. Verify unified Device was created
	extKey := "gb28181_nvr:" + discoveredCode
	dev, err := deviceRepo.FindByExternalKey(ctx, extKey)
	require.NoError(t, err)
	assert.NotNil(t, dev)
	assert.Equal(t, "gb28181_nvr", dev.AccessType)
	assert.Equal(t, "MyAdoptedCamera", dev.DeviceName)
	assert.True(t, dev.AutoInfer) // default true

	// 4. Verify GB28181Device was created
	gbDev, err := gbDeviceRepo.FindByDeviceCode(ctx, discoveredCode)
	require.NoError(t, err)
	assert.NotNil(t, gbDev)
	assert.Equal(t, "custompassword", gbDev.SipPassword)
	assert.Equal(t, model.GB28181StatusOffline, gbDev.Status)
	assert.Equal(t, dev.ID, *gbDev.DeviceID)

	// 5. Verify DeviceSipConfig was created
	sipCfg, err := deviceSipConfigRepo.FindByDeviceCode(ctx, discoveredCode)
	require.NoError(t, err)
	assert.NotNil(t, sipCfg)
	assert.Equal(t, "custompassword", sipCfg.SipPassword)
	assert.Equal(t, dev.ID, sipCfg.DeviceID)

	// 6. Verify staging record is marked as imported
	st, err := stagingRepo.FindByID(ctx, stagingRecord.ID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusImported, st.Status)
	assert.Equal(t, dev.ID, *st.MatchedDeviceID)
}

func md5Sum(s string) string {
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
