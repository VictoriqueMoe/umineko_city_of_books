import { useCallback } from "react";
import { useFollowStats } from "../api/queries/misc";
import { useFollowUser, useUnfollowUser } from "../api/mutations/misc";

export function useFollow(userId: string) {
    const { stats, loading, refresh } = useFollowStats(userId);
    const followMutation = useFollowUser();
    const unfollowMutation = useUnfollowUser();

    const toggleFollow = useCallback(async () => {
        if (!stats || !userId) {
            return;
        }
        try {
            if (stats.is_following) {
                await unfollowMutation.mutateAsync(userId);
            } else {
                await followMutation.mutateAsync(userId);
            }
            await refresh();
        } catch {
            return;
        }
    }, [stats, userId, followMutation, unfollowMutation, refresh]);

    return { stats, loading, toggleFollow };
}
