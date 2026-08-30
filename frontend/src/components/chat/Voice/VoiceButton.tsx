import { Button } from "../../Button/Button";
import type { VoiceStatus } from "../../../hooks/useVoiceChat";
import styles from "./Voice.module.css";

interface VoiceButtonProps {
    enabled: boolean;
    status: VoiceStatus;
    presenceCount: number;
    error?: string | null;
    onJoin: () => void;
    onLeave: () => void;
}

export function VoiceButton({ enabled, status, presenceCount, error, onJoin, onLeave }: VoiceButtonProps) {
    if (!enabled) {
        return null;
    }

    const label = presenceCount > 0 ? `\u{1F399} Voice · ${presenceCount}` : "\u{1F399} Voice";

    return (
        <>
            {status === "connected" ? (
                <Button variant="ghost" size="small" onClick={onLeave} title="Leave voice">
                    {"\u{1F50A} Leave voice"}
                </Button>
            ) : (
                <Button
                    variant="ghost"
                    size="small"
                    onClick={onJoin}
                    disabled={status === "connecting"}
                    title="Join voice"
                >
                    {status === "connecting" ? "Joining…" : label}
                </Button>
            )}
            {error && (
                <span className={styles.joinError} role="alert" title={error}>
                    {error}
                </span>
            )}
        </>
    );
}
