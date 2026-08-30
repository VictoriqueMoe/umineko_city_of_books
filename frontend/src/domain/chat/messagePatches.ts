import type { ChatMessage, ChatRoomMember, ReactionGroup, SiteRole } from "../../types/api";

export interface SiteRoleChangePayload {
    user_id: string;
    role: SiteRole | "";
}

export interface ChatMemberUpdatedPayload {
    room_id: string;
    user_id: string;
    nickname: string;
    display_name: string;
    username: string;
    member_avatar_url: string;
    nickname_locked: boolean;
    timeout_until: string;
    timeout_set_by_staff: boolean;
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

export interface ChatMessageDeletedPayload {
    room_id: string;
    message_id: string;
}

export interface ChatReactionPayload {
    room_id: string;
    message_id: string;
    emoji: string;
    user_id: string;
    display_name: string;
    count?: number;
}

export type ReactionDelta = 1 | -1;

export function applyLocalMemberChangeToMembers(current: ChatRoomMember[], member: ChatRoomMember): ChatRoomMember[] {
    return current.map(m => (m.user.id === member.user.id ? member : m));
}

export function applyLocalMemberChangeToMessages(current: ChatMessage[], member: ChatRoomMember): ChatMessage[] {
    return current.map(m => {
        if (m.sender.id !== member.user.id) {
            return m;
        }

        return {
            ...m,
            sender_nickname: member.nickname || undefined,
            sender_member_avatar_url: member.member_avatar_url || undefined,
        };
    });
}

export function applyChatMemberUpdateToMembers(
    current: ChatRoomMember[],
    payload: ChatMemberUpdatedPayload,
): ChatRoomMember[] {
    return current.map(m => {
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
    });
}

export function applyChatMemberUpdateToMessages(
    current: ChatMessage[],
    payload: ChatMemberUpdatedPayload,
): ChatMessage[] {
    return current.map(m => {
        const senderMatches = m.sender.id === payload.user_id;
        const replyMatches = m.reply_to?.sender_id === payload.user_id;
        if (!senderMatches && !replyMatches) {
            return m;
        }

        const next = { ...m };
        if (senderMatches) {
            next.sender_nickname = payload.nickname || undefined;
            next.sender_member_avatar_url = payload.member_avatar_url || undefined;
        }
        if (replyMatches && m.reply_to) {
            next.reply_to = {
                ...m.reply_to,
                sender_name: resolveReplySenderName(payload),
            };
        }

        return next;
    });
}

function resolveReplySenderName(payload: ChatMemberUpdatedPayload): string {
    if (payload.nickname.trim() !== "") {
        return payload.nickname;
    }
    if (payload.display_name.trim() !== "") {
        return payload.display_name;
    }

    return payload.username;
}

export function applySiteRoleChangeToMembers(
    current: ChatRoomMember[],
    payload: SiteRoleChangePayload,
): ChatRoomMember[] {
    const nextRole = payload.role || undefined;

    return current.map(m => {
        if (m.user.id !== payload.user_id) {
            return m;
        }

        return { ...m, user: { ...m.user, role: nextRole } };
    });
}

export function applySiteRoleChangeToMessages(current: ChatMessage[], payload: SiteRoleChangePayload): ChatMessage[] {
    const nextRole = payload.role || undefined;

    return current.map(m => {
        if (m.sender.id !== payload.user_id) {
            return m;
        }

        return { ...m, sender: { ...m.sender, role: nextRole } };
    });
}

export function applyChatMessagePinned(current: ChatMessage[], payload: ChatMessagePinnedPayload): ChatMessage[] {
    return current.map(m => {
        if (m.id !== payload.message_id) {
            return m;
        }

        return { ...m, pinned: true, pinned_at: payload.pinned_at, pinned_by: payload.pinned_by };
    });
}

export function applyChatMessageUnpinned(current: ChatMessage[], payload: ChatMessageUnpinnedPayload): ChatMessage[] {
    return current.map(m => {
        if (m.id !== payload.message_id) {
            return m;
        }

        return { ...m, pinned: false, pinned_at: undefined, pinned_by: undefined };
    });
}

export function applyChatMessageDeleted(current: ChatMessage[], payload: ChatMessageDeletedPayload): ChatMessage[] {
    return current.filter(m => m.id !== payload.message_id);
}

export function applyChatMessageEdited(current: ChatMessage[], updated: ChatMessage): ChatMessage[] {
    return current.map(m => {
        if (m.id !== updated.id) {
            return m;
        }

        return {
            ...m,
            body: updated.body,
            edited_at: updated.edited_at,
            media: updated.media ?? m.media,
        };
    });
}

function toggleReactionInGroups(
    groups: ReactionGroup[],
    emoji: string,
    delta: ReactionDelta,
    viewerReacted: boolean | undefined,
    displayName: string,
    authoritativeCount?: number,
): ReactionGroup[] {
    const idx = groups.findIndex(g => g.emoji === emoji);
    if (idx === -1) {
        if (delta < 0) {
            return groups;
        }

        const initialCount = authoritativeCount !== undefined ? authoritativeCount : 1;
        if (initialCount <= 0) {
            return groups;
        }

        const names = displayName ? [displayName] : [];

        return [
            ...groups,
            { emoji, count: initialCount, viewer_reacted: viewerReacted ?? false, display_names: names },
        ];
    }

    const existing = groups[idx];
    const nextCount =
        authoritativeCount !== undefined ? Math.max(0, authoritativeCount) : Math.max(0, existing.count + delta);
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

export function applyReaction(
    current: ChatMessage[],
    payload: ChatReactionPayload,
    viewerUserId: string,
    delta: ReactionDelta,
): ChatMessage[] {
    const viewerReacted = payload.user_id === viewerUserId ? delta > 0 : undefined;

    return current.map(m => {
        if (m.id !== payload.message_id) {
            return m;
        }

        return {
            ...m,
            reactions: toggleReactionInGroups(
                m.reactions ?? [],
                payload.emoji,
                delta,
                viewerReacted,
                payload.display_name,
                payload.count,
            ),
        };
    });
}
