import { useCallback, useEffect, useState } from "react";
import { getSidebarActivity } from "../api/endpoints";
import { parseServerDate } from "../utils/time";
import { useAuth } from "./useAuth";

const POLL_INTERVAL_MS = 30_000;
const STORAGE_PREFIX = "sidebarLastVisited";

function storageKey(userId: string): string {
    return `${STORAGE_PREFIX}:${userId}`;
}

function readLastVisited(userId: string): Record<string, string> {
    try {
        const raw = window.localStorage.getItem(storageKey(userId));
        if (!raw) {
            return {};
        }
        const parsed = JSON.parse(raw);
        if (parsed && typeof parsed === "object") {
            return parsed as Record<string, string>;
        }
        return {};
    } catch {
        return {};
    }
}

function writeLastVisited(userId: string, map: Record<string, string>): void {
    try {
        window.localStorage.setItem(storageKey(userId), JSON.stringify(map));
    } catch {
        /* quota or disabled storage, silently ignore */
    }
}

interface VisitedState {
    userId: string | null;
    lastVisited: Record<string, string>;
}

export function useSidebarBadges() {
    const { user } = useAuth();
    const [latestActivity, setLatestActivity] = useState<Record<string, string>>({});
    const userId = user?.id ?? null;
    const [visitedState, setVisitedState] = useState<VisitedState>(() => ({
        userId,
        lastVisited: userId ? readLastVisited(userId) : {},
    }));
    if (visitedState.userId !== userId) {
        setVisitedState({
            userId,
            lastVisited: userId ? readLastVisited(userId) : {},
        });
    }

    useEffect(() => {
        if (!user) {
            return;
        }
        let cancelled = false;
        const load = async () => {
            try {
                const resp = await getSidebarActivity();
                if (!cancelled) {
                    setLatestActivity(resp.activity ?? {});
                }
            } catch {
                /* silent */
            }
        };
        load();
        const timer = window.setInterval(load, POLL_INTERVAL_MS);
        return () => {
            cancelled = true;
            window.clearInterval(timer);
        };
    }, [user]);

    const lastVisited = visitedState.lastVisited;

    const hasUnread = useCallback(
        (key: string): boolean => {
            if (!user) {
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
        [user, latestActivity, lastVisited],
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
            const current = readLastVisited(userId);
            const next = { ...current, [key]: now };
            writeLastVisited(userId, next);
            setVisitedState({ userId, lastVisited: next });
        },
        [userId],
    );

    return { hasUnread, hasAnyUnread, markVisited };
}
