import {useCallback, useEffect, useState} from "react";
import type {FollowStats} from "../types/api";
import {followUser, getFollowStats, unfollowUser} from "../api/endpoints";

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
        if (stats.is_following) {
            await unfollowUser(userId);
        } else {
            await followUser(userId);
        }
        const updated = await getFollowStats(userId);
        setStats(updated);
    }, [stats, userId]);

    return { stats, loading, toggleFollow };
}
