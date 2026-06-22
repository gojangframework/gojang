package utils

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

type sentEmail struct {
	from string
	to   []string
	msg  []byte
}

type fakeEmailSender struct {
	mu       sync.Mutex
	sent     []sentEmail
	block    chan struct{}
	sendDone chan struct{}
}

func (f *fakeEmailSender) SendMail(ctx context.Context, from string, to []string, msg []byte) error {
	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	f.mu.Lock()
	f.sent = append(f.sent, sentEmail{
		from: from,
		to:   append([]string(nil), to...),
		msg:  append([]byte(nil), msg...),
	})
	f.mu.Unlock()

	if f.sendDone != nil {
		select {
		case f.sendDone <- struct{}{}:
		default:
		}
	}

	return nil
}

func (f *fakeEmailSender) last() sentEmail {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sent[len(f.sent)-1]
}

func TestEmailServiceQueuesAndSendsEmail(t *testing.T) {
	sender := &fakeEmailSender{sendDone: make(chan struct{}, 1)}
	service := newEmailServiceWithSender(sender, EmailConfig{
		FromAddress:     "noreply@example.com",
		FromDisplayName: "Gojang",
		MaxSendRate:     100,
		QueueSize:       4,
		WorkerCount:     1,
		SendTimeout:     time.Second,
	})
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = service.Shutdown(ctx)
	}()

	if err := service.SendEmail(&EmailMessage{
		To:      []string{"user@example.com"},
		Cc:      []string{"cc@example.com"},
		Bcc:     []string{"bcc@example.com"},
		Subject: "Hello",
		Body:    "Welcome",
		IsHTML:  false,
	}); err != nil {
		t.Fatalf("SendEmail() error = %v", err)
	}

	select {
	case <-sender.sendDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for email to send")
	}

	got := sender.last()
	if got.from != "noreply@example.com" {
		t.Fatalf("from = %q, want noreply@example.com", got.from)
	}
	if len(got.to) != 3 {
		t.Fatalf("recipients = %v, want To/Cc/Bcc", got.to)
	}
	if !bytes.Contains(got.msg, []byte("From: \"Gojang\" <noreply@example.com>")) {
		t.Fatalf("message missing formatted From header:\n%s", string(got.msg))
	}
	if bytes.Contains(got.msg, []byte("Bcc:")) {
		t.Fatalf("message should not include Bcc header:\n%s", string(got.msg))
	}
}

func TestEmailServiceReturnsQueueFull(t *testing.T) {
	sender := &fakeEmailSender{block: make(chan struct{})}
	service := newEmailServiceWithSender(sender, EmailConfig{
		FromAddress: "noreply@example.com",
		MaxSendRate: 100,
		QueueSize:   1,
		WorkerCount: 1,
		SendTimeout: time.Second,
	})
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = service.Shutdown(ctx)
	}()
	defer close(sender.block)

	msg := &EmailMessage{To: []string{"user@example.com"}, Subject: "Hello", Body: "Welcome"}
	if err := service.SendEmail(msg); err != nil {
		t.Fatalf("first SendEmail() error = %v", err)
	}
	if err := service.SendEmail(msg); err != nil {
		t.Fatalf("second SendEmail() error = %v", err)
	}
	if err := service.SendEmail(msg); err == nil || !strings.Contains(err.Error(), "queue is full") {
		t.Fatalf("third SendEmail() error = %v, want queue full", err)
	}
}

func TestEmailServiceShutdownRejectsNewEmail(t *testing.T) {
	sender := &fakeEmailSender{}
	service := newEmailServiceWithSender(sender, EmailConfig{
		FromAddress: "noreply@example.com",
		MaxSendRate: 100,
		QueueSize:   1,
		WorkerCount: 1,
		SendTimeout: time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := service.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}

	err := service.SendPlainEmail([]string{"user@example.com"}, "Hello", "Welcome")
	if err == nil || !strings.Contains(err.Error(), "shutting down") {
		t.Fatalf("SendPlainEmail() error = %v, want shutting down", err)
	}
}

func TestNewEmailServiceValidatesConfig(t *testing.T) {
	if _, err := NewEmailService(EmailConfig{FromAddress: "noreply@example.com"}); err == nil {
		t.Fatal("NewEmailService() error = nil, want missing SMTP host error")
	}
	if _, err := NewEmailService(EmailConfig{SMTPHost: "smtp.example.com"}); err == nil {
		t.Fatal("NewEmailService() error = nil, want missing from address error")
	}
}
