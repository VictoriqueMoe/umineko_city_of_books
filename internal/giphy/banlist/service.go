package banlist

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"umineko_city_of_books/internal/repository"
)

type Kind string

const (
	KindGif  Kind = "gif"
	KindUser Kind = "user"
)

type (
	Service interface {
		ContainsGif(id string) bool
		ContainsUser(username string) bool
		List(ctx context.Context) ([]Entry, error)
		Add(ctx context.Context, kind Kind, value, reason string, createdBy *string) error
		Remove(ctx context.Context, kind Kind, value string) error
	}

	Entry struct {
		Kind      Kind      `json:"kind"`
		Value     string    `json:"value"`
		Reason    string    `json:"reason"`
		CreatedAt time.Time `json:"created_at"`
		CreatedBy *string   `json:"created_by,omitempty"`
	}

	service struct {
		repo  repository.BannedGiphyRepository
		mu    sync.RWMutex
		gifs  map[string]struct{}
		users map[string]struct{}
	}
)

var (
	ErrInvalidKind   = errors.New("invalid kind")
	ErrValueRequired = errors.New("value required")
)

func NewService(ctx context.Context, repo repository.BannedGiphyRepository) (Service, error) {
	s := &service{
		repo:  repo,
		gifs:  make(map[string]struct{}),
		users: make(map[string]struct{}),
	}
	rows, err := repo.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		switch Kind(r.Kind) {
		case KindGif:
			s.gifs[r.Value] = struct{}{}
		case KindUser:
			s.users[strings.ToLower(r.Value)] = struct{}{}
		}
	}
	return s, nil
}

func (s *service) ContainsGif(id string) bool {
	if id == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.gifs[id]
	return ok
}

func (s *service) ContainsUser(username string) bool {
	if username == "" {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.users[strings.ToLower(username)]
	return ok
}

func (s *service) List(ctx context.Context) ([]Entry, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(rows))
	for _, r := range rows {
		out = append(out, Entry{
			Kind:      Kind(r.Kind),
			Value:     r.Value,
			Reason:    r.Reason,
			CreatedAt: r.CreatedAt,
			CreatedBy: r.CreatedBy,
		})
	}
	return out, nil
}

func (s *service) Add(ctx context.Context, kind Kind, value, reason string, createdBy *string) error {
	if kind != KindGif && kind != KindUser {
		return ErrInvalidKind
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return ErrValueRequired
	}
	if err := s.repo.Add(ctx, string(kind), value, reason, createdBy); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch kind {
	case KindGif:
		s.gifs[value] = struct{}{}
	case KindUser:
		s.users[strings.ToLower(value)] = struct{}{}
	}
	return nil
}

func (s *service) Remove(ctx context.Context, kind Kind, value string) error {
	if kind != KindGif && kind != KindUser {
		return ErrInvalidKind
	}
	if err := s.repo.Remove(ctx, string(kind), value); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	switch kind {
	case KindGif:
		delete(s.gifs, value)
	case KindUser:
		delete(s.users, strings.ToLower(value))
	}
	return nil
}
