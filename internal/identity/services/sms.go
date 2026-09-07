package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type SMSMessage struct {
	To   string `json:"to"`
	Body string `json:"body"`
}

type SMSSender interface {
	Send(context.Context, SMSMessage) error
}

type LoggingSMSSender struct { log *slog.Logger }

func NewLoggingSMSSender(log *slog.Logger) *LoggingSMSSender {
	if log == nil { log = slog.Default() }
	return &LoggingSMSSender{log: log}
}

func (s *LoggingSMSSender) Send(ctx context.Context, msg SMSMessage) error {
	_ = ctx
	// Development/test transport only. OTP contents are intentionally not logged.
	s.log.Info("sms suppressed in development transport", "to", maskPhone(msg.To))
	return nil
}

type WebhookSMSSender struct {
	url   string
	token string
	http  *http.Client
}

func NewWebhookSMSSender(url, token string) (*WebhookSMSSender, error) {
	if strings.TrimSpace(url) == "" { return nil, errors.New("SMS_WEBHOOK_URL is required") }
	return &WebhookSMSSender{
		url: strings.TrimSpace(url), token: strings.TrimSpace(token),
		http: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (s *WebhookSMSSender) Send(ctx context.Context, msg SMSMessage) error {
	body, err := json.Marshal(msg)
	if err != nil { return err }
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(body))
	if err != nil { return err }
	req.Header.Set("Content-Type", "application/json")
	if s.token != "" { req.Header.Set("Authorization", "Bearer "+s.token) }
	resp, err := s.http.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("sms provider returned status %d", resp.StatusCode)
	}
	return nil
}

func maskPhone(v string) string {
	if len(v) <= 4 { return "****" }
	return "****" + v[len(v)-4:]
}