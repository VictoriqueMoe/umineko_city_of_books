package dto

import (
	"umineko_city_of_books/internal/role"

	"github.com/google/uuid"
)

type (
	AdminUserItem struct {
		ID          uuid.UUID `json:"id"`
		Username    string    `json:"username"`
		DisplayName string    `json:"display_name"`
		AvatarURL   string    `json:"avatar_url"`
		Role        role.Role `json:"role,omitempty"`
		Banned      bool      `json:"banned"`
		CreatedAt   string    `json:"created_at"`
	}

	AdminUserListResponse struct {
		Users  []AdminUserItem `json:"users"`
		Total  int             `json:"total"`
		Limit  int             `json:"limit"`
		Offset int             `json:"offset"`
	}

	AdminUserDetailResponse struct {
		AdminUserItem
		BanReason     string `json:"ban_reason,omitempty"`
		BannedAt      string `json:"banned_at,omitempty"`
		TheoryCount   int    `json:"theory_count"`
		ResponseCount int    `json:"response_count"`
	}

	AdminStatsResponse struct {
		TotalUsers      int              `json:"total_users"`
		TotalTheories   int              `json:"total_theories"`
		TotalResponses  int              `json:"total_responses"`
		TotalVotes      int              `json:"total_votes"`
		NewUsers24h     int              `json:"new_users_24h"`
		NewUsers7d      int              `json:"new_users_7d"`
		NewUsers30d     int              `json:"new_users_30d"`
		NewTheories24h  int              `json:"new_theories_24h"`
		NewTheories7d   int              `json:"new_theories_7d"`
		NewTheories30d  int              `json:"new_theories_30d"`
		NewResponses24h int              `json:"new_responses_24h"`
		NewResponses7d  int              `json:"new_responses_7d"`
		NewResponses30d int              `json:"new_responses_30d"`
		MostActiveUsers []MostActiveUser `json:"most_active_users"`
	}

	MostActiveUser struct {
		ID          uuid.UUID `json:"id"`
		Username    string    `json:"username"`
		DisplayName string    `json:"display_name"`
		AvatarURL   string    `json:"avatar_url"`
		ActionCount int       `json:"action_count"`
	}

	AuditLogEntryResponse struct {
		ID         int       `json:"id"`
		ActorID    uuid.UUID `json:"actor_id"`
		ActorName  string    `json:"actor_name"`
		Action     string    `json:"action"`
		TargetType string    `json:"target_type"`
		TargetID   string    `json:"target_id"`
		Details    string    `json:"details"`
		CreatedAt  string    `json:"created_at"`
	}

	AuditLogListResponse struct {
		Entries []AuditLogEntryResponse `json:"entries"`
		Total   int                     `json:"total"`
		Limit   int                     `json:"limit"`
		Offset  int                     `json:"offset"`
	}

	SettingsResponse struct {
		Settings map[string]string `json:"settings"`
	}

	UpdateSettingsRequest struct {
		Settings map[string]string `json:"settings"`
	}

	SetRoleRequest struct {
		Role string `json:"role"`
	}

	BanUserRequest struct {
		Reason string `json:"reason"`
	}
)
