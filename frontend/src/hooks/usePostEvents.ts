import { REALTIME_EVENTS } from "../api/realtime/events";
import { useRealtimeEvent } from "../api/realtime/useRealtime";

export function usePostLikeEvents(postId: string, onLike: (delta: number) => void): void {
    useRealtimeEvent(REALTIME_EVENTS.POST_LIKE, event => {
        if (event.data.post_id !== postId) {
            return;
        }

        onLike(event.data.delta);
    });
}

export function usePostCommentEvents(postId: string, onComment: () => void): void {
    useRealtimeEvent(REALTIME_EVENTS.POST_COMMENT, event => {
        if (event.data.post_id !== postId) {
            return;
        }

        onComment();
    });
}
