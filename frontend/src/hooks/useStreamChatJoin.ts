import { useEffect, useState } from "react";
import { useJoinStreamChat } from "./mutations/stream";
import { useAuth } from "./useAuth";

export interface UseStreamChatJoinResult {
    joined: boolean;
    failed: boolean;
}

export function useStreamChatJoin(streamId: string, isLive: boolean): UseStreamChatJoinResult {
    const { user } = useAuth();
    const { mutateAsync } = useJoinStreamChat();

    const [joinedStreamId, setJoinedStreamId] = useState<string | null>(null);
    const [failedStreamId, setFailedStreamId] = useState<string | null>(null);

    const canJoin = isLive && !!user;

    useEffect(() => {
        if (!canJoin) {
            return;
        }

        let cancelled = false;

        mutateAsync(streamId)
            .then(() => {
                if (cancelled) {
                    return;
                }

                setJoinedStreamId(streamId);
            })
            .catch(() => {
                if (cancelled) {
                    return;
                }

                setFailedStreamId(streamId);
            });

        return () => {
            cancelled = true;
        };
    }, [streamId, canJoin, mutateAsync]);

    return {
        joined: joinedStreamId === streamId,
        failed: failedStreamId === streamId,
    };
}
