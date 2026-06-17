import { useMemo } from "react";
import { buildLadder, buildSnake, cellCenter, cellTopLeft, GRID, CELL, LADDERS, SNAKES, VIEW } from "./board";
import styles from "./SnakesLaddersBoard.module.css";

export interface BoardToken {
    color: string;
    ring: string;
    initial: string;
}

interface SnakesLaddersBoardProps {
    positions: number[];
    tokens: BoardToken[];
    lastTo?: number | null;
}

const SNAKE_PALETTE: Array<[string, string]> = [
    ["#3aa76d", "#1c6f48"],
    ["#3a9aa7", "#1c6671"],
    ["#9072c6", "#5b3f97"],
    ["#c2566f", "#8c2c46"],
    ["#5fa03a", "#3a6c1f"],
];

const START_Y = VIEW + 42;

function tokenPosition(cell: number, slot: number, slotCount: number): { x: number; y: number } {
    const spread = slotCount > 1 ? 17 : 0;
    const shift = (slot - (slotCount - 1) / 2) * spread;
    if (cell <= 0) {
        return { x: VIEW / 2 + shift * 1.6, y: START_Y };
    }
    const c = cellCenter(cell);
    return { x: c.x + shift, y: c.y + 6 };
}

export function SnakesLaddersBoard({ positions, tokens, lastTo }: SnakesLaddersBoardProps) {
    const cells = useMemo(() => {
        const out: Array<{ n: number; x: number; y: number; dark: boolean }> = [];
        for (let n = 1; n <= GRID * GRID; n++) {
            const tl = cellTopLeft(n);
            const col = Math.round(tl.x / CELL);
            const row = Math.round(tl.y / CELL);
            out.push({ n, x: tl.x, y: tl.y, dark: (col + row) % 2 === 0 });
        }
        return out;
    }, []);

    const ladders = useMemo(() => {
        return Object.entries(LADDERS).map(([from, to]) => ({
            key: `l-${from}`,
            geo: buildLadder(Number(from), to),
        }));
    }, []);

    const snakes = useMemo(() => {
        return Object.entries(SNAKES).map(([from, to], i) => ({
            key: `s-${from}`,
            geo: buildSnake(Number(from), to),
            colors: SNAKE_PALETTE[i % SNAKE_PALETTE.length],
        }));
    }, []);

    return (
        <svg
            className={styles.board}
            viewBox={`-12 -12 ${VIEW + 24} ${START_Y + 60}`}
            role="img"
            aria-label="Snakes and ladders board"
        >
            <defs>
                <linearGradient id="sl-frame" x1="0" y1="0" x2="1" y2="1">
                    <stop offset="0" stopColor="#e3c069" />
                    <stop offset="0.5" stopColor="#b88a32" />
                    <stop offset="1" stopColor="#7c5b1d" />
                </linearGradient>
                <linearGradient id="sl-rail" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0" stopColor="#c98b46" />
                    <stop offset="1" stopColor="#8a5a22" />
                </linearGradient>
                {snakes.map((s, i) => (
                    <linearGradient key={s.key} id={`sl-snake-${i}`} x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0" stopColor={s.colors[0]} />
                        <stop offset="1" stopColor={s.colors[1]} />
                    </linearGradient>
                ))}
                <radialGradient id="sl-goal" cx="0.5" cy="0.5" r="0.7">
                    <stop offset="0" stopColor="#f6e7a8" />
                    <stop offset="1" stopColor="#e0b94e" />
                </radialGradient>
                <filter id="sl-shadow" x="-30%" y="-30%" width="160%" height="160%">
                    <feDropShadow dx="0" dy="3" stdDeviation="3" floodColor="#000" floodOpacity="0.35" />
                </filter>
            </defs>

            <rect x={-12} y={-12} width={VIEW + 24} height={START_Y + 60} rx={22} fill="url(#sl-frame)" />
            <rect x={0} y={0} width={VIEW} height={VIEW} rx={8} fill="#efe2c2" />

            {cells.map(cell => {
                const isGoal = cell.n === 100;
                return (
                    <g key={cell.n}>
                        <rect
                            x={cell.x}
                            y={cell.y}
                            width={CELL}
                            height={CELL}
                            fill={isGoal ? "url(#sl-goal)" : cell.dark ? "#e7d4a8" : "#f4e9cd"}
                            stroke="#cbb27e"
                            strokeWidth={1}
                        />
                        <text x={cell.x + 8} y={cell.y + 24} className={styles.cellNumber}>
                            {cell.n}
                        </text>
                        {isGoal && (
                            <text x={cell.x + CELL / 2} y={cell.y + CELL - 22} className={styles.goalMark}>
                                ★
                            </text>
                        )}
                    </g>
                );
            })}

            {ladders.map(l => (
                <g key={l.key} strokeLinecap="round" filter="url(#sl-shadow)">
                    {l.geo.rails.map((rail, i) => (
                        <line
                            key={`rail-${i}`}
                            x1={rail[0].x}
                            y1={rail[0].y}
                            x2={rail[1].x}
                            y2={rail[1].y}
                            stroke="url(#sl-rail)"
                            strokeWidth={9}
                        />
                    ))}
                    {l.geo.rungs.map((rung, i) => (
                        <line
                            key={`rung-${i}`}
                            x1={rung[0].x}
                            y1={rung[0].y}
                            x2={rung[1].x}
                            y2={rung[1].y}
                            stroke="#9a6526"
                            strokeWidth={7}
                        />
                    ))}
                </g>
            ))}

            {snakes.map((s, i) => (
                <g key={s.key} filter="url(#sl-shadow)">
                    <path d={s.geo.body} fill={`url(#sl-snake-${i})`} stroke={s.colors[1]} strokeWidth={2.5} />
                    <path
                        d={s.geo.belly}
                        fill="none"
                        stroke="#ffffff"
                        strokeOpacity={0.35}
                        strokeWidth={3}
                        strokeLinecap="round"
                        strokeDasharray="2 16"
                    />
                    <path d={s.geo.tongue} fill="none" stroke="#d2354b" strokeWidth={3.5} strokeLinecap="round" />
                    <circle cx={s.geo.head.x} cy={s.geo.head.y} r={s.geo.headRadius} fill={s.colors[1]} />
                    <circle
                        cx={s.geo.head.x}
                        cy={s.geo.head.y}
                        r={s.geo.headRadius}
                        fill={`url(#sl-snake-${i})`}
                        fillOpacity={0.55}
                    />
                    {s.geo.eyes.map((eye, j) => (
                        <g key={`eye-${j}`}>
                            <circle cx={eye.x} cy={eye.y} r={6} fill="#fdf6e3" />
                            <circle
                                cx={eye.x + s.geo.pupilOffset.x}
                                cy={eye.y + s.geo.pupilOffset.y}
                                r={3}
                                fill="#1a1a1a"
                            />
                        </g>
                    ))}
                </g>
            ))}

            <text x={VIEW / 2} y={START_Y + 34} className={styles.startLabel}>
                START
            </text>

            {positions.map((pos, slot) => {
                const token = tokens[slot];
                if (!token) {
                    return null;
                }
                const p = tokenPosition(pos, slot, positions.length);
                return (
                    <g
                        key={`token-${slot}`}
                        className={styles.token}
                        style={{ transform: `translate(${p.x}px, ${p.y}px)` }}
                        filter="url(#sl-shadow)"
                    >
                        <circle cx={0} cy={0} r={21} fill={token.ring} />
                        <circle cx={0} cy={0} r={17} fill={token.color} />
                        <text x={0} y={6} className={styles.tokenLabel}>
                            {token.initial}
                        </text>
                    </g>
                );
            })}

            {lastTo && lastTo > 0 && (
                <rect
                    x={cellTopLeft(lastTo).x + 3}
                    y={cellTopLeft(lastTo).y + 3}
                    width={CELL - 6}
                    height={CELL - 6}
                    fill="none"
                    stroke="#d4af37"
                    strokeWidth={4}
                    rx={6}
                    className={styles.lastPulse}
                />
            )}
        </svg>
    );
}
