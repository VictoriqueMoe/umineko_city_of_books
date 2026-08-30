import type {
    ChatMessage,
    GameRoom,
    GameType,
    LiveStream,
    Notification,
    OCSummary,
    SecretProgressEvent,
    SecretSolvedEvent,
    SiteRole,
    SpectatorMessage,
    User,
    WatchPartyControlChangedEvent,
    WatchPartyEndedEvent,
    WatchPartyKickedEvent,
    WatchPartyParticipantEvent,
    WatchPartyParticipantLeftEvent,
    WatchPartyStartedEvent,
} from "../../types/api";
import type { PongFrame } from "../../games/pong/types";

export const REALTIME_EVENTS = {
    PONG: "pong",

    NOTIFICATION: "notification",

    ROLE_CHANGED: "role_changed",
    BAN_CHANGED: "ban_changed",
    LOCK_CHANGED: "lock_changed",
    PROFILE_CHANGED: "profile_changed",
    PERMISSIONS_CHANGED: "permissions_changed",

    TOP_DETECTIVE_CHANGED: "top_detective_changed",
    TOP_GM_CHANGED: "top_gm_changed",
    TOP_CHESS_CHANGED: "top_chess_changed",
    TOP_CHECKERS_CHANGED: "top_checkers_changed",
    TOP_OTHELLO_CHANGED: "top_othello_changed",
    TOP_MINESWEEPER_CHANGED: "top_minesweeper_changed",
    VANITY_ROLES_CHANGED: "vanity_roles_changed",
    RULES_PAGE_CHANGED: "rules_page_changed",

    NEW_ANNOUNCEMENT: "new_announcement",

    CHATBOTS_CHANGED: "chatbots_changed",

    SIDEBAR_ACTIVITY: "sidebar_activity",

    CHAT_MESSAGE: "chat_message",
    CHAT_MESSAGE_EDITED: "chat_message_edited",
    CHAT_MESSAGE_DELETED: "chat_message_deleted",
    CHAT_MESSAGE_PINNED: "chat_message_pinned",
    CHAT_MESSAGE_UNPINNED: "chat_message_unpinned",
    CHAT_REACTION_ADDED: "chat_reaction_added",
    CHAT_REACTION_REMOVED: "chat_reaction_removed",
    CHAT_MEMBER_JOINED: "chat_member_joined",
    CHAT_MEMBER_LEFT: "chat_member_left",
    CHAT_MEMBER_UPDATED: "chat_member_updated",
    CHAT_KICKED: "chat_kicked",
    CHAT_ROOM_INVITED: "chat_room_invited",
    CHAT_ROOM_UPDATED: "chat_room_updated",
    CHAT_ROOM_DELETED: "chat_room_deleted",
    CHAT_PRESENCE_CHANGED: "chat_presence_changed",
    CHAT_READ: "chat_read",
    CHAT_READ_RECEIPT: "chat_read_receipt",
    CHAT_UNREAD_BUMPED: "chat_unread_bumped",
    CHAT_AUDIO: "chat_audio",
    TYPING: "typing",
    VOICE_PRESENCE: "voice_presence",

    WATCH_PARTY_STARTED: "watch_party_started",
    WATCH_PARTY_ENDED: "watch_party_ended",
    WATCH_PARTY_PARTICIPANT_JOINED: "watch_party_participant_joined",
    WATCH_PARTY_PARTICIPANT_LEFT: "watch_party_participant_left",
    WATCH_PARTY_CONTROL_CHANGED: "watch_party_control_changed",
    WATCH_PARTY_KICKED: "watch_party_kicked",

    GAME_ROOM_STARTED: "game_room_started",
    GAME_ROOM_ACTION: "game_room_action",
    GAME_ROOM_FINISHED: "game_room_finished",
    GAME_ROOM_DECLINED: "game_room_declined",
    GAME_ROOM_CANCELLED: "game_room_cancelled",
    GAME_ROOM_PRESENCE: "game_room_presence",
    GAME_DRAW_OFFERED: "game_draw_offered",
    GAME_DRAW_DECLINED: "game_draw_declined",
    GAME_YOUR_TURN: "game_your_turn",
    GAME_FORFEIT_WARNING: "game_forfeit_warning",
    GAME_FORFEIT_CLEARED: "game_forfeit_cleared",
    LIVE_GAMES_COUNT: "live_games_count",
    PLAYER_CHAT_MESSAGE: "player_chat_message",
    SPECTATOR_CHAT_MESSAGE: "spectator_chat_message",
    GAME_PONG_FRAME: "game_pong_frame",

    MYSTERY_ATTEMPT_CREATED: "mystery_attempt_created",
    MYSTERY_SOLVED: "mystery_solved",
    MYSTERY_WINNER_ADDED: "mystery_winner_added",
    MYSTERY_CLUE_ADDED: "mystery_clue_added",
    MYSTERY_CLUE_UPDATED: "mystery_clue_updated",
    MYSTERY_PAUSED: "mystery_paused",
    MYSTERY_GM_AWAY: "mystery_gm_away",

    POST_LIKE: "post_like",
    POST_COMMENT: "post_comment",

    USER_OCS_CHANGED: "user_ocs_changed",

    SECRET_PROGRESS: "secret_progress",
    SECRET_SOLVED: "secret_solved",
    SECRET_CLOSED: "secret_closed",

    STREAM_LIVE: "stream_live",
    STREAM_OFFLINE: "stream_offline",
    STREAM_VIEWERS: "stream_viewers",
    STREAM_TITLE: "stream_title",
} as const satisfies Record<string, RealtimeEventName>;

type EmptyEventPayload = Record<string, never>;

export type PongPayload = EmptyEventPayload;

export type NotificationPayload = Notification;

export interface RoleChangedPayload {
    user_id: string;
    role: SiteRole | "";
}

export interface BanChangedPayload {
    user_id: string;
    banned: boolean;
    ban_reason?: string;
}

export interface LockChangedPayload {
    user_id: string;
    locked: boolean;
    lock_reason?: string;
}

export interface ProfileChangedPayload {
    user_id: string;
    display_name?: string;
    avatar_url?: string;
}

export type PermissionsChangedPayload = EmptyEventPayload;

export interface TopUserIdsPayload {
    user_ids: string[];
}

export type TopDetectiveChangedPayload = TopUserIdsPayload;

export type TopGmChangedPayload = TopUserIdsPayload;

export type TopChessChangedPayload = TopUserIdsPayload;

export type TopCheckersChangedPayload = TopUserIdsPayload;

export type TopOthelloChangedPayload = TopUserIdsPayload;

export type TopMinesweeperChangedPayload = TopUserIdsPayload;

export type VanityRolesChangedPayload = EmptyEventPayload;

export type RulesPageChangedPayload = EmptyEventPayload;

export interface NewAnnouncementPayload {
    id: string;
    title: string;
    author_id: string;
}

export type ChatbotsChangedPayload = EmptyEventPayload;

export interface SidebarActivityPayload {
    key: string;
    at: string;
}

export type ChatMessagePayload = ChatMessage;

export type ChatMessageEditedPayload = ChatMessage;

export interface ChatMessageDeletedPayload {
    room_id: string;
    message_id: string;
}

export interface ChatMessagePinnedPayload {
    room_id: string;
    message_id: string;
    pinned_at: string;
    pinned_by: string;
}

export interface ChatMessageUnpinnedPayload {
    room_id: string;
    message_id: string;
}

export interface ChatReactionPayload {
    room_id: string;
    message_id: string;
    emoji: string;
    user_id: string;
    display_name: string;
    count: number;
}

export type ChatReactionAddedPayload = ChatReactionPayload;

export type ChatReactionRemovedPayload = ChatReactionPayload;

export interface ChatMemberJoinedPayload {
    room_id: string;
    user: User;
    ghost?: boolean;
}

export interface ChatMemberLeftPayload {
    room_id: string;
    user_id: string;
    ghost?: boolean;
}

export interface ChatMemberUpdatedPayload {
    room_id: string;
    user_id: string;
    nickname: string;
    display_name: string;
    username: string;
    member_avatar_url: string;
    nickname_locked: boolean;
    timeout_until: string;
    timeout_set_by_staff: boolean;
}

export interface ChatKickedPayload {
    room_id: string;
    reason?: string;
}

export interface ChatRoomInvitedPayload {
    room_id: string;
}

export interface ChatRoomUpdatedPayload {
    room_id: string;
    name: string;
    description: string;
    tags: string[];
    is_public: boolean;
    is_rp: boolean;
}

export interface ChatRoomDeletedPayload {
    room_id: string;
}

export interface ChatPresenceChangedPayload {
    room_id: string;
    user_id: string;
    state: "" | "active" | "idle";
}

export interface ChatReadPayload {
    room_id: string;
    total: number;
}

export interface ChatReadReceiptPayload {
    room_id: string;
    user_id: string;
    read_at: string;
}

export interface ChatUnreadBumpedPayload {
    room_id: string;
    total: number;
}

export interface ChatAudioPayload {
    room_id: string;
    url: string;
    volume: number;
}

export interface TypingPayload {
    room_id: string;
    user_id: string;
}

export interface VoicePresencePayload {
    room_id: string;
    participants: string[];
    count: number;
}

export type WatchPartyStartedPayload = WatchPartyStartedEvent;

export type WatchPartyEndedPayload = WatchPartyEndedEvent;

export type WatchPartyParticipantJoinedPayload = WatchPartyParticipantEvent;

export type WatchPartyParticipantLeftPayload = WatchPartyParticipantLeftEvent;

export type WatchPartyControlChangedPayload = WatchPartyControlChangedEvent;

export type WatchPartyKickedPayload = WatchPartyKickedEvent;

export interface GameRoomBroadcastPayload {
    room_id: string;
    room: GameRoom;
}

export type GameRoomStartedPayload = GameRoomBroadcastPayload;

export interface GameRoomActionPayload extends GameRoomBroadcastPayload {
    ply?: number;
    by_slot: number;
    action?: unknown;
    reason?: string;
}

export interface GameRoomFinishedPayload extends GameRoomBroadcastPayload {
    reason?: string;
    draw_agreed_by?: string;
    resigned_by?: string;
    abandoned_by?: string;
}

export interface GameRoomDeclinedPayload extends GameRoomBroadcastPayload {
    by_user_id: string;
}

export interface GameRoomCancelledPayload extends GameRoomBroadcastPayload {
    by_user_id: string;
}

export interface GameRoomPresencePayload {
    room_id: string;
    user_id: string;
    connected: boolean;
    as_player: boolean;
    watcher_count: number;
    disconnected_at?: string;
    grace_seconds?: number;
}

export interface GameDrawOfferedPayload extends GameRoomBroadcastPayload {
    by_user_id: string;
}

export interface GameDrawDeclinedPayload extends GameRoomBroadcastPayload {
    by_user_id: string;
}

export interface GameYourTurnPayload {
    room_id: string;
    game_type: GameType;
}

export interface GameForfeitWarningPayload {
    room_id: string;
    game_type: GameType;
    disconnected_at: string;
    grace_seconds: number;
}

export interface GameForfeitClearedPayload {
    room_id: string;
}

export interface LiveGamesCountPayload {
    count: number;
}

export interface GameChatMessagePayload {
    room_id: string;
    message: SpectatorMessage;
}

export type PlayerChatMessagePayload = GameChatMessagePayload;

export type SpectatorChatMessagePayload = GameChatMessagePayload;

export type GamePongFramePayload = PongFrame;

export interface MysteryAttemptCreatedPayload {
    mystery_id: string;
    attempt_id: string;
    parent_id: string | null;
    author_id: string;
    author_username: string;
    author_display_name: string;
    author_avatar_url: string;
}

export interface MysterySolvedPayload {
    mystery_id: string;
    attempt_id?: string;
    winner_id?: string;
}

export interface MysteryWinnerAddedPayload {
    mystery_id: string;
    winner_id: string;
}

export interface MysteryClueAddedPayload {
    mystery_id: string;
    truth_type: string;
    player_id?: string;
}

export interface MysteryClueUpdatedPayload {
    mystery_id: string;
}

export interface MysteryPausedPayload {
    mystery_id: string;
    paused: boolean;
}

export interface MysteryGmAwayPayload {
    mystery_id: string;
    gm_away: boolean;
}

export interface PostLikePayload {
    post_id: string;
    delta: number;
}

export interface PostCommentPayload {
    post_id: string;
    comment_id: string;
}

export interface UserOCsChangedPayload {
    action: "created" | "updated" | "deleted";
    oc: OCSummary;
}

export type SecretProgressPayload = SecretProgressEvent;

export type SecretSolvedPayload = SecretSolvedEvent;

export interface SecretClosedPayload {
    secret_id: string;
    secret_title: string;
    solver: User;
}

export type StreamLivePayload = LiveStream;

export interface StreamOfflinePayload {
    streamId: string;
}

export interface StreamViewersPayload {
    streamId: string;
    viewerCount: number;
}

export interface StreamTitlePayload {
    streamId: string;
    title: string;
}

export type RealtimeEvent =
    | { type: "pong"; data: PongPayload }
    | { type: "notification"; data: NotificationPayload }
    | { type: "role_changed"; data: RoleChangedPayload }
    | { type: "ban_changed"; data: BanChangedPayload }
    | { type: "lock_changed"; data: LockChangedPayload }
    | { type: "profile_changed"; data: ProfileChangedPayload }
    | { type: "permissions_changed"; data: PermissionsChangedPayload }
    | { type: "top_detective_changed"; data: TopDetectiveChangedPayload }
    | { type: "top_gm_changed"; data: TopGmChangedPayload }
    | { type: "top_chess_changed"; data: TopChessChangedPayload }
    | { type: "top_checkers_changed"; data: TopCheckersChangedPayload }
    | { type: "top_othello_changed"; data: TopOthelloChangedPayload }
    | { type: "top_minesweeper_changed"; data: TopMinesweeperChangedPayload }
    | { type: "vanity_roles_changed"; data: VanityRolesChangedPayload }
    | { type: "rules_page_changed"; data: RulesPageChangedPayload }
    | { type: "new_announcement"; data: NewAnnouncementPayload }
    | { type: "chatbots_changed"; data: ChatbotsChangedPayload }
    | { type: "sidebar_activity"; data: SidebarActivityPayload }
    | { type: "chat_message"; data: ChatMessagePayload }
    | { type: "chat_message_edited"; data: ChatMessageEditedPayload }
    | { type: "chat_message_deleted"; data: ChatMessageDeletedPayload }
    | { type: "chat_message_pinned"; data: ChatMessagePinnedPayload }
    | { type: "chat_message_unpinned"; data: ChatMessageUnpinnedPayload }
    | { type: "chat_reaction_added"; data: ChatReactionAddedPayload }
    | { type: "chat_reaction_removed"; data: ChatReactionRemovedPayload }
    | { type: "chat_member_joined"; data: ChatMemberJoinedPayload }
    | { type: "chat_member_left"; data: ChatMemberLeftPayload }
    | { type: "chat_member_updated"; data: ChatMemberUpdatedPayload }
    | { type: "chat_kicked"; data: ChatKickedPayload }
    | { type: "chat_room_invited"; data: ChatRoomInvitedPayload }
    | { type: "chat_room_updated"; data: ChatRoomUpdatedPayload }
    | { type: "chat_room_deleted"; data: ChatRoomDeletedPayload }
    | { type: "chat_presence_changed"; data: ChatPresenceChangedPayload }
    | { type: "chat_read"; data: ChatReadPayload }
    | { type: "chat_read_receipt"; data: ChatReadReceiptPayload }
    | { type: "chat_unread_bumped"; data: ChatUnreadBumpedPayload }
    | { type: "chat_audio"; data: ChatAudioPayload }
    | { type: "typing"; data: TypingPayload }
    | { type: "voice_presence"; data: VoicePresencePayload }
    | { type: "watch_party_started"; data: WatchPartyStartedPayload }
    | { type: "watch_party_ended"; data: WatchPartyEndedPayload }
    | { type: "watch_party_participant_joined"; data: WatchPartyParticipantJoinedPayload }
    | { type: "watch_party_participant_left"; data: WatchPartyParticipantLeftPayload }
    | { type: "watch_party_control_changed"; data: WatchPartyControlChangedPayload }
    | { type: "watch_party_kicked"; data: WatchPartyKickedPayload }
    | { type: "game_room_started"; data: GameRoomStartedPayload }
    | { type: "game_room_action"; data: GameRoomActionPayload }
    | { type: "game_room_finished"; data: GameRoomFinishedPayload }
    | { type: "game_room_declined"; data: GameRoomDeclinedPayload }
    | { type: "game_room_cancelled"; data: GameRoomCancelledPayload }
    | { type: "game_room_presence"; data: GameRoomPresencePayload }
    | { type: "game_draw_offered"; data: GameDrawOfferedPayload }
    | { type: "game_draw_declined"; data: GameDrawDeclinedPayload }
    | { type: "game_your_turn"; data: GameYourTurnPayload }
    | { type: "game_forfeit_warning"; data: GameForfeitWarningPayload }
    | { type: "game_forfeit_cleared"; data: GameForfeitClearedPayload }
    | { type: "live_games_count"; data: LiveGamesCountPayload }
    | { type: "player_chat_message"; data: PlayerChatMessagePayload }
    | { type: "spectator_chat_message"; data: SpectatorChatMessagePayload }
    | { type: "game_pong_frame"; data: GamePongFramePayload }
    | { type: "mystery_attempt_created"; data: MysteryAttemptCreatedPayload }
    | { type: "mystery_solved"; data: MysterySolvedPayload }
    | { type: "mystery_winner_added"; data: MysteryWinnerAddedPayload }
    | { type: "mystery_clue_added"; data: MysteryClueAddedPayload }
    | { type: "mystery_clue_updated"; data: MysteryClueUpdatedPayload }
    | { type: "mystery_paused"; data: MysteryPausedPayload }
    | { type: "mystery_gm_away"; data: MysteryGmAwayPayload }
    | { type: "post_like"; data: PostLikePayload }
    | { type: "post_comment"; data: PostCommentPayload }
    | { type: "user_ocs_changed"; data: UserOCsChangedPayload }
    | { type: "secret_progress"; data: SecretProgressPayload }
    | { type: "secret_solved"; data: SecretSolvedPayload }
    | { type: "secret_closed"; data: SecretClosedPayload }
    | { type: "stream_live"; data: StreamLivePayload }
    | { type: "stream_offline"; data: StreamOfflinePayload }
    | { type: "stream_viewers"; data: StreamViewersPayload }
    | { type: "stream_title"; data: StreamTitlePayload };

export type RealtimeEventName = RealtimeEvent["type"];

export type RealtimeEventOf<K extends RealtimeEventName> = Extract<RealtimeEvent, { type: K }>;

export type RealtimeEventPayload<K extends RealtimeEventName> = RealtimeEventOf<K>["data"];
