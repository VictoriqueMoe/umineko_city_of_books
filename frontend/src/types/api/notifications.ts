import type { PaginationFields, User } from "./common";

export type NotificationType =
    | "theory_response"
    | "response_reply"
    | "theory_upvote"
    | "response_upvote"
    | "chat_message"
    | "report"
    | "report_resolved"
    | "new_follower"
    | "post_liked"
    | "post_commented"
    | "post_comment_reply"
    | "mention"
    | "art_liked"
    | "art_commented"
    | "art_comment_reply"
    | "comment_liked"
    | "content_edited"
    | "mystery_attempt"
    | "mystery_reply"
    | "mystery_attempt_vote"
    | "mystery_solved"
    | "mystery_paused_notif"
    | "mystery_unpaused"
    | "mystery_gm_away_notif"
    | "mystery_gm_back_notif"
    | "mystery_solved_all"
    | "mystery_comment_reply"
    | "mystery_private_clue"
    | "journal_update"
    | "journal_commented"
    | "journal_comment_reply"
    | "journal_comment_liked"
    | "journal_followed"
    | "journal_archived"
    | "chat_mention"
    | "chat_room_message"
    | "chat_room_invite"
    | "chat_reply"
    | "chat_room_banned"
    | "chat_room_kicked"
    | "chat_room_unbanned"
    | "secret_comment_reply"
    | "secret_commented"
    | "secret_comment_liked"
    | "secret_solved_by_other"
    | "fanfic_commented"
    | "fanfic_comment_reply"
    | "fanfic_comment_liked"
    | "fanfic_favourited"
    | "ship_commented"
    | "ship_comment_reply"
    | "ship_comment_liked"
    | "oc_commented"
    | "oc_comment_reply"
    | "oc_comment_liked"
    | "oc_favourited"
    | "announcement_commented"
    | "announcement_comment_reply"
    | "announcement_comment_liked"
    | "suggestion_posted"
    | "suggestion_resolved"
    | "content_shared"
    | "game_invite"
    | "game_your_turn"
    | "game_finished"
    | "stream_live"
    | "mystery_created"
    | "theory_created"
    | "theory_refuted";

export interface Notification {
    id: number;
    type: NotificationType;
    reference_id: string;
    reference_type: string;
    actor: User;
    message?: string;
    read: boolean;
    created_at: string;
    count: number;
}

export interface NotificationListResponse extends PaginationFields {
    notifications: Notification[];
}
