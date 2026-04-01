import { useCallback, useEffect, useState } from "react";
import type { FollowStats } from "../types/api";
import { followUser, getFollowStats, unfollowUser } from "../api/endpoints";

export function useFollow(userId: string) {
    const [stats, setStats] = useState<FollowStats | null>(null);
    const [loading, setLoading] = useState(!!userId);

    useEffect(() => {
        if (!userId) {
            return;
        }
        let cancelled = false;
        getFollowStats(userId)
            .then(data => {
                if (!cancelled) {
                    setStats(data);
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

    const toggleFollow = useCallback(async () => {
        if (!stats) {
            return;
        }
        const wasFollowing = stats.is_following;
        setStats({
            ...stats,
            is_following: !wasFollowing,
            follower_count: stats.follower_count + (wasFollowing ? -1 : 1),
        });
        try {
            if (wasFollowing) {
                await unfollowUser(userId);
            } else {
                await followUser(userId);
            }
            const updated = await getFollowStats(userId);
            setStats(updated);
        } catch {
            setStats(stats);
        }
    }, [stats, userId]);

    return { stats, loading, toggleFollow };
}
