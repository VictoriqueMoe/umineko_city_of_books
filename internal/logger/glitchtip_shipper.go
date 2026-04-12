package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type (
	glitchtipShipper struct {
		endpoint   string
		authHeader string
		dsnURL     string
		client     *http.Client

		mu      sync.Mutex
		batch   []glitchtipLogItem
		quit    chan struct{}
		running bool
	}

	glitchtipLogItem struct {
		Timestamp      float64                       `json:"timestamp"`
		Level          string                        `json:"level"`
		Body           string                        `json:"body"`
		SeverityNumber int                           `json:"severity_number"`
		Attributes     map[string]glitchtipAttribute `json:"attributes,omitempty"`
	}

	glitchtipAttribute struct {
		Value interface{} `json:"value"`
		Type  string      `json:"type"`
	}

	glitchtipEnvelopeHeader struct {
		DSN string `json:"dsn"`
	}

	glitchtipItemHeader struct {
		Type        string `json:"type"`
		ItemCount   int    `json:"item_count"`
		ContentType string `json:"content_type"`
	}

	glitchtipItemPayload struct {
		Items []glitchtipLogItem `json:"items"`
	}
)

const (
	batchSize     = 100
	flushInterval = 5 * time.Second
)

func newGlitchtipShipper(dsn string) (*glitchtipShipper, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	if u.User == nil {
		return nil, fmt.Errorf("dsn missing auth")
	}
	publicKey := u.User.Username()
	path := strings.TrimPrefix(u.Path, "/")
	if path == "" {
		return nil, fmt.Errorf("dsn missing project id")
	}
	projectID := path

	endpoint := fmt.Sprintf("%s://%s/api/%s/envelope/", u.Scheme, u.Host, projectID)
	authHeader := fmt.Sprintf("Sentry sentry_version=7, sentry_key=%s, sentry_client=umineko-custom/1.0", publicKey)

	s := &glitchtipShipper{
		endpoint:   endpoint,
		authHeader: authHeader,
		dsnURL:     dsn,
		client:     &http.Client{Timeout: 10 * time.Second},
		quit:       make(chan struct{}),
	}
	return s, nil
}

func (s *glitchtipShipper) start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go s.flushLoop()
}

func (s *glitchtipShipper) stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.quit)
	s.mu.Unlock()
	s.flush()
}

func (s *glitchtipShipper) flushLoop() {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.quit:
			return
		case <-ticker.C:
			s.flush()
		}
	}
}

func (s *glitchtipShipper) Write(p []byte) (int, error) {
	return len(p), nil
}

func (s *glitchtipShipper) WriteLevel(level zerolog.Level, p []byte) (int, error) {
	var parsed map[string]interface{}
	if err := json.Unmarshal(p, &parsed); err != nil {
		return len(p), nil
	}
	msg, _ := parsed["message"].(string)
	if msg == "" {
		return len(p), nil
	}

	attrs := make(map[string]glitchtipAttribute)
	for k, v := range parsed {
		if k == "message" || k == "level" || k == "time" {
			continue
		}
		attrs[k] = toGlitchtipAttribute(v)
	}

	item := glitchtipLogItem{
		Timestamp:      float64(time.Now().UnixNano()) / 1e9,
		Level:          levelToString(level),
		Body:           msg,
		SeverityNumber: levelToSeverity(level),
		Attributes:     attrs,
	}

	s.mu.Lock()
	s.batch = append(s.batch, item)
	shouldFlush := len(s.batch) >= batchSize
	s.mu.Unlock()

	if shouldFlush {
		go s.flush()
	}
	return len(p), nil
}

func (s *glitchtipShipper) flush() {
	s.mu.Lock()
	if len(s.batch) == 0 {
		s.mu.Unlock()
		return
	}
	items := s.batch
	s.batch = nil
	s.mu.Unlock()

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	if err := enc.Encode(glitchtipEnvelopeHeader{DSN: s.dsnURL}); err != nil {
		return
	}
	if err := enc.Encode(glitchtipItemHeader{
		Type:        "log",
		ItemCount:   len(items),
		ContentType: "application/vnd.sentry.items.log+json",
	}); err != nil {
		return
	}
	if err := enc.Encode(glitchtipItemPayload{Items: items}); err != nil {
		return
	}

	req, err := http.NewRequest("POST", s.endpoint, &buf)
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/x-sentry-envelope")
	req.Header.Set("X-Sentry-Auth", s.authHeader)

	resp, err := s.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

func toGlitchtipAttribute(v interface{}) glitchtipAttribute {
	switch val := v.(type) {
	case string:
		return glitchtipAttribute{Value: val, Type: "string"}
	case bool:
		return glitchtipAttribute{Value: val, Type: "boolean"}
	case float64:
		if val == float64(int64(val)) {
			return glitchtipAttribute{Value: int64(val), Type: "integer"}
		}
		return glitchtipAttribute{Value: val, Type: "double"}
	}
	return glitchtipAttribute{Value: fmt.Sprintf("%v", v), Type: "string"}
}

func levelToString(level zerolog.Level) string {
	switch level {
	case zerolog.TraceLevel:
		return "trace"
	case zerolog.DebugLevel:
		return "debug"
	case zerolog.InfoLevel:
		return "info"
	case zerolog.WarnLevel:
		return "warn"
	case zerolog.ErrorLevel:
		return "error"
	case zerolog.FatalLevel:
		return "fatal"
	case zerolog.PanicLevel:
		return "fatal"
	}
	return "info"
}

func levelToSeverity(level zerolog.Level) int {
	switch level {
	case zerolog.TraceLevel:
		return 1
	case zerolog.DebugLevel:
		return 5
	case zerolog.InfoLevel:
		return 9
	case zerolog.WarnLevel:
		return 13
	case zerolog.ErrorLevel:
		return 17
	case zerolog.FatalLevel:
		return 21
	case zerolog.PanicLevel:
		return 21
	}
	return 9
}
