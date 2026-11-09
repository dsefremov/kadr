package mail

import (
	"context"
	"encoding/json"
	"fmt"
	"net/smtp"
	"strings"
)

type OutboxRow struct {
	ID      int64
	Event   string
	Payload json.RawMessage
}

type SMTP struct {
	Host string
	Port int
	From string
}

func (s *SMTP) Send(ctx context.Context, to, subject, body string) error {
	if s.Host == "" {
		return fmt.Errorf("smtp not configured")
	}
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", s.From),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")
	return smtp.SendMail(addr, nil, s.From, []string{to}, []byte(msg))
}

func RenderOutbox(row OutboxRow, publicURL string) (to, subject, body string, err error) {
	var p struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
		Body    string `json:"body"`
		Link    string `json:"link"`
	}
	if err := json.Unmarshal(row.Payload, &p); err != nil {
		return "", "", "", err
	}
	to = p.To
	subject = p.Subject
	body = p.Body
	if p.Link != "" {
		body = strings.ReplaceAll(body, "{{link}}", publicURL+p.Link)
	}
	return to, subject, body, nil
}
