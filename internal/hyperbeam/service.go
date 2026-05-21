package hyperbeam

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"umineko_city_of_books/internal/config"
	"umineko_city_of_books/internal/logger"
)

type (
	Service interface {
		Enabled() bool
		CreateVM(ctx context.Context, opts CreateVMOptions) (*VM, error)
		GetVMStatus(ctx context.Context, sessionID string) (*VMStatus, error)
		SetControlRole(ctx context.Context, vmBaseURL, userIdentifier string, hasControl bool) error
		TerminateVM(ctx context.Context, sessionID string) error
	}

	CreateVMOptions struct {
		StartURL string         `json:"start_url,omitempty"`
		Region   string         `json:"region,omitempty"`
		Timeout  *VMTimeoutOpts `json:"timeout,omitempty"`
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

	UserPerms struct {
		ControlDisabled  bool `json:"control_disabled"`
		ControlExclusive bool `json:"control_exclusive,omitempty"`
	}

	service struct {
		apiKey     string
		baseURL    string
		httpClient *http.Client
	}
)

const (
	defaultBaseURL  = "https://engine.hyperbeam.com/v0"
	requestTimeout  = 15 * time.Second
	ControlRoleName = "control"
)

func NewService() Service {
	svc := &service{
		apiKey:  config.Cfg.HyperbeamAPIKey,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}
	if !svc.Enabled() {
		logger.Log.Warn().Msg("HYPERBEAM_API_KEY is not set: watch parties are disabled")
	}
	return svc
}

func (s *service) Enabled() bool {
	return s.apiKey != ""
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

func (s *service) SetControlRole(ctx context.Context, vmBaseURL, userIdentifier string, hasControl bool) error {
	if !s.Enabled() {
		return ErrDisabled
	}
	if vmBaseURL == "" {
		return fmt.Errorf("hyperbeam: empty vm base url")
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
	logger.Log.Info().
		Str("vm_base_url", vmBaseURL).
		Str("user_identifier", userIdentifier).
		Bool("has_control", hasControl).
		Str("path", path).
		Msg("hyperbeam role change")
	if err := s.do(ctx, http.MethodPost, fullURL, body, nil); err != nil {
		logger.Log.Warn().Err(err).Str("user_identifier", userIdentifier).Str("path", path).Msg("hyperbeam role change failed")
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
	var reqBody io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode hyperbeam request: %w", err)
		}
		reqBody = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return fmt.Errorf("build hyperbeam request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call hyperbeam: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return &APIError{StatusCode: resp.StatusCode, Body: string(raw)}
	}

	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode hyperbeam response: %w", err)
	}

	return nil
}
