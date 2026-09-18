import { useState, type ComponentType, type ReactNode } from "react";
import { useNavigate, useParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useGameRoom } from "../../hooks/queries/gameRoom";
import { useAcceptGameInvite, useDeclineGameInvite } from "../../hooks/mutations/gameRoom";
import { GameChat } from "../../components/games/chat/GameChat";
import { LatencyBanner } from "../../components/games/LatencyBanner";
import { Button } from "../../components/Button/Button";
import { ConfirmDialog } from "../../components/ConfirmDialog/ConfirmDialog";
import { REALTIME_GAME_TYPES } from "../../domain/games/latency";
import { useServerPing } from "../../hooks/useServerPing";
import type { GameRoom, User } from "../../types/api";
import styles from "./GamesPages.module.css";

export interface GameBoardProps<TState = unknown, TStats = unknown> {
    room: GameRoom<TState, TStats>;
    viewer: User | null;
    isSpectator: boolean;
}

interface GameRoomShellProps<TState, TStats> {
    gameName: string;
    inviteCopy: (opponentName: string) => ReactNode;
    Board: ComponentType<GameBoardProps<TState, TStats>>;
}

export function GameRoomShell<TState, TStats>({ gameName, inviteCopy, Board }: GameRoomShellProps<TState, TStats>) {
    const { id } = useParams<{ id: string }>();
    const { user } = useAuth();
    const navigate = useNavigate();
    const { room, loading, error, refetch } = useGameRoom<TState, TStats>(id);
    const [acceptError, setAcceptError] = useState("");
    const [confirmSlowLink, setConfirmSlowLink] = useState(false);
    const acceptInvite = useAcceptGameInvite();
    const declineInvite = useDeclineGameInvite();

    usePageTitle(room ? `${gameName} - ${room.players.map(p => p.display_name).join(" vs ")}` : gameName);

    const isParticipant = user && room ? room.players.some(p => p.user_id === user.id) : false;
    const isInvitee = user && room ? room.created_by !== user.id && isParticipant : false;
    const awaitingAccept = room?.status === "pending" && isInvitee && REALTIME_GAME_TYPES.has(room.game_type);

    const ping = useServerPing(awaitingAccept);

    if (!id) {
        return null;
    }

    if (loading && !room) {
        return <div className={styles.page}>Loading...</div>;
    }

    if (error && !room) {
        return (
            <div className={styles.page}>
                <div className={styles.error}>{error}</div>
                <Button onClick={() => navigate("/games/live")}>Back</Button>
            </div>
        );
    }

    if (!room) {
        return null;
    }

    if (room.status === "declined") {
        return (
            <div className={styles.page}>
                <h2 className={styles.heading}>{gameName}</h2>
                <p>This match never started. The invite was declined or cancelled.</p>
                <div className={styles.actions}>
                    <Button onClick={() => navigate("/games")}>My Games</Button>
                </div>
            </div>
        );
    }

    if (room.status === "pending") {
        if (!isParticipant) {
            return (
                <div className={styles.page}>
                    <h2 className={styles.heading}>{gameName}</h2>
                    <p>This match hasn't started yet - invites are private.</p>
                    <div className={styles.actions}>
                        <Button onClick={() => navigate("/games/live")}>Live Games</Button>
                    </div>
                </div>
            );
        }

        const opponent = room.players.find(p => p.user_id !== user?.id);

        const acceptNow = async () => {
            setConfirmSlowLink(false);
            setAcceptError("");
            try {
                await acceptInvite.mutateAsync(room.id);
                await refetch();
            } catch (err) {
                setAcceptError(err instanceof Error ? err.message : "Failed to accept invite");
            }
        };

        const handleAccept = async () => {
            if (ping.grade === "poor") {
                setConfirmSlowLink(true);
                return;
            }

            await acceptNow();
        };

        const handleDecline = async () => {
            setAcceptError("");
            try {
                await declineInvite.mutateAsync(room.id);
            } catch (err) {
                setAcceptError(err instanceof Error ? err.message : "Failed to decline invite");
                return;
            }
            navigate("/games");
        };

        return (
            <div className={styles.page}>
                <h2 className={styles.heading}>{gameName}</h2>
                {isInvitee ? (
                    <p>{inviteCopy(opponent?.display_name ?? "Someone")}</p>
                ) : (
                    <p>
                        Waiting for <bdi>{opponent?.display_name ?? "opponent"}</bdi> to accept.
                    </p>
                )}
                <div className={styles.actions}>
                    {isInvitee && (
                        <>
                            <Button variant="primary" onClick={handleAccept}>
                                Accept
                            </Button>
                            <Button variant="ghost" onClick={handleDecline}>
                                Decline
                            </Button>
                        </>
                    )}
                    <Button variant="ghost" onClick={() => navigate("/games")}>
                        Back
                    </Button>
                </div>
                {acceptError && <div className={styles.error}>{acceptError}</div>}
                {isInvitee && <LatencyBanner roundTripMs={ping.roundTripMs} grade={ping.grade} />}
                <ConfirmDialog
                    open={confirmSlowLink}
                    title="Your connection is slow"
                    body={
                        <>
                            <p>
                                Your connection to the game server is {Math.round(ping.roundTripMs ?? 0)}ms. {gameName}{" "}
                                is played in real time, so your moves will reach the server late and you will have to
                                aim ahead of what you see.
                            </p>
                            <p>You can still accept, but the match may not be playable.</p>
                        </>
                    }
                    confirmLabel="Accept anyway"
                    cancelLabel="Go back"
                    onConfirm={acceptNow}
                    onCancel={() => setConfirmSlowLink(false)}
                />
            </div>
        );
    }

    return (
        <div className={`${styles.page} ${styles.gamePage}`}>
            <div className={styles.boardColumn}>
                <Board room={room} viewer={user} isSpectator={!isParticipant} />
            </div>
            <div className={styles.chatColumn}>
                <GameChat
                    roomId={room.id}
                    variant={isParticipant ? "player" : "spectator"}
                    watcherCount={room.watcher_count}
                />
            </div>
        </div>
    );
}
