import { useEffect, useMemo, useRef, useState } from "react";
import type { GameRoom, SnakesLaddersState, SnakesLaddersStats, User } from "../../../types/api.ts";
import { Button } from "../../Button/Button.tsx";
import { DisconnectBanner } from "../DisconnectBanner.tsx";
import { GameOverPanel } from "../GameOverPanel.tsx";
import { GamePlayerBar } from "../GamePlayerBar.tsx";
import { GameStatsGrid } from "../GameStatsGrid.tsx";
import { gameResultLabel, performResignWithConfirm, useDisconnectForfeit } from "../gameRoomHelpers.ts";
import type { BoardToken } from "./SnakesLaddersBoard.tsx";
import { SnakesLaddersBoard } from "./SnakesLaddersBoard.tsx";
import styles from "./SnakesAndLaddersBoardView.module.css";

const SLOT_ONE = 0;
const SLOT_TWO = 1;

interface SnakesAndLaddersBoardViewProps {
    room: GameRoom;
    viewer: User | null;
    isSpectator: boolean;
    onRoll: () => Promise<void>;
    onResign: () => Promise<void>;
}

const PIP_LAYOUT: Record<number, Array<[number, number]>> = {
    1: [[50, 50]],
    2: [
        [30, 30],
        [70, 70],
    ],
    3: [
        [30, 30],
        [50, 50],
        [70, 70],
    ],
    4: [
        [30, 30],
        [70, 30],
        [30, 70],
        [70, 70],
    ],
    5: [
        [30, 30],
        [70, 30],
        [50, 50],
        [30, 70],
        [70, 70],
    ],
    6: [
        [30, 26],
        [70, 26],
        [30, 50],
        [70, 50],
        [30, 74],
        [70, 74],
    ],
};

const DICE_ROLL_MS = 600;
const GLIDE_MS = 600;
const PAUSE_MS = 250;

function DiceFace({ value, rolling }: { value: number; rolling: boolean }) {
    const pips = PIP_LAYOUT[value] ?? [];
    return (
        <svg
            className={`${styles.dice} ${rolling ? styles.diceRolling : ""}`}
            viewBox="0 0 100 100"
            role="img"
            aria-label={`Die showing ${value}`}
        >
            <rect x="4" y="4" width="92" height="92" rx="18" fill="#f6efda" stroke="#b88a32" strokeWidth="4" />
            {pips.map((p, i) => (
                <circle key={i} cx={p[0]} cy={p[1]} r="9" fill="#3a2a12" />
            ))}
        </svg>
    );
}

function isSnakesLaddersStats(x: unknown): x is SnakesLaddersStats {
    if (!x || typeof x !== "object") {
        return false;
    }
    return "total_rolls" in x && "final_p0" in x;
}

function tokenFor(name: string | undefined, color: string, ring: string): BoardToken {
    const initial = name && name.length > 0 ? name[0].toUpperCase() : "?";
    return { color, ring, initial };
}

export function SnakesAndLaddersBoardView({
    room,
    viewer,
    isSpectator,
    onRoll,
    onResign,
}: SnakesAndLaddersBoardViewProps) {
    const [error, setError] = useState("");
    const [submitting, setSubmitting] = useState(false);

    const state = room.state as Partial<SnakesLaddersState> | undefined;
    const positions = state?.positions ?? [0, 0];
    const last = state?.last;
    const rolls = state?.rolls ?? 0;

    const [displayPositions, setDisplayPositions] = useState<number[]>(positions);
    const [diceRolling, setDiceRolling] = useState(false);
    const animatedRollsRef = useRef(-1);
    const timersRef = useRef<number[]>([]);

    const moverSlot = last?.slot ?? -1;
    const fromCell = last?.from ?? 0;
    const steppedCell = last?.stepped ?? 0;
    const toCell = last?.to ?? 0;
    const p0 = positions[0];
    const p1 = positions[1];

    useEffect(() => {
        const clearTimers = () => {
            for (const t of timersRef.current) {
                clearTimeout(t);
            }
            timersRef.current = [];
        };

        if (rolls === 0 || moverSlot < 0 || animatedRollsRef.current === rolls) {
            animatedRollsRef.current = rolls;
            return clearTimers;
        }

        const firstSight = animatedRollsRef.current < 0;
        animatedRollsRef.current = rolls;
        if (firstSight) {
            return clearTimers;
        }

        clearTimers();

        const moverAt = (cell: number) => {
            const next = [p0, p1];
            next[moverSlot] = cell;
            return next;
        };

        timersRef.current.push(
            window.setTimeout(() => {
                setDiceRolling(true);
                setDisplayPositions(moverAt(fromCell));
            }, 0),
        );
        timersRef.current.push(
            window.setTimeout(() => {
                setDiceRolling(false);
                setDisplayPositions(moverAt(steppedCell));
            }, DICE_ROLL_MS),
        );
        if (toCell !== steppedCell) {
            timersRef.current.push(
                window.setTimeout(
                    () => {
                        setDisplayPositions(moverAt(toCell));
                    },
                    DICE_ROLL_MS + GLIDE_MS + PAUSE_MS,
                ),
            );
        }

        return clearTimers;
    }, [rolls, moverSlot, fromCell, steppedCell, toCell, p0, p1]);

    const viewerId = viewer?.id ?? null;
    const isMyTurn = !isSpectator && viewerId !== null && room.turn_user_id === viewerId && room.status === "active";

    const { offlinePlayer, forfeitRemaining, liveDurationSeconds } = useDisconnectForfeit(room);

    const slot0 = room.players.find(p => p.slot === SLOT_ONE);
    const slot1 = room.players.find(p => p.slot === SLOT_TWO);

    const tokens = useMemo<BoardToken[]>(
        () => [
            tokenFor(slot0?.display_name, "#c0392b", "#f1c40f"),
            tokenFor(slot1?.display_name, "#2e6db4", "#f6e7a8"),
        ],
        [slot0?.display_name, slot1?.display_name],
    );

    async function handleRoll() {
        if (!isMyTurn || submitting) {
            return;
        }

        setSubmitting(true);
        setError("");

        try {
            await onRoll();
        } catch (err) {
            setError(err instanceof Error ? err.message : "Roll failed");
        } finally {
            setSubmitting(false);
        }
    }

    async function handleResign() {
        if (submitting) {
            return;
        }
        await performResignWithConfirm(onResign, setSubmitting, setError);
    }

    const result = gameResultLabel(room, viewerId, isSpectator);
    const isOver = room.status === "finished" || room.status === "abandoned";
    const statsAvailable = isSnakesLaddersStats(room.stats);
    const showStats = statsAvailable && (isOver || (room.status === "active" && isSpectator));
    const stats = statsAvailable ? (room.stats as SnakesLaddersStats) : null;

    function describeLast(): string {
        if (!last) {
            return "";
        }

        const mover = last.slot === SLOT_ONE ? slot0 : slot1;
        const name = mover?.display_name ?? (last.slot === SLOT_ONE ? "Player 1" : "Player 2");

        if (last.to === last.from) {
            return `${name} rolled ${last.roll} and overshot 100 - staying put.`;
        }
        if (last.to > last.stepped) {
            return `${name} rolled ${last.roll}, found a ladder at ${last.stepped} and climbed to ${last.to}.`;
        }
        if (last.to < last.stepped) {
            return `${name} rolled ${last.roll}, hit a snake at ${last.stepped} and slid to ${last.to}.`;
        }
        return `${name} rolled ${last.roll} and moved to ${last.to}.`;
    }

    const lastText = describeLast();

    return (
        <div className={styles.wrapper}>
            <GamePlayerBar
                room={room}
                slot0Label="Player 1"
                slot1Label="Player 2"
                liveDurationSeconds={liveDurationSeconds}
            />

            <DisconnectBanner offlinePlayer={offlinePlayer} forfeitRemaining={forfeitRemaining} />

            {error && <div className={styles.error}>{error}</div>}

            <div className={styles.boardContainer}>
                <SnakesLaddersBoard positions={displayPositions} tokens={tokens} lastTo={last?.to ?? null} />
            </div>

            {room.status === "active" && (
                <div className={styles.rollRow}>
                    {last && <DiceFace value={last.roll} rolling={diceRolling} />}
                    <div className={styles.rollInfo}>
                        {lastText && <span className={styles.lastText}>{lastText}</span>}
                        {!isSpectator && (
                            <Button variant="primary" onClick={handleRoll} disabled={!isMyTurn || submitting}>
                                {isMyTurn ? (submitting ? "Rolling..." : "Roll the die") : "Waiting for opponent..."}
                            </Button>
                        )}
                    </div>
                </div>
            )}

            <GameOverPanel isOver={isOver} showChildren={showStats} resultText={result.text} resultTone={result.tone}>
                {showStats && stats && (
                    <GameStatsGrid
                        slot0Name={slot0?.display_name ?? "Player 1"}
                        slot1Name={slot1?.display_name ?? "Player 2"}
                        isOver={isOver}
                        rows={[
                            { slot0: stats.final_p0, label: "Square", slot1: stats.final_p1 },
                            { slot0: stats.rolls_p0, label: "Rolls", slot1: stats.rolls_p1 },
                            { slot0: stats.ladders_p0, label: "Ladders climbed", slot1: stats.ladders_p1 },
                            { slot0: stats.snakes_p0, label: "Snakes hit", slot1: stats.snakes_p1 },
                        ]}
                        totalLabel="Total rolls"
                        totalValue={stats.total_rolls}
                        durationSeconds={liveDurationSeconds}
                    />
                )}
            </GameOverPanel>

            {room.status === "active" && !isSpectator && (
                <div className={styles.controls}>
                    <Button variant="danger" onClick={handleResign} disabled={submitting}>
                        Resign
                    </Button>
                </div>
            )}
        </div>
    );
}
