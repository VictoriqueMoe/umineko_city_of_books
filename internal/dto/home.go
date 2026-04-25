package dto

import "github.com/google/uuid"

type (
	HomeActivityAuthor struct {
		ID          uuid.UUID `json:"id"`
		Username    string    `json:"username"`
		DisplayName string    `json:"display_name"`
		AvatarURL   string    `json:"avatar_url"`
	}

	HomeActivityEntry struct {
		Kind      string             `json:"kind"`
		ID        uuid.UUID          `json:"id"`
		Title     string             `json:"title"`
		Excerpt   string             `json:"excerpt"`
		Corner    string             `json:"corner"`
		URL       string             `json:"url"`
		CreatedAt string             `json:"created_at"`
		Author    HomeActivityAuthor `json:"author"`
	}

	HomeMember struct {
		ID          uuid.UUID `json:"id"`
		Username    string    `json:"username"`
		DisplayName string    `json:"display_name"`
		AvatarURL   string    `json:"avatar_url"`
		CreatedAt   string    `json:"created_at"`
	}

	HomePublicRoom struct {
		ID            uuid.UUID `json:"id"`
		Name          string    `json:"name"`
		Description   string    `json:"description"`
		MemberCount   int       `json:"member_count"`
		LastMessageAt *string   `json:"last_message_at"`
	}

	HomeCornerActivity struct {
		Corner        string  `json:"corner"`
		PostCount     int     `json:"post_count"`
		UniquePosters int     `json:"unique_posters"`
		LastPostAt    *string `json:"last_post_at"`
	}

	HomeActivityResponse struct {
		OnlineCount    int                  `json:"online_count"`
		RecentActivity []HomeActivityEntry  `json:"recent_activity"`
		RecentMembers  []HomeMember         `json:"recent_members"`
		PublicRooms    []HomePublicRoom     `json:"public_rooms"`
		CornerActivity []HomeCornerActivity `json:"corner_activity"`
	}

	SidebarActivityResponse struct {
		Activity map[string]string `json:"activity"`
	}

	SidebarLastVisitedResponse struct {
		Visited map[string]string `json:"visited"`
	}

	MarkSidebarVisitedRequest struct {
		Key string `json:"key"`
	}
)
