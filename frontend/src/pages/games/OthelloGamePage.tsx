import { useState } from "react";
import { useNavigate, useParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useGameRoom } from "../../api/queries/gameRoom";
import {
    useAcceptGameInvite,
    useDeclineGameInvite,
    useResignGame,
    useSubmitGameAction,
} from "../../api/mutations/gameRoom";
import { OthelloBoardView } from "../../components/games/othello/OthelloBoardView";
import { GameChat } from "../../components/games/chat/GameChat.tsx";
import { Button } from "../../components/Button/Button";
import styles from "./GamesPages.module.css";

export function OthelloGamePage() {
    const { id } = useParams<{ id: string }>();
    const { user } = useAuth();
    const navigate = useNavigate();
    const { room, loading, error, refetch } = useGameRoom(id);
    const [acceptError, setAcceptError] = useState("");
    const acceptInvite = useAcceptGameInvite();
    const declineInvite = useDeclineGameInvite();
    const submitAction = useSubmitGameAction(room?.id ?? "");
    const resign = useResignGame();

    usePageTitle(room ? `Othello - ${room.players.map(p => p.display_name).join(" vs ")}` : "Othello");

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

    const isParticipant = user ? room.players.some(p => p.user_id === user.id) : false;
    const isInvitee = user ? room.created_by !== user.id && isParticipant : false;

    if (room.status === "pending") {
        if (!isParticipant) {
            return (
                <div className={styles.page}>
                    <h2 className={styles.heading}>Othello</h2>
                    <p>This match hasn't started yet - invites are private.</p>
                    <div className={styles.actions}>
                        <Button onClick={() => navigate("/games/live")}>Live Games</Button>
                    </div>
                </div>
            );
        }
        const opponent = room.players.find(p => p.user_id !== user?.id);
        return (
            <div className={styles.page}>
                <h2 className={styles.heading}>Othello</h2>
                {isInvitee ? (
                    <p>
                        {opponent?.display_name ?? "Someone"} has invited you to an othello game. Accept to start - you
                        will play as white.
                    </p>
                ) : (
                    <p>Waiting for {opponent?.display_name ?? "opponent"} to accept.</p>
                )}
                <div className={styles.actions}>
                    {isInvitee && (
                        <>
                            <Button
                                variant="primary"
                                onClick={async () => {
                                    setAcceptError("");
                                    try {
                                        await acceptInvite.mutateAsync(room.id);
                                        await refetch();
                                    } catch (err) {
                                        setAcceptError(err instanceof Error ? err.message : "Failed to accept invite");
                                    }
                                }}
                            >
                                Accept
                            </Button>
                            <Button
                                variant="ghost"
                                onClick={async () => {
                                    await declineInvite.mutateAsync(room.id);
                                    navigate("/games");
                                }}
                            >
                                Decline
                            </Button>
                        </>
                    )}
                    <Button variant="ghost" onClick={() => navigate("/games")}>
                        Back
                    </Button>
                </div>
                {acceptError && <div className={styles.error}>{acceptError}</div>}
            </div>
        );
    }

    async function handleMove(move: { square: string }) {
        await submitAction.mutateAsync({ square: move.square });
    }

    async function handleResign() {
        await resign.mutateAsync(room!.id);
    }

    return (
        <div className={`${styles.page} ${styles.gamePage}`}>
            <div className={styles.boardColumn}>
                <OthelloBoardView
                    room={room}
                    viewer={user}
                    isSpectator={!isParticipant}
                    onMove={handleMove}
                    onResign={handleResign}
                />
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
