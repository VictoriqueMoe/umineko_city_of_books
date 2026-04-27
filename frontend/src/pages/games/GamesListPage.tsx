import { useEffect } from "react";
import { Link, useNavigate } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useMyGameRooms } from "../../api/queries/gameRoom";
import { useDeclineGameInvite, useCancelGameInvite } from "../../api/mutations/gameRoom";
import type { GameRoom } from "../../types/api";
import { GAME_TYPES, gameTypeLabel } from "../../games/registry";
import { Button } from "../../components/Button/Button";
import { InfoPanel } from "../../components/InfoPanel/InfoPanel";
import { formatFullDateTime } from "../../utils/time";
import styles from "./GamesPages.module.css";

function statusBadgeClass(status: string): string {
    if (status === "pending") {
        return styles.statusPending;
    }
    if (status === "active") {
        return styles.statusActive;
    }
    return styles.statusFinished;
}

export function GamesListPage() {
    usePageTitle("Games");
    const { user, loading: authLoading } = useAuth();
    const navigate = useNavigate();
    const { rooms, loading, error, refresh } = useMyGameRooms();
    const declineInvite = useDeclineGameInvite();
    const cancelInvite = useCancelGameInvite();

    useEffect(() => {
        if (!authLoading && !user) {
            navigate("/login");
        }
    }, [user, authLoading, navigate]);

    if (authLoading || !user) {
        return null;
    }

    const pendingIncoming = rooms.filter(r => r.status === "pending" && r.created_by !== user.id);
    const pendingOutgoing = rooms.filter(r => r.status === "pending" && r.created_by === user.id);
    const active = rooms.filter(r => r.status === "active");
    const finished = rooms.filter(r => r.status === "finished" || r.status === "declined" || r.status === "abandoned");

    function handleOpenInvite(room: GameRoom) {
        navigate(`/games/${room.game_type}/${room.id}`);
    }

    async function handleDecline(room: GameRoom) {
        try {
            await declineInvite.mutateAsync(room.id);
            await refresh();
        } catch {
            // ignore
        }
    }

    async function handleCancel(room: GameRoom) {
        if (!window.confirm("Cancel this invite?")) {
            return;
        }
        try {
            await cancelInvite.mutateAsync(room.id);
            await refresh();
        } catch {
            // ignore
        }
    }

    return (
        <div className={styles.page}>
            <h2 className={styles.heading}>My Games</h2>

            <InfoPanel title="Games are in beta">
                <p>
                    The Games feature is brand new and still being iterated on. Rules, UI and the data layout for saved
                    games may change. If something misbehaves or feels off, please report it on the suggestions page or
                    directly to staff.
                </p>
                <p>
                    To <strong>start a game</strong>, open the hub for the game you want to play and hit{" "}
                    <em>Start a new game</em>. You invite another player by username or from your mutual followers, and
                    the match begins as soon as they accept. Each game hub has its own <em>How to play</em> panel with
                    the rules specific to that game.
                </p>
            </InfoPanel>

            <div className={styles.actions}>
                <Link to="/games/live">
                    <Button variant="secondary">Live Games</Button>
                </Link>
            </div>

            <h3 className={styles.sectionTitle}>Games you can play</h3>
            <div className={styles.tileRow}>
                {GAME_TYPES.filter(g => g.available).map(g => (
                    <Link key={g.type} to={g.hubPath} className={styles.tile}>
                        <span className={styles.tileTitle}>{g.label}</span>
                        <span className={styles.tileTagline}>{g.tagline}</span>
                    </Link>
                ))}
            </div>

            {error && <div className={styles.error}>{error}</div>}

            <h3 className={styles.sectionTitle}>Invites for you</h3>
            {loading ? (
                <p className={styles.empty}>Loading...</p>
            ) : pendingIncoming.length === 0 ? (
                <p className={styles.empty}>No pending invites.</p>
            ) : (
                <div className={styles.gameList}>
                    {pendingIncoming.map(r => {
                        const opponent = r.players.find(p => p.user_id !== user.id);
                        return (
                            <div key={r.id} className={styles.gameRow}>
                                <div className={styles.gameRowContent}>
                                    <span className={styles.opponentLine}>
                                        {opponent?.display_name ?? "Unknown"} invited you to{" "}
                                        {gameTypeLabel(r.game_type)}
                                    </span>
                                    <span className={styles.subline}>{formatFullDateTime(r.created_at)}</span>
                                </div>
                                <div className={styles.inviteActions}>
                                    <Button variant="primary" size="small" onClick={() => handleOpenInvite(r)}>
                                        View and accept
                                    </Button>
                                    <Button variant="ghost" size="small" onClick={() => handleDecline(r)}>
                                        Decline
                                    </Button>
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}

            <h3 className={styles.sectionTitle}>Waiting on opponent</h3>
            {pendingOutgoing.length === 0 ? (
                <p className={styles.empty}>None.</p>
            ) : (
                <div className={styles.gameList}>
                    {pendingOutgoing.map(r => {
                        const opponent = r.players.find(p => p.user_id !== user.id);
                        return (
                            <div key={r.id} className={styles.gameRow}>
                                <div className={styles.gameRowContent}>
                                    <span className={styles.opponentLine}>
                                        {gameTypeLabel(r.game_type)} vs {opponent?.display_name ?? "Unknown"}
                                    </span>
                                    <span className={styles.subline}>Invited {formatFullDateTime(r.created_at)}</span>
                                </div>
                                <div className={styles.inviteActions}>
                                    <Button variant="ghost" size="small" onClick={() => handleCancel(r)}>
                                        Cancel
                                    </Button>
                                </div>
                            </div>
                        );
                    })}
                </div>
            )}

            <h3 className={styles.sectionTitle}>Active games</h3>
            {active.length === 0 ? (
                <p className={styles.empty}>None in progress.</p>
            ) : (
                <div className={styles.gameList}>
                    {active.map(r => {
                        const opponent = r.players.find(p => p.user_id !== user.id);
                        const yourTurn = r.turn_user_id === user.id;
                        return (
                            <Link key={r.id} to={`/games/${r.game_type}/${r.id}`} className={styles.gameRow}>
                                <div className={styles.gameRowContent}>
                                    <span className={styles.opponentLine}>
                                        {gameTypeLabel(r.game_type)} vs {opponent?.display_name ?? "Unknown"}
                                    </span>
                                    <span className={styles.subline}>
                                        {yourTurn ? "Your turn" : "Their turn"} — updated{" "}
                                        {formatFullDateTime(r.updated_at)}
                                    </span>
                                </div>
                                <span className={`${styles.statusBadge} ${statusBadgeClass(r.status)}`}>
                                    {yourTurn ? "your turn" : r.status}
                                </span>
                            </Link>
                        );
                    })}
                </div>
            )}

            <h3 className={styles.sectionTitle}>Past games</h3>
            {finished.length === 0 ? (
                <p className={styles.empty}>None yet.</p>
            ) : (
                <div className={styles.gameList}>
                    {finished.map(r => {
                        const opponent = r.players.find(p => p.user_id !== user.id);
                        let outcome: string;
                        if (r.status === "finished") {
                            if (!r.winner_user_id) {
                                outcome = "Draw";
                            } else if (r.winner_user_id === user.id) {
                                outcome = "Won";
                            } else {
                                outcome = "Lost";
                            }
                        } else {
                            outcome = r.status;
                        }
                        return (
                            <Link key={r.id} to={`/games/${r.game_type}/${r.id}`} className={styles.gameRow}>
                                <div className={styles.gameRowContent}>
                                    <span className={styles.opponentLine}>
                                        {gameTypeLabel(r.game_type)} vs {opponent?.display_name ?? "Unknown"}
                                    </span>
                                    <span className={styles.subline}>
                                        {outcome} — {formatFullDateTime(r.finished_at ?? r.updated_at)}
                                    </span>
                                </div>
                                <span className={`${styles.statusBadge} ${statusBadgeClass(r.status)}`}>
                                    {r.status}
                                </span>
                            </Link>
                        );
                    })}
                </div>
            )}
        </div>
    );
}
