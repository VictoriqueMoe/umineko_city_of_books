package dto

import "github.com/google/uuid"

type (
	NotificationType string

	NotifyParams struct {
		RecipientID   uuid.UUID
		Type          NotificationType
		ReferenceID   uuid.UUID
		ReferenceType string
		ActorID       uuid.UUID
		Message       string
		EmailSubject  string
		EmailBody     string
	}

	NotificationResponse struct {
		ID            int              `json:"id"`
		Type          NotificationType `json:"type"`
		ReferenceID   uuid.UUID        `json:"reference_id"`
		ReferenceType string           `json:"reference_type"`
		Actor         UserResponse     `json:"actor"`
		Message       string           `json:"message,omitempty"`
		Read          bool             `json:"read"`
		CreatedAt     string           `json:"created_at"`
	}

	NotificationListResponse struct {
		Notifications []NotificationResponse `json:"notifications"`
		Total         int                    `json:"total"`
		Limit         int                    `json:"limit"`
		Offset        int                    `json:"offset"`
	}

	UnreadCountResponse struct {
		Count int `json:"count"`
	}
)

const (
	NotifTheoryResponse           NotificationType = "theory_response"
	NotifResponseReply            NotificationType = "response_reply"
	NotifTheoryUpvote             NotificationType = "theory_upvote"
	NotifResponseUpvote           NotificationType = "response_upvote"
	NotifChatMessage              NotificationType = "chat_message"
	NotifReport                   NotificationType = "report"
	NotifNewFollower              NotificationType = "new_follower"
	NotifPostLiked                NotificationType = "post_liked"
	NotifPostCommented            NotificationType = "post_commented"
	NotifPostCommentReply         NotificationType = "post_comment_reply"
	NotifMention                  NotificationType = "mention"
	NotifArtLiked                 NotificationType = "art_liked"
	NotifArtCommented             NotificationType = "art_commented"
	NotifArtCommentReply          NotificationType = "art_comment_reply"
	NotifCommentLiked             NotificationType = "comment_liked"
	NotifReportResolved           NotificationType = "report_resolved"
	NotifContentEdited            NotificationType = "content_edited"
	NotifMysteryAttempt           NotificationType = "mystery_attempt"
	NotifMysteryReply             NotificationType = "mystery_reply"
	NotifMysteryVote              NotificationType = "mystery_attempt_vote"
	NotifMysterySolved            NotificationType = "mystery_solved"
	NotifMysteryPaused            NotificationType = "mystery_paused_notif"
	NotifMysteryUnpaused          NotificationType = "mystery_unpaused"
	NotifMysteryGmAway            NotificationType = "mystery_gm_away_notif"
	NotifMysteryGmBack            NotificationType = "mystery_gm_back_notif"
	NotifMysterySolvedAll         NotificationType = "mystery_solved_all"
	NotifMysteryCommentReply      NotificationType = "mystery_comment_reply"
	NotifMysteryPrivateClue       NotificationType = "mystery_private_clue"
	NotifJournalUpdate            NotificationType = "journal_update"
	NotifJournalCommented         NotificationType = "journal_commented"
	NotifJournalCommentReply      NotificationType = "journal_comment_reply"
	NotifJournalCommentLiked      NotificationType = "journal_comment_liked"
	NotifJournalFollowed          NotificationType = "journal_followed"
	NotifJournalArchived          NotificationType = "journal_archived"
	NotifChatMention              NotificationType = "chat_mention"
	NotifChatRoomMessage          NotificationType = "chat_room_message"
	NotifChatRoomInvite           NotificationType = "chat_room_invite"
	NotifChatReply                NotificationType = "chat_reply"
	NotifShipCommented            NotificationType = "ship_commented"
	NotifShipCommentReply         NotificationType = "ship_comment_reply"
	NotifShipCommentLiked         NotificationType = "ship_comment_liked"
	NotifAnnouncementCommented    NotificationType = "announcement_commented"
	NotifAnnouncementCommentReply NotificationType = "announcement_comment_reply"
	NotifAnnouncementCommentLiked NotificationType = "announcement_comment_liked"
	NotifFanficCommented          NotificationType = "fanfic_commented"
	NotifFanficCommentReply       NotificationType = "fanfic_comment_reply"
	NotifFanficCommentLiked       NotificationType = "fanfic_comment_liked"
	NotifFanficFavourited         NotificationType = "fanfic_favourited"
	NotifSuggestionPosted         NotificationType = "suggestion_posted"
	NotifSuggestionResolved       NotificationType = "suggestion_resolved"
	NotifContentShared            NotificationType = "content_shared"
)
