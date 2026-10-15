package hyperbeam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/httpclient"
	"umineko_city_of_books/internal/logger"
	"umineko_city_of_books/internal/settings"
)

type (
	Service interface {
		Enabled() bool
		CreateVM(ctx context.Context, opts CreateVMOptions) (*VM, error)
		GetVMStatus(ctx context.Context, sessionID string) (*VMStatus, error)
		SetControlRole(ctx context.Context, vmBaseURL, adminToken, userIdentifier string, hasControl bool) error
		TerminateVM(ctx context.Context, sessionID string) error
	}

	CreateVMOptions struct {
		StartURL string         `json:"start_url,omitempty"`
		Region   string         `json:"region,omitempty"`
		Timeout  *VMTimeoutOpts `json:"timeout,omitempty"`
		Width    int            `json:"width,omitempty"`
		Height   int            `json:"height,omitempty"`
		FPS      int            `json:"fps,omitempty"`
		Adblock  bool           `json:"adblock,omitempty"`
		WebGL    bool           `json:"webgl,omitempty"`
		Dark     bool           `json:"dark,omitempty"`
		Quality  *VMQualityOpts `json:"quality,omitempty"`
	}

	VMQualityOpts struct {
		Mode string `json:"mode,omitempty"`
	}

	VMTimeoutOpts struct {
		Offline  int `json:"offline,omitempty"`
		Absolute int `json:"absolute,omitempty"`
	}

	VM struct {
		SessionID  string `json:"session_id"`
		EmbedURL   string `json:"embed_url"`
		AdminToken string `json:"admin_token"`
	}

	VMStatus struct {
		SessionID       string `json:"session_id"`
		TerminationDate string `json:"termination_date,omitempty"`
		EmbedURL        string `json:"embed_url"`
	}

	service struct {
		settingsSvc settings.Service
		baseURL     string
		client      *httpclient.Client
	}
)

const (
	defaultBaseURL  = "https://engine.hyperbeam.com/v0"
	requestTimeout  = 15 * time.Second
	ControlRoleName = "control"
)

func NewService(settingsSvc settings.Service) Service {
	svc := &service{
		settingsSvc: settingsSvc,
		baseURL:     defaultBaseURL,
		client:      httpclient.New(requestTimeout),
	}
	if !svc.Enabled() {
		logger.Log.Warn().Msg("no hyperbeam api key is set: virtual browser watch parties are disabled until one is saved in admin settings")
	}
	return svc
}

func (s *service) apiKey() string {
	return strings.TrimSpace(s.settingsSvc.Get(context.Background(), config.SettingHyperbeamAPIKey))
}

func (s *service) Enabled() bool {
	return s.apiKey() != ""
}

func (s *service) CreateVM(ctx context.Context, opts CreateVMOptions) (*VM, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	var out VM
	if err := s.do(ctx, http.MethodPost, s.baseURL+"/vm", opts, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *service) GetVMStatus(ctx context.Context, sessionID string) (*VMStatus, error) {
	if !s.Enabled() {
		return nil, ErrDisabled
	}
	if sessionID == "" {
		return nil, fmt.Errorf("hyperbeam: empty session id")
	}
	var out VMStatus
	if err := s.do(ctx, http.MethodGet, s.baseURL+"/vm/"+sessionID, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *service) SetControlRole(ctx context.Context, vmBaseURL, adminToken, userIdentifier string, hasControl bool) error {
	if !s.Enabled() {
		return ErrDisabled
	}
	if vmBaseURL == "" {
		return fmt.Errorf("hyperbeam: empty vm base url")
	}
	if adminToken == "" {
		return ErrMissingAdminToken
	}
	if userIdentifier == "" {
		return fmt.Errorf("hyperbeam: empty user identifier")
	}
	path := "/removeRoles"
	if hasControl {
		path = "/addRoles"
	}
	body := []any{
		[]string{userIdentifier},
		[]string{ControlRoleName},
	}
	fullURL := strings.TrimRight(vmBaseURL, "/") + path
	logger.Ctx(ctx).Info().
		Str("vm_base_url", vmBaseURL).
		Str("user_identifier", userIdentifier).
		Bool("has_control", hasControl).
		Str("path", path).
		Msg("hyperbeam role change")
	if err := s.doAs(ctx, adminToken, http.MethodPost, fullURL, body, nil); err != nil {
		logger.Ctx(ctx).Warn().Err(err).Str("user_identifier", userIdentifier).Str("path", path).Msg("hyperbeam role change failed")
		return err
	}
	return nil
}

func (s *service) TerminateVM(ctx context.Context, sessionID string) error {
	if !s.Enabled() {
		return ErrDisabled
	}
	if sessionID == "" {
		return fmt.Errorf("hyperbeam: empty session id")
	}
	return s.do(ctx, http.MethodDelete, s.baseURL+"/vm/"+sessionID, nil, nil)
}

func extractErrorCode(raw []byte) string {
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	return payload.Code
}

func ExtractVMBaseURL(embedURL string) (string, error) {
	if embedURL == "" {
		return "", fmt.Errorf("empty embed_url")
	}

	u, err := url.Parse(embedURL)
	if err != nil {
		return "", fmt.Errorf("parse embed_url: %w", err)
	}

	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("embed_url missing scheme/host")
	}

	path := strings.TrimRight(u.Path, "/")
	return fmt.Sprintf("%s://%s%s", u.Scheme, u.Host, path), nil
}

func (s *service) do(ctx context.Context, method, fullURL string, body any, out any) error {
	return s.doAs(ctx, s.apiKey(), method, fullURL, body, out)
}

func (s *service) doAs(ctx context.Context, bearer, method, fullURL string, body any, out any) error {
	err := s.client.JSON(ctx, httpclient.Request{
		Method: method,
		URL:    fullURL,
		Bearer: bearer,
		JSON:   body,
	}, out)
	if err == nil {
		return nil
	}

	if statusErr, ok := errors.AsType[*httpclient.StatusError](err); ok {
		return &APIError{StatusCode: statusErr.StatusCode, Code: extractErrorCode(statusErr.Body), Body: string(statusErr.Body)}
	}

	return fmt.Errorf("call hyperbeam: %w", err)
}
