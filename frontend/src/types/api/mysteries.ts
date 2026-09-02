import type { CommentBase, PaginationFields, PostMedia, User } from "./common";

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
    keep_open_after_solve: boolean;
    solver_count: number;
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

export interface MysteryComment extends CommentBase {
    replies?: MysteryComment[];
}

export interface MysteryAttachment {
    id: number;
    file_url: string;
    file_name: string;
    file_size: number;
}

export interface KnoxContract {
    culprit_named_early: boolean;
    no_supernatural: boolean;
    passages_declared: boolean;
    no_unknown_poison: boolean;
    no_outsider: boolean;
    no_lucky_accident: boolean;
    detective_not_culprit: boolean;
    clues_shown: boolean;
    narrator_hides_nothing: boolean;
    no_unannounced_twins: boolean;
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
    keep_open_after_solve: boolean;
    knox_contract: KnoxContract;
    knox_contract_published: boolean;
    knox_contract_locked: boolean;
    solver_count: number;
    viewer_has_solved: boolean;
    winner?: User;
    solved_at?: string;
    paused_at?: string;
    paused_duration_seconds: number;
    clues: MysteryClue[];
    attempts: MysteryAttempt[];
    comments: MysteryComment[];
    attachments?: MysteryAttachment[];
    media?: PostMedia[];
    player_count: number;
    created_at: string;
}

export interface MysteryListResponse extends PaginationFields {
    mysteries: Mystery[];
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
