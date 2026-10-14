import { REALTIME_COMMANDS, sendRealtime } from "./outbound";
import { subscribeRealtimeEpoch } from "./pipeline";

export const POST_WATCH_DEBOUNCE_MS = 200;
export const MAX_WATCHED_POSTS = 200;

const watchCounts = new Map<string, number>();

let flushTimer: ReturnType<typeof setTimeout> | null = null;
let lastSentKey = "";
let reconnectBound = false;

function cancelFlush(): void {
    if (flushTimer === null) {
        return;
    }

    clearTimeout(flushTimer);
    flushTimer = null;
}

function flushWatchedPosts(): void {
    cancelFlush();

    const postIds = Array.from(watchCounts.keys()).slice(-MAX_WATCHED_POSTS);
    const key = postIds.join(",");

    if (key === lastSentKey) {
        return;
    }

    lastSentKey = key;
    sendRealtime({ type: REALTIME_COMMANDS.WATCH_POSTS, data: { post_ids: postIds } });
}

function scheduleFlush(): void {
    cancelFlush();
    flushTimer = setTimeout(flushWatchedPosts, POST_WATCH_DEBOUNCE_MS);
}

function resendAfterReconnect(): void {
    lastSentKey = "";
    flushWatchedPosts();
}

function bindReconnect(): void {
    if (reconnectBound) {
        return;
    }

    reconnectBound = true;
    subscribeRealtimeEpoch(resendAfterReconnect);
}

export function watchPost(postId: string): () => void {
    bindReconnect();

    const count = watchCounts.get(postId) ?? 0;
    watchCounts.set(postId, count + 1);

    if (count === 0) {
        scheduleFlush();
    }

    let released = false;

    return () => {
        if (released) {
            return;
        }
        released = true;

        const remaining = (watchCounts.get(postId) ?? 1) - 1;
        if (remaining > 0) {
            watchCounts.set(postId, remaining);
            return;
        }

        watchCounts.delete(postId);
        scheduleFlush();
    };
}
