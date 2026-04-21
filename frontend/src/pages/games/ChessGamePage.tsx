import { useNavigate, useParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useGameRoom } from "../../hooks/useGameRoom";
import * as api from "../../api/endpoints";
import { ChessBoardView } from "../../components/chess/ChessBoardView";
import { SpectatorChat } from "../../components/chess/SpectatorChat";
import { Button } from "../../components/Button/Button";
import styles from "./GamesPages.module.css";

export function ChessGamePage() {
    const { id } = useParams<{ id: string }>();
    const { user } = useAuth();
    const navigate = useNavigate();
    const { room, loading, error, refetch } = useGameRoom(id);

    usePageTitle(room ? `Chess - ${room.players.map(p => p.display_name).join(" vs ")}` : "Chess");

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
                    <h2 className={styles.heading}>Chess</h2>
                    <p>This match hasn't started yet — invites are private.</p>
                    <div className={styles.actions}>
                        <Button onClick={() => navigate("/games/live")}>Live Games</Button>
                    </div>
                </div>
            );
        }
        const opponent = room.players.find(p => p.user_id !== user?.id);
        return (
            <div className={styles.page}>
                <h2 className={styles.heading}>Chess</h2>
                {isInvitee ? (
                    <p>
                        {opponent?.display_name ?? "Someone"} has invited you to a chess game. Accept to start — you
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

    async function handleMove(move: { from: string; to: string; promotion?: string }) {
        await api.submitGameAction(room!.id, {
            from: move.from,
            to: move.to,
            promotion: move.promotion ?? "",
        });
    }

    async function handleResign() {
        await api.resignGame(room!.id);
    }

    return (
        <div className={`${styles.page} ${styles.gamePage}`}>
            <div className={styles.boardColumn}>
                <ChessBoardView
                    room={room}
                    viewer={user}
                    isSpectator={!isParticipant}
                    onMove={handleMove}
                    onResign={handleResign}
                />
            </div>
            {!isParticipant && (
                <div className={styles.chatColumn}>
                    <SpectatorChat roomId={room.id} watcherCount={room.watcher_count} />
                </div>
            )}
        </div>
    );
}
