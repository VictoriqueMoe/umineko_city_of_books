import { useEffect, useState } from "react";
import { Link } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { useNotifications } from "../../hooks/useNotifications";
import type { WSMessage } from "../../types/api";
import { Toast } from "../Toast/Toast";

const WARNING_WINDOW_SECONDS = 10;

interface PendingForfeit {
    roomId: string;
    gameType: string;
    forfeitAt: number;
}

interface FinalToast {
    roomId: string;
    gameType: string;
}

export function GameForfeitWarning() {
    const { user } = useAuth();
    const { addWSListener } = useNotifications();
    const [pending, setPending] = useState<PendingForfeit | null>(null);
    const [now, setNow] = useState(() => Date.now());
    const [finalToast, setFinalToast] = useState<FinalToast | null>(null);

    useEffect(() => {
        if (!pending) {
            return;
        }
        const id = window.setInterval(() => setNow(Date.now()), 1000);
        return () => window.clearInterval(id);
    }, [pending]);

    useEffect(() => {
        return addWSListener((msg: WSMessage) => {
            if (msg.type === "game_forfeit_warning") {
                const data = msg.data as {
                    room_id?: string;
                    game_type?: string;
                    disconnected_at?: string;
                    grace_seconds?: number;
                };
                if (!data.room_id || !data.disconnected_at || typeof data.grace_seconds !== "number") {
                    return;
                }
                const startedAt = Date.parse(data.disconnected_at);
                if (Number.isNaN(startedAt)) {
                    return;
                }
                setPending({
                    roomId: data.room_id,
                    gameType: data.game_type ?? "chess",
                    forfeitAt: startedAt + data.grace_seconds * 1000,
                });
                return;
            }
            if (msg.type === "game_forfeit_cleared") {
                const data = msg.data as { room_id?: string };
                setPending(prev => (prev && prev.roomId === data.room_id ? null : prev));
                return;
            }
            if (msg.type === "game_room_finished") {
                const data = msg.data as { room_id?: string; abandoned_by?: string };
                if (!data.room_id || !data.abandoned_by || !user) {
                    return;
                }
                if (data.abandoned_by !== user.id) {
                    return;
                }
                setPending(prev => {
                    const gameType = prev && prev.roomId === data.room_id ? prev.gameType : "chess";
                    setFinalToast({ roomId: data.room_id!, gameType });
                    return null;
                });
            }
        });
    }, [addWSListener, user]);

    const remainingSeconds = pending ? Math.max(0, Math.ceil((pending.forfeitAt - now) / 1000)) : null;
    const showCountdownToast =
        pending !== null &&
        remainingSeconds !== null &&
        remainingSeconds > 0 &&
        remainingSeconds <= WARNING_WINDOW_SECONDS;

    if (finalToast) {
        return (
            <Toast variant="error" duration={10000} onDismiss={() => setFinalToast(null)}>
                You forfeited the {finalToast.gameType} game by disconnecting.{" "}
                <Link to={`/games/${finalToast.gameType}/${finalToast.roomId}`} style={{ color: "inherit" }}>
                    View game
                </Link>
            </Toast>
        );
    }

    if (showCountdownToast && pending) {
        return (
            <Toast variant="error" duration={0}>
                Forfeiting your {pending.gameType} game in {remainingSeconds}s.{" "}
                <Link to={`/games/${pending.gameType}/${pending.roomId}`} style={{ color: "inherit" }}>
                    Return to game
                </Link>
            </Toast>
        );
    }

    return null;
}
