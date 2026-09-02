import type { CommentBase, User } from "./common";

export interface SecretComment extends CommentBase {
    replies?: SecretComment[];
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
