import type { Dispatch, SetStateAction } from "react";
import type { ChatMessage, ChatRoomMember, PostMedia, ReactionGroup } from "../types/api";
import { markChatRoomRead } from "../api/endpoints";

export interface ChatMessageMediaAddedPayload {
    room_id: string;
    message_id: string;
    media: PostMedia;
}

export interface ChatReactionPayload {
    room_id: string;
    message_id: string;
    emoji: string;
    user_id: string;
}

export function handleIncomingChatMessage(
    chatMsg: ChatMessage,
    activeRoomId: string | null,
    setMessages: Dispatch<SetStateAction<ChatMessage[]>>,
    scrollToBottom: () => void,
): boolean {
    if (chatMsg.room_id !== activeRoomId) {
        return false;
    }
    setMessages(prev => {
        if (prev.some(m => m.id === chatMsg.id)) {
            return prev;
        }
        return [...prev, chatMsg];
    });
    scrollToBottom();
    if (document.visibilityState === "visible" && document.hasFocus()) {
        markChatRoomRead(chatMsg.room_id).catch(() => {});
    }
    return true;
}

export function handleIncomingChatMessageMedia(
    payload: ChatMessageMediaAddedPayload,
    activeRoomId: string | null,
    setMessages: Dispatch<SetStateAction<ChatMessage[]>>,
): boolean {
    if (payload.room_id !== activeRoomId) {
        return false;
    }
    setMessages(prev => {
        let changed = false;
        const next = prev.map(m => {
            if (m.id !== payload.message_id) {
                return m;
            }
            const existing = m.media ?? [];
            if (existing.some(x => x.id === payload.media.id)) {
                return m;
            }
            changed = true;
            return { ...m, media: [...existing, payload.media] };
        });
        return changed ? next : prev;
    });
    return true;
}

export interface ChatMemberUpdatedPayload {
    room_id: string;
    user_id: string;
    nickname: string;
    member_avatar_url: string;
    nickname_locked: boolean;
    timeout_until: string;
    timeout_set_by_staff: boolean;
}

export function applyChatMemberUpdate(
    payload: ChatMemberUpdatedPayload,
    setMembers: Dispatch<SetStateAction<ChatRoomMember[]>>,
    setMessages: Dispatch<SetStateAction<ChatMessage[]>>,
): void {
    setMembers(prev =>
        prev.map(m => {
            if (m.user.id !== payload.user_id) {
                return m;
            }
            return {
                ...m,
                nickname: payload.nickname,
                member_avatar_url: payload.member_avatar_url,
                nickname_locked: payload.nickname_locked,
                timeout_until: payload.timeout_until || undefined,
                timeout_set_by_staff: payload.timeout_set_by_staff,
            };
        }),
    );
    setMessages(prev =>
        prev.map(m => {
            if (m.sender.id !== payload.user_id) {
                return m;
            }
            return {
                ...m,
                sender_nickname: payload.nickname || undefined,
                sender_member_avatar_url: payload.member_avatar_url || undefined,
            };
        }),
    );
}

export interface ChatMessagePinnedPayload {
    room_id: string;
    message_id: string;
    pinned_at: string;
    pinned_by: string;
}

export interface ChatMessageUnpinnedPayload {
    room_id: string;
    message_id: string;
}

export function applyChatMessagePinned(
    payload: ChatMessagePinnedPayload,
    setMessages: Dispatch<SetStateAction<ChatMessage[]>>,
): void {
    setMessages(prev =>
        prev.map(m => {
            if (m.id !== payload.message_id) {
                return m;
            }
            return { ...m, pinned: true, pinned_at: payload.pinned_at, pinned_by: payload.pinned_by };
        }),
    );
}

export function applyChatMessageUnpinned(
    payload: ChatMessageUnpinnedPayload,
    setMessages: Dispatch<SetStateAction<ChatMessage[]>>,
): void {
    setMessages(prev =>
        prev.map(m => {
            if (m.id !== payload.message_id) {
                return m;
            }
            return { ...m, pinned: false, pinned_at: undefined, pinned_by: undefined };
        }),
    );
}

export interface ChatMessageDeletedPayload {
    room_id: string;
    message_id: string;
}

export function applyChatMessageDeleted(
    payload: ChatMessageDeletedPayload,
    setMessages: Dispatch<SetStateAction<ChatMessage[]>>,
): void {
    setMessages(prev => prev.filter(m => m.id !== payload.message_id));
}

function toggleReactionInGroups(
    groups: ReactionGroup[],
    emoji: string,
    delta: number,
    viewerReacted: boolean | undefined,
): ReactionGroup[] {
    const idx = groups.findIndex(g => g.emoji === emoji);
    if (idx === -1) {
        if (delta < 0) {
            return groups;
        }
        return [...groups, { emoji, count: 1, viewer_reacted: viewerReacted ?? false, display_names: [] }];
    }
    const existing = groups[idx];
    const nextCount = Math.max(0, existing.count + delta);
    if (nextCount === 0) {
        return groups.filter((_, i) => i !== idx);
    }
    const next = groups.slice();
    next[idx] = {
        ...existing,
        count: nextCount,
        viewer_reacted: viewerReacted ?? existing.viewer_reacted,
    };
    return next;
}

export function applyReactionAdded(
    payload: ChatReactionPayload,
    viewerUserId: string,
    setMessages: Dispatch<SetStateAction<ChatMessage[]>>,
): void {
    const viewerReacted = payload.user_id === viewerUserId ? true : undefined;
    setMessages(prev =>
        prev.map(m => {
            if (m.id !== payload.message_id) {
                return m;
            }
            return {
                ...m,
                reactions: toggleReactionInGroups(m.reactions ?? [], payload.emoji, 1, viewerReacted),
            };
        }),
    );
}

export function applyReactionRemoved(
    payload: ChatReactionPayload,
    viewerUserId: string,
    setMessages: Dispatch<SetStateAction<ChatMessage[]>>,
): void {
    const viewerReacted = payload.user_id === viewerUserId ? false : undefined;
    setMessages(prev =>
        prev.map(m => {
            if (m.id !== payload.message_id) {
                return m;
            }
            return {
                ...m,
                reactions: toggleReactionInGroups(m.reactions ?? [], payload.emoji, -1, viewerReacted),
            };
        }),
    );
}
