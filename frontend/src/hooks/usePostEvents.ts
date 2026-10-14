import { useEffect } from "react";
import { REALTIME_EVENTS } from "../api/realtime/events";
import { watchPost } from "../api/realtime/postWatch";
import { useRealtimeEvent } from "../api/realtime/useRealtime";

function useWatchedPost(postId: string): void {
    useEffect(() => {
        if (!postId) {
            return;
        }

        return watchPost(postId);
    }, [postId]);
}

export function usePostLikeEvents(postId: string, onLike: (delta: number) => void): void {
    useWatchedPost(postId);

    useRealtimeEvent(REALTIME_EVENTS.POST_LIKE, event => {
        if (event.data.post_id !== postId) {
            return;
        }

        onLike(event.data.delta);
    });
}

export function usePostCommentEvents(postId: string, onComment: () => void): void {
    useWatchedPost(postId);

    useRealtimeEvent(REALTIME_EVENTS.POST_COMMENT, event => {
        if (event.data.post_id !== postId) {
            return;
        }

        onComment();
    });
}
