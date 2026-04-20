import type { Dispatch, SetStateAction } from "react";
import type { ChatMessage, ChatRoomMember, ReactionGroup } from "../types/api";
import { markChatRoomRead } from "../api/endpoints";
import { playMessageSound } from "./sound";

export interface ChatReactionPayload {
    room_id: string;
    message_id: string;
    emoji: string;
    user_id: string;
    display_name: string;
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

export interface MaybePlayChatSoundOpts {
    senderId: string;
    currentUserId: string;
    roomMuted: boolean;
    enabled: boolean;
}

export function maybePlayChatMessageSound(opts: MaybePlayChatSoundOpts): void {
    if (!opts.enabled) {
        return;
    }
    if (opts.roomMuted) {
        return;
    }
    if (opts.senderId === opts.currentUserId) {
        return;
    }
    if (document.visibilityState === "visible") {
        return;
    }
    playMessageSound();
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

export function applyLocalMemberChange(
    member: ChatRoomMember,
    setMembers: Dispatch<SetStateAction<ChatRoomMember[]>>,
    setMessages: Dispatch<SetStateAction<ChatMessage[]>>,
): void {
    setMembers(prev => prev.map(m => (m.user.id === member.user.id ? member : m)));
    setMessages(prev =>
        prev.map(m => {
            if (m.sender.id !== member.user.id) {
                return m;
            }
            return {
                ...m,
                sender_nickname: member.nickname || undefined,
                sender_member_avatar_url: member.member_avatar_url || undefined,
            };
        }),
    );
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

export function applyChatMessageEdited(
    updated: ChatMessage,
    setMessages: Dispatch<SetStateAction<ChatMessage[]>>,
): void {
    setMessages(prev =>
        prev.map(m => {
            if (m.id !== updated.id) {
                return m;
            }
            return {
                ...m,
                body: updated.body,
                edited_at: updated.edited_at,
                media: updated.media ?? m.media,
            };
        }),
    );
}

function toggleReactionInGroups(
    groups: ReactionGroup[],
    emoji: string,
    delta: number,
    viewerReacted: boolean | undefined,
    displayName: string,
): ReactionGroup[] {
    const idx = groups.findIndex(g => g.emoji === emoji);
    if (idx === -1) {
        if (delta < 0) {
            return groups;
        }
        const names = displayName ? [displayName] : [];
        return [...groups, { emoji, count: 1, viewer_reacted: viewerReacted ?? false, display_names: names }];
    }
    const existing = groups[idx];
    const nextCount = Math.max(0, existing.count + delta);
    if (nextCount === 0) {
        return groups.filter((_, i) => i !== idx);
    }
    const existingNames = existing.display_names ?? [];
    let nextNames = existingNames;
    if (displayName) {
        if (delta > 0) {
            if (!existingNames.includes(displayName)) {
                nextNames = [...existingNames, displayName];
            }
        } else {
            const removeAt = existingNames.indexOf(displayName);
            if (removeAt !== -1) {
                nextNames = existingNames.filter((_, i) => i !== removeAt);
            }
        }
    }
    const next = groups.slice();
    next[idx] = {
        ...existing,
        count: nextCount,
        viewer_reacted: viewerReacted ?? existing.viewer_reacted,
        display_names: nextNames,
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
                reactions: toggleReactionInGroups(
                    m.reactions ?? [],
                    payload.emoji,
                    1,
                    viewerReacted,
                    payload.display_name,
                ),
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
                reactions: toggleReactionInGroups(
                    m.reactions ?? [],
                    payload.emoji,
                    -1,
                    viewerReacted,
                    payload.display_name,
                ),
            };
        }),
    );
}
