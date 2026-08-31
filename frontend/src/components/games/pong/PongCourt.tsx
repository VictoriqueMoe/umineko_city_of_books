import { useCallback, useEffect, useRef, type PointerEvent as ReactPointerEvent } from "react";
import { playPongEvents } from "../../../games/pong/audio";
import { usePongFrames } from "../../../games/pong/hooks/usePongFrames";
import { usePongInput } from "../../../games/pong/hooks/usePongInput";
import { sampleAt, serverNow } from "../../../games/pong/interpolate";
import {
    PONG_EVENT_HIT_P0,
    PONG_EVENT_HIT_P1,
    PONG_EVENT_SCORE,
    PONG_RENDER_DELAY_MS,
} from "../../../games/pong/types";
import type { GameRoom, PongState } from "../../../types/api";
import styles from "./PongCourt.module.css";

interface PongCourtProps {
    room: GameRoom;
    state: PongState;
    mySlot: number | null;
    isSpectator: boolean;
}

interface CourtColours {
    background: string;
    line: string;
    ball: string;
    trailRgb: string;
    local: string;
    opponent: string;
    flash: string;
    text: string;
    muted: string;
}

const TRAIL_MS = 100;
const TRAIL_MAX = 24;
const FLASH_MS = 60;
const SHAKE_MS = 120;
const SHAKE_PX = 2;
const SCRIM = "rgba(0, 0, 0, 0.55)";
const SLOTS = [0, 1];
const COURT_FONT = '"Courier New", ui-monospace, monospace';

function clamp(value: number, min: number, max: number): number {
    if (value < min) {
        return min;
    }
    if (value > max) {
        return max;
    }
    return value;
}

function readColours(): CourtColours {
    const root = getComputedStyle(document.documentElement);
    const token = (name: string, fallback: string) => {
        const raw = root.getPropertyValue(name).trim();
        return raw === "" ? fallback : raw;
    };
    const trailRgb = token("--gold-rgb", "212, 168, 75");

    return {
        background: token("--bg-void", "#0a0612"),
        line: `rgba(${trailRgb}, 0.28)`,
        ball: token("--gold-light", "#f0d590"),
        trailRgb,
        local: token("--gold", "#d4a84b"),
        opponent: token("--purple-light", "#9d7bc9"),
        flash: token("--text", "#e8e0f0"),
        text: token("--text", "#e8e0f0"),
        muted: token("--text-muted", "#a89bb8"),
    };
}

export function PongCourt({ room, state, mySlot, isSpectator }: PongCourtProps) {
    const canvasRef = useRef<HTMLCanvasElement>(null);
    const animRef = useRef(0);
    const drawRef = useRef<() => void>(() => {});
    const coloursRef = useRef<CourtColours | null>(null);
    const sizeRef = useRef({ cssWidth: 0, cssHeight: 0, dpr: 1 });
    const trailRef = useRef<{ x: number; y: number; at: number }[]>([]);
    const localYRef = useRef<number | null>(null);
    const lastFiredTRef = useRef(0);
    const flashRef = useRef<[number, number]>([0, 0]);
    const shakeRef = useRef(0);
    const reduceMotionRef = useRef(false);

    const active = room.status === "active";
    const localSlot = mySlot ?? 0;
    const enabled = active && !isSpectator && mySlot !== null;

    const frames = usePongFrames(room.id);
    const { targetRef, seedTargetY, setTargetFromPointer } = usePongInput({
        roomId: room.id,
        enabled,
        courtHeight: state.height,
        paddleHeight: state.paddle_height,
        initialY: state.paddle_y[localSlot],
    });

    useEffect(() => {
        const query = window.matchMedia("(prefers-reduced-motion: reduce)");
        reduceMotionRef.current = query.matches;

        const onChange = (event: MediaQueryListEvent) => {
            reduceMotionRef.current = event.matches;
        };

        query.addEventListener("change", onChange);

        return () => query.removeEventListener("change", onChange);
    }, []);

    useEffect(() => {
        const canvas = canvasRef.current;
        if (!canvas) {
            return;
        }
        const parent = canvas.parentElement;
        if (!parent) {
            return;
        }

        const resize = () => {
            const cssWidth = parent.clientWidth;
            const cssHeight = parent.clientHeight;
            if (cssWidth <= 0 || cssHeight <= 0) {
                return;
            }

            const dpr = window.devicePixelRatio || 1;
            canvas.width = Math.round(cssWidth * dpr);
            canvas.height = Math.round(cssHeight * dpr);
            sizeRef.current = { cssWidth, cssHeight, dpr };
            coloursRef.current = readColours();

            drawRef.current();
        };

        const observer = new ResizeObserver(resize);
        observer.observe(parent);
        resize();

        return () => observer.disconnect();
    }, []);

    useEffect(() => {
        const canvas = canvasRef.current;
        if (!canvas) {
            return;
        }
        const ctx = canvas.getContext("2d");
        if (!ctx) {
            return;
        }

        const draw = () => {
            const { cssWidth, cssHeight, dpr } = sizeRef.current;
            if (cssWidth <= 0 || cssHeight <= 0) {
                return;
            }

            const colours = coloursRef.current ?? readColours();
            coloursRef.current = colours;

            const now = Date.now();

            const buffer = frames.current;
            const sample = buffer.length > 0 ? sampleAt(buffer, serverNow(buffer, now) - PONG_RENDER_DELAY_MS) : null;

            const phase = sample ? sample.phase : state.phase;
            const scores = sample ? sample.scores : state.scores;
            const connected = sample ? sample.connected : ([true, true] as [boolean, boolean]);
            const serveInMs = sample ? sample.serveInMs : state.phase_remain_ms;
            const ballX = sample ? sample.ballX : state.ball_x;
            const ballY = sample ? sample.ballY : state.ball_y;
            const stale = sample ? sample.stale : false;
            const paddleY: [number, number] = sample
                ? [sample.paddleY[0], sample.paddleY[1]]
                : [state.paddle_y[0], state.paddle_y[1]];

            if (sample) {
                for (const at of sample.events) {
                    if (at.t <= lastFiredTRef.current) {
                        continue;
                    }
                    lastFiredTRef.current = at.t;
                    playPongEvents(at.events);
                    if ((at.events & PONG_EVENT_HIT_P0) !== 0) {
                        flashRef.current[0] = now;
                    }
                    if ((at.events & PONG_EVENT_HIT_P1) !== 0) {
                        flashRef.current[1] = now;
                    }
                    if ((at.events & PONG_EVENT_SCORE) !== 0) {
                        shakeRef.current = now;
                    }
                }
            }

            if (enabled) {
                const half = state.paddle_height / 2;

                if (sample) {
                    seedTargetY(sample.paddleY[localSlot]);
                }

                const predicted = clamp(targetRef.current, half, state.height - half);

                localYRef.current = predicted;
                paddleY[localSlot] = predicted;
            }

            if (!reduceMotionRef.current && phase === "rally") {
                const trail = trailRef.current;
                trail.push({ x: ballX, y: ballY, at: now });

                let expired = 0;
                while (expired < trail.length && now - trail[expired].at > TRAIL_MS) {
                    expired += 1;
                }
                const overflow = Math.max(expired, trail.length - TRAIL_MAX);
                if (overflow > 0) {
                    trail.splice(0, overflow);
                }
            } else {
                trailRef.current = [];
            }

            const shaking = !reduceMotionRef.current && now - shakeRef.current < SHAKE_MS;
            const shakeX = shaking ? (Math.random() - 0.5) * 2 * SHAKE_PX : 0;
            const shakeY = shaking ? (Math.random() - 0.5) * 2 * SHAKE_PX : 0;

            const scale = Math.min(cssWidth / state.width, cssHeight / state.height);
            const offsetX = (cssWidth - state.width * scale) / 2;
            const offsetY = (cssHeight - state.height * scale) / 2;

            ctx.setTransform(1, 0, 0, 1, 0, 0);
            ctx.clearRect(0, 0, canvas.width, canvas.height);
            ctx.fillStyle = colours.background;
            ctx.fillRect(0, 0, canvas.width, canvas.height);
            ctx.setTransform(dpr * scale, 0, 0, dpr * scale, (offsetX + shakeX) * dpr, (offsetY + shakeY) * dpr);

            ctx.save();
            ctx.strokeStyle = colours.line;
            ctx.lineWidth = 6;
            ctx.setLineDash([20, 26]);
            ctx.beginPath();
            ctx.moveTo(state.width / 2, 0);
            ctx.lineTo(state.width / 2, state.height);
            ctx.stroke();
            ctx.restore();

            ctx.save();
            ctx.fillStyle = colours.line;
            ctx.font = `bold ${Math.round(state.height * 0.2)}px ${COURT_FONT}`;
            ctx.textBaseline = "top";
            ctx.textAlign = "right";
            ctx.fillText(String(scores[0]), state.width / 2 - state.width * 0.05, state.height * 0.05);
            ctx.textAlign = "left";
            ctx.fillText(String(scores[1]), state.width / 2 + state.width * 0.05, state.height * 0.05);
            ctx.restore();

            if (trailRef.current.length > 1) {
                ctx.save();
                for (const [index, point] of trailRef.current.entries()) {
                    const fade = (index + 1) / trailRef.current.length;
                    ctx.fillStyle = `rgba(${colours.trailRgb}, ${(fade * 0.3).toFixed(3)})`;
                    ctx.beginPath();
                    ctx.arc(point.x, point.y, state.ball_radius * (0.45 + fade * 0.5), 0, Math.PI * 2);
                    ctx.fill();
                }
                ctx.restore();
            }

            for (const slot of SLOTS) {
                const x = slot === 0 ? state.paddle_inset : state.width - state.paddle_inset - state.paddle_width;
                const mine = mySlot !== null && !isSpectator ? slot === localSlot : slot === 0;
                const flashing = now - flashRef.current[slot] < FLASH_MS;

                ctx.save();
                ctx.globalAlpha = connected[slot] ? 1 : 0.35;
                ctx.fillStyle = flashing ? colours.flash : mine ? colours.local : colours.opponent;
                ctx.beginPath();
                ctx.roundRect(
                    x,
                    paddleY[slot] - state.paddle_height / 2,
                    state.paddle_width,
                    state.paddle_height,
                    state.paddle_width / 2,
                );
                ctx.fill();
                ctx.restore();
            }

            ctx.save();
            ctx.fillStyle = colours.ball;
            ctx.beginPath();
            ctx.arc(ballX, ballY, state.ball_radius, 0, Math.PI * 2);
            ctx.fill();
            ctx.restore();

            if (phase === "countdown" || phase === "serve") {
                ctx.save();
                ctx.textAlign = "center";
                ctx.textBaseline = "middle";
                ctx.fillStyle = colours.text;
                ctx.font = `bold ${Math.round(state.height * 0.16)}px ${COURT_FONT}`;
                ctx.fillText(String(Math.max(0, Math.ceil(serveInMs / 1000))), state.width / 2, state.height * 0.32);
                ctx.fillStyle = colours.muted;
                ctx.font = `${Math.round(state.height * 0.05)}px ${COURT_FONT}`;
                ctx.fillText(phase === "countdown" ? "GET READY" : "NEXT SERVE", state.width / 2, state.height * 0.72);
                ctx.restore();
            }

            if (stale && active) {
                ctx.save();
                ctx.fillStyle = SCRIM;
                ctx.fillRect(0, 0, state.width, state.height);
                ctx.textAlign = "center";
                ctx.textBaseline = "middle";
                ctx.fillStyle = colours.muted;
                ctx.font = `${Math.round(state.height * 0.06)}px ${COURT_FONT}`;
                ctx.fillText("Reconnecting...", state.width / 2, state.height / 2);
                ctx.restore();
            }
        };

        drawRef.current = draw;

        if (!active) {
            draw();

            return () => {
                drawRef.current = () => {};
            };
        }

        const loop = () => {
            draw();
            animRef.current = window.requestAnimationFrame(loop);
        };

        animRef.current = window.requestAnimationFrame(loop);

        return () => {
            window.cancelAnimationFrame(animRef.current);
            drawRef.current = () => {};
        };
    }, [active, enabled, frames, isSpectator, localSlot, mySlot, seedTargetY, state, targetRef]);

    const handlePointer = useCallback(
        (event: ReactPointerEvent<HTMLCanvasElement>) => {
            if (!enabled) {
                return;
            }
            setTargetFromPointer(event.clientY, event.currentTarget.getBoundingClientRect());
        },
        [enabled, setTargetFromPointer],
    );

    const handlePointerDown = useCallback(
        (event: ReactPointerEvent<HTMLCanvasElement>) => {
            if (!enabled) {
                return;
            }
            event.currentTarget.setPointerCapture(event.pointerId);
            handlePointer(event);
        },
        [enabled, handlePointer],
    );

    return (
        <canvas
            ref={canvasRef}
            className={`${styles.canvas} ${enabled ? styles.interactive : ""}`}
            role="img"
            aria-label="Pong court"
            onPointerDown={handlePointerDown}
            onPointerMove={handlePointer}
        />
    );
}
