import { streamChatPopoutPath } from "../domain/live/streamUrl";

export const STREAM_CHAT_POPOUT_CLOSED = "stream-chat-popout-closed";

const POPOUT_FEATURES = "popup=yes,width=420,height=760,resizable=yes,scrollbars=yes";

export function streamChatPopoutName(streamId: string): string {
    return `stream-chat-${streamId}`;
}

export function openStreamChatPopout(username: string, streamId: string): Window | null {
    return window.open(streamChatPopoutPath(username), streamChatPopoutName(streamId), POPOUT_FEATURES);
}

export function streamChatPopoutOrigin(): string {
    return window.location.origin;
}

export function notifyStreamChatPopoutClosed(opener: Window | null, streamId: string): void {
    if (!opener || opener.closed) {
        return;
    }

    opener.postMessage({ type: STREAM_CHAT_POPOUT_CLOSED, streamId }, streamChatPopoutOrigin());
}

export function readStreamChatPopoutClosed(event: MessageEvent): string | null {
    if (event.origin !== streamChatPopoutOrigin()) {
        return null;
    }

    const data = event.data as { type?: unknown; streamId?: unknown } | null | undefined;
    if (!data || data.type !== STREAM_CHAT_POPOUT_CLOSED || typeof data.streamId !== "string") {
        return null;
    }

    return data.streamId;
}
