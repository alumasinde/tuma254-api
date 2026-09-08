package services

import (
	"context"
	"sync"
)

type CapturedSMS struct {
	To   string
	Body string
}

type CaptureSMSSender struct {
	mu       sync.RWMutex
	messages []CapturedSMS
}

func NewCaptureSMSSender() *CaptureSMSSender {
	return &CaptureSMSSender{}
}

func (s *CaptureSMSSender) Send(ctx context.Context, msg SMSMessage) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, CapturedSMS{To: msg.To, Body: msg.Body})
	return nil
}

func (s *CaptureSMSSender) Latest() (CapturedSMS, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.messages) == 0 {
		return CapturedSMS{}, false
	}
	return s.messages[len(s.messages)-1], true
}
