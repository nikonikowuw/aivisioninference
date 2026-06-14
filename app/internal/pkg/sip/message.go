package sip

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Message struct {
	IsRequest  bool
	Method     string // Request method: REGISTER, MESSAGE, INVITE, BYE, ACK, INFO
	Recipient  string // Request-URI
	StatusCode int    // Status code for responses (e.g., 200, 401)
	Reason     string // Status line reason (e.g., OK, Unauthorized)
	Headers    map[string]string
	Body       []byte
}

// GetHeader returns the value of a header, case-insensitively.
func (m *Message) GetHeader(name string) string {
	nameLower := strings.ToLower(name)
	for k, v := range m.Headers {
		if strings.ToLower(k) == nameLower {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// ParseMessage parses a raw SIP message byte slice.
func ParseMessage(data []byte) (*Message, error) {
	parts := bytes.SplitN(data, []byte("\r\n\r\n"), 2)
	if len(parts) < 1 {
		// Fallback to \n\n if \r\n\r\n is missing
		parts = bytes.SplitN(data, []byte("\n\n"), 2)
	}

	headerPart := parts[0]
	var bodyPart []byte
	if len(parts) > 1 {
		bodyPart = parts[1]
	}

	lines := strings.Split(string(headerPart), "\n")
	if len(lines) == 0 || len(lines[0]) == 0 {
		return nil, errors.New("empty SIP message")
	}

	startLine := strings.TrimSpace(lines[0])
	startTokens := strings.SplitN(startLine, " ", 3)
	if len(startTokens) < 3 {
		return nil, fmt.Errorf("invalid start line: %s", startLine)
	}

	msg := &Message{
		Headers: make(map[string]string),
	}

	if strings.HasPrefix(startTokens[0], "SIP/") {
		// Response status line: SIP/2.0 200 OK
		msg.IsRequest = false
		code, err := strconv.Atoi(startTokens[1])
		if err != nil {
			return nil, fmt.Errorf("invalid status code: %s", startTokens[1])
		}
		msg.StatusCode = code
		msg.Reason = startTokens[2]
	} else {
		// Request line: REGISTER sip:34020000002000000001@192.168.1.1 SIP/2.0
		msg.IsRequest = true
		msg.Method = startTokens[0]
		msg.Recipient = startTokens[1]
	}

	// Parse headers using textproto Reader style parsing
	headerStr := strings.ReplaceAll(string(headerPart), "\r", "")
	headerLines := strings.Split(headerStr, "\n")[1:]

	var currentKey string
	for _, hl := range headerLines {
		if hl == "" {
			continue
		}
		// Check for header continuation line (starts with space or tab)
		if (hl[0] == ' ' || hl[0] == '\t') && currentKey != "" {
			msg.Headers[currentKey] += " " + strings.TrimSpace(hl)
			continue
		}

		parts := strings.SplitN(hl, ":", 2)
		if len(parts) != 2 {
			continue // skip malformed header
		}
		currentKey = strings.TrimSpace(parts[0])
		msg.Headers[currentKey] = strings.TrimSpace(parts[1])
	}

	// Extract body based on Content-Length
	contentLengthStr := msg.GetHeader("Content-Length")
	if contentLengthStr != "" {
		contentLength, err := strconv.Atoi(contentLengthStr)
		if err == nil && contentLength > 0 {
			if len(bodyPart) < contentLength {
				return nil, fmt.Errorf("incomplete body: expected %d bytes, got %d", contentLength, len(bodyPart))
			}
			msg.Body = bodyPart[:contentLength]
		}
	} else if len(bodyPart) > 0 {
		msg.Body = bodyPart
	}

	return msg, nil
}

// String serializes the SIP message back to raw bytes string.
func (m *Message) String() string {
	var sb strings.Builder
	if m.IsRequest {
		sb.WriteString(fmt.Sprintf("%s %s SIP/2.0\r\n", m.Method, m.Recipient))
	} else {
		sb.WriteString(fmt.Sprintf("SIP/2.0 %d %s\r\n", m.StatusCode, m.Reason))
	}

	// Ensure Content-Length matches the body size
	if len(m.Body) > 0 {
		m.Headers["Content-Length"] = strconv.Itoa(len(m.Body))
	} else if _, ok := m.Headers["Content-Length"]; !ok {
		m.Headers["Content-Length"] = "0"
	}

	for k, v := range m.Headers {
		sb.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	sb.WriteString("\r\n")
	if len(m.Body) > 0 {
		sb.Write(m.Body)
	}
	return sb.String()
}
