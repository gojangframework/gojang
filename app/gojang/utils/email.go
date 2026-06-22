package utils

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"net/mail"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// EmailConfig configures SMTP email delivery and queue behavior.
type EmailConfig struct {
	SMTPHost        string
	SMTPPort        int
	SMTPUser        string
	SMTPPass        string
	FromAddress     string
	FromDisplayName string
	MaxSendRate     int
	QueueSize       int
	WorkerCount     int
	SendTimeout     time.Duration
}

// EmailMessage represents an email to send.
type EmailMessage struct {
	To      []string
	Cc      []string
	Bcc     []string
	Subject string
	Body    string
	IsHTML  bool
}

type emailJob struct {
	msg EmailMessage
}

type emailSender interface {
	SendMail(ctx context.Context, from string, to []string, msg []byte) error
}

type smtpEmailSender struct {
	addr string
	auth smtp.Auth
}

func (s smtpEmailSender) SendMail(ctx context.Context, from string, to []string, msg []byte) error {
	result := make(chan error, 1)
	go func() {
		result <- smtp.SendMail(s.addr, s.auth, from, to, msg)
	}()

	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// EmailService queues email for asynchronous SMTP delivery.
type EmailService struct {
	sender      emailSender
	fromAddress string
	fromHeader  string
	queue       chan emailJob
	limiter     *rate.Limiter
	sendTimeout time.Duration

	mu     sync.Mutex
	closed bool

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewEmailService creates an SMTP-backed email queue.
func NewEmailService(cfg EmailConfig) (*EmailService, error) {
	if strings.TrimSpace(cfg.SMTPHost) == "" {
		return nil, fmt.Errorf("SMTP host is required")
	}
	if strings.TrimSpace(cfg.FromAddress) == "" {
		return nil, fmt.Errorf("from address is required")
	}

	port := cfg.SMTPPort
	if port <= 0 {
		port = 587
	}

	var auth smtp.Auth
	if cfg.SMTPUser != "" || cfg.SMTPPass != "" {
		auth = smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPHost)
	}

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, port)
	return newEmailServiceWithSender(smtpEmailSender{addr: addr, auth: auth}, cfg), nil
}

func newEmailServiceWithSender(sender emailSender, cfg EmailConfig) *EmailService {
	maxSendRate := cfg.MaxSendRate
	if maxSendRate <= 0 {
		maxSendRate = 14
	}

	queueSize := cfg.QueueSize
	if queueSize <= 0 {
		queueSize = 1000
	}

	workerCount := cfg.WorkerCount
	if workerCount <= 0 {
		workerCount = maxSendRate
	}
	if workerCount > queueSize {
		workerCount = queueSize
	}
	if workerCount <= 0 {
		workerCount = 1
	}

	sendTimeout := cfg.SendTimeout
	if sendTimeout <= 0 {
		sendTimeout = 15 * time.Second
	}

	fromHeader := cfg.FromAddress
	if cfg.FromDisplayName != "" {
		fromHeader = (&mail.Address{Name: cfg.FromDisplayName, Address: cfg.FromAddress}).String()
	}

	ctx, cancel := context.WithCancel(context.Background())
	service := &EmailService{
		sender:      sender,
		fromAddress: cfg.FromAddress,
		fromHeader:  fromHeader,
		queue:       make(chan emailJob, queueSize),
		limiter:     rate.NewLimiter(rate.Limit(maxSendRate), 1),
		sendTimeout: sendTimeout,
		cancel:      cancel,
	}

	for range workerCount {
		service.wg.Add(1)
		go service.worker(ctx)
	}

	return service
}

func (e *EmailService) worker(ctx context.Context) {
	defer e.wg.Done()

	for job := range e.queue {
		if err := e.sendQueuedEmail(ctx, &job.msg); err != nil {
			Warnw("email.job_failed", "error", err, "to", job.msg.To, "subject", job.msg.Subject)
		}
	}
}

func (e *EmailService) sendQueuedEmail(ctx context.Context, msg *EmailMessage) error {
	if err := e.limiter.Wait(ctx); err != nil {
		return fmt.Errorf("email worker stopped before send: %w", err)
	}

	sendCtx, cancel := context.WithTimeout(ctx, e.sendTimeout)
	defer cancel()

	raw, err := e.buildMessage(msg)
	if err != nil {
		return err
	}

	recipients := append([]string(nil), msg.To...)
	recipients = append(recipients, msg.Cc...)
	recipients = append(recipients, msg.Bcc...)

	if err := e.sender.SendMail(sendCtx, e.fromAddress, recipients, raw); err != nil {
		Errorw("email.send_failed", "error", err, "to", msg.To, "subject", msg.Subject)
		return fmt.Errorf("failed to send email: %w", err)
	}

	Infow("email.sent", "to", msg.To, "subject", msg.Subject)
	return nil
}

func (e *EmailService) buildMessage(msg *EmailMessage) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("email message is required")
	}

	var body bytes.Buffer
	writeHeader(&body, "From", e.fromHeader)
	writeHeader(&body, "To", strings.Join(msg.To, ", "))
	if len(msg.Cc) > 0 {
		writeHeader(&body, "Cc", strings.Join(msg.Cc, ", "))
	}
	writeHeader(&body, "Subject", mime.QEncoding.Encode("UTF-8", msg.Subject))
	writeHeader(&body, "MIME-Version", "1.0")
	if msg.IsHTML {
		writeHeader(&body, "Content-Type", `text/html; charset="UTF-8"`)
	} else {
		writeHeader(&body, "Content-Type", `text/plain; charset="UTF-8"`)
	}
	writeHeader(&body, "Content-Transfer-Encoding", "base64")
	body.WriteString("\r\n")

	encoded := make([]byte, base64.StdEncoding.EncodedLen(len([]byte(msg.Body))))
	base64.StdEncoding.Encode(encoded, []byte(msg.Body))
	for len(encoded) > 76 {
		body.Write(encoded[:76])
		body.WriteString("\r\n")
		encoded = encoded[76:]
	}
	body.Write(encoded)
	body.WriteString("\r\n")

	return body.Bytes(), nil
}

func writeHeader(buf *bytes.Buffer, key, value string) {
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(value)
	buf.WriteString("\r\n")
}

func cloneEmailMessage(msg *EmailMessage) EmailMessage {
	return EmailMessage{
		To:      append([]string(nil), msg.To...),
		Cc:      append([]string(nil), msg.Cc...),
		Bcc:     append([]string(nil), msg.Bcc...),
		Subject: msg.Subject,
		Body:    msg.Body,
		IsHTML:  msg.IsHTML,
	}
}

// SendEmail queues an email for asynchronous delivery.
func (e *EmailService) SendEmail(msg *EmailMessage) error {
	if msg == nil {
		return fmt.Errorf("email message is required")
	}
	if len(msg.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	job := emailJob{msg: cloneEmailMessage(msg)}

	e.mu.Lock()
	defer e.mu.Unlock()

	if e.closed {
		return fmt.Errorf("email service is shutting down")
	}

	select {
	case e.queue <- job:
		Infow("email.queued", "to", job.msg.To, "subject", job.msg.Subject)
		return nil
	default:
		Warnw("email.queue_full", "to", job.msg.To, "subject", job.msg.Subject)
		return fmt.Errorf("email queue is full")
	}
}

// Shutdown stops accepting new email and drains queued jobs until ctx expires.
func (e *EmailService) Shutdown(ctx context.Context) error {
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		return nil
	}
	e.closed = true
	close(e.queue)
	e.mu.Unlock()

	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		e.cancel()
		return nil
	case <-ctx.Done():
		e.cancel()
		<-done
		return ctx.Err()
	}
}

// SendPlainEmail queues a plain text email.
func (e *EmailService) SendPlainEmail(to []string, subject, body string) error {
	return e.SendEmail(&EmailMessage{
		To:      to,
		Subject: subject,
		Body:    body,
		IsHTML:  false,
	})
}

// SendHTMLEmail queues an HTML email.
func (e *EmailService) SendHTMLEmail(to []string, subject, htmlBody string) error {
	return e.SendEmail(&EmailMessage{
		To:      to,
		Subject: subject,
		Body:    htmlBody,
		IsHTML:  true,
	})
}
