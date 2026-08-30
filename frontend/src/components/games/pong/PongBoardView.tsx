import { useState } from "react";
import { isPongMuted, setPongMuted } from "../../../games/pong/audio";
import type { GameRoom, PongState, PongStats, User } from "../../../types/api";
import { Button } from "../../Button/Button";
import { DisconnectBanner } from "../DisconnectBanner";
import { GameOverPanel } from "../GameOverPanel";
import { GamePlayerBar } from "../GamePlayerBar";
import { GameStatsGrid } from "../GameStatsGrid";
import { gameResultLabel, getMySlot, performResignWithConfirm, useDisconnectForfeit } from "../gameRoomHelpers";
import shell from "../boardShell.module.css";
import { PongCourt } from "./PongCourt";
import styles from "./PongBoardView.module.css";

interface PongBoardViewProps {
    room: GameRoom<PongState, PongStats>;
    viewer: User | null;
    isSpectator: boolean;
    onResign: () => Promise<void>;
}

function isPongStats(x: unknown): x is PongStats {
    if (!x || typeof x !== "object") {
        return false;
    }
    return "points_p0" in x && "points_p1" in x;
}

function reasonText(reason: string): string {
    switch (reason) {
        case "win":
            return "on points";
        case "forfeit":
            return "by forfeit";
        case "resign":
            return "by resignation";
        case "abandoned":
            return "by abandonment";
        case "time_limit":
            return "on the time limit";
        case "timeout":
            return "due to inactivity";
        default:
            return reason ? `by ${reason.replace(/_/g, " ")}` : "";
    }
}

export function PongBoardView({ room, viewer, isSpectator, onResign }: PongBoardViewProps) {
    const [error, setError] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [muted, setMuted] = useState(() => isPongMuted());

    const { offlinePlayer, forfeitRemaining, liveDurationSeconds } = useDisconnectForfeit(room);

    const viewerId = viewer?.id ?? null;
    const mySlot = getMySlot(room, viewerId);
    const state = room.state;

    const result = gameResultLabel(room, viewerId, isSpectator);
    const isOver = room.status === "finished" || room.status === "abandoned";
    const stats = isPongStats(room.stats) ? room.stats : null;
    const showStats = stats !== null && (isOver || (room.status === "active" && isSpectator));

    async function handleResign() {
        if (submitting) {
            return;
        }
        await performResignWithConfirm(onResign, setSubmitting, setError);
    }

    function toggleSound() {
        const next = !muted;
        setPongMuted(next);
        setMuted(next);
    }

    const scoreLine = `Score ${state.scores[0]} - ${state.scores[1]}, first to ${state.points_to_win}`;

    return (
        <div className={shell.wrapper}>
            <GamePlayerBar room={room} slot0Label="P1" slot1Label="P2" liveDurationSeconds={liveDurationSeconds} />

            <DisconnectBanner offlinePlayer={offlinePlayer} forfeitRemaining={forfeitRemaining} />

            {error && <div className={shell.error}>{error}</div>}

            <div className={`${shell.info} ${styles.scoreLine}`}>{scoreLine}</div>

            <div className={styles.court}>
                <PongCourt room={room} state={state} mySlot={mySlot} isSpectator={isSpectator} />
            </div>

            <GameOverPanel
                isOver={isOver}
                showChildren={showStats}
                resultText={result.text}
                resultTone={result.tone}
                reasonText={stats ? reasonText(stats.result_reason) : ""}
            >
                {showStats && stats && (
                    <GameStatsGrid
                        slot0Name={room.players.find(p => p.slot === 0)?.display_name ?? "P1"}
                        slot1Name={room.players.find(p => p.slot === 1)?.display_name ?? "P2"}
                        isOver={isOver}
                        rows={[
                            { slot0: stats.points_p0, label: "Points", slot1: stats.points_p1 },
                            { slot0: stats.hits_p0, label: "Paddle hits", slot1: stats.hits_p1 },
                            { slot0: stats.longest_rally_p0, label: "Longest rally", slot1: stats.longest_rally_p1 },
                        ]}
                        totalLabel="Top speed"
                        totalValue={stats.top_speed}
                        durationSeconds={liveDurationSeconds}
                    />
                )}
            </GameOverPanel>

            <div className={styles.controls}>
                <span className={styles.hint}>
                    {isSpectator || mySlot === null
                        ? "Watching a live match"
                        : "Move the mouse, drag, or hold the arrow keys to steer your paddle"}
                </span>
                <div className={shell.controls}>
                    <Button variant="ghost" onClick={toggleSound} aria-pressed={!muted}>
                        {muted ? "Sound off" : "Sound on"}
                    </Button>
                    {room.status === "active" && !isSpectator && (
                        <Button variant="danger" onClick={handleResign} disabled={submitting}>
                            Resign
                        </Button>
                    )}
                </div>
            </div>
        </div>
    );
}
