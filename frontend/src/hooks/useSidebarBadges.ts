import { useCallback, useEffect, useMemo, useState } from "react";
import { useSidebarActivity, useSidebarLastVisited } from "../api/queries/sidebar";
import { useMarkSidebarVisited } from "../api/mutations/sidebar";
import type { WSMessage } from "../types/api";
import { parseServerDate } from "../utils/time";
import { useAuth } from "./useAuth";
import { useNotifications } from "./useNotifications";

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

function clearLegacyVisited(userId: string): void {
    try {
        window.localStorage.removeItem(legacyStorageKey(userId));
    } catch {
        return;
    }
}

export function useSidebarBadges() {
    const { user } = useAuth();
    const { addWSListener } = useNotifications();
    const userId = user?.id ?? null;

    const { data: activityResp } = useSidebarActivity();
    const { data: visitedResp, refresh: refreshVisited } = useSidebarLastVisited();
    const markVisitedMutation = useMarkSidebarVisited();

    const [activityOverlay, setActivityOverlay] = useState<Record<string, string>>({});

    const latestActivity = useMemo<Record<string, string>>(
        () => ({
            ...(activityResp?.activity ?? {}),
            ...activityOverlay,
        }),
        [activityResp, activityOverlay],
    );
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
            const results = await Promise.allSettled(keys.map(key => markVisitedMutation.mutateAsync(key)));
            if (cancelled) {
                return;
            }
            for (const r of results) {
                if (r.status === "rejected") {
                    return;
                }
            }
            clearLegacyVisited(userId);
            void refreshVisited();
        };
        void run();
        return () => {
            cancelled = true;
        };
    }, [userId, markVisitedMutation, refreshVisited]);

    useEffect(() => {
        if (!userId) {
            return;
        }
        return addWSListener((msg: WSMessage) => {
            if (msg.type !== "sidebar_activity") {
                return;
            }
            const data = msg.data as { key?: string; at?: string };
            if (!data.key || !data.at) {
                return;
            }
            const key = data.key;
            const at = data.at;
            setActivityOverlay(prev => {
                const existing = prev[key] ?? activityResp?.activity?.[key];
                if (existing && existing >= at) {
                    return prev;
                }
                return { ...prev, [key]: at };
            });
        });
    }, [userId, addWSListener, activityResp]);

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
            markVisitedMutation.mutate(key);
        },
        [userId, hasUnread, markVisitedMutation],
    );

    return { hasUnread, hasAnyUnread, markVisited };
}
