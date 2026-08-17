import { useState, type ComponentType, type ReactNode } from "react";
import { useNavigate, useParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useGameRoom } from "../../api/queries/gameRoom";
import { useAcceptGameInvite, useDeclineGameInvite } from "../../api/mutations/gameRoom";
import { GameChat } from "../../components/games/chat/GameChat";
import { Button } from "../../components/Button/Button";
import type { GameRoom, User } from "../../types/api";
import styles from "./GamesPages.module.css";

export interface GameBoardProps {
    room: GameRoom;
    viewer: User | null;
    isSpectator: boolean;
}

interface GameRoomShellProps {
    gameName: string;
    inviteCopy: (opponentName: string) => ReactNode;
    Board: ComponentType<GameBoardProps>;
}

export function GameRoomShell({ gameName, inviteCopy, Board }: GameRoomShellProps) {
    const { id } = useParams<{ id: string }>();
    const { user } = useAuth();
    const navigate = useNavigate();
    const { room, loading, error, refetch } = useGameRoom(id);
    const [acceptError, setAcceptError] = useState("");
    const acceptInvite = useAcceptGameInvite();
    const declineInvite = useDeclineGameInvite();

    usePageTitle(room ? `${gameName} - ${room.players.map(p => p.display_name).join(" vs ")}` : gameName);

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
                    <h2 className={styles.heading}>{gameName}</h2>
                    <p>This match hasn't started yet - invites are private.</p>
                    <div className={styles.actions}>
                        <Button onClick={() => navigate("/games/live")}>Live Games</Button>
                    </div>
                </div>
            );
        }

        const opponent = room.players.find(p => p.user_id !== user?.id);

        const handleAccept = async () => {
            setAcceptError("");
            try {
                await acceptInvite.mutateAsync(room.id);
                await refetch();
            } catch (err) {
                setAcceptError(err instanceof Error ? err.message : "Failed to accept invite");
            }
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
                    <p>Waiting for {opponent?.display_name ?? "opponent"} to accept.</p>
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
