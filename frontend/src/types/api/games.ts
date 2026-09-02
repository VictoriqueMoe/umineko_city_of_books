import type { User } from "./common";

export type GameType = "chess" | "checkers" | "othello" | "minesweeper" | "snakes_and_ladders" | "pong";
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

export interface CheckersLastMove {
    from: string;
    path: string[];
    captured: string[];
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
    last_move?: CheckersLastMove;
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

export interface OthelloLastMove {
    square: string;
    slot: number;
    flipped: string[];
}

export interface OthelloState {
    board: string;
    turn: number;
    black_moves: number;
    white_moves: number;
    black_passes: number;
    white_passes: number;
    black_flips: number;
    white_flips: number;
    last_move?: OthelloLastMove;
}

export interface OthelloStats {
    total_moves: number;
    black_moves: number;
    white_moves: number;
    black_passes: number;
    white_passes: number;
    black_discs: number;
    white_discs: number;
    black_flips: number;
    white_flips: number;
    black_corners: number;
    white_corners: number;
    result_reason: string;
    duration_seconds: number;
    final_board: string;
}

export type MinesweeperPhase = "char_select" | "playing" | "finished";

export interface MinesweeperState {
    phase: MinesweeperPhase;
    width: number;
    height: number;
    mine_count: number;
    characters: [string, string];
    started_at?: string;
    finished_at?: string;
    revealed: [boolean[], boolean[]];
    flagged: [boolean[], boolean[]];
    revealed_count: [number, number];
    values: [number[], number[]];
    mines?: boolean[];
    mines_placed: boolean;
    pending_clicks: [[number, number] | null, [number, number] | null];
    winner_slot?: number;
    reason?: string;
    hit_mine_x?: number;
    hit_mine_y?: number;
}

export interface MinesweeperStats {
    duration_seconds: number;
    revealed_p0: number;
    revealed_p1: number;
    flags_p0: number;
    flags_p1: number;
    reason: string;
}

export type PongPhase = "countdown" | "serve" | "rally" | "finished";

export interface PongState {
    phase: PongPhase;
    width: number;
    height: number;
    ball_radius: number;
    paddle_width: number;
    paddle_height: number;
    paddle_inset: number;
    points_to_win: number;
    ball_x: number;
    ball_y: number;
    ball_vx: number;
    ball_vy: number;
    ball_speed: number;
    paddle_y: [number, number];
    scores: [number, number];
    hits: [number, number];
    longest_rally: [number, number];
    rally_hits: number;
    top_speed: number;
    serve_to_slot: number;
    serve_vx: number;
    serve_vy: number;
    phase_remain_ms: number;
    ticks: number;
    started_at?: string;
    finished_at?: string;
    winner_slot?: number;
    reason?: string;
}

export interface PongStats {
    points_p0: number;
    points_p1: number;
    hits_p0: number;
    hits_p1: number;
    longest_rally_p0: number;
    longest_rally_p1: number;
    top_speed: number;
    result_reason: string;
    duration_seconds: number;
}

export interface SnakesLaddersLast {
    slot: number;
    roll: number;
    from: number;
    stepped: number;
    to: number;
}

export interface SnakesLaddersState {
    positions: [number, number];
    turn: number;
    rolls: number;
    ladders_climbed: [number, number];
    snakes_hit: [number, number];
    last?: SnakesLaddersLast;
}

export interface SnakesLaddersStats {
    total_rolls: number;
    rolls_p0: number;
    rolls_p1: number;
    ladders_p0: number;
    ladders_p1: number;
    snakes_p0: number;
    snakes_p1: number;
    final_p0: number;
    final_p1: number;
    result_reason: string;
    duration_seconds: number;
}

export interface GameRoom<TState = unknown, TStats = unknown> {
    id: string;
    game_type: GameType;
    status: GameStatus;
    state: TState;
    turn_user_id?: string;
    winner_user_id?: string;
    result?: string;
    created_by: string;
    created_at: string;
    updated_at: string;
    finished_at?: string;
    players: GameRoomPlayer[];
    watcher_count: number;
    stats?: TStats;
    draw_offer_from_user_id?: string;
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
