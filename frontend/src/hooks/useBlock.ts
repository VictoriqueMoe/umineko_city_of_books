import { useCallback, useEffect, useState } from "react";
import { blockUser, getBlockStatus, unblockUser, type BlockStatus } from "../api/endpoints";

export function useBlock(userId: string) {
    const [status, setStatus] = useState<BlockStatus | null>(null);
    const [loading, setLoading] = useState(!!userId);

    useEffect(() => {
        if (!userId) {
            return;
        }
        let cancelled = false;
        getBlockStatus(userId)
            .then(data => {
                if (!cancelled) {
                    setStatus(data);
                }
            })
            .catch(() => {})
            .finally(() => {
                if (!cancelled) {
                    setLoading(false);
                }
            });
        return () => {
            cancelled = true;
        };
    }, [userId]);

    const toggleBlock = useCallback(async () => {
        if (!status) {
            return;
        }
        const wasBlocking = status.blocking;
        setStatus({ ...status, blocking: !wasBlocking });
        try {
            if (wasBlocking) {
                await unblockUser(userId);
            } else {
                await blockUser(userId);
            }
            const updated = await getBlockStatus(userId);
            setStatus(updated);
        } catch {
            setStatus(status);
        }
    }, [status, userId]);

    return { status, loading, toggleBlock };
}
