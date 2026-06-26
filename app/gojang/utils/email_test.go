package utils

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
)

type sentEmail struct {
	msg EmailMessage
}

type fakeQueuedSender struct {
	provider    string
	mu          sync.Mutex
	sent        []sentEmail
	block       chan struct{}
	sendStarted chan struct{}
	sendDone    chan struct{}
}

func (f *fakeQueuedSender) Provider() string {
	if f.provider == "" {
		return "fake"
	}
	return f.provider
}

func (f *fakeQueuedSender) Send(ctx context.Context, msg *EmailMessage) error {
	if f.sendStarted != nil {
		select {
		case f.sendStarted <- struct{}{}:
		default:
		}
	}

	if f.block != nil {
		select {
		case <-f.block:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	f.mu.Lock()
	f.sent = append(f.sent, sentEmail{msg: cloneEmailMessage(msg)})
	f.mu.Unlock()

	if f.sendDone != nil {
		select {
		case f.sendDone <- struct{}{}:
		default:
		}
	}

	return nil
}

func (f *fakeQueuedSender) last() sentEmail {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sent[len(f.sent)-1]
}

type fakeSESSender struct {
	sent []*sesv2.SendEmailInput
}

func (f *fakeSESSender) SendEmail(ctx context.Context, input *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	f.sent = append(f.sent, input)
	return &sesv2.SendEmailOutput{MessageId: aws.String("msg-1")}, nil
}

func TestEmailServiceQueuesAndSendsEmail(t *testing.T) {
	sender := &fakeQueuedSender{provider: "smtp", sendDone: make(chan struct{}, 1)}
	service := newEmailServiceWithSender(sender, EmailConfig{
		MaxSendRate: 100,
		QueueSize:   4,
		WorkerCount: 1,
		SendTimeout: time.Second,
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

	got := sender.last().msg
	if got.Subject != "Hello" {
		t.Fatalf("subject = %q, want Hello", got.Subject)
	}
	if len(got.To) != 1 || got.To[0] != "user@example.com" {
		t.Fatalf("to = %v", got.To)
	}
}

func TestEmailServiceReturnsQueueFull(t *testing.T) {
	sender := &fakeQueuedSender{
		block:       make(chan struct{}),
		sendStarted: make(chan struct{}, 1),
	}
	service := newEmailServiceWithSender(sender, EmailConfig{
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

	select {
	case <-sender.sendStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first email to start sending")
	}

	if err := service.SendEmail(msg); err != nil {
		t.Fatalf("second SendEmail() error = %v", err)
	}
	if err := service.SendEmail(msg); err == nil || !strings.Contains(err.Error(), "queue is full") {
		t.Fatalf("third SendEmail() error = %v, want queue full", err)
	}
}

func TestEmailServiceShutdownRejectsNewEmail(t *testing.T) {
	sender := &fakeQueuedSender{}
	service := newEmailServiceWithSender(sender, EmailConfig{
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

func TestNewEmailServiceProviderSelection(t *testing.T) {
	sender, err := newConfiguredEmailSender(EmailConfig{
		SESAccessKeyID:      "key",
		SESSecretAccessKey:  "secret",
		SESRegion:           "us-east-1",
		SESFromEmailAddress: "ses@example.com",
		SMTPHost:            "smtp.example.com",
		FromAddress:         "smtp@example.com",
	})
	if err != nil {
		t.Fatalf("newConfiguredEmailSender() error = %v", err)
	}
	if sender.Provider() != "ses" {
		t.Fatalf("provider = %q, want ses", sender.Provider())
	}

	sender, err = newConfiguredEmailSender(EmailConfig{
		SMTPHost:    "smtp.example.com",
		SMTPPort:    587,
		FromAddress: "smtp@example.com",
	})
	if err != nil {
		t.Fatalf("newConfiguredEmailSender() SMTP error = %v", err)
	}
	if sender.Provider() != "smtp" {
		t.Fatalf("provider = %q, want smtp", sender.Provider())
	}
}

func TestPartialSESConfigFallsBackToSMTP(t *testing.T) {
	sender, err := newConfiguredEmailSender(EmailConfig{
		SESAccessKeyID: "partial-key",
		SMTPHost:       "smtp.example.com",
		SMTPPort:       587,
		FromAddress:    "smtp@example.com",
	})
	if err != nil {
		t.Fatalf("newConfiguredEmailSender() error = %v", err)
	}
	if sender.Provider() != "smtp" {
		t.Fatalf("provider = %q, want smtp", sender.Provider())
	}
}

func TestNewEmailServiceValidatesConfig(t *testing.T) {
	if _, err := NewEmailService(EmailConfig{}); err == nil {
		t.Fatal("NewEmailService() error = nil, want missing provider error")
	}
	if _, err := NewEmailService(EmailConfig{SMTPHost: "smtp.example.com"}); err == nil {
		t.Fatal("NewEmailService() error = nil, want missing from address error")
	}
}

func TestSESSenderBuildsEmailInput(t *testing.T) {
	client := &fakeSESSender{}
	sender := &sesSender{client: client, from: "Gojang <noreply@example.com>"}

	msg := &EmailMessage{
		To:      []string{"user@example.com"},
		Cc:      []string{"cc@example.com"},
		Subject: "Hello",
		Body:    "<p>Welcome</p>",
		IsHTML:  true,
	}
	input := sender.buildSendEmailInput(msg)

	if aws.ToString(input.FromEmailAddress) != "Gojang <noreply@example.com>" {
		t.Fatalf("from = %q", aws.ToString(input.FromEmailAddress))
	}
	if got := aws.ToString(input.Content.Simple.Subject.Data); got != "Hello" {
		t.Fatalf("subject = %q", got)
	}
	if got := aws.ToString(input.Content.Simple.Body.Html.Data); got != "<p>Welcome</p>" {
		t.Fatalf("html = %q", got)
	}
	if len(input.Destination.ToAddresses) != 1 || input.Destination.ToAddresses[0] != "user@example.com" {
		t.Fatalf("to = %v", input.Destination.ToAddresses)
	}
}

func TestSendPasswordResetEmail(t *testing.T) {
	sender := &fakeQueuedSender{sendDone: make(chan struct{}, 1)}
	service := newEmailServiceWithSender(sender, EmailConfig{
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

	resetURL := "https://app.example.com/reset-password?token=abc"
	if err := service.SendPasswordResetEmail("user@example.com", resetURL); err != nil {
		t.Fatalf("SendPasswordResetEmail() error = %v", err)
	}

	select {
	case <-sender.sendDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for password reset email")
	}

	got := sender.last().msg
	if got.Subject != "Gojang password reset" {
		t.Fatalf("subject = %q", got.Subject)
	}
	if !strings.Contains(got.Body, resetURL) {
		t.Fatalf("body missing reset URL: %s", got.Body)
	}
}
