import { Button } from "../../Button/Button";
import type { VoiceStatus } from "./useVoiceChat";

interface VoiceButtonProps {
    enabled: boolean;
    status: VoiceStatus;
    presenceCount: number;
    onJoin: () => void;
    onLeave: () => void;
}

export function VoiceButton({ enabled, status, presenceCount, onJoin, onLeave }: VoiceButtonProps) {
    if (!enabled) {
        return null;
    }

    if (status === "connected") {
        return (
            <Button variant="ghost" size="small" onClick={onLeave} title="Leave voice">
                {"\u{1F50A} Leave voice"}
            </Button>
        );
    }

    const label = presenceCount > 0 ? `\u{1F399} Voice · ${presenceCount}` : "\u{1F399} Voice";

    return (
        <Button variant="ghost" size="small" onClick={onJoin} disabled={status === "connecting"} title="Join voice">
            {status === "connecting" ? "Joining…" : label}
        </Button>
    );
}
