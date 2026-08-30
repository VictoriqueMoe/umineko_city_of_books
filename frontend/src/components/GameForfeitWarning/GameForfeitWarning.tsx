import { useEffect, useState } from "react";
import { Link } from "react-router";
import { useGameForfeitWarning } from "../../hooks/useGameForfeitWarning";
import { Toast } from "../Toast/Toast";

const WARNING_WINDOW_SECONDS = 10;

export function GameForfeitWarning() {
    const { pending, finalForfeit, dismissFinalForfeit } = useGameForfeitWarning();
    const [now, setNow] = useState(() => Date.now());

    useEffect(() => {
        if (!pending) {
            return;
        }
        const id = window.setInterval(() => setNow(Date.now()), 1000);
        return () => window.clearInterval(id);
    }, [pending]);

    const remainingSeconds = pending ? Math.max(0, Math.ceil((pending.forfeitAt - now) / 1000)) : null;
    const showCountdownToast =
        pending !== null &&
        remainingSeconds !== null &&
        remainingSeconds > 0 &&
        remainingSeconds <= WARNING_WINDOW_SECONDS;

    if (finalForfeit) {
        return (
            <Toast variant="error" duration={10000} onDismiss={dismissFinalForfeit}>
                You forfeited the {finalForfeit.gameType} game by disconnecting.{" "}
                <Link to={`/games/${finalForfeit.gameType}/${finalForfeit.roomId}`} style={{ color: "inherit" }}>
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
