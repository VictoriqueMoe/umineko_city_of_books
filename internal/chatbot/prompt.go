package chatbot

import (
	"context"
	"fmt"
	"strings"

	"umineko_city_of_books/internal/openai"
	"umineko_city_of_books/internal/repository"

	"github.com/google/uuid"
)

type (
	promptRow struct {
		AuthorID    uuid.UUID
		DisplayName string
		Username    string
		Body        string
		IsSystem    bool
		ReplyToName string
		ReplyToBody string
	}
)

func truncate(body string, limit int) string {
	if len(body) <= limit {
		return body
	}

	return body[:limit] + "..."
}

func fitBudget(messages []openai.Message) []openai.Message {
	if len(messages) <= 1 {
		return messages
	}

	total := 0
	for i := range messages {
		total += len(messages[i].Content)
	}

	drop := 0
	for drop < len(messages)-1 && total > promptCharBudget {
		total -= len(messages[drop].Content)
		drop++
	}

	return messages[drop:]
}

func (s *service) buildMessages(ctx context.Context, j job, tune tuning) []openai.Message {
	if j.ev.IsDM {
		return s.dmHistory(ctx, j.ev, j.bot.UserID, tune)
	}

	if j.useChain {
		return s.replyChain(ctx, j.ev, j.bot.UserID, tune)
	}

	return []openai.Message{{Role: "user", Content: authored(j.ev.Body, speaker(j.ev.SenderHandle, j.ev.SenderName), messageBodyMax)}}
}

func (s *service) dmHistory(ctx context.Context, ev botEvent, botUserID uuid.UUID, tune tuning) []openai.Message {
	limit := tune.contextMessages
	if limit <= 0 {
		limit = 1
	}

	rows, err := s.chatRepo.GetMessagesForMember(ctx, ev.ScopeID, ev.SenderID, limit)
	if err != nil {
		return []openai.Message{{Role: "user", Content: truncate(ev.Body, messageBodyMax)}}
	}

	out := make([]openai.Message, 0, len(rows))
	for i := range rows {
		if rows[i].IsSystem {
			continue
		}

		out = append(out, rowToMessage(chatPromptRow(rows[i]), botUserID, messageBodyMax))
	}

	if len(out) == 0 {
		return []openai.Message{{Role: "user", Content: truncate(ev.Body, messageBodyMax)}}
	}

	return fitBudget(out)
}

func (s *service) replyChain(ctx context.Context, ev botEvent, botUserID uuid.UUID, tune tuning) []openai.Message {
	depth := tune.maxReplyChain
	if depth <= 0 {
		depth = 1
	}

	chain := make([]openai.Message, 0, depth+1)
	cursor := ev.ParentID

	for range depth {
		if cursor == nil {
			break
		}

		row, parentID, ok := s.parentRow(ctx, ev, *cursor)
		if !ok {
			break
		}

		if !row.IsSystem {
			chain = append(chain, rowToMessage(row, botUserID, messageBodyMax))
		}

		cursor = parentID
	}

	ordered := make([]openai.Message, 0, len(chain)+1)
	for i := len(chain) - 1; i >= 0; i-- {
		ordered = append(ordered, chain[i])
	}

	return fitBudget(append(ordered, openai.Message{Role: "user", Content: authored(ev.Body, speaker(ev.SenderHandle, ev.SenderName), messageBodyMax)}))
}

func (s *service) parentRow(ctx context.Context, ev botEvent, id uuid.UUID) (promptRow, *uuid.UUID, bool) {
	if ev.Surface.gameBoard() {
		row, err := s.postRepo.GetCommentByID(ctx, id)
		if err != nil || row == nil || row.EntityID != ev.ScopeID.String() {
			return promptRow{}, nil, false
		}

		return commentPromptRow(*row), row.ParentID, true
	}

	row, err := s.chatRepo.GetMessageByID(ctx, id)
	if err != nil || row == nil || row.RoomID != ev.ScopeID {
		return promptRow{}, nil, false
	}

	return chatPromptRow(*row), row.ReplyToID, true
}

func chatPromptRow(row repository.ChatMessageRow) promptRow {
	out := promptRow{
		AuthorID:    row.SenderID,
		DisplayName: row.SenderDisplayName,
		Username:    row.SenderUsername,
		Body:        row.Body,
		IsSystem:    row.IsSystem,
	}

	if strings.TrimSpace(row.SenderNickname) != "" {
		out.DisplayName = row.SenderNickname
	}

	if row.ReplyToBody != nil {
		out.ReplyToBody = *row.ReplyToBody
	}

	if row.ReplyToSenderName != nil {
		out.ReplyToName = *row.ReplyToSenderName
	}

	return out
}

func commentPromptRow(row repository.CommentRow) promptRow {
	return promptRow{
		AuthorID:    row.UserID,
		DisplayName: row.AuthorDisplayName,
		Username:    row.AuthorUsername,
		Body:        row.Body,
	}
}

func rowToMessage(row promptRow, botUserID uuid.UUID, limit int) openai.Message {
	if row.AuthorID == botUserID {
		return openai.Message{Role: "assistant", Content: truncate(row.Body, limit)}
	}

	return openai.Message{Role: "user", Content: authored(row.Body, speaker(row.Username, row.DisplayName)+replyContext(row), limit)}
}

func replyContext(row promptRow) string {
	quoted := strings.TrimSpace(row.ReplyToBody)
	if quoted == "" {
		return ""
	}

	target := strings.TrimSpace(row.ReplyToName)
	if target == "" {
		return fmt.Sprintf(" (replying to %q)", truncate(quoted, replyQuoteMax))
	}

	return fmt.Sprintf(" (replying to %s: %q)", target, truncate(quoted, replyQuoteMax))
}

func speaker(handle, name string) string {
	handle = strings.TrimSpace(handle)
	name = strings.TrimSpace(name)

	switch {
	case handle == "":
		return name
	case name == "" || strings.EqualFold(handle, name):
		return "@" + handle
	default:
		return fmt.Sprintf("@%s (%s)", handle, name)
	}
}

func authored(body, name string, limit int) string {
	if name == "" {
		return truncate(body, limit)
	}

	return fmt.Sprintf("%s: %s", name, truncate(body, limit))
}
