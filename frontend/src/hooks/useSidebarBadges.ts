import { useCallback, useEffect, useState } from "react";
import { getSidebarActivity, getSidebarLastVisited, markSidebarVisited } from "../api/endpoints";
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
        /* quota or disabled storage, silently ignore */
    }
}

async function migrateLegacyVisited(userId: string): Promise<void> {
    const legacy = readLegacyVisited(userId);
    if (!legacy) {
        return;
    }
    const keys = Object.keys(legacy);
    if (keys.length === 0) {
        clearLegacyVisited(userId);
        return;
    }
    const results = await Promise.allSettled(keys.map(key => markSidebarVisited(key)));
    for (let i = 0; i < results.length; i++) {
        if (results[i].status === "rejected") {
            return;
        }
    }
    clearLegacyVisited(userId);
}

export function useSidebarBadges() {
    const { user } = useAuth();
    const { addWSListener, wsEpoch } = useNotifications();
    const userId = user?.id ?? null;
    const [latestActivity, setLatestActivity] = useState<Record<string, string>>({});
    const [lastVisited, setLastVisited] = useState<Record<string, string>>({});

    useEffect(() => {
        if (!userId) {
            return;
        }
        let cancelled = false;

        const run = async () => {
            try {
                await migrateLegacyVisited(userId);
            } catch {
                /* silent; next mount retries */
            }
            if (cancelled) {
                return;
            }
            try {
                const [activityResp, visitedResp] = await Promise.all([getSidebarActivity(), getSidebarLastVisited()]);
                if (cancelled) {
                    return;
                }
                setLatestActivity(activityResp.activity ?? {});
                setLastVisited(visitedResp.visited ?? {});
            } catch {
                /* silent */
            }
        };

        void run();
        return () => {
            cancelled = true;
        };
    }, [userId, wsEpoch]);

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
            setLatestActivity(prev => {
                const existing = prev[key];
                if (existing && existing >= at) {
                    return prev;
                }
                return { ...prev, [key]: at };
            });
        });
    }, [userId, addWSListener]);

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
            for (let i = 0; i < keys.length; i++) {
                if (hasUnread(keys[i])) {
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
            const now = new Date().toISOString();
            setLastVisited(prev => ({ ...prev, [key]: now }));
            markSidebarVisited(key).catch(() => {
                /* silent; next poll reconciles */
            });
        },
        [userId],
    );

    return { hasUnread, hasAnyUnread, markVisited };
}
