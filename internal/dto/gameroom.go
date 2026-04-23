package dto

import (
	"encoding/json"

	"github.com/google/uuid"
)

type (
	GameType   string
	GameStatus string

	GameRoomPlayer struct {
		UserID         uuid.UUID    `json:"user_id"`
		Username       string       `json:"username"`
		DisplayName    string       `json:"display_name"`
		AvatarURL      string       `json:"avatar_url"`
		Role           string       `json:"role"`
		Slot           int          `json:"slot"`
		Joined         bool         `json:"joined"`
		Connected      bool         `json:"connected"`
		DisconnectedAt *string      `json:"disconnected_at,omitempty"`
		User           UserResponse `json:"user"`
	}

	GameRoom struct {
		ID           uuid.UUID        `json:"id"`
		GameType     GameType         `json:"game_type"`
		Status       GameStatus       `json:"status"`
		State        json.RawMessage  `json:"state"`
		TurnUserID   *uuid.UUID       `json:"turn_user_id,omitempty"`
		WinnerID     *uuid.UUID       `json:"winner_user_id,omitempty"`
		Result       string           `json:"result,omitempty"`
		CreatedBy    uuid.UUID        `json:"created_by"`
		CreatedAt    string           `json:"created_at"`
		UpdatedAt    string           `json:"updated_at"`
		FinishedAt   *string          `json:"finished_at,omitempty"`
		Players      []GameRoomPlayer `json:"players"`
		WatcherCount int              `json:"watcher_count"`
		Stats        json.RawMessage  `json:"stats,omitempty"`
	}

	SpectatorMessage struct {
		ID        string       `json:"id"`
		UserID    uuid.UUID    `json:"user_id"`
		User      UserResponse `json:"user"`
		Body      string       `json:"body"`
		CreatedAt string       `json:"created_at"`
	}

	SpectatorChatRequest struct {
		Body string `json:"body"`
	}

	SpectatorChatResponse struct {
		Messages []SpectatorMessage `json:"messages"`
	}

	GameRoomListResponse struct {
		Rooms []GameRoom `json:"rooms"`
		Total int        `json:"total"`
	}

	GameInviteRequest struct {
		GameType   GameType  `json:"game_type"`
		OpponentID uuid.UUID `json:"opponent_id"`
	}

	GameActionRequest struct {
		Action json.RawMessage `json:"action"`
	}

	GameMove struct {
		Ply    int             `json:"ply"`
		UserID uuid.UUID       `json:"user_id"`
		Action json.RawMessage `json:"action"`
	}

	GameScoreboardRow struct {
		User        UserResponse `json:"user"`
		Wins        int          `json:"wins"`
		Losses      int          `json:"losses"`
		Draws       int          `json:"draws"`
		GamesPlayed int          `json:"games_played"`
		WinRate     float64      `json:"win_rate"`
	}

	GameScoreboardResponse struct {
		GameType GameType            `json:"game_type"`
		Rows     []GameScoreboardRow `json:"rows"`
	}
)

const (
	GameTypeChess GameType = "chess"

	GameStatusPending   GameStatus = "pending"
	GameStatusActive    GameStatus = "active"
	GameStatusFinished  GameStatus = "finished"
	GameStatusDeclined  GameStatus = "declined"
	GameStatusAbandoned GameStatus = "abandoned"
)
