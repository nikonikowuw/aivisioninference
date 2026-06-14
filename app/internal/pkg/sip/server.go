package sip

import (
	"fmt"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Server struct {
	listenIP    string
	listenPort  int
	conn        *net.UDPConn
	handlers    *Handlers
	mu          sync.RWMutex
	running     bool
	stopped     chan struct{}
	lastError   error
	startTime   time.Time
	messageChan chan *udpPacket
}

type udpPacket struct {
	data       []byte
	remoteAddr *net.UDPAddr
}

type Handlers struct {
	OnRequest func(msg *Message, remoteAddr *net.UDPAddr)
}

func NewServer(ip string, port int, handlers *Handlers) *Server {
	return &Server{
		listenIP:    ip,
		listenPort:  port,
		handlers:    handlers,
		stopped:     make(chan struct{}),
		messageChan: make(chan *udpPacket, 1024),
	}
}

func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil
	}

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", s.listenIP, s.listenPort))
	if err != nil {
		s.lastError = err
		s.mu.Unlock()
		return err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		s.lastError = err
		s.mu.Unlock()
		return err
	}

	s.conn = conn
	s.running = true
	s.startTime = time.Now()
	s.lastError = nil
	s.stopped = make(chan struct{})
	s.mu.Unlock()

	zap.L().Info("Go SIP Server listener started", zap.String("addr", addr.String()))

	// Start worker pool
	for i := 0; i < 8; i++ {
		go s.workerLoop()
	}

	go s.readLoop()

	return nil
}

func (s *Server) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	if s.conn != nil {
		s.conn.Close()
	}
	close(s.stopped)
	s.mu.Unlock()
	zap.L().Info("Go SIP Server listener stopped")
}

func (s *Server) Send(msg *Message, addr *net.UDPAddr) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.running || s.conn == nil {
		return fmt.Errorf("SIP server is not running")
	}
	raw := []byte(msg.String())
	_, err := s.conn.WriteToUDP(raw, addr)
	return err
}

func (s *Server) readLoop() {
	buf := make([]byte, 65535)
	for {
		s.mu.RLock()
		running := s.running
		s.mu.RUnlock()
		if !running {
			break
		}

		n, remoteAddr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			s.mu.RLock()
			running := s.running
			s.mu.RUnlock()
			if running {
				zap.L().Warn("SIP server UDP read error", zap.Error(err))
			}
			break
		}

		raw := make([]byte, n)
		copy(raw, buf[:n])

		select {
		case s.messageChan <- &udpPacket{data: raw, remoteAddr: remoteAddr}:
		default:
			zap.L().Warn("SIP server message channel full, dropping message", zap.String("from", remoteAddr.String()))
		}
	}
}

func (s *Server) workerLoop() {
	for {
		select {
		case packet := <-s.messageChan:
			s.handleRawMessage(packet.data, packet.remoteAddr)
		case <-s.stopped:
			return
		}
	}
}

func (s *Server) handleRawMessage(data []byte, remoteAddr *net.UDPAddr) {
	msg, err := ParseMessage(data)
	if err != nil {
		zap.L().Debug("Failed to parse incoming SIP message", zap.Error(err), zap.String("raw", string(data)))
		return
	}

	if s.handlers != nil && s.handlers.OnRequest != nil {
		s.handlers.OnRequest(msg, remoteAddr)
	}
}

func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *Server) GetState() (string, string, time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status := "stopped"
	if s.running {
		status = "running"
	} else if s.lastError != nil {
		status = "failed"
	}
	addr := fmt.Sprintf("%s:%d", s.listenIP, s.listenPort)
	return status, addr, s.startTime, s.lastError
}
