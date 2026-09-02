import { useCallback, useState } from "react";
import { REALTIME_EVENTS } from "../../api/realtime/events";
import { useRealtimeEvent } from "../../api/realtime/useRealtime";
import {
    applyChatMemberUpdateToMessages,
    applyChatMessageDeleted,
    applyChatMessageEdited,
    applyChatMessagePinned,
    applyChatMessageUnpinned,
    applyReaction,
    applySiteRoleChangeToMessages,
} from "../../domain/chat/messagePatches";
import { appendIfUnknown } from "../../domain/chat/messageStore";
import { playMessageSound, playRemoteAudio } from "../../platform/sound";
import type { ChatMessage, UserProfile } from "../../types/api";
import { useChatMessageHandlers } from "../useChatMessageHandlers";
import { useMessageHistory } from "../useMessageHistory";
import { useTypingIndicator } from "../useTypingIndicator";
import { useDebouncedMarkChatRoomRead } from "./useDebouncedMarkChatRoomRead";

const CHAT_SESSION_EVENTS = [
    REALTIME_EVENTS.CHAT_MESSAGE,
    REALTIME_EVENTS.CHAT_MESSAGE_EDITED,
    REALTIME_EVENTS.CHAT_MESSAGE_DELETED,
    REALTIME_EVENTS.CHAT_MESSAGE_PINNED,
    REALTIME_EVENTS.CHAT_MESSAGE_UNPINNED,
    REALTIME_EVENTS.CHAT_REACTION_ADDED,
    REALTIME_EVENTS.CHAT_REACTION_REMOVED,
    REALTIME_EVENTS.CHAT_MEMBER_UPDATED,
    REALTIME_EVENTS.CHAT_AUDIO,
    REALTIME_EVENTS.CHAT_ROOM_DELETED,
    REALTIME_EVENTS.ROLE_CHANGED,
    REALTIME_EVENTS.TYPING,
] as const;

const DEFAULT_AUDIO_VOLUME = 0.5;

type MessageHistory = ReturnType<typeof useMessageHistory>;

export type ChatSessionStatus = "idle" | "live" | "ended";

export type ChatSessionScrollMode = "smooth" | "instant";

export interface ChatSessionSound {
    enabled: boolean;
    muted: boolean;
}

export interface UseChatSessionOptions {
    roomId: string | undefined;
    user: UserProfile | null;
    maxMessages?: number;
    scrollMode?: ChatSessionScrollMode;
    sound?: ChatSessionSound;
    onEditError?: (message: string) => void;
    editLastBlocked?: boolean;
}

export interface ChatSessionHistory {
    hasMore: boolean;
    loadingMore: boolean;
    setMessages: MessageHistory["setMessages"];
    addMessage: MessageHistory["addMessage"];
    seedMessages: MessageHistory["seedMessages"];
    loadUntilMessage: MessageHistory["loadUntilMessage"];
    resync: MessageHistory["resync"];
}

export interface ChatSessionScroll {
    containerRef: MessageHistory["containerRef"];
    contentRef: MessageHistory["contentRef"];
    endRef: MessageHistory["endRef"];
    onScroll: () => void;
    toBottom: MessageHistory["scrollToBottom"];
    toBottomInstant: MessageHistory["scrollToBottomInstant"];
}

export interface ChatSessionEditing {
    messageId: string | null;
    start: (message: ChatMessage) => void;
    cancel: () => void;
    save: (message: ChatMessage, body: string) => Promise<void>;
    remove: (message: ChatMessage) => Promise<void>;
    editLast: () => void;
}

export interface ChatSessionTyping {
    userIds: string[];
    note: (userId: string) => void;
    clear: (userId: string) => void;
    reset: () => void;
}

export interface ChatSession {
    status: ChatSessionStatus;
    messages: ChatMessage[];
    history: ChatSessionHistory;
    scroll: ChatSessionScroll;
    editing: ChatSessionEditing;
    typing: ChatSessionTyping;
}

function resolveStatus(
    roomId: string | undefined,
    boundRoomId: string | null,
    endedRoomId: string | null,
): ChatSessionStatus {
    if (roomId === undefined) {
        return boundRoomId === null ? "idle" : "ended";
    }

    return endedRoomId === roomId ? "ended" : "live";
}

function shouldPlayMessageSound(
    sound: ChatSessionSound | undefined,
    senderId: string,
    viewerId: string | undefined,
): boolean {
    if (!sound || !sound.enabled || sound.muted) {
        return false;
    }
    if (senderId === viewerId) {
        return false;
    }

    return document.visibilityState !== "visible" || !document.hasFocus();
}

export function useChatSession(options: UseChatSessionOptions): ChatSession {
    const { roomId, user, maxMessages, scrollMode = "smooth", sound, onEditError, editLastBlocked } = options;

    const [editingMessageId, setEditingMessageId] = useState<string | null>(null);
    const [boundRoomId, setBoundRoomId] = useState<string | null>(roomId ?? null);
    const [endedRoomId, setEndedRoomId] = useState<string | null>(null);

    const history = useMessageHistory(roomId, editingMessageId === null ? maxMessages : undefined);
    const typing = useTypingIndicator(roomId);
    const markReadDebounced = useDebouncedMarkChatRoomRead();

    const { handleDeleteMessage, handleEditMessage, handleEditLast } = useChatMessageHandlers({
        user,
        messages: history.messages,
        setMessages: history.setMessages,
        setEditingMessageId,
        onError: onEditError,
        editLastBlocked,
    });

    if (roomId !== undefined && roomId !== boundRoomId) {
        setBoundRoomId(roomId);
        setEndedRoomId(null);
    }

    const status = resolveStatus(roomId, boundRoomId, endedRoomId);
    const live = status === "live";

    useRealtimeEvent(CHAT_SESSION_EVENTS, event => {
        if (!live || roomId === undefined || user === null) {
            return;
        }

        switch (event.type) {
            case "chat_room_deleted": {
                if (event.data.room_id === roomId) {
                    setEndedRoomId(roomId);
                }

                return;
            }
            case "chat_message": {
                const message = event.data;
                if (message.room_id !== roomId) {
                    return;
                }

                typing.clearUser(message.sender.id);
                history.setMessages(current => appendIfUnknown(current, message));

                if (scrollMode === "instant") {
                    history.scrollToBottomInstant();
                } else {
                    history.scrollToBottom();
                }

                markReadDebounced(roomId);

                if (shouldPlayMessageSound(sound, message.sender.id, user.id)) {
                    playMessageSound();
                }

                return;
            }
            case "chat_message_edited": {
                const updated = event.data;
                if (updated.room_id !== roomId) {
                    return;
                }

                history.setMessages(current => applyChatMessageEdited(current, updated));

                return;
            }
            case "chat_message_deleted": {
                if (event.data.room_id !== roomId) {
                    return;
                }

                history.setMessages(current => applyChatMessageDeleted(current, event.data));

                return;
            }
            case "chat_message_pinned": {
                if (event.data.room_id !== roomId) {
                    return;
                }

                history.setMessages(current => applyChatMessagePinned(current, event.data));

                return;
            }
            case "chat_message_unpinned": {
                if (event.data.room_id !== roomId) {
                    return;
                }

                history.setMessages(current => applyChatMessageUnpinned(current, event.data));

                return;
            }
            case "chat_reaction_added": {
                if (event.data.room_id !== roomId) {
                    return;
                }

                history.setMessages(current => applyReaction(current, event.data, user.id, 1));

                return;
            }
            case "chat_reaction_removed": {
                if (event.data.room_id !== roomId) {
                    return;
                }

                history.setMessages(current => applyReaction(current, event.data, user.id, -1));

                return;
            }
            case "chat_member_updated": {
                if (event.data.room_id !== roomId) {
                    return;
                }

                history.setMessages(current => applyChatMemberUpdateToMessages(current, event.data));

                return;
            }
            case "role_changed": {
                if (!event.data.user_id) {
                    return;
                }

                history.setMessages(current => applySiteRoleChangeToMessages(current, event.data));

                return;
            }
            case "chat_audio": {
                if (event.data.room_id !== roomId || !event.data.url) {
                    return;
                }

                playRemoteAudio(event.data.url, event.data.volume ?? DEFAULT_AUDIO_VOLUME);

                return;
            }
            case "typing": {
                if (event.data.room_id !== roomId) {
                    return;
                }

                typing.noteTyping(event.data.user_id);

                return;
            }
        }
    });

    const startEditing = useCallback((message: ChatMessage) => {
        setEditingMessageId(message.id);
    }, []);

    const cancelEditing = useCallback(() => {
        setEditingMessageId(null);
    }, []);

    const { handleScroll } = history;
    const onScroll = useCallback(() => {
        if (!live) {
            return;
        }

        handleScroll();
    }, [live, handleScroll]);

    return {
        status,
        messages: history.messages,
        history: {
            hasMore: live && history.hasMore,
            loadingMore: live && history.loadingMore,
            setMessages: history.setMessages,
            addMessage: history.addMessage,
            seedMessages: history.seedMessages,
            loadUntilMessage: history.loadUntilMessage,
            resync: history.resync,
        },
        scroll: {
            containerRef: history.containerRef,
            contentRef: history.contentRef,
            endRef: history.endRef,
            onScroll,
            toBottom: history.scrollToBottom,
            toBottomInstant: history.scrollToBottomInstant,
        },
        editing: {
            messageId: editingMessageId,
            start: startEditing,
            cancel: cancelEditing,
            save: handleEditMessage,
            remove: handleDeleteMessage,
            editLast: handleEditLast,
        },
        typing: {
            userIds: typing.typingUserIds,
            note: typing.noteTyping,
            clear: typing.clearUser,
            reset: typing.reset,
        },
    };
}
