import type { PongPhase } from "../../types/api";

export type { PongPhase };

export interface PongFrame {
    room_id: string;
    t: number;
    phase: PongPhase;
    ball_x: number;
    ball_y: number;
    ball_vx: number;
    ball_vy: number;
    paddle_y: [number, number];
    scores: [number, number];
    ack: [number, number];
    connected: [boolean, boolean];
    serve_in_ms: number;
    events: number;
}

export const PONG_FRAME_TYPE = "game_pong_frame";
export const PONG_INPUT_TYPE = "game_room_input";

export const PONG_INPUT_INTERVAL_MS = 50;
export const PONG_INPUT_KEEPALIVE_MS = 500;
export const PONG_INPUT_EPSILON = 4;
export const PONG_RENDER_DELAY_MS = 100;
export const PONG_STALE_MS = 600;
export const PONG_BUFFER = 12;
export const PONG_PADDLE_MAX_SPEED = 1100;

export const PONG_EVENT_HIT_P0 = 1;
export const PONG_EVENT_HIT_P1 = 2;
export const PONG_EVENT_WALL = 4;
export const PONG_EVENT_SCORE = 8;
export const PONG_EVENT_SERVE = 16;
