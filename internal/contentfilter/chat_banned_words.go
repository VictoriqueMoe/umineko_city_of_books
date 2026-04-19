package contentfilter

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"

	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

const (
	MatchModeSubstring = "substring"
	MatchModeWholeWord = "whole_word"
	MatchModeRegex     = "regex"

	BannedWordActionDelete = "delete"
	BannedWordActionKick   = "kick"
)

var ErrInvalidRegex = errors.New("invalid regex pattern")

type (
	ChatBannedWordsRule struct {
		repo  repository.ChatBannedWordRepository
		cache sync.Map
	}

	ChatBannedWordMatch struct {
		RuleID    uuid.UUID
		Scope     string
		Pattern   string
		Action    string
		MatchedOn string
	}

	compiledRule struct {
		id      uuid.UUID
		scope   string
		pattern string
		action  string
		re      *regexp.Regexp
	}
)

func NewChatBannedWordsRule(repo repository.ChatBannedWordRepository) *ChatBannedWordsRule {
	return &ChatBannedWordsRule{repo: repo}
}

func (r *ChatBannedWordsRule) CheckForRoom(ctx context.Context, roomID uuid.UUID, texts ...string) (*ChatBannedWordMatch, error) {
	rows, err := r.repo.ListApplicable(ctx, roomID)
	if err != nil {
		return nil, fmt.Errorf("list applicable banned words: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	for i := 0; i < len(rows); i++ {
		row := rows[i]
		compiled, err := r.compile(row)
		if err != nil {
			continue
		}
		for j := 0; j < len(texts); j++ {
			text := texts[j]
			if text == "" {
				continue
			}
			if match := compiled.re.FindString(text); match != "" {
				return &ChatBannedWordMatch{
					RuleID:    compiled.id,
					Scope:     compiled.scope,
					Pattern:   compiled.pattern,
					Action:    compiled.action,
					MatchedOn: match,
				}, nil
			}
		}
	}
	return nil, nil
}

func (r *ChatBannedWordsRule) compile(row repository.ChatBannedWordRow) (*compiledRule, error) {
	if cached, ok := r.cache.Load(row.ID); ok {
		return cached.(*compiledRule), nil
	}
	expr, err := CompileBannedWordPattern(row.Pattern, row.MatchMode, row.CaseSensitive)
	if err != nil {
		return nil, err
	}
	compiled := &compiledRule{
		id:      row.ID,
		scope:   row.Scope,
		pattern: row.Pattern,
		action:  row.Action,
		re:      expr,
	}
	r.cache.Store(row.ID, compiled)
	return compiled, nil
}

func (r *ChatBannedWordsRule) Invalidate(id uuid.UUID) {
	r.cache.Delete(id)
}

func CompileBannedWordPattern(pattern, mode string, caseSensitive bool) (*regexp.Regexp, error) {
	var expr string
	switch mode {
	case MatchModeSubstring:
		expr = regexp.QuoteMeta(pattern)
	case MatchModeWholeWord:
		expr = `\b` + regexp.QuoteMeta(pattern) + `\b`
	case MatchModeRegex:
		expr = pattern
	default:
		return nil, fmt.Errorf("%w: unknown match mode %q", ErrInvalidRegex, mode)
	}
	if !caseSensitive {
		expr = `(?i)` + expr
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRegex, err)
	}
	return re, nil
}

func ValidateBannedWordAction(action string) error {
	if action != BannedWordActionDelete && action != BannedWordActionKick {
		return fmt.Errorf("invalid action %q", action)
	}
	return nil
}

func ValidateBannedWordMode(mode string) error {
	switch mode {
	case MatchModeSubstring, MatchModeWholeWord, MatchModeRegex:
		return nil
	}
	return fmt.Errorf("invalid match mode %q", mode)
}
