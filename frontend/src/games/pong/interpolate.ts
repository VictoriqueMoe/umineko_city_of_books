import { PONG_BUFFER, PONG_RENDER_DELAY_MS, PONG_STALE_MS, type PongFrame, type PongPhase } from "./types";

export interface BufferedFrame {
    frame: PongFrame;
    receivedAt: number;
    offset: number;
}

export interface PongEventAt {
    t: number;
    events: number;
}

export interface PongSample {
    t: number;
    phase: PongPhase;
    ballX: number;
    ballY: number;
    paddleY: [number, number];
    scores: [number, number];
    ack: [number, number];
    connected: [boolean, boolean];
    serveInMs: number;
    events: PongEventAt[];
    stale: boolean;
}

const OFFSET_SMOOTHING = 0.08;

function clamp01(value: number): number {
    if (value < 0) {
        return 0;
    }
    if (value > 1) {
        return 1;
    }
    return value;
}

function lerp(from: number, to: number, alpha: number): number {
    return from + (to - from) * alpha;
}

export function pushFrame(buf: BufferedFrame[], frame: PongFrame, now: number): BufferedFrame[] {
    const newest = buf.length > 0 ? buf[buf.length - 1] : null;
    const sampled = frame.t - now;

    if (!newest) {
        return [{ frame, receivedAt: now, offset: sampled }];
    }

    if (frame.t === newest.frame.t) {
        return buf;
    }
    if (frame.t < newest.frame.t || now - newest.receivedAt > PONG_STALE_MS) {
        return [{ frame, receivedAt: now, offset: sampled }];
    }

    const offset = newest.offset + (sampled - newest.offset) * OFFSET_SMOOTHING;

    const next = [...buf, { frame, receivedAt: now, offset }];
    if (next.length > PONG_BUFFER) {
        next.splice(0, next.length - PONG_BUFFER);
    }

    return next;
}

export function serverNow(buf: BufferedFrame[], now: number): number {
    if (buf.length === 0) {
        return 0;
    }

    return buf[buf.length - 1].offset + now;
}

export function sampleAt(buf: BufferedFrame[], renderT: number): PongSample | null {
    if (buf.length === 0) {
        return null;
    }

    const newest = buf[buf.length - 1].frame;
    const stale = renderT + PONG_RENDER_DELAY_MS - newest.t > PONG_STALE_MS;

    let base = -1;
    const crossed: PongEventAt[] = [];
    for (const [index, entry] of buf.entries()) {
        if (entry.frame.t > renderT) {
            continue;
        }
        base = index;
        if (entry.frame.events !== 0) {
            crossed.push({ t: entry.frame.t, events: entry.frame.events });
        }
    }

    const anchor = base < 0 ? 0 : base;
    const a = buf[anchor].frame;
    const b = base >= 0 && base + 1 < buf.length ? buf[base + 1].frame : a;
    const span = b.t - a.t;
    const alpha = span > 0 ? clamp01((renderT - a.t) / span) : 0;

    return {
        t: lerp(a.t, b.t, alpha),
        phase: a.phase,
        ballX: lerp(a.ball_x, b.ball_x, alpha),
        ballY: lerp(a.ball_y, b.ball_y, alpha),
        paddleY: [lerp(a.paddle_y[0], b.paddle_y[0], alpha), lerp(a.paddle_y[1], b.paddle_y[1], alpha)],
        scores: [a.scores[0], a.scores[1]],
        ack: [a.ack[0], a.ack[1]],
        connected: [a.connected[0], a.connected[1]],
        serveInMs: a.serve_in_ms,
        events: crossed,
        stale,
    };
}
