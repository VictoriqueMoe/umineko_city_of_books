export interface Quote {
    text: string;
    textHtml: string;
    textJp?: string;
    textJpHtml?: string;
    characterId: string;
    character: string;
    audioId: string;
    episode: number;
    contentType: string;
    hasRedTruth: boolean;
    hasBlueTruth: boolean;
    hasGoldTruth: boolean;
    hasPurpleTruth: boolean;
    index: number;
    arc?: string;
    audioCharMap?: Record<string, string>;
    audioTextMap?: Record<string, string>;
}

export interface QuoteBrowseResponse {
    character: string;
    characterId: string;
    quotes: Quote[];
    total: number;
    limit: number;
    offset: number;
}

export interface QuoteSearchResult {
    quote: Quote;
    score: number;
}

export interface QuoteSearchResponse {
    results: QuoteSearchResult[];
    total: number;
    limit: number;
    offset: number;
}

import type { SiteRole } from "../utils/permissions";

export interface User {
    id: string;
    username: string;
    display_name: string;
    avatar_url?: string;
    role?: SiteRole;
    banned?: boolean;
    ban_reason?: string;
}

export interface EvidenceItem {
    id: number;
    audio_id?: string;
    quote_index?: number;
    note: string;
    lang: string;
    sort_order: number;
}

export interface Theory {
    id: string;
    title: string;
    body: string;
    episode: number;
    series: string;
    author: User;
    vote_score: number;
    with_love_count: number;
    without_love_count: number;
    user_vote?: number;
    credibility_score: number;
    created_at: string;
}

export interface TheoryDetail extends Theory {
    evidence: EvidenceItem[];
    responses: Response[];
}

export interface TheoryListResponse {
    theories: Theory[];
    total: number;
    limit: number;
    offset: number;
}

export interface Response {
    id: string;
    parent_id?: string;
    author: User;
    side: "with_love" | "without_love";
    body: string;
    evidence: EvidenceItem[];
    replies?: Response[];
    vote_score: number;
    user_vote?: number;
    created_at: string;
}

export interface EvidenceInput {
    audio_id?: string;
    quote_index?: number;
    note: string;
    lang?: string;
}

export interface CreateTheoryPayload {
    title: string;
    body: string;
    episode: number;
    series: string;
    evidence: EvidenceInput[];
}

export interface CreateResponsePayload {
    parent_id?: string;
    side: "with_love" | "without_love";
    body: string;
    evidence: EvidenceInput[];
}

export interface VotePayload {
    value: number;
}

export interface UserProfile {
    id: string;
    username: string;
    display_name: string;
    bio: string;
    avatar_url: string;
    banner_url: string;
    banner_position: number;
    favourite_character: string;
    gender: string;
    pronoun_subject: string;
    pronoun_possessive: string;
    role?: SiteRole;
    online: boolean;
    social_twitter: string;
    social_discord: string;
    social_waifulist: string;
    social_tumblr: string;
    social_github: string;
    website: string;
    dms_enabled: boolean;
    episode_progress: number;
    higurashi_arc_progress: number;
    ciconia_chapter_progress: number;
    secrets: string[];
    dob?: string;
    dob_public?: boolean;
    email?: string;
    email_public?: boolean;
    email_notifications?: boolean;
    play_message_sound?: boolean;
    play_notification_sound?: boolean;
    home_page?: string;
    game_board_sort?: string;
    theme?: string;
    font?: string;
    wide_layout?: boolean;
    created_at: string;
    stats: UserStats;
    banned?: boolean;
    ban_reason?: string;
}

export interface UserStats {
    theory_count: number;
    response_count: number;
    votes_received: number;
    ship_count: number;
    mystery_count: number;
    fanfic_count: number;
}

export interface UpdateProfilePayload {
    display_name: string;
    bio: string;
    avatar_url: string;
    banner_url: string;
    banner_position: number;
    favourite_character: string;
    gender: string;
    pronoun_subject: string;
    pronoun_possessive: string;
    social_twitter: string;
    social_discord: string;
    social_waifulist: string;
    social_tumblr: string;
    social_github: string;
    website: string;
    dms_enabled: boolean;
    episode_progress: number;
    higurashi_arc_progress: number;
    ciconia_chapter_progress: number;
    dob: string;
    dob_public: boolean;
    email: string;
    email_public: boolean;
    email_notifications: boolean;
    play_message_sound: boolean;
    play_notification_sound: boolean;
    home_page: string;
    game_board_sort: string;
}

export interface ChangePasswordPayload {
    old_password: string;
    new_password: string;
}

export interface DeleteAccountPayload {
    password: string;
}

export interface ActivityItem {
    type: string;
    theory_id: string;
    theory_title: string;
    side?: string;
    body: string;
    created_at: string;
}

export interface ActivityListResponse {
    items: ActivityItem[];
    total: number;
    limit: number;
    offset: number;
}

export interface PostMedia {
    id: number;
    media_url: string;
    media_type: "image" | "video";
    thumbnail_url?: string;
    sort_order: number;
}

export interface PostEmbed {
    url: string;
    type: "link" | "youtube";
    title?: string;
    description?: string;
    image?: string;
    site_name?: string;
    video_id?: string;
}

export interface PollOption {
    id: number;
    label: string;
    vote_count: number;
    percent: number;
}

export interface Poll {
    id: string;
    options: PollOption[];
    total_votes: number;
    user_voted_option: number | null;
    expired: boolean;
    expires_at: string;
    duration_seconds: number;
}

export interface SharedContentPreview {
    id: string;
    content_type: string;
    title?: string;
    body?: string;
    image_url?: string;
    media?: PostMedia[];
    author?: User;
    deleted: boolean;
    url: string;
    difficulty?: string;
    solved?: boolean;
    series?: string;
    vote_score?: number;
    credibility_score?: number;
    rating?: string;
    word_count?: number;
    chapter_count?: number;
    corner?: string;
    like_count?: number;
    comment_count?: number;
}

export interface Post {
    id: string;
    author: User;
    body: string;
    media: PostMedia[];
    embeds?: PostEmbed[];
    poll?: Poll;
    shared_content?: SharedContentPreview;
    share_count: number;
    like_count: number;
    comment_count: number;
    view_count: number;
    user_liked: boolean;
    resolved_status?: string;
    created_at: string;
    updated_at?: string;
}

export interface PostDetail extends Post {
    comments: PostComment[];
    liked_by: User[];
    viewer_blocked: boolean;
}

export interface PostComment {
    id: string;
    parent_id?: string;
    author: User;
    body: string;
    media: PostMedia[];
    embeds?: PostEmbed[];
    like_count: number;
    user_liked: boolean;
    replies?: PostComment[];
    created_at: string;
    updated_at?: string;
}

export interface PostListResponse {
    posts: Post[];
    total: number;
    limit: number;
    offset: number;
}

export interface FollowStats {
    follower_count: number;
    following_count: number;
    is_following: boolean;
    follows_you: boolean;
}

export type JournalWork = "general" | "umineko" | "higurashi" | "ciconia" | "higanbana" | "roseguns";

export interface Journal {
    id: string;
    title: string;
    body: string;
    work: JournalWork;
    author: User;
    follower_count: number;
    is_following: boolean;
    is_archived: boolean;
    comment_count: number;
    created_at: string;
    updated_at?: string;
    last_author_activity_at: string;
    archived_at?: string;
}

export interface JournalComment extends PostComment {
    is_author: boolean;
}

export interface JournalDetail extends Journal {
    comments: JournalComment[];
}

export interface JournalListResponse {
    journals: Journal[];
    total: number;
    limit: number;
    offset: number;
}

export interface CreateJournalPayload {
    title: string;
    body: string;
    work: JournalWork;
}

export type FeedTab = "following" | "everyone";

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
    | "announcement_commented"
    | "announcement_comment_reply"
    | "announcement_comment_liked"
    | "suggestion_posted"
    | "suggestion_resolved"
    | "content_shared"
    | "game_invite"
    | "game_your_turn"
    | "game_finished";

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

export interface NotificationListResponse {
    notifications: Notification[];
    total: number;
    limit: number;
    offset: number;
}

export interface WSMessage {
    type: string;
    data: unknown;
}

export interface AdminUserItem {
    id: string;
    username: string;
    display_name: string;
    avatar_url: string;
    role?: SiteRole;
    banned: boolean;
    created_at: string;
}

export interface AdminUserListResponse {
    users: AdminUserItem[];
    total: number;
    limit: number;
    offset: number;
}

export interface AdminUserDetail extends AdminUserItem {
    ip?: string;
    ban_reason?: string;
    banned_at?: string;
    theory_count: number;
    response_count: number;
    mystery_score_adjustment: number;
    detective_score: number;
    gm_score_adjustment: number;
    gm_score: number;
}

export interface AdminStats {
    total_users: number;
    total_theories: number;
    total_responses: number;
    total_votes: number;
    total_posts: number;
    total_comments: number;
    new_users_24h: number;
    new_users_7d: number;
    new_users_30d: number;
    new_theories_24h: number;
    new_theories_7d: number;
    new_theories_30d: number;
    new_responses_24h: number;
    new_responses_7d: number;
    new_responses_30d: number;
    new_posts_24h: number;
    new_posts_7d: number;
    new_posts_30d: number;
    posts_by_corner: Record<string, number>;
    most_active_users: {
        id: string;
        username: string;
        display_name: string;
        avatar_url: string;
        action_count: number;
    }[];
}

export interface AuditLogEntry {
    id: number;
    actor_id: string;
    actor_name: string;
    action: string;
    target_type: string;
    target_id: string;
    details: string;
    created_at: string;
}

export interface AuditLogListResponse {
    entries: AuditLogEntry[];
    total: number;
    limit: number;
    offset: number;
}

export interface SiteSettings {
    [key: string]: string;
}

export interface Art {
    id: string;
    author: User;
    corner: string;
    art_type: string;
    title: string;
    description: string;
    image_url: string;
    thumbnail_url: string;
    gallery_id?: string;
    tags: string[];
    like_count: number;
    comment_count: number;
    view_count: number;
    user_liked: boolean;
    is_spoiler: boolean;
    created_at: string;
    updated_at?: string;
}

export interface ArtDetail extends Art {
    comments: ArtComment[];
    liked_by: User[];
    viewer_blocked: boolean;
}

export interface ArtComment {
    id: string;
    parent_id?: string;
    author: User;
    body: string;
    media: PostMedia[];
    embeds?: PostEmbed[];
    like_count: number;
    user_liked: boolean;
    replies?: ArtComment[];
    created_at: string;
    updated_at?: string;
}

export interface ArtListResponse {
    art: Art[];
    total: number;
    limit: number;
    offset: number;
}

export interface TagCount {
    tag: string;
    count: number;
}

export interface Gallery {
    id: string;
    author: User;
    name: string;
    description: string;
    cover_image_url: string;
    cover_thumbnail_url: string;
    preview_images?: { thumbnail: string; full: string }[];
    art_count: number;
    created_at: string;
    updated_at?: string;
}

export interface GalleryDetailResponse {
    gallery: Gallery;
    art: Art[];
    total: number;
    limit: number;
    offset: number;
}

export interface ChatRoom {
    id: string;
    name: string;
    description: string;
    type: "dm" | "group";
    is_public: boolean;
    is_rp: boolean;
    is_system: boolean;
    system_kind?: string;
    tags: string[];
    viewer_role?: string;
    viewer_muted: boolean;
    viewer_ghost: boolean;
    is_member: boolean;
    member_count: number;
    members: User[];
    created_at: string;
    last_message_at?: string;
    unread?: boolean;
}

export interface ChatRoomMember {
    user: User;
    role: string;
    joined_at: string;
    nickname: string;
    member_avatar_url: string;
    nickname_locked: boolean;
    timeout_until?: string;
    timeout_set_by_staff?: boolean;
    presence?: "active" | "idle" | "";
    ghost?: boolean;
}

export interface ChatRoomBan {
    user: User;
    banned_by?: User;
    reason: string;
    created_at: string;
}

export type BannedWordMatchMode = "substring" | "whole_word" | "regex";
export type BannedWordAction = "delete" | "kick";

export interface BannedWordRule {
    id: string;
    scope: "global" | "room";
    room_id?: string;
    pattern: string;
    match_mode: BannedWordMatchMode;
    case_sensitive: boolean;
    action: BannedWordAction;
    created_by_id?: string;
    created_by_name?: string;
    created_at: string;
}

export interface CreateBannedWordRequest {
    pattern: string;
    match_mode: BannedWordMatchMode;
    case_sensitive: boolean;
    action: BannedWordAction;
}

export interface ChatMessageReplyPreview {
    id: string;
    sender_id: string;
    sender_name: string;
    body_preview: string;
}

export interface ReactionGroup {
    emoji: string;
    count: number;
    viewer_reacted: boolean;
    display_names: string[];
}

export interface ChatMessage {
    id: string;
    room_id: string;
    sender: User;
    body: string;
    is_system: boolean;
    created_at: string;
    media?: PostMedia[];
    reply_to?: ChatMessageReplyPreview;
    pinned: boolean;
    pinned_at?: string;
    pinned_by?: string;
    edited_at?: string;
    reactions: ReactionGroup[];
    sender_nickname?: string;
    sender_member_avatar_url?: string;
}

export interface ChatMessageListResponse {
    messages: ChatMessage[];
    total: number;
}

export interface Mystery {
    id: string;
    title: string;
    body: string;
    difficulty: string;
    author: User;
    solved: boolean;
    paused: boolean;
    gm_away: boolean;
    free_for_all: boolean;
    winner?: User;
    solved_at?: string;
    paused_at?: string;
    paused_duration_seconds: number;
    attempt_count: number;
    clue_count: number;
    created_at: string;
}

export interface MysteryClue {
    id: number;
    body: string;
    truth_type: string;
    sort_order: number;
    player_id?: string;
}

export interface MysteryAttempt {
    id: string;
    parent_id?: string;
    author: User;
    body: string;
    is_winner: boolean;
    vote_score: number;
    user_vote?: number;
    replies?: MysteryAttempt[];
    created_at: string;
}

export interface MysteryComment {
    id: string;
    parent_id?: string;
    author: User;
    body: string;
    media: PostMedia[];
    like_count: number;
    user_liked: boolean;
    replies?: MysteryComment[];
    created_at: string;
    updated_at?: string;
}

export interface MysteryAttachment {
    id: number;
    file_url: string;
    file_name: string;
    file_size: number;
}

export interface MysteryDetail {
    id: string;
    title: string;
    body: string;
    difficulty: string;
    author: User;
    solved: boolean;
    paused: boolean;
    gm_away: boolean;
    free_for_all: boolean;
    winner?: User;
    solved_at?: string;
    paused_at?: string;
    paused_duration_seconds: number;
    clues: MysteryClue[];
    attempts: MysteryAttempt[];
    comments: MysteryComment[];
    attachments?: MysteryAttachment[];
    player_count: number;
    created_at: string;
}

export interface MysteryListResponse {
    mysteries: Mystery[];
    total: number;
    limit: number;
    offset: number;
}

export interface SecretComment {
    id: string;
    parent_id?: string;
    author: User;
    body: string;
    media: PostMedia[];
    like_count: number;
    user_liked: boolean;
    replies?: SecretComment[];
    created_at: string;
    updated_at?: string;
}

export interface SecretLeaderboardEntry {
    user: User;
    pieces_collected: number;
    solved: boolean;
}

export interface SecretSummary {
    id: string;
    title: string;
    description: string;
    total_pieces: number;
    solved: boolean;
    solver?: User;
    solved_at?: string;
    viewer_progress: number;
    comment_count: number;
}

export interface SecretDetailResponse extends SecretSummary {
    riddle: string;
    leaderboard: SecretLeaderboardEntry[];
    comments: SecretComment[];
}

export interface SecretSolverEntry {
    user: User;
    solved_count: number;
    last_solved_at: string;
}

export interface SecretListResponse {
    secrets: SecretSummary[];
    solvers_leaderboard: SecretSolverEntry[];
}

export interface SecretProgressEvent {
    secret_id: string;
    user: User;
    pieces_collected: number;
    total_pieces: number;
}

export interface SecretSolvedEvent {
    secret_id: string;
    solver: User;
    solved_at: string;
}

export interface MysteryLeaderboardEntry {
    user: User;
    score: number;
    easy_solved: number;
    medium_solved: number;
    hard_solved: number;
    nightmare_solved: number;
    score_adjustment: number;
}

export interface MysteryLeaderboardResponse {
    entries: MysteryLeaderboardEntry[];
}

export interface GMLeaderboardEntry {
    user: User;
    score: number;
    mystery_count: number;
    player_count: number;
}

export interface GMLeaderboardResponse {
    entries: GMLeaderboardEntry[];
}

export interface FanficCharacter {
    series: string;
    character_id?: string;
    character_name: string;
    sort_order: number;
}

export interface Fanfic {
    id: string;
    author: User;
    title: string;
    summary: string;
    series: string;
    rating: string;
    language: string;
    status: string;
    is_oneshot: boolean;
    contains_lemons: boolean;
    cover_image_url?: string;
    cover_thumbnail_url?: string;
    genres: string[];
    tags: string[];
    characters: FanficCharacter[];
    is_pairing: boolean;
    word_count: number;
    chapter_count: number;
    favourite_count: number;
    view_count: number;
    comment_count: number;
    user_favourited: boolean;
    published_at: string;
    created_at: string;
    updated_at?: string;
}

export interface FanficChapterSummary {
    id: string;
    chapter_number: number;
    title: string;
    word_count: number;
}

export interface FanficChapter {
    id: string;
    chapter_number: number;
    title: string;
    body: string;
    word_count: number;
    has_prev: boolean;
    has_next: boolean;
    created_at: string;
    updated_at?: string;
}

export interface FanficDetail extends Fanfic {
    chapters: FanficChapterSummary[];
    comments: PostComment[];
    reading_progress: number;
    viewer_blocked: boolean;
}

export interface FanficListResponse {
    fanfics: Fanfic[];
    total: number;
    limit: number;
    offset: number;
}

export interface Announcement {
    id: string;
    title: string;
    body: string;
    author: User;
    pinned: boolean;
    created_at: string;
    updated_at: string;
    comments?: AnnouncementComment[];
}

export interface AnnouncementComment {
    id: string;
    parent_id?: string;
    author: User;
    body: string;
    media: PostMedia[];
    embeds?: PostEmbed[];
    like_count: number;
    user_liked: boolean;
    replies?: AnnouncementComment[];
    created_at: string;
    updated_at?: string;
}

export interface ShipCharacter {
    series: string;
    character_id?: string;
    character_name: string;
    sort_order: number;
}

export interface Ship {
    id: string;
    author: User;
    title: string;
    description: string;
    image_url?: string;
    thumbnail_url?: string;
    characters: ShipCharacter[];
    vote_score: number;
    user_vote?: number;
    comment_count: number;
    is_crackship: boolean;
    created_at: string;
    updated_at?: string;
}

export interface ShipComment {
    id: string;
    parent_id?: string;
    author: User;
    body: string;
    media: PostMedia[];
    embeds?: PostEmbed[];
    like_count: number;
    user_liked: boolean;
    replies?: ShipComment[];
    created_at: string;
    updated_at?: string;
}

export interface ShipDetail extends Ship {
    comments: ShipComment[];
    viewer_blocked: boolean;
}

export interface ShipListResponse {
    ships: Ship[];
    total: number;
    limit: number;
    offset: number;
}

export interface CharacterListEntry {
    id: string;
    name: string;
    group?: "main" | "additional";
}

export interface CharacterListResponse {
    series: string;
    characters: CharacterListEntry[];
}

export interface AnnouncementListResponse {
    announcements: Announcement[];
    total: number;
    limit: number;
    offset: number;
}

export type GameType = "chess" | "checkers";
export type GameStatus = "pending" | "active" | "finished" | "declined" | "abandoned";

export interface GameRoomPlayer {
    user_id: string;
    username: string;
    display_name: string;
    avatar_url: string;
    role: string;
    slot: number;
    joined: boolean;
    connected: boolean;
    disconnected_at?: string;
    user: User;
}

export interface ChessState {
    fen: string;
    pgn: string;
}

export interface ChessStats {
    total_ply: number;
    white_moves: number;
    black_moves: number;
    white_captures: number;
    black_captures: number;
    white_checks: number;
    black_checks: number;
    result_reason: string;
    duration_seconds: number;
    final_fen: string;
}

export interface CheckersState {
    board: string;
    turn: number;
    total_moves: number;
    red_captures: number;
    black_captures: number;
    red_crownings: number;
    black_crownings: number;
    moves_since_capture: number;
}

export interface CheckersStats {
    total_moves: number;
    red_moves: number;
    black_moves: number;
    red_captures: number;
    black_captures: number;
    red_crownings: number;
    black_crownings: number;
    result_reason: string;
    duration_seconds: number;
    final_board: string;
    red_pieces_left: number;
    black_pieces_left: number;
}

export interface GameRoom {
    id: string;
    game_type: GameType;
    status: GameStatus;
    state: ChessState | CheckersState | Record<string, unknown>;
    turn_user_id?: string;
    winner_user_id?: string;
    result?: string;
    created_by: string;
    created_at: string;
    updated_at: string;
    finished_at?: string;
    players: GameRoomPlayer[];
    watcher_count: number;
    stats?: ChessStats | CheckersStats | Record<string, unknown>;
}

export interface SpectatorMessage {
    id: string;
    user_id: string;
    user: User;
    body: string;
    created_at: string;
}

export interface SpectatorChatResponse {
    messages: SpectatorMessage[];
}

export interface GameRoomListResponse {
    rooms: GameRoom[];
    total: number;
}

export interface GameScoreboardRow {
    user: User;
    wins: number;
    losses: number;
    draws: number;
    games_played: number;
    win_rate: number;
}

export interface GameScoreboardResponse {
    game_type: GameType;
    rows: GameScoreboardRow[];
}
