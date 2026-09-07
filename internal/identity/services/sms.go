package services

import (
	"context"
	"log/slog"
)

type SMSMessage struct {
	To   string
	Body string
}

type SMSSender interface {
	Send(context.Context, SMSMessage) error
}

type LoggingSMSSender struct {
	log *slog.Logger
}

func NewLoggingSMSSender(log *slog.Logger) *LoggingSMSSender {
	if log == nil {
		log = slog.Default()
	}
	return &LoggingSMSSender{log: log}
}

func (s *LoggingSMSSender) Send(ctx context.Context, msg SMSMessage) error {
	_ = ctx
	// Development/test transport only. Do not log OTP values.
	s.log.Info("sms queued", "to", maskPhone(msg.To))
	return nil
}

func maskPhone(v string) string {
	if len(v) <= 4 {
		return "****"
	}
	return "****" + v[len(v)-4:]
}