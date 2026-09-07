package model

import (
	"github.com/google/uuid"
)

type (
	SecretSolver struct {
		UserID      uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
		Role        string
		UnlockedAt  string
	}

	SecretLeaderboardRow struct {
		UserID      uuid.UUID
		Username    string
		DisplayName string
		AvatarURL   string
		Role        string
		Pieces      int
	}

	SecretSolverRow struct {
		UserID       uuid.UUID
		Username     string
		DisplayName  string
		AvatarURL    string
		Role         string
		SolvedCount  int
		LastSolvedAt string
	}
)
