import { useCallback } from "react";
import { useBlockStatus } from "../api/queries/misc";
import { useBlockUser, useUnblockUser } from "../api/mutations/misc";

export function useBlock(userId: string) {
    const { status, loading, refresh } = useBlockStatus(userId);
    const blockMutation = useBlockUser();
    const unblockMutation = useUnblockUser();

    const toggleBlock = useCallback(async () => {
        if (!status || loading || !userId) {
            return;
        }
        try {
            if (status.blocking) {
                await unblockMutation.mutateAsync(userId);
            } else {
                await blockMutation.mutateAsync(userId);
            }
            await refresh();
        } catch {
            return;
        }
    }, [status, loading, userId, blockMutation, unblockMutation, refresh]);

    return { status, loading, toggleBlock };
}
