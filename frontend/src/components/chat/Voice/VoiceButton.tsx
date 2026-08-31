import { useState } from "react";
import { Button } from "../../Button/Button";
import { ConfirmDialog } from "../../ConfirmDialog/ConfirmDialog";
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
    const [confirming, setConfirming] = useState(false);

    if (!enabled) {
        return null;
    }

    const label = presenceCount > 0 ? `\u{1F399} Voice · ${presenceCount}` : "\u{1F399} Voice";
    const alreadyTalking =
        presenceCount === 1 ? "One person is already in the call." : `${presenceCount} people are already in the call.`;

    function confirmJoin() {
        setConfirming(false);
        onJoin();
    }

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
                    onClick={() => setConfirming(true)}
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
            <ConfirmDialog
                open={confirming}
                title="Join the voice call?"
                body={
                    <>
                        <p>Your microphone will be switched on and the others will hear you.</p>
                        {presenceCount > 0 && <p>{alreadyTalking}</p>}
                    </>
                }
                confirmLabel="Join voice"
                onConfirm={confirmJoin}
                onCancel={() => setConfirming(false)}
            />
        </>
    );
}
