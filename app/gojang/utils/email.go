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

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"golang.org/x/time/rate"
)

// EmailConfig configures email delivery and queue behavior.
type EmailConfig struct {
	SMTPHost        string
	SMTPPort        int
	SMTPUser        string
	SMTPPass        string
	FromAddress     string
	FromDisplayName string

	SESAccessKeyID      string
	SESSecretAccessKey  string
	SESRegion           string
	SESFromEmailAddress string
	SESFromDisplayName  string

	MaxSendRate int
	QueueSize   int
	WorkerCount int
	SendTimeout time.Duration
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
	Send(ctx context.Context, msg *EmailMessage) error
	Provider() string
}

type smtpEmailSender struct {
	addr        string
	auth        smtp.Auth
	fromAddress string
	fromHeader  string
}

func (s *smtpEmailSender) Provider() string {
	return "smtp"
}

func (s *smtpEmailSender) Send(ctx context.Context, msg *EmailMessage) error {
	raw, err := s.buildMessage(msg)
	if err != nil {
		return err
	}

	recipients := append([]string(nil), msg.To...)
	recipients = append(recipients, msg.Cc...)
	recipients = append(recipients, msg.Bcc...)

	result := make(chan error, 1)
	go func() {
		result <- smtp.SendMail(s.addr, s.auth, s.fromAddress, recipients, raw)
	}()

	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *smtpEmailSender) buildMessage(msg *EmailMessage) ([]byte, error) {
	if msg == nil {
		return nil, fmt.Errorf("email message is required")
	}

	var body bytes.Buffer
	writeHeader(&body, "From", s.fromHeader)
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

type sesEmailSender interface {
	SendEmail(ctx context.Context, params *sesv2.SendEmailInput, optFns ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error)
}

type sesSender struct {
	client sesEmailSender
	from   string
}

func (s *sesSender) Provider() string {
	return "ses"
}

func (s *sesSender) Send(ctx context.Context, msg *EmailMessage) error {
	_, err := s.client.SendEmail(ctx, s.buildSendEmailInput(msg))
	return err
}

func (s *sesSender) buildSendEmailInput(msg *EmailMessage) *sesv2.SendEmailInput {
	destination := &types.Destination{
		ToAddresses: append([]string(nil), msg.To...),
	}
	if len(msg.Cc) > 0 {
		destination.CcAddresses = append([]string(nil), msg.Cc...)
	}
	if len(msg.Bcc) > 0 {
		destination.BccAddresses = append([]string(nil), msg.Bcc...)
	}

	content := &types.EmailContent{
		Simple: &types.Message{
			Subject: &types.Content{
				Data:    aws.String(msg.Subject),
				Charset: aws.String("UTF-8"),
			},
			Body: &types.Body{},
		},
	}
	if msg.IsHTML {
		content.Simple.Body.Html = &types.Content{
			Data:    aws.String(msg.Body),
			Charset: aws.String("UTF-8"),
		}
	} else {
		content.Simple.Body.Text = &types.Content{
			Data:    aws.String(msg.Body),
			Charset: aws.String("UTF-8"),
		}
	}

	return &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(s.from),
		Destination:      destination,
		Content:          content,
	}
}

// EmailService queues email for asynchronous delivery.
type EmailService struct {
	sender      emailSender
	queue       chan emailJob
	limiter     *rate.Limiter
	sendTimeout time.Duration

	mu     sync.Mutex
	closed bool

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewEmailService creates a queued email service. SES is preferred when fully
// configured; otherwise SMTP is used when SMTP_HOST is configured.
func NewEmailService(cfg EmailConfig) (*EmailService, error) {
	sender, err := newConfiguredEmailSender(cfg)
	if err != nil {
		return nil, err
	}
	return newEmailServiceWithSender(sender, cfg), nil
}

func newConfiguredEmailSender(cfg EmailConfig) (emailSender, error) {
	if sesAvailable(cfg) {
		region := strings.TrimSpace(cfg.SESRegion)
		if region == "" {
			region = "us-east-1"
		}
		creds := credentials.NewStaticCredentialsProvider(
			cfg.SESAccessKeyID,
			cfg.SESSecretAccessKey,
			"",
		)
		awsCfg, err := awsconfig.LoadDefaultConfig(
			context.TODO(),
			awsconfig.WithRegion(region),
			awsconfig.WithCredentialsProvider(creds),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS SDK configuration: %w", err)
		}

		from := cfg.SESFromEmailAddress
		if cfg.SESFromDisplayName != "" {
			from = (&mail.Address{Name: cfg.SESFromDisplayName, Address: cfg.SESFromEmailAddress}).String()
		}
		return &sesSender{client: sesv2.NewFromConfig(awsCfg), from: from}, nil
	}

	if strings.TrimSpace(cfg.SMTPHost) != "" {
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

		fromHeader := cfg.FromAddress
		if cfg.FromDisplayName != "" {
			fromHeader = (&mail.Address{Name: cfg.FromDisplayName, Address: cfg.FromAddress}).String()
		}

		return &smtpEmailSender{
			addr:        fmt.Sprintf("%s:%d", cfg.SMTPHost, port),
			auth:        auth,
			fromAddress: cfg.FromAddress,
			fromHeader:  fromHeader,
		}, nil
	}

	return nil, fmt.Errorf("email provider is not configured")
}

func sesAvailable(cfg EmailConfig) bool {
	return strings.TrimSpace(cfg.SESAccessKeyID) != "" &&
		strings.TrimSpace(cfg.SESSecretAccessKey) != "" &&
		strings.TrimSpace(cfg.SESFromEmailAddress) != ""
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

	ctx, cancel := context.WithCancel(context.Background())
	service := &EmailService{
		sender:      sender,
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

func (e *EmailService) Provider() string {
	if e == nil || e.sender == nil {
		return ""
	}
	return e.sender.Provider()
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

	if err := e.sender.Send(sendCtx, msg); err != nil {
		Errorw("email.send_failed", "provider", e.sender.Provider(), "error", err, "to", msg.To, "subject", msg.Subject)
		return fmt.Errorf("failed to send email: %w", err)
	}

	Infow("email.sent", "provider", e.sender.Provider(), "to", msg.To, "subject", msg.Subject)
	return nil
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
		Infow("email.queued", "provider", e.sender.Provider(), "to", job.msg.To, "subject", job.msg.Subject)
		return nil
	default:
		Warnw("email.queue_full", "provider", e.sender.Provider(), "to", job.msg.To, "subject", job.msg.Subject)
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

// SendPasswordResetEmail queues a password reset email with a reset link.
func (e *EmailService) SendPasswordResetEmail(to, resetURL string) error {
	subject := "Gojang password reset"
	htmlBody := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head><meta charset="UTF-8"><title>Password Reset</title></head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #222;">
  <div style="max-width: 600px; margin: 0 auto; padding: 24px;">
    <h2>Password reset request</h2>
    <p>Use the link below to choose a new password for your Gojang account.</p>
    <p style="margin: 28px 0;">
      <a href="%s" style="background-color: #2563eb; color: #fff; padding: 12px 20px; text-decoration: none; border-radius: 4px; display: inline-block;">Reset password</a>
    </p>
    <p>If the button does not work, copy and paste this link into your browser:</p>
    <p style="word-break: break-all;"><a href="%s">%s</a></p>
    <p>If you did not request this reset, you can ignore this email.</p>
    <p style="color: #64748b; font-size: 12px; margin-top: 28px;">This link expires in 1 hour.</p>
  </div>
</body>
</html>`, resetURL, resetURL, resetURL)

	return e.SendHTMLEmail([]string{to}, subject, htmlBody)
}
