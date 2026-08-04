export const STREAM_CHAT_POPOUT_CLOSED = "stream-chat-popout-closed";

const POPOUT_FEATURES = "popup=yes,width=420,height=760,resizable=yes,scrollbars=yes";

export function streamChatPopoutPath(streamId: string): string {
    return `/live/${streamId}/chat`;
}

export function streamChatPopoutName(streamId: string): string {
    return `stream-chat-${streamId}`;
}

export function openStreamChatPopout(streamId: string): Window | null {
    return window.open(streamChatPopoutPath(streamId), streamChatPopoutName(streamId), POPOUT_FEATURES);
}
