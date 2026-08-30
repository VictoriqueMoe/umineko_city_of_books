import { useEffect, useRef, useState } from "react";
import { useLocation } from "react-router";
import type { WatchPartySession } from "../../types/api";

const OPEN_FAILED = "Could not open that watch party.";

export interface RoomWatchPartyInviteOptions {
    roomReady: boolean;
    loaded: boolean;
    sessions: readonly WatchPartySession[];
    join: (sessionId: string) => Promise<void>;
    onError: (message: string) => void;
}

export interface RoomWatchPartyInvite {
    invitedPartyMissing: boolean;
}

export function useRoomWatchPartyInvite(options: RoomWatchPartyInviteOptions): RoomWatchPartyInvite {
    const { roomReady, loaded, sessions, join, onError } = options;
    const location = useLocation();

    const invitedPartyId = new URLSearchParams(location.search).get("party");
    const handledPartyRef = useRef<string | null>(null);
    const [invitedPartyOpened, setInvitedPartyOpened] = useState(false);

    const resolved = !!invitedPartyId && roomReady && loaded;
    const invitedPartyMissing = resolved && !invitedPartyOpened && !sessions.some(s => s.id === invitedPartyId);

    useEffect(() => {
        if (!resolved || invitedPartyMissing || !invitedPartyId) {
            return;
        }
        if (handledPartyRef.current === invitedPartyId) {
            return;
        }

        handledPartyRef.current = invitedPartyId;

        join(invitedPartyId)
            .then(() => {
                setInvitedPartyOpened(true);
            })
            .catch(() => {
                onError(OPEN_FAILED);
            });
    }, [invitedPartyId, resolved, invitedPartyMissing, join, onError]);

    return { invitedPartyMissing };
}
