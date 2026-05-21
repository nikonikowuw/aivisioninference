package mail

import (
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
	"time"
)

// ParseMessage parses common plain text and HTML email content.
func ParseMessage(uid uint32, raw []byte) (*InboundMessage, error) {
	msg, err := mail.ReadMessage(strings.NewReader(string(raw)))
	if err != nil {
		return nil, err
	}

	subject := decodeHeader(msg.Header.Get("Subject"))
	from := decodeHeader(msg.Header.Get("From"))
	to := decodeHeader(msg.Header.Get("To"))
	messageID := strings.TrimSpace(msg.Header.Get("Message-ID"))

	var emailDate *time.Time
	if dateHeader := msg.Header.Get("Date"); dateHeader != "" {
		if parsed, err := mail.ParseDate(dateHeader); err == nil {
			emailDate = &parsed
		}
	}

	mediaType, params, _ := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	textBody, htmlBody := readBodies(mediaType, params, msg.Body)
	textBody = strings.TrimSpace(textBody)
	htmlBody = strings.TrimSpace(htmlBody)

	return &InboundMessage{
		UID:       uid,
		MessageID: messageID,
		From:      from,
		To:        to,
		Subject:   subject,
		TextBody:  textBody,
		HTMLBody:  htmlBody,
		Summary:   buildSummary(textBody, htmlBody),
		Date:      emailDate,
		RawSize:   int64(len(raw)),
	}, nil
}

func readBodies(mediaType string, params map[string]string, body io.Reader) (string, string) {
	switch {
	case strings.HasPrefix(mediaType, "multipart/"):
		return readMultipartBodies(params["boundary"], body)
	case strings.EqualFold(mediaType, "text/html"):
		data, _ := io.ReadAll(body)
		return "", string(data)
	default:
		data, _ := io.ReadAll(body)
		return string(data), ""
	}
}

func readMultipartBodies(boundary string, body io.Reader) (string, string) {
	if boundary == "" {
		data, _ := io.ReadAll(body)
		return string(data), ""
	}
	reader := multipart.NewReader(body, boundary)
	var textBody, htmlBody strings.Builder
	for {
		part, err := reader.NextPart()
		if err != nil {
			break
		}
		contentType, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		data, _ := io.ReadAll(part)
		switch {
		case strings.EqualFold(contentType, "text/plain"):
			textBody.Write(data)
		case strings.EqualFold(contentType, "text/html"):
			htmlBody.Write(data)
		case strings.HasPrefix(contentType, "multipart/"):
			nestedType, nestedParams, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
			text, html := readBodies(nestedType, nestedParams, strings.NewReader(string(data)))
			textBody.WriteString(text)
			htmlBody.WriteString(html)
		}
	}
	return textBody.String(), htmlBody.String()
}

func decodeHeader(value string) string {
	decoded, err := new(mime.WordDecoder).DecodeHeader(value)
	if err != nil {
		return value
	}
	return decoded
}

func buildSummary(textBody, htmlBody string) string {
	summary := textBody
	if summary == "" {
		summary = stripHTML(htmlBody)
	}
	summary = strings.Join(strings.Fields(summary), " ")
	if len(summary) > 512 {
		return summary[:512]
	}
	return summary
}

func stripHTML(input string) string {
	var out strings.Builder
	inTag := false
	for _, r := range input {
		switch r {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag {
				out.WriteRune(r)
			}
		}
	}
	return out.String()
}
