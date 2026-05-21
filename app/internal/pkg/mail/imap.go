package mail

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/textproto"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// IMAPClient implements the small IMAP subset needed for connection tests and inbox sync.
type IMAPClient struct {
	cfg IMAPConfig
	seq uint64
}

// NewIMAPClient creates an IMAPClient.
func NewIMAPClient(cfg IMAPConfig) *IMAPClient {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.Mailbox == "" {
		cfg.Mailbox = "INBOX"
	}
	return &IMAPClient{cfg: cfg}
}

// TestConnection verifies login and mailbox selection.
func (c *IMAPClient) TestConnection(ctx context.Context) error {
	conn, err := c.connect(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer c.logout(conn)

	if err := c.login(conn); err != nil {
		return err
	}
	return c.selectMailbox(conn)
}

// FetchRecent fetches recent messages from the configured mailbox.
func (c *IMAPClient) FetchRecent(ctx context.Context, limit int) ([]InboundMessage, error) {
	if !c.cfg.Enabled {
		return nil, errors.New("imap is disabled")
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	conn, err := c.connect(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	defer c.logout(conn)

	if err := c.login(conn); err != nil {
		return nil, err
	}
	exists, err := c.selectMailboxExists(conn)
	if err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, nil
	}

	start := exists - limit + 1
	if start < 1 {
		start = 1
	}
	tag := c.nextTag()
	if _, err := fmt.Fprintf(conn, "%s UID FETCH %d:%d (UID RFC822)\r\n", tag, start, exists); err != nil {
		return nil, err
	}
	responses, err := readUntilTagged(conn.Reader, tag)
	if err != nil {
		return nil, err
	}
	return parseFetchResponses(responses), nil
}

func (c *IMAPClient) connect(ctx context.Context) (*imapConn, error) {
	if !c.cfg.Enabled {
		return nil, errors.New("imap is disabled")
	}
	if c.cfg.Host == "" || c.cfg.Port == 0 {
		return nil, errors.New("imap host or port is empty")
	}
	addr := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)
	conn, err := dialContext(ctx, addr, c.cfg.Timeout, strings.EqualFold(c.cfg.Encryption, EncryptionTLS), c.cfg.Host)
	if err != nil {
		return nil, fmt.Errorf("connect imap server: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	} else {
		_ = conn.SetDeadline(time.Now().Add(c.cfg.Timeout))
	}
	imapConn := &imapConn{Conn: conn, Reader: textproto.NewReader(bufio.NewReader(conn))}
	if _, err := imapConn.Reader.ReadLine(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("read imap greeting: %w", err)
	}
	if strings.EqualFold(c.cfg.Encryption, EncryptionSTARTTLS) {
		if err := c.startTLS(imapConn); err != nil {
			conn.Close()
			return nil, err
		}
	}
	return imapConn, nil
}

func (c *IMAPClient) startTLS(conn *imapConn) error {
	tag := c.nextTag()
	if _, err := fmt.Fprintf(conn, "%s STARTTLS\r\n", tag); err != nil {
		return err
	}
	if err := readTaggedOK(conn.Reader, tag); err != nil {
		return fmt.Errorf("imap starttls: %w", err)
	}
	tlsConn := tls.Client(conn.Conn, &tls.Config{ServerName: c.cfg.Host, MinVersion: tls.VersionTLS12})
	if err := tlsConn.Handshake(); err != nil {
		return fmt.Errorf("imap tls handshake: %w", err)
	}
	conn.Conn = tlsConn
	conn.Reader = textproto.NewReader(bufio.NewReader(tlsConn))
	return nil
}

func (c *IMAPClient) login(conn *imapConn) error {
	tag := c.nextTag()
	if _, err := fmt.Fprintf(conn, "%s LOGIN %s %s\r\n", tag, quoteIMAP(c.cfg.Username), quoteIMAP(c.cfg.Password)); err != nil {
		return err
	}
	if err := readTaggedOK(conn.Reader, tag); err != nil {
		return fmt.Errorf("imap login: %w", err)
	}
	return nil
}

func (c *IMAPClient) selectMailbox(conn *imapConn) error {
	_, err := c.selectMailboxExists(conn)
	return err
}

func (c *IMAPClient) selectMailboxExists(conn *imapConn) (int, error) {
	tag := c.nextTag()
	if _, err := fmt.Fprintf(conn, "%s SELECT %s\r\n", tag, quoteIMAP(c.cfg.Mailbox)); err != nil {
		return 0, err
	}
	lines, err := readUntilTagged(conn.Reader, tag)
	if err != nil {
		return 0, fmt.Errorf("imap select mailbox: %w", err)
	}
	exists := 0
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "*" && strings.EqualFold(fields[2], "EXISTS") {
			if n, convErr := strconv.Atoi(fields[1]); convErr == nil {
				exists = n
			}
		}
	}
	return exists, nil
}

func (c *IMAPClient) logout(conn *imapConn) {
	tag := c.nextTag()
	_, _ = fmt.Fprintf(conn, "%s LOGOUT\r\n", tag)
}

func (c *IMAPClient) nextTag() string {
	n := atomic.AddUint64(&c.seq, 1)
	return fmt.Sprintf("A%04d", n)
}

type imapConn struct {
	net.Conn
	Reader *textproto.Reader
}

func readTaggedOK(r *textproto.Reader, tag string) error {
	lines, err := readUntilTagged(r, tag)
	if err != nil {
		return err
	}
	if len(lines) == 0 || !strings.Contains(strings.ToUpper(lines[len(lines)-1]), " OK") {
		return errors.New("imap command failed")
	}
	return nil
}

func readUntilTagged(r *textproto.Reader, tag string) ([]string, error) {
	var lines []string
	for {
		line, err := r.ReadLine()
		if err != nil {
			return lines, err
		}
		lines = append(lines, line)
		if strings.HasPrefix(line, tag+" ") {
			if strings.Contains(strings.ToUpper(line), " OK") {
				return lines, nil
			}
			return lines, fmt.Errorf("imap command failed: %s", line)
		}
	}
}

var uidPattern = regexp.MustCompile(`UID ([0-9]+)`)

func parseFetchResponses(lines []string) []InboundMessage {
	messages := make([]InboundMessage, 0)
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if !strings.HasPrefix(line, "* ") || !strings.Contains(line, "FETCH") {
			continue
		}
		uid := parseUID(line)
		raw := collectLiteral(lines, &i)
		if len(raw) == 0 {
			continue
		}
		parsed, err := ParseMessage(uid, []byte(raw))
		if err != nil {
			continue
		}
		messages = append(messages, *parsed)
	}
	return messages
}

func parseUID(line string) uint32 {
	match := uidPattern.FindStringSubmatch(line)
	if len(match) != 2 {
		return 0
	}
	n, _ := strconv.ParseUint(match[1], 10, 32)
	return uint32(n)
}

func collectLiteral(lines []string, idx *int) string {
	var b strings.Builder
	for *idx+1 < len(lines) {
		*idx = *idx + 1
		line := lines[*idx]
		if strings.HasPrefix(line, ")") || strings.Contains(line, " FETCH ") || strings.HasPrefix(line, "A") {
			*idx = *idx - 1
			break
		}
		b.WriteString(line)
		b.WriteString("\r\n")
	}
	return b.String()
}

func quoteIMAP(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}
