package email

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/settings"

	mail "github.com/wneessen/go-mail"
)

type (
	Service interface {
		Send(ctx context.Context, to, subject, body string) error
	}

	service struct {
		settingsSvc settings.Service
		mu          sync.RWMutex
		client      *mail.Client
	}
)

func NewService(settingsSvc settings.Service) Service {
	svc := &service{settingsSvc: settingsSvc}
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
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		logger.Log.Warn().Msg("SMTP not configured, skipping email send")
		return nil
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
