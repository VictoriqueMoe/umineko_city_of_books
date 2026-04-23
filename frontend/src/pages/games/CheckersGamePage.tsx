import { useNavigate, useParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useGameRoom } from "../../hooks/useGameRoom";
import * as api from "../../api/endpoints";
import { CheckersBoardView } from "../../components/games/checkers/CheckersBoardView";
import { SpectatorChat } from "../../components/games/chess/SpectatorChat";
import { PlayerChat } from "../../components/games/chess/PlayerChat";
import { Button } from "../../components/Button/Button";
import styles from "./GamesPages.module.css";

export function CheckersGamePage() {
    const { id } = useParams<{ id: string }>();
    const { user } = useAuth();
    const navigate = useNavigate();
    const { room, loading, error, refetch } = useGameRoom(id);

    usePageTitle(room ? `Checkers - ${room.players.map(p => p.display_name).join(" vs ")}` : "Checkers");

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
                    <h2 className={styles.heading}>Checkers</h2>
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
                <h2 className={styles.heading}>Checkers</h2>
                {isInvitee ? (
                    <p>
                        {opponent?.display_name ?? "Someone"} has invited you to a checkers game. Accept to start - you
                        will play as black.
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
                                    await api.acceptGameInvite(room.id);
                                    await refetch();
                                }}
                            >
                                Accept
                            </Button>
                            <Button
                                variant="ghost"
                                onClick={async () => {
                                    await api.declineGameInvite(room.id);
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
            </div>
        );
    }

    async function handleMove(move: { from: string; path: string[] }) {
        await api.submitGameAction(room!.id, {
            from: move.from,
            path: move.path,
        });
    }

    async function handleResign() {
        await api.resignGame(room!.id);
    }

    return (
        <div className={`${styles.page} ${styles.gamePage}`}>
            <div className={styles.boardColumn}>
                <CheckersBoardView
                    room={room}
                    viewer={user}
                    isSpectator={!isParticipant}
                    onMove={handleMove}
                    onResign={handleResign}
                />
            </div>
            <div className={styles.chatColumn}>
                {isParticipant ? (
                    <PlayerChat roomId={room.id} />
                ) : (
                    <SpectatorChat roomId={room.id} watcherCount={room.watcher_count} />
                )}
            </div>
        </div>
    );
}
