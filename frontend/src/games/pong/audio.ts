import { PONG_EVENT_HIT_P0, PONG_EVENT_HIT_P1, PONG_EVENT_SCORE, PONG_EVENT_SERVE, PONG_EVENT_WALL } from "./types";

const MUTE_KEY = "pong_muted";

let context: AudioContext | null = null;
let muted = readMuted();

function readMuted(): boolean {
    try {
        return window.localStorage.getItem(MUTE_KEY) === "1";
    } catch {
        return false;
    }
}

export function isPongMuted(): boolean {
    return muted;
}

export function setPongMuted(next: boolean): void {
    muted = next;
    try {
        window.localStorage.setItem(MUTE_KEY, next ? "1" : "0");
    } catch {}
}

function audioContext(): AudioContext | null {
    if (typeof window === "undefined" || typeof window.AudioContext === "undefined") {
        return null;
    }
    if (!context) {
        context = new window.AudioContext();
    }
    if (context.state === "suspended") {
        context.resume().catch(() => {});
    }

    return context;
}

function blip(type: OscillatorType, from: number, to: number, peak: number, duration: number, delay: number): void {
    const ctx = audioContext();
    if (!ctx) {
        return;
    }

    const start = ctx.currentTime + delay;
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();

    osc.type = type;
    osc.frequency.setValueAtTime(from, start);
    osc.frequency.exponentialRampToValueAtTime(Math.max(to, 1), start + duration);
    gain.gain.setValueAtTime(peak, start);
    gain.gain.exponentialRampToValueAtTime(0.0001, start + duration);

    osc.connect(gain);
    gain.connect(ctx.destination);
    osc.start(start);
    osc.stop(start + duration);
}

export function playPongEvents(events: number): void {
    if (muted || events === 0) {
        return;
    }

    if ((events & PONG_EVENT_HIT_P0) !== 0) {
        blip("square", 420, 260, 0.14, 0.06, 0);
    }
    if ((events & PONG_EVENT_HIT_P1) !== 0) {
        blip("square", 520, 320, 0.14, 0.06, 0);
    }
    if ((events & PONG_EVENT_WALL) !== 0) {
        blip("triangle", 200, 150, 0.09, 0.05, 0);
    }
    if ((events & PONG_EVENT_SERVE) !== 0) {
        blip("sine", 660, 660, 0.08, 0.08, 0);
    }
    if ((events & PONG_EVENT_SCORE) !== 0) {
        blip("sine", 500, 500, 0.16, 0.12, 0);
        blip("sine", 330, 330, 0.16, 0.2, 0.13);
    }
}
