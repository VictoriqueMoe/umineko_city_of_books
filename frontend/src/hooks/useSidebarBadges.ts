import { useCallback, useEffect, useMemo, useState } from "react";
import { REALTIME_EVENTS } from "../api/realtime/events";
import { useRealtimeEvent } from "../api/realtime/useRealtime";
import { useSidebarActivity, useSidebarLastVisited } from "./queries/sidebar";
import { useMarkSidebarVisited } from "./mutations/sidebar";
import { parseServerDate } from "../utils/time";
import { useAuth } from "./useAuth";

const LEGACY_STORAGE_PREFIX = "sidebarLastVisited";

function legacyStorageKey(userId: string): string {
    return `${LEGACY_STORAGE_PREFIX}:${userId}`;
}

function readLegacyVisited(userId: string): Record<string, string> | null {
    try {
        const raw = window.localStorage.getItem(legacyStorageKey(userId));
        if (!raw) {
            return null;
        }
        const parsed = JSON.parse(raw);
        if (parsed && typeof parsed === "object") {
            return parsed as Record<string, string>;
        }
        return null;
    } catch {
        return null;
    }
}

function isNewer(candidate: string, existing: string): boolean {
    const candidateDate = parseServerDate(candidate);
    if (!candidateDate) {
        return false;
    }

    const existingDate = parseServerDate(existing);
    if (!existingDate) {
        return true;
    }

    return candidateDate.getTime() > existingDate.getTime();
}

function clearLegacyVisited(userId: string): void {
    try {
        window.localStorage.removeItem(legacyStorageKey(userId));
    } catch {
        return;
    }
}

export function useSidebarBadges() {
    const { user } = useAuth();
    const userId = user?.id ?? null;

    const { data: activityResp } = useSidebarActivity();
    const { data: visitedResp, refresh: refreshVisited } = useSidebarLastVisited();
    const { mutate: markVisitedMutate, mutateAsync: markVisitedAsync } = useMarkSidebarVisited();

    const [activityOverlay, setActivityOverlay] = useState<Record<string, string>>({});

    const latestActivity = useMemo<Record<string, string>>(() => {
        const merged: Record<string, string> = { ...(activityResp?.activity ?? {}) };

        for (const [key, at] of Object.entries(activityOverlay)) {
            const serverAt = merged[key];
            if (!serverAt || isNewer(at, serverAt)) {
                merged[key] = at;
            }
        }

        return merged;
    }, [activityResp, activityOverlay]);
    const lastVisited = useMemo<Record<string, string>>(() => visitedResp?.visited ?? {}, [visitedResp]);

    useEffect(() => {
        if (!userId) {
            return;
        }
        const legacy = readLegacyVisited(userId);
        if (!legacy) {
            return;
        }
        const keys = Object.keys(legacy);
        if (keys.length === 0) {
            clearLegacyVisited(userId);
            return;
        }
        let cancelled = false;
        const run = async () => {
            const results = await Promise.allSettled(keys.map(key => markVisitedAsync(key)));
            if (cancelled) {
                return;
            }
            for (const r of results) {
                if (r.status === "rejected") {
                    return;
                }
            }
            clearLegacyVisited(userId);
            refreshVisited();
        };
        run();
        return () => {
            cancelled = true;
        };
    }, [userId, markVisitedAsync, refreshVisited]);

    useRealtimeEvent(REALTIME_EVENTS.SIDEBAR_ACTIVITY, event => {
        if (!userId) {
            return;
        }

        const { key, at } = event.data;
        if (!key || !at) {
            return;
        }

        setActivityOverlay(prev => {
            const existing = prev[key] ?? activityResp?.activity?.[key];
            if (existing && !isNewer(at, existing)) {
                return prev;
            }
            return { ...prev, [key]: at };
        });
    });

    const hasUnread = useCallback(
        (key: string): boolean => {
            if (!userId) {
                return false;
            }
            const latest = latestActivity[key];
            if (!latest) {
                return false;
            }
            const latestDate = parseServerDate(latest);
            if (!latestDate) {
                return false;
            }
            const visited = lastVisited[key];
            if (!visited) {
                return true;
            }
            const visitedDate = parseServerDate(visited);
            if (!visitedDate) {
                return true;
            }
            return latestDate.getTime() > visitedDate.getTime();
        },
        [userId, latestActivity, lastVisited],
    );

    const hasAnyUnread = useCallback(
        (keys: string[]): boolean => {
            for (const key of keys) {
                if (hasUnread(key)) {
                    return true;
                }
            }
            return false;
        },
        [hasUnread],
    );

    const markVisited = useCallback(
        (key: string) => {
            if (!userId) {
                return;
            }
            if (!hasUnread(key)) {
                return;
            }
            markVisitedMutate(key);
        },
        [userId, hasUnread, markVisitedMutate],
    );

    const markAllVisited = useCallback(() => {
        if (!userId) {
            return;
        }
        const unreadKeys = Object.keys(latestActivity).filter(key => hasUnread(key));
        if (unreadKeys.length === 0) {
            return;
        }
        for (const key of unreadKeys) {
            markVisitedMutate(key);
        }
    }, [userId, latestActivity, hasUnread, markVisitedMutate]);

    const anyUnread = useMemo(
        () => Object.keys(latestActivity).some(key => hasUnread(key)),
        [latestActivity, hasUnread],
    );

    return { hasUnread, hasAnyUnread, markVisited, markAllVisited, anyUnread };
}
