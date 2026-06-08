import { useEffect, useMemo, useRef, useState } from "react";
import type { GameRoom, MinesweeperState, MinesweeperStats, User } from "../../../types/api";
import { Button } from "../../Button/Button";
import { DisconnectBanner } from "../DisconnectBanner";
import { GameOverPanel } from "../GameOverPanel";
import { GamePlayerBar } from "../GamePlayerBar";
import { GameStatsGrid } from "../GameStatsGrid";
import { gameResultLabel, getMySlot, performResignWithConfirm, useDisconnectForfeit } from "../gameRoomHelpers";
import { findCharacter } from "../../../games/minesweeper/characters";
import type { CharacterDef, CharacterId } from "../../../games/minesweeper/types";
import { useCharacterMood } from "../../../games/minesweeper/hooks/useCharacterMood";
import { useGameAudio } from "../../../games/minesweeper/hooks/useGameAudio";
import { useMinesweeperView } from "../../../games/minesweeper/hooks/useMinesweeperView";
import { useIsMobile } from "../../../hooks/useIsMobile";
import { MinesweeperBoard } from "./MinesweeperBoard";
import { MinesweeperCharacterSelect } from "./MinesweeperCharacterSelect";
import { MinesweeperLightningCanvas } from "./MinesweeperLightningCanvas";
import { MinesweeperVsIntro } from "./MinesweeperVsIntro";
import styles from "./MinesweeperBoardView.module.css";

interface MinesweeperBoardViewProps {
    room: GameRoom;
    viewer: User | null;
    isSpectator: boolean;
    onAction: (action: Record<string, unknown>) => Promise<void>;
    onResign: () => Promise<void>;
}

function isMinesweeperStats(x: unknown): x is MinesweeperStats {
    if (!x || typeof x !== "object") {
        return false;
    }
    return "revealed_p0" in x && "revealed_p1" in x;
}

function formatReason(reason: string, loserName?: string): string {
    switch (reason) {
        case "mine_hit":
            return loserName ? `after ${loserName} hit a mine` : "after a mine was hit";
        case "completed":
            return "by clearing the board";
        case "forfeit":
            return loserName ? `after ${loserName} forfeited` : "by forfeit";
        case "resign":
            return loserName ? `after ${loserName} resigned` : "by resignation";
        case "abandoned":
            return "by abandonment";
        case "timeout":
            return "due to inactivity";
        default:
            return reason ? `by ${reason.replace(/_/g, " ")}` : "";
    }
}

export function MinesweeperBoardView({ room, viewer, isSpectator, onAction, onResign }: MinesweeperBoardViewProps) {
    const [error, setError] = useState("");
    const [submitting, setSubmitting] = useState(false);
    const [explosionActive, setExplosionActive] = useState(false);
    const [localPendingClick, setLocalPendingClick] = useState<{ x: number; y: number } | null>(null);
    const [flagMode, setFlagMode] = useState(false);
    const isMobile = useIsMobile();
    const lastAudioPhaseRef = useRef<string | null>(null);
    const lastExplosionKeyRef = useRef<string | null>(null);

    const state = room.state as MinesweeperState | undefined;
    const viewerId = viewer?.id ?? null;
    const mySlot = getMySlot(room, viewerId);
    const visibleSlot = mySlot ?? 0;
    const opponentSlot = 1 - visibleSlot;

    const myCharId = state?.characters?.[visibleSlot] ?? "";
    const oppCharId = state?.characters?.[opponentSlot] ?? "";
    const myCharacter = useMemo<CharacterDef | undefined>(() => findCharacter(myCharId), [myCharId]);
    const opponentCharacter = useMemo<CharacterDef | undefined>(() => findCharacter(oppCharId), [oppCharId]);

    const roomFinished = room.status === "finished" || room.status === "abandoned";
    const { clientPhase, markIntroPlayed } = useMinesweeperView(state ?? null, roomFinished);
    const { myExpr, opExpr } = useCharacterMood({
        state: state ?? null,
        mySlot: visibleSlot,
        myCharacter,
        opponentCharacter,
    });
    const winnerSlot = state?.winner_slot;
    const audioPrimaryChar =
        isSpectator && winnerSlot !== undefined ? (state?.characters?.[winnerSlot] ?? "") : myCharId;
    const audioSecondaryChar =
        isSpectator && winnerSlot !== undefined ? (state?.characters?.[1 - winnerSlot] ?? "") : oppCharId;
    const audio = useGameAudio(
        (audioPrimaryChar || "") as CharacterId | "",
        (audioSecondaryChar || "") as CharacterId | "",
    );

    const { offlinePlayer, forfeitRemaining, liveDurationSeconds } = useDisconnectForfeit(room);

    useEffect(() => {
        if (!state) {
            return;
        }
        const prev = lastAudioPhaseRef.current;
        const next = roomFinished ? "finished" : (state.phase ?? null);
        if (next === prev) {
            return;
        }
        lastAudioPhaseRef.current = next;
        if (prev === null) {
            return;
        }
        if (next === "playing" && prev === "char_select" && !isSpectator) {
            audio.play("start");
        }
        if (next === "finished" && prev !== "finished") {
            if (isSpectator) {
                audio.play("win");
            } else {
                const iWon = state.winner_slot === visibleSlot;
                audio.play(iWon ? "win" : "lose");
            }
        }
    }, [state, roomFinished, isSpectator, audio, visibleSlot]);

    const minesPlacedNow = state?.mines_placed ?? false;
    const [trackedMinesPlaced, setTrackedMinesPlaced] = useState(minesPlacedNow);
    if (trackedMinesPlaced !== minesPlacedNow) {
        setTrackedMinesPlaced(minesPlacedNow);
        if (minesPlacedNow && localPendingClick !== null) {
            setLocalPendingClick(null);
        }
    }

    useEffect(() => {
        if (!state || state.reason !== "mine_hit") {
            return;
        }
        const key = `${state.finished_at ?? ""}-${state.hit_mine_x ?? ""}-${state.hit_mine_y ?? ""}`;
        if (lastExplosionKeyRef.current === key) {
            return;
        }
        const firstObservation = lastExplosionKeyRef.current === null;
        lastExplosionKeyRef.current = key;
        if (firstObservation) {
            return;
        }
        setExplosionActive(true);
        const timer = setTimeout(() => {
            setExplosionActive(false);
        }, 1600);
        return () => clearTimeout(timer);
    }, [state]);

    async function submit(payload: Record<string, unknown>) {
        if (submitting) {
            return;
        }
        setSubmitting(true);
        setError("");
        try {
            await onAction(payload);
        } catch (err) {
            setError(err instanceof Error ? err.message : "Action failed");
        } finally {
            setSubmitting(false);
        }
    }

    async function handleReveal(x: number, y: number) {
        if (state && !state.mines_placed) {
            setLocalPendingClick({ x, y });
        }
        await submit({ type: "reveal", x, y });
    }

    async function handleFlag(x: number, y: number) {
        await submit({ type: "flag", x, y });
    }

    async function handleSelect(character: CharacterId) {
        await submit({ type: "select_character", character });
    }

    async function handleResign() {
        if (submitting) {
            return;
        }
        await performResignWithConfirm(onResign, setSubmitting, setError);
    }

    const result = gameResultLabel(room, viewerId, isSpectator);
    const isOver = room.status === "finished" || room.status === "abandoned";
    const statsAvailable = isMinesweeperStats(room.stats);
    const finishedWinnerSlot = (state as MinesweeperState | undefined)?.winner_slot;
    const loserPlayer =
        finishedWinnerSlot !== undefined ? room.players.find(p => p.slot !== finishedWinnerSlot) : undefined;
    const loserName = loserPlayer?.display_name;
    const reasonText =
        statsAvailable && room.stats ? formatReason((room.stats as MinesweeperStats).reason, loserName) : "";

    if (!state) {
        return <div className={styles.wrapper}>Loading game...</div>;
    }

    const slotMineCount = state.mine_count ?? 0;
    let myFlags = 0;
    const flaggedArr = state.flagged?.[visibleSlot] ?? [];
    for (let i = 0; i < flaggedArr.length; i++) {
        if (flaggedArr[i]) {
            myFlags++;
        }
    }
    const totalSafe = (state.width ?? 0) * (state.height ?? 0) - slotMineCount;
    let opponentFlags = 0;
    const oppFlaggedArr = state.flagged?.[opponentSlot] ?? [];
    for (let i = 0; i < oppFlaggedArr.length; i++) {
        if (oppFlaggedArr[i]) {
            opponentFlags++;
        }
    }
    const visiblePlayer = room.players.find(p => p.slot === visibleSlot);
    const opponentPlayer = room.players.find(p => p.slot === opponentSlot);
    const visibleName = visiblePlayer?.display_name ?? (isSpectator ? `Player ${visibleSlot + 1}` : "You");
    const opponentName = opponentPlayer?.display_name ?? `Player ${opponentSlot + 1}`;
    const leftCellSize = isSpectator ? 22 : 30;
    const rightCellSize = isSpectator ? 22 : 14;
    const leftBannerClass = isSpectator ? styles.bannerNeutral : styles.bannerSelf;
    const rightBannerClass = isSpectator ? styles.bannerNeutral : styles.bannerOpp;
    const leftNameplateClass = isSpectator ? styles.nameplateNeutral : styles.nameplateSelf;
    const rightNameplateClass = isSpectator ? styles.nameplateNeutral : styles.nameplateOpp;

    return (
        <div className={styles.wrapper}>
            <GamePlayerBar room={room} slot0Label="P1" slot1Label="P2" liveDurationSeconds={liveDurationSeconds} />

            <DisconnectBanner offlinePlayer={offlinePlayer} forfeitRemaining={forfeitRemaining} />

            {error && <div className={styles.error}>{error}</div>}

            {clientPhase === "char_select" && (
                <MinesweeperCharacterSelect
                    state={state}
                    mySlot={mySlot}
                    isSpectator={isSpectator}
                    submitting={submitting}
                    onSelect={handleSelect}
                />
            )}

            {clientPhase === "vs_intro" && (
                <MinesweeperVsIntro
                    myCharacter={myCharacter}
                    opponentCharacter={opponentCharacter}
                    onDone={markIntroPlayed}
                />
            )}

            {(clientPhase === "playing" || clientPhase === "finished") && (
                <div className={styles.battlefield}>
                    {explosionActive && (
                        <div className={styles.lightningOverlay}>
                            <MinesweeperLightningCanvas active />
                        </div>
                    )}
                    <section className={styles.column}>
                        <div className={`${styles.banner} ${leftBannerClass}`}>
                            {myExpr.image && (
                                <img className={styles.bannerImg} src={myExpr.image} alt={myCharacter?.name ?? ""} />
                            )}
                            <div className={`${styles.nameplate} ${leftNameplateClass}`}>
                                <span className={styles.nameplatePlayer}>{visibleName}</span>
                                {myCharacter && <span className={styles.nameplateChar}>as {myCharacter.name}</span>}
                            </div>
                        </div>
                        <div className={styles.statsRow}>
                            <div className={styles.stat}>
                                <span className={styles.statLabel}>Progress</span>
                                <span className={styles.statValue}>
                                    {state.revealed_count[visibleSlot] ?? 0} / {totalSafe}
                                </span>
                            </div>
                            <div className={styles.stat}>
                                <span className={styles.statLabel}>Flags</span>
                                <span className={`${styles.statValue} ${styles.statValueFlag}`}>
                                    {myFlags} / {state.mine_count ?? 0}
                                </span>
                            </div>
                        </div>
                        {isMobile && !isSpectator && clientPhase === "playing" && (
                            <div className={styles.modeToggle}>
                                <button
                                    type="button"
                                    className={`${styles.modeButton} ${!flagMode ? styles.modeButtonActive : ""}`}
                                    onClick={() => setFlagMode(false)}
                                    aria-pressed={!flagMode}
                                >
                                    Reveal
                                </button>
                                <button
                                    type="button"
                                    className={`${styles.modeButton} ${flagMode ? styles.modeButtonActive : ""}`}
                                    onClick={() => setFlagMode(true)}
                                    aria-pressed={flagMode}
                                >
                                    ⚑ Flag
                                </button>
                            </div>
                        )}
                        <MinesweeperBoard
                            state={state}
                            slot={visibleSlot}
                            interactive={!isSpectator && clientPhase === "playing"}
                            cellSize={leftCellSize}
                            flagMode={isMobile && flagMode}
                            pendingClick={isSpectator ? null : localPendingClick}
                            onReveal={handleReveal}
                            onFlag={handleFlag}
                        />
                    </section>

                    <section className={styles.column}>
                        <div className={`${styles.banner} ${rightBannerClass}`}>
                            {opExpr.image && (
                                <img
                                    className={styles.bannerImg}
                                    src={opExpr.image}
                                    alt={opponentCharacter?.name ?? ""}
                                />
                            )}
                            <div className={`${styles.nameplate} ${rightNameplateClass}`}>
                                <span className={styles.nameplatePlayer}>{opponentName}</span>
                                {opponentCharacter && (
                                    <span className={styles.nameplateChar}>as {opponentCharacter.name}</span>
                                )}
                            </div>
                        </div>
                        <div className={styles.statsRow}>
                            <div className={styles.stat}>
                                <span className={styles.statLabel}>Progress</span>
                                <span className={styles.statValue}>
                                    {state.revealed_count[opponentSlot] ?? 0} / {totalSafe}
                                </span>
                            </div>
                            {isSpectator && (
                                <div className={styles.stat}>
                                    <span className={styles.statLabel}>Flags</span>
                                    <span className={`${styles.statValue} ${styles.statValueFlag}`}>
                                        {opponentFlags} / {state.mine_count ?? 0}
                                    </span>
                                </div>
                            )}
                        </div>
                        <MinesweeperBoard
                            state={state}
                            slot={opponentSlot}
                            interactive={false}
                            cellSize={rightCellSize}
                        />
                    </section>
                </div>
            )}

            <GameOverPanel
                isOver={isOver}
                showChildren={statsAvailable && (isOver || (room.status === "active" && isSpectator))}
                resultText={result.text}
                resultTone={result.tone}
                reasonText={reasonText}
            >
                {statsAvailable && room.stats && (
                    <GameStatsGrid
                        slot0Name={room.players.find(p => p.slot === 0)?.display_name ?? "P1"}
                        slot1Name={room.players.find(p => p.slot === 1)?.display_name ?? "P2"}
                        isOver={isOver}
                        rows={[
                            {
                                slot0: (room.stats as MinesweeperStats).revealed_p0,
                                label: "Cells revealed",
                                slot1: (room.stats as MinesweeperStats).revealed_p1,
                            },
                            {
                                slot0: (room.stats as MinesweeperStats).flags_p0,
                                label: "Flags placed",
                                slot1: (room.stats as MinesweeperStats).flags_p1,
                            },
                        ]}
                        totalLabel="Duration"
                        totalValue={(room.stats as MinesweeperStats).duration_seconds}
                        durationSeconds={liveDurationSeconds}
                    />
                )}
            </GameOverPanel>

            {room.status === "active" && !isSpectator && clientPhase === "playing" && (
                <div className={styles.controls}>
                    <Button variant="danger" onClick={handleResign} disabled={submitting}>
                        Resign
                    </Button>
                </div>
            )}
        </div>
    );
}
