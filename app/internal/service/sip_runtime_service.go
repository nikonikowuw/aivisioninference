package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/niko-admin/niko-admin/internal/model"
	"github.com/niko-admin/niko-admin/internal/pkg/sip"
	"github.com/niko-admin/niko-admin/internal/repository"
)

type SIPRuntimeService struct {
	platformConfigRepo   *repository.GB28181PlatformConfigRepository
	deviceRepo           *repository.DeviceRepository
	gbDeviceRepo         *repository.GB28181DeviceRepository
	deviceSipConfigRepo  *repository.DeviceSipConfigRepository
	discoveredDeviceRepo *repository.DiscoveredDeviceRepository
	sipSvc               *SIPService

	server   *sip.Server
	serverMu sync.RWMutex

	// Keepalive & Registration tracking
	nonces sync.Map // nonce -> timestamp

	// Lifecycle control via context cancel
	cancel   context.CancelFunc
	cancelMu sync.Mutex

	// Restart serialization
	restartMu     sync.Mutex
	restartRunning bool
}

func NewSIPRuntimeService(
	platformConfigRepo *repository.GB28181PlatformConfigRepository,
	deviceRepo *repository.DeviceRepository,
	gbDeviceRepo *repository.GB28181DeviceRepository,
	deviceSipConfigRepo *repository.DeviceSipConfigRepository,
	discoveredDeviceRepo *repository.DiscoveredDeviceRepository,
	sipSvc *SIPService,
) *SIPRuntimeService {
	s := &SIPRuntimeService{
		platformConfigRepo:   platformConfigRepo,
		deviceRepo:           deviceRepo,
		gbDeviceRepo:         gbDeviceRepo,
		deviceSipConfigRepo:  deviceSipConfigRepo,
		discoveredDeviceRepo: discoveredDeviceRepo,
		sipSvc:               sipSvc,
	}
	return s
}

func (s *SIPRuntimeService) NotifyConfigChange(ctx context.Context, needRestart bool) {
	zap.L().Info("SIP config changed", zap.Bool("needRestart", needRestart))
	if needRestart {
		go func() {
			if err := s.Restart(context.Background()); err != nil {
				zap.L().Error("SIP runtime restart failed", zap.Error(err))
			}
		}()
	}
}

func (s *SIPRuntimeService) Start(ctx context.Context) error {
	s.serverMu.Lock()
	defer s.serverMu.Unlock()

	cfg, err := s.platformConfigRepo.Get(ctx)
	if err != nil {
		return fmt.Errorf("load platform config: %w", err)
	}

	if !cfg.Enabled {
		zap.L().Info("Go SIP Runtime is disabled; listener will not start")
		return nil
	}

	if s.server != nil && s.server.IsRunning() {
		return nil
	}

	// 11.2 实现启动前端口冲突检查
	if err := s.checkPortConflict(cfg.ListenIP, cfg.ListenPort); err != nil {
		zap.L().Error("SIP port conflict detected", zap.String("ip", cfg.ListenIP), zap.Int("port", cfg.ListenPort), zap.Error(err))
		return fmt.Errorf("port %d is already in use: %w", cfg.ListenPort, err)
	}

	handlers := &sip.Handlers{
		OnRequest: s.handleRequest,
	}

	s.server = sip.NewServer(cfg.ListenIP, cfg.ListenPort, handlers)
	err = s.server.Start()
	if err != nil {
		zap.L().Error("Failed to start SIP listener", zap.Error(err))
		return err
	}

	// Start heartbeat and cleanup loops with cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	s.cancelMu.Lock()
	s.cancel = cancel
	s.cancelMu.Unlock()
	go s.heartbeatLoop(ctx, cfg.HeartbeatTimeout)
	go s.sessionCleanupLoop(ctx)

	return nil
}

func (s *SIPRuntimeService) Stop(ctx context.Context) error {
	s.serverMu.Lock()
	defer s.serverMu.Unlock()

	// Cancel background goroutines
	s.cancelMu.Lock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.cancelMu.Unlock()

	if s.server != nil {
		s.server.Stop()
	}
	return nil
}

func (s *SIPRuntimeService) Restart(ctx context.Context) error {
	s.restartMu.Lock()
	if s.restartRunning {
		s.restartMu.Unlock()
		zap.L().Warn("SIP runtime restart already in progress, skipping")
		return nil
	}
	s.restartRunning = true
	s.restartMu.Unlock()

	defer func() {
		s.restartMu.Lock()
		s.restartRunning = false
		s.restartMu.Unlock()
	}()

	if err := s.Stop(ctx); err != nil {
		zap.L().Warn("SIP runtime stop before restart failed", zap.Error(err))
	}
	// Add a tiny sleep to allow socket cleanup
	time.Sleep(200 * time.Millisecond)
	return s.Start(ctx)
}

// IsRunning returns true if the SIP server is currently running.
func (s *SIPRuntimeService) IsRunning() bool {
	s.serverMu.RLock()
	defer s.serverMu.RUnlock()
	return s.server != nil && s.server.IsRunning()
}

func (s *SIPRuntimeService) GetStatus() map[string]interface{} {
	s.serverMu.RLock()
	srv := s.server
	s.serverMu.RUnlock()

	status := "stopped"
	addr := ""
	var startTime time.Time
	var lastErr error

	if srv != nil {
		status, addr, startTime, lastErr = srv.GetState()
	}

	res := map[string]interface{}{
		"status":     status,
		"addr":       addr,
		"start_time": startTime,
	}
	if lastErr != nil {
		res["error"] = lastErr.Error()
	}

	return res
}

func (s *SIPRuntimeService) heartbeatLoop(ctx context.Context, timeout int) {
	if timeout <= 0 {
		timeout = 180 // Default 3 minutes
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	zap.L().Info("SIP heartbeat timeout scanner started", zap.Int("timeout_seconds", timeout))

	for {
		select {
		case <-ticker.C:
			timeoutCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			offline, err := s.sipSvc.CheckHeartbeatTimeout(timeoutCtx, time.Duration(timeout)*time.Second)
			cancel()
			if err != nil {
				zap.L().Error("SIP heartbeat timeout check failed", zap.Error(err))
			} else if len(offline) > 0 {
				zap.L().Info("SIP heartbeat timeout scanner found offline devices", zap.Int("count", len(offline)))
			}
		case <-ctx.Done():
			zap.L().Info("SIP heartbeat timeout scanner stopped")
			return
		}
	}
}

func (s *SIPRuntimeService) sessionCleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			zap.L().Debug("Running SIP session cleanup")
		case <-ctx.Done():
			return
		}
	}
}

func (s *SIPRuntimeService) checkPortConflict(ip string, port int) error {
	addr := fmt.Sprintf("%s:%d", ip, port)
	l, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}
	_ = l.Close()
	return nil
}

// ====== UAC Methods (Outgoing Requests) ======

func (s *SIPRuntimeService) SendCatalogQuery(ctx context.Context, deviceID string) (string, error) {
	gbDev, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
	if err != nil {
		return "", err
	}
	if gbDev.Status != model.GB28181StatusOnline || gbDev.RegisterAddress == "" {
		return "", fmt.Errorf("device is offline")
	}

	cfg, _ := s.platformConfigRepo.Get(ctx)
	sn := s.nextSN()
	xmlBody := CatalogueQueryXML(strconv.Itoa(sn))

	recipient := fmt.Sprintf("sip:%s@%s:%d", deviceID, gbDev.RegisterAddress, gbDev.RegisterPort)
	req := s.newRequest("MESSAGE", recipient)
	req.Headers["Content-Type"] = "Application/MANSCDP+xml"
	req.Body = []byte(xmlBody)

	req.Headers["From"] = fmt.Sprintf("<sip:%s@%s>;tag=%s", cfg.SipID, cfg.SipDomain, sip.GenerateNonce()[:8])
	req.Headers["To"] = fmt.Sprintf("<sip:%s@%s>", deviceID, cfg.SipDomain)

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", gbDev.RegisterAddress, gbDev.RegisterPort))
	if err != nil {
		return "", err
	}

	err = s.server.Send(req, addr)
	if err != nil {
		return "", err
	}

	return strconv.Itoa(sn), nil
}

func (s *SIPRuntimeService) SendInvite(ctx context.Context, deviceID, channelID string, rtpPort int, ssrc string) (string, error) {
	gbDev, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
	if err != nil {
		return "", err
	}
	if gbDev.Status != model.GB28181StatusOnline || gbDev.RegisterAddress == "" {
		return "", fmt.Errorf("device is offline")
	}

	cfg, _ := s.platformConfigRepo.Get(ctx)

	// Generate SDP
	rtpIP := cfg.RtpIP
	if rtpIP == "" {
		rtpIP = cfg.ListenIP
	}

	sdp := s.buildLiveSDP(cfg.SipID, rtpIP, rtpPort, ssrc)

	recipient := fmt.Sprintf("sip:%s@%s:%d", channelID, gbDev.RegisterAddress, gbDev.RegisterPort)
	req := s.newRequest("INVITE", recipient)
	req.Headers["Content-Type"] = "application/sdp"
	req.Headers["Subject"] = fmt.Sprintf("%s:%s,%s:0", channelID, ssrc, cfg.SipID)
	req.Body = []byte(sdp)

	req.Headers["From"] = fmt.Sprintf("<sip:%s@%s>;tag=%s", cfg.SipID, cfg.SipDomain, sip.GenerateNonce()[:8])
	req.Headers["To"] = fmt.Sprintf("<sip:%s@%s>", channelID, cfg.SipDomain)
	req.Headers["Contact"] = fmt.Sprintf("<sip:%s@%s:%d>", cfg.SipID, cfg.ListenIP, cfg.ListenPort)

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", gbDev.RegisterAddress, gbDev.RegisterPort))
	if err != nil {
		return "", err
	}

	err = s.server.Send(req, addr)
	if err != nil {
		return "", err
	}

	return req.GetHeader("Call-ID"), nil
}

func (s *SIPRuntimeService) SendPlaybackInvite(ctx context.Context, deviceID, channelID string, rtpPort int, ssrc string, start, end time.Time) (string, error) {
	gbDev, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
	if err != nil {
		return "", err
	}
	if gbDev.Status != model.GB28181StatusOnline || gbDev.RegisterAddress == "" {
		return "", fmt.Errorf("device is offline")
	}

	cfg, _ := s.platformConfigRepo.Get(ctx)
	
	// Generate SDP
	rtpIP := cfg.RtpIP
	if rtpIP == "" {
		rtpIP = cfg.ListenIP
	}

	sdp := s.buildPlaybackSDP(cfg.SipID, rtpIP, rtpPort, ssrc, start, end)

	recipient := fmt.Sprintf("sip:%s@%s:%d", channelID, gbDev.RegisterAddress, gbDev.RegisterPort)
	req := s.newRequest("INVITE", recipient)
	req.Headers["Content-Type"] = "application/sdp"
	req.Headers["Subject"] = fmt.Sprintf("%s:%s,%s:0", channelID, ssrc, cfg.SipID)
	req.Body = []byte(sdp)

	req.Headers["From"] = fmt.Sprintf("<sip:%s@%s>;tag=%s", cfg.SipID, cfg.SipDomain, sip.GenerateNonce()[:8])
	req.Headers["To"] = fmt.Sprintf("<sip:%s@%s>", channelID, cfg.SipDomain)
	req.Headers["Contact"] = fmt.Sprintf("<sip:%s@%s:%d>", cfg.SipID, cfg.ListenIP, cfg.ListenPort)

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", gbDev.RegisterAddress, gbDev.RegisterPort))
	if err != nil {
		return "", err
	}

	err = s.server.Send(req, addr)
	if err != nil {
		return "", err
	}

	return req.GetHeader("Call-ID"), nil
}

func (s *SIPRuntimeService) SendPlaybackControl(ctx context.Context, deviceID, channelID, callID, action string, speed float64, stamp int64) error {
	gbDev, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
	if err != nil {
		return err
	}

	recipient := fmt.Sprintf("sip:%s@%s:%d", channelID, gbDev.RegisterAddress, gbDev.RegisterPort)
	req := s.newRequest("INFO", recipient)
	req.Headers["Call-ID"] = callID
	req.Headers["Content-Type"] = "Application/MANSRTSP"

	var body string
	switch action {
	case "pause":
		body = fmt.Sprintf("PAUSE RTSP/1.0\r\nCSeq: %d\r\nPauseTime: now\r\n", s.nextSN())
	case "play":
		body = fmt.Sprintf("PLAY RTSP/1.0\r\nCSeq: %d\r\nRange: npt=now-\r\nScale: %.2f\r\n", s.nextSN(), speed)
	case "seek":
		body = fmt.Sprintf("PLAY RTSP/1.0\r\nCSeq: %d\r\nRange: npt=%d-\r\n", s.nextSN(), stamp)
	default:
		return fmt.Errorf("unsupported playback action: %s", action)
	}
	req.Body = []byte(body)

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", gbDev.RegisterAddress, gbDev.RegisterPort))
	if err != nil {
		return err
	}

	return s.server.Send(req, addr)
}

func (s *SIPRuntimeService) SendBye(ctx context.Context, deviceID, channelID, callID string) error {
	gbDev, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
	if err != nil {
		return err
	}

	cfg, _ := s.platformConfigRepo.Get(ctx)
	recipient := fmt.Sprintf("sip:%s@%s:%d", channelID, gbDev.RegisterAddress, gbDev.RegisterPort)
	req := s.newRequest("BYE", recipient)
	req.Headers["Call-ID"] = callID

	req.Headers["From"] = fmt.Sprintf("<sip:%s@%s>;tag=%s", cfg.SipID, cfg.SipDomain, sip.GenerateNonce()[:8])
	req.Headers["To"] = fmt.Sprintf("<sip:%s@%s>", channelID, cfg.SipDomain)

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", gbDev.RegisterAddress, gbDev.RegisterPort))
	if err != nil {
		return err
	}

	return s.server.Send(req, addr)
}

func (s *SIPRuntimeService) buildLiveSDP(sipID, rtpIP string, rtpPort int, ssrc string) string {
	return fmt.Sprintf("v=0\r\n"+
		"o=%s 0 0 IN IP4 %s\r\n"+
		"s=Play\r\n"+
		"c=IN IP4 %s\r\n"+
		"t=0 0\r\n"+
		"m=video %d RTP/AVP 96 97 98\r\n"+
		"a=recvonly\r\n"+
		"a=rtpmap:96 PS/90000\r\n"+
		"a=rtpmap:97 MPEG4/90000\r\n"+
		"a=rtpmap:98 H264/90000\r\n"+
		"y=%s\r\n", sipID, rtpIP, rtpIP, rtpPort, ssrc)
}

func (s *SIPRuntimeService) buildPlaybackSDP(sipID, rtpIP string, rtpPort int, ssrc string, start, end time.Time) string {
	return fmt.Sprintf("v=0\r\n"+
		"o=%s 0 0 IN IP4 %s\r\n"+
		"s=Playback\r\n"+
		"u=%s:0\r\n"+
		"c=IN IP4 %s\r\n"+
		"t=%d %d\r\n"+
		"m=video %d RTP/AVP 96 97 98\r\n"+
		"a=recvonly\r\n"+
		"a=rtpmap:96 PS/90000\r\n"+
		"y=%s\r\n", sipID, rtpIP, sipID, rtpIP, start.Unix(), end.Unix(), rtpPort, ssrc)
}

func (s *SIPRuntimeService) nextSN() int {
	return int(time.Now().UnixNano() % 1000000)
}

func (s *SIPRuntimeService) newRequest(method, recipient string) *sip.Message {
	msg := &sip.Message{
		IsRequest: true,
		Method:    method,
		Recipient: recipient,
		Headers:   make(map[string]string),
	}
	msg.Headers["Call-ID"] = sip.GenerateNonce()
	msg.Headers["CSeq"] = "1 " + method
	msg.Headers["Max-Forwards"] = "70"
	msg.Headers["User-Agent"] = "GoSIPServer/1.0"
	return msg
}

// ====== UAS Methods (Incoming Requests/Responses) ======

func (s *SIPRuntimeService) handleRequest(msg *sip.Message, remoteAddr *net.UDPAddr) {
	if !msg.IsRequest {
		s.handleResponse(msg, remoteAddr)
		return
	}
	switch msg.Method {
	case "REGISTER":
		s.handleRegister(msg, remoteAddr)
	case "MESSAGE":
		s.handleMessage(msg, remoteAddr)
	case "BYE":
		s.handleBye(msg, remoteAddr)
	default:
		resp := s.newResponse(msg, 501, "Not Implemented")
		_ = s.server.Send(resp, remoteAddr)
	}
}

func (s *SIPRuntimeService) handleResponse(msg *sip.Message, remoteAddr *net.UDPAddr) {
	cseq := msg.GetHeader("CSeq")
	if strings.Contains(cseq, "INVITE") && msg.StatusCode == 200 {
		zap.L().Info("INVITE 200 OK received, sending ACK", zap.String("call_id", msg.GetHeader("Call-ID")))

		ack := s.newRequest("ACK", msg.Recipient)
		ack.Headers["Call-ID"] = msg.GetHeader("Call-ID")
		ack.Headers["From"] = msg.GetHeader("From")
		ack.Headers["To"] = msg.GetHeader("To")
		ack.Headers["Via"] = msg.GetHeader("Via")
		ack.Headers["CSeq"] = "1 ACK"

		_ = s.server.Send(ack, remoteAddr)

		ctx := context.Background()
		_ = s.sipSvc.HandleInviteOK(ctx, msg.GetHeader("Call-ID"), string(msg.Body))
	}
}

func (s *SIPRuntimeService) handleRegister(msg *sip.Message, remoteAddr *net.UDPAddr) {
	ctx := context.Background()
	toHeader := msg.GetHeader("To")
	deviceID := extractSipID(toHeader)
	if len(deviceID) != 20 {
		resp := s.newResponse(msg, 400, "Bad Request (Invalid Device ID)")
		_ = s.server.Send(resp, remoteAddr)
		return
	}

	authHeader := msg.GetHeader("Authorization")
	expiresHeader := msg.GetHeader("Expires")
	expires := 3600
	if expiresHeader != "" {
		if val, err := strconv.Atoi(expiresHeader); err == nil {
			expires = val
		}
	}

	if expires == 0 {
		_ = s.sipSvc.HandleUnregister(ctx, deviceID)
		resp := s.newResponse(msg, 200, "OK")
		_ = s.server.Send(resp, remoteAddr)
		return
	}

	platformCfg, err := s.platformConfigRepo.Get(ctx)
	if err != nil {
		resp := s.newResponse(msg, 500, "Internal Server Error")
		_ = s.server.Send(resp, remoteAddr)
		return
	}

	if authHeader == "" {
		s.send401Challenge(msg, platformCfg.SipRealm, remoteAddr)
		return
	}

	password := platformCfg.SipPassword
	gbDev, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
	if err == nil && gbDev != nil && gbDev.SipPassword != "" {
		password = gbDev.SipPassword
	}

	if !sip.VerifyDigest(authHeader, "REGISTER", password) {
		zap.L().Warn("SIP Register Digest verification failed", zap.String("device", deviceID))
		resp := s.newResponse(msg, 401, "Unauthorized (Incorrect Response)")
		_ = s.server.Send(resp, remoteAddr)
		return
	}

	if err != nil || gbDev == nil {
		s.handleUnknownDeviceRegister(ctx, deviceID, remoteAddr)
	} else {
		_ = s.sipSvc.HandleRegister(ctx, deviceID, remoteAddr.IP.String(), remoteAddr.Port)
	}

	resp := s.newResponse(msg, 200, "OK")
	resp.Headers["Date"] = time.Now().UTC().Format(time.RFC1123)
	resp.Headers["Expires"] = strconv.Itoa(expires)
	_ = s.server.Send(resp, remoteAddr)
}

func (s *SIPRuntimeService) send401Challenge(msg *sip.Message, realm string, remoteAddr *net.UDPAddr) {
	resp := s.newResponse(msg, 401, "Unauthorized")
	nonce := sip.GenerateNonce()
	s.nonces.Store(nonce, time.Now())
	resp.Headers["WWW-Authenticate"] = fmt.Sprintf(`Digest realm="%s", nonce="%s", algorithm=MD5`, realm, nonce)
	_ = s.server.Send(resp, remoteAddr)
}

func (s *SIPRuntimeService) handleUnknownDeviceRegister(ctx context.Context, deviceID string, remoteAddr *net.UDPAddr) {
	if s.discoveredDeviceRepo == nil {
		return
	}
	staging, err := s.discoveredDeviceRepo.FindByGB28181Code(ctx, deviceID)
	now := time.Now()
	if err != nil || staging == nil {
		staging = &model.DiscoveredDevice{
			GB28181Code:     deviceID,
			AccessType:      "gb28181",
			DeviceIP:        remoteAddr.IP.String(),
			Status:          model.StatusPending,
			Source:          model.SourceGB28181,
			DeviceName:      "GB28181-" + deviceID,
		}
		staging.CreatedAt = now
		staging.UpdatedAt = now
		_ = s.discoveredDeviceRepo.Create(ctx, staging)
	} else {
		if staging.Status == model.StatusIgnored {
			updates := map[string]interface{}{"updated_at": now, "device_ip": remoteAddr.IP.String()}
			_ = s.discoveredDeviceRepo.Update(ctx, staging.ID, updates)
		} else {
			updates := map[string]interface{}{"updated_at": now, "status": model.StatusPending, "device_ip": remoteAddr.IP.String()}
			_ = s.discoveredDeviceRepo.Update(ctx, staging.ID, updates)
		}
	}
}

type KeepaliveXML struct {
	XMLName  xml.Name `xml:"Notify"`
	CmdType  string   `xml:"CmdType"`
	SN       int      `xml:"SN"`
	DeviceID string   `xml:"DeviceID"`
	Status   string   `xml:"Status"`
}

func (s *SIPRuntimeService) handleMessage(msg *sip.Message, remoteAddr *net.UDPAddr) {
	ctx := context.Background()
	body := msg.Body
	if len(body) == 0 {
		resp := s.newResponse(msg, 400, "Bad Request (Empty Body)")
		_ = s.server.Send(resp, remoteAddr)
		return
	}

	var keepalive KeepaliveXML
	if err := xml.Unmarshal(body, &keepalive); err == nil && strings.EqualFold(keepalive.CmdType, "Keepalive") {
		deviceID := keepalive.DeviceID
		gbDev, err := s.gbDeviceRepo.FindByDeviceCode(ctx, deviceID)
		if err != nil || gbDev == nil {
			resp := s.newResponse(msg, 403, "Forbidden (Unregistered Device)")
			_ = s.server.Send(resp, remoteAddr)
			return
		}
		_ = s.sipSvc.HandleHeartbeat(ctx, gbDev)
		resp := s.newResponse(msg, 200, "OK")
		_ = s.server.Send(resp, remoteAddr)
		return
	}

	if strings.Contains(string(body), "<CmdType>Catalog</CmdType>") {
		channels, err := ParseCatalogueResponse(string(body))
		if err == nil {
			deviceID := extractSipID(msg.GetHeader("From"))
			_ = s.sipSvc.SyncCatalogChannels(ctx, deviceID, channels)
		}
		resp := s.newResponse(msg, 200, "OK")
		_ = s.server.Send(resp, remoteAddr)
		return
	}

	resp := s.newResponse(msg, 200, "OK")
	_ = s.server.Send(resp, remoteAddr)
}

func (s *SIPRuntimeService) handleBye(msg *sip.Message, remoteAddr *net.UDPAddr) {
	resp := s.newResponse(msg, 200, "OK")
	_ = s.server.Send(resp, remoteAddr)
}

func (s *SIPRuntimeService) newResponse(req *sip.Message, statusCode int, reason string) *sip.Message {
	resp := &sip.Message{
		IsRequest:  false,
		StatusCode: statusCode,
		Reason:     reason,
		Headers:    make(map[string]string),
	}
	copyHeaders := []string{"Via", "From", "To", "Call-ID", "CSeq", "User-Agent"}
	for _, h := range copyHeaders {
		val := req.GetHeader(h)
		if val != "" {
			resp.Headers[h] = val
		}
	}
	resp.Headers["User-Agent"] = "GoSIPServer/1.0"
	return resp
}

func extractSipID(headerVal string) string {
	start := strings.Index(headerVal, "sip:")
	if start == -1 {
		return ""
	}
	sub := headerVal[start+4:]
	end := strings.Index(sub, "@")
	if end == -1 {
		end = strings.Index(sub, ">")
	}
	if end == -1 {
		return sub
	}
	return sub[:end]
}
