package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/settings"

	"github.com/wneessen/go-mail"
)

type (
	Service interface {
		Send(ctx context.Context, to, subject, body string) error
		SendTest(ctx context.Context, to, subject, body string) error
		Enabled(ctx context.Context) bool
	}

	service struct {
		settingsSvc settings.Service
		mu          sync.RWMutex
		client      *mail.Client
		httpClient  *http.Client
	}

	cloudflareAddress struct {
		Address string `json:"address"`
		Name    string `json:"name"`
	}

	cloudflareSendRequest struct {
		To      string            `json:"to"`
		From    cloudflareAddress `json:"from"`
		Subject string            `json:"subject"`
		HTML    string            `json:"html"`
		Text    string            `json:"text"`
	}

	cloudflareSendResponse struct {
		Success bool `json:"success"`
	}
)

var (
	htmlTagRe = regexp.MustCompile(`<[^>]*>`)

	ErrNotConfigured = errors.New("email provider is not configured")
)

func NewService(settingsSvc settings.Service) Service {
	svc := &service{
		settingsSvc: settingsSvc,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
	svc.buildClient()
	return svc
}

func (s *service) buildClient() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		_ = s.client.Close()
		s.client = nil
	}

	ctx := context.Background()
	host := s.settingsSvc.Get(ctx, config.SettingSMTPHost)
	if host == "" {
		return
	}

	port := s.settingsSvc.GetInt(ctx, config.SettingSMTPPort)
	username := s.settingsSvc.Get(ctx, config.SettingSMTPUsername)
	password := s.settingsSvc.Get(ctx, config.SettingSMTPPassword)

	var opts []mail.Option
	opts = append(opts, mail.WithPort(port))

	if username != "" && password != "" {
		opts = append(opts, mail.WithSMTPAuth(mail.SMTPAuthPlain))
		opts = append(opts, mail.WithUsername(username))
		opts = append(opts, mail.WithPassword(password))
	} else {
		opts = append(opts, mail.WithTLSPolicy(mail.NoTLS))
	}

	client, err := mail.NewClient(host, opts...)
	if err != nil {
		logger.Log.Error().Err(err).Msg("failed to create SMTP client")
		return
	}

	from := s.settingsSvc.Get(ctx, config.SettingSMTPFrom)
	if from == "" {
		logger.Log.Warn().Msg("SMTP from address not set, skipping connection test")
		s.client = client
		return
	}

	if err := client.DialWithContext(ctx); err != nil {
		logger.Log.Error().Err(err).Str("host", host).Int("port", port).Msg("SMTP connection test failed")
		_ = client.Close()
		return
	}
	_ = client.Close()

	s.client = client
	logger.Log.Info().Str("host", host).Int("port", port).Msg("SMTP client configured and verified")
}

func (s *service) Send(ctx context.Context, to, subject, body string) error {
	err := s.send(ctx, to, subject, body)
	if errors.Is(err, ErrNotConfigured) {
		logger.Log.Warn().Msg("email provider not configured, skipping email send")
		return nil
	}

	return err
}

func (s *service) SendTest(ctx context.Context, to, subject, body string) error {
	return s.send(ctx, to, subject, body)
}

func (s *service) Enabled(ctx context.Context) bool {
	provider := config.EmailProvider(s.settingsSvc.Get(ctx, config.SettingEmailProvider))

	if provider == config.EmailProviderCloudflare {
		return s.settingsSvc.Get(ctx, config.SettingCloudflareAccountID) != "" &&
			s.settingsSvc.Get(ctx, config.SettingCloudflareAPIToken) != "" &&
			s.settingsSvc.Get(ctx, config.SettingCloudflareEmailFrom) != ""
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.client != nil
}

func (s *service) send(ctx context.Context, to, subject, body string) error {
	provider := config.EmailProvider(s.settingsSvc.Get(ctx, config.SettingEmailProvider))

	if provider == config.EmailProviderCloudflare {
		return s.sendCloudflare(ctx, to, subject, body)
	}

	return s.sendSMTP(ctx, to, subject, body)
}

func (s *service) sendSMTP(ctx context.Context, to, subject, body string) error {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return ErrNotConfigured
	}

	from := s.settingsSvc.Get(ctx, config.SettingSMTPFrom)

	msg := mail.NewMsg()
	if err := msg.From(from); err != nil {
		return fmt.Errorf("set from address: %w", err)
	}
	if err := msg.To(to); err != nil {
		return fmt.Errorf("set to address: %w", err)
	}
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextHTML, body)

	if err := client.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}

func (s *service) sendCloudflare(ctx context.Context, to, subject, body string) error {
	accountID := s.settingsSvc.Get(ctx, config.SettingCloudflareAccountID)
	apiToken := s.settingsSvc.Get(ctx, config.SettingCloudflareAPIToken)
	from := s.settingsSvc.Get(ctx, config.SettingCloudflareEmailFrom)
	if accountID == "" || apiToken == "" || from == "" {
		return ErrNotConfigured
	}

	payload := cloudflareSendRequest{
		To:      to,
		From:    cloudflareAddress{Address: from, Name: s.settingsSvc.Get(ctx, config.SettingSiteName)},
		Subject: subject,
		HTML:    body,
		Text:    htmlToText(body),
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal cloudflare request: %w", err)
	}

	endpoint := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/email/sending/send", accountID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("build cloudflare request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send cloudflare email: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read cloudflare response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cloudflare email failed: status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed cloudflareSendResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return fmt.Errorf("decode cloudflare response: %w", err)
	}
	if !parsed.Success {
		return fmt.Errorf("cloudflare email rejected: %s", string(respBody))
	}

	return nil
}

func htmlToText(html string) string {
	text := htmlTagRe.ReplaceAllString(html, "")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	return strings.TrimSpace(text)
}

type MailSettingListener struct {
	svc *service
}

func NewMailSettingListener(svc Service) *MailSettingListener {
	return &MailSettingListener{svc: svc.(*service)}
}

func (l *MailSettingListener) OnSettingChanged(_ config.SiteSettingKey, _ string) {}

func (l *MailSettingListener) OnSettingsBatchChanged(keys []config.SiteSettingKey) {
	for _, key := range keys {
		if strings.HasPrefix(string(key), "smtp_") {
			l.svc.buildClient()
			return
		}
	}
}
