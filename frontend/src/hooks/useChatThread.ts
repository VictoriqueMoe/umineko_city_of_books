import { useCallback, useEffect, useMemo, useRef, useState, type Dispatch, type SetStateAction } from "react";
import { useParams } from "react-router";
import { REALTIME_EVENTS } from "../api/realtime/events";
import { REALTIME_COMMANDS, sendRealtime } from "../api/realtime/outbound";
import { useRealtimeEvent, useRealtimeStatus } from "../api/realtime/useRealtime";
import { type ReplyTarget } from "../components/chat/ChatComposer/ChatComposer";
import { getRoomDisplayName } from "../domain/chat/dm";
import { patchRoom } from "../domain/chat/dmRoster";
import { roomCapabilities, type RoomCapabilities } from "../domain/chat/roomPolicy";
import { buildMentionMatcher } from "../domain/mentions";
import type { ChatMessage, ChatRoom, UserProfile } from "../types/api";
import {
    useChatSession,
    type ChatSessionEditing,
    type ChatSessionHistory,
    type ChatSessionScroll,
    type ChatSessionStatus,
} from "./chat/useChatSession";
import { useRoomScopedOverride } from "./chat/useRoomScopedOverride";
import { useResetOnChange } from "./useResetOnChange";
import {
    useAddChatMessageReaction,
    useMarkChatRoomRead,
    usePinChatMessage,
    useRemoveChatMessageReaction,
    useSetChatRoomMuted,
    useUnpinChatMessage,
} from "./mutations/chat";
import { useUserRooms, useUserRoomsCache } from "./queries/chat";
import { useAuth } from "./useAuth";
import { usePageTitle } from "./usePageTitle";
import { usePresenceReporter } from "./usePresenceReporter";
import { useSiteInfo } from "./useSiteInfo";
import { useVoiceChat, type VoiceChatState } from "./useVoiceChat";

const TOAST_MS = 4000;
const DEFAULT_MAX_MESSAGES = 300;
const IDLE_TITLE = "Chat";
const GROUP_TITLE = "Chat Room";

const MUTED_MESSAGE = "Notifications muted";
const UNMUTED_MESSAGE = "Notifications unmuted";
const MUTE_FAILED = "Failed to update mute";
const REACTION_FAILED = "Failed to update reaction";
const PIN_FAILED = "Failed to update pin";

export interface ChatThreadOptions {
    maxMessages?: number;
    requireMembership?: boolean;
}

export interface ChatThreadRoom {
    data: ChatRoom | null;
    id: string | undefined;
    loading: boolean;
    set: Dispatch<SetStateAction<ChatRoom | null>>;
    toggleMute: () => Promise<void>;
    mutePending: boolean;
    voice: VoiceChatState & { enabled: boolean };
}

export interface ChatThreadSession {
    viewer: UserProfile | null;
    status: ChatSessionStatus;
    messages: ChatMessage[];
    hasMore: boolean;
    loadingMore: boolean;
    containerRef: ChatSessionScroll["containerRef"];
    contentRef: ChatSessionScroll["contentRef"];
    endRef: ChatSessionScroll["endRef"];
    onScroll: ChatSessionScroll["onScroll"];
    toBottom: ChatSessionScroll["toBottom"];
    editingMessageId: ChatSessionEditing["messageId"];
    startEditing: ChatSessionEditing["start"];
    cancelEditing: ChatSessionEditing["cancel"];
    deleteMessage: ChatSessionEditing["remove"];
    editMessage: ChatSessionEditing["save"];
    editLast: ChatSessionEditing["editLast"];
    replyingTo: ReplyTarget | null;
    setReplyingTo: Dispatch<SetStateAction<ReplyTarget | null>>;
    typingUserIds: string[];
    notifyTyping: () => void;
    matchesViewerMention: ((body: string) => boolean) | null;
    readReceipts: Record<string, Record<string, string>>;
    onSent: (message: ChatMessage) => void;
    toggleReaction: (message: ChatMessage, emoji: string) => Promise<void>;
    togglePin: (message: ChatMessage) => Promise<void>;
    history: ChatSessionHistory;
}

export interface ChatThreadToast {
    message: string | null;
    show: Dispatch<SetStateAction<string | null>>;
}

export interface ChatThread {
    thread: ChatThreadRoom;
    session: ChatThreadSession;
    capabilities: RoomCapabilities;
    toast: ChatThreadToast;
}

function threadTitle(room: ChatRoom | null, viewer: UserProfile | null): string {
    if (!room) {
        return IDLE_TITLE;
    }

    if (room.type === "dm" && viewer) {
        return getRoomDisplayName(room, viewer);
    }

    return room.name || GROUP_TITLE;
}

export function useChatThread(options: ChatThreadOptions = {}): ChatThread {
    const { maxMessages = DEFAULT_MAX_MESSAGES, requireMembership = false } = options;

    const { roomId } = useParams<{ roomId: string }>();
    const { user } = useAuth();
    const realtimeEpoch = useRealtimeStatus();
    const matchesViewerMention = useMemo(() => buildMentionMatcher(user?.username), [user?.username]);

    const { rooms, loading: roomsLoading } = useUserRooms(!!user);
    const { patchRooms } = useUserRoomsCache();
    const baseRoom = roomId ? (rooms.find(candidate => candidate.id === roomId) ?? null) : null;
    const [room, setRoom] = useRoomScopedOverride<ChatRoom | null>(roomId, baseRoom);
    const loading = !!roomId && roomsLoading;

    const capabilities = roomCapabilities(room, user);
    const attached = !requireMembership || !!room;

    const voiceParticipants = useMemo(() => room?.voice_participants ?? [], [room?.voice_participants]);
    const voice = useVoiceChat(roomId ?? "", voiceParticipants);
    const voiceEnabled = useSiteInfo()?.voice_enabled ?? false;

    usePageTitle(threadTitle(room, user));
    usePresenceReporter(roomId);

    const [replyingTo, setReplyingTo] = useState<ReplyTarget | null>(null);
    const [toast, setToast] = useState<string | null>(null);
    const [readReceipts, setReadReceipts] = useState<Record<string, Record<string, string>>>({});
    const [mutePending, setMutePending] = useState(false);

    useResetOnChange(roomId, () => {
        setReplyingTo(null);
    });

    const session = useChatSession({
        roomId: attached ? roomId : undefined,
        user,
        maxMessages,
        sound: {
            enabled: user?.private?.play_message_sound ?? true,
            muted: room?.viewer_muted ?? false,
        },
        onEditError: setToast,
    });
    const { addMessage, resync } = session.history;
    const { toBottom } = session.scroll;

    const didResyncMountRef = useRef(false);
    useEffect(() => {
        if (!didResyncMountRef.current) {
            didResyncMountRef.current = true;
            return;
        }

        resync().catch(() => {});
    }, [realtimeEpoch, resync]);

    useEffect(() => {
        if (!toast) {
            return;
        }

        const t = setTimeout(() => setToast(null), TOAST_MS);

        return () => clearTimeout(t);
    }, [toast]);

    useEffect(() => {
        if (!roomId || !attached) {
            return;
        }

        sendRealtime({ type: REALTIME_COMMANDS.JOIN_ROOM, data: { room_id: roomId } });

        return () => {
            sendRealtime({ type: REALTIME_COMMANDS.LEAVE_ROOM, data: { room_id: roomId } });
        };
    }, [roomId, attached, realtimeEpoch]);

    const markRead = useMarkChatRoomRead().mutate;
    const lastMarkedReadRef = useRef<string | null>(null);
    const roomLastMessageAt = room?.last_message_at;
    useEffect(() => {
        if (!roomId || !attached) {
            return;
        }

        const signal = `${roomId}:${roomLastMessageAt ?? ""}`;
        if (lastMarkedReadRef.current === signal) {
            return;
        }

        lastMarkedReadRef.current = signal;
        markRead(roomId);
    }, [roomId, attached, roomLastMessageAt, markRead]);

    useEffect(() => {
        if (!roomId || !attached) {
            return;
        }

        const openRoomId = roomId;
        const handleFocus = () => markRead(openRoomId);

        window.addEventListener("focus", handleFocus);

        return () => {
            window.removeEventListener("focus", handleFocus);
        };
    }, [roomId, attached, markRead]);

    useRealtimeEvent(REALTIME_EVENTS.CHAT_READ_RECEIPT, event => {
        if (!user) {
            return;
        }

        const { room_id: receiptRoomId, user_id: readerId, read_at: readAt } = event.data;

        const receiptRoom =
            receiptRoomId === roomId ? room : (rooms.find(candidate => candidate.id === receiptRoomId) ?? null);
        if (roomCapabilities(receiptRoom, user).readReceipts !== "pairwise") {
            return;
        }

        setReadReceipts(prev => {
            const held = prev[receiptRoomId] ?? {};
            if (held[readerId] && held[readerId] >= readAt) {
                return prev;
            }

            return { ...prev, [receiptRoomId]: { ...held, [readerId]: readAt } };
        });
    });

    const setMutedMutation = useSetChatRoomMuted();
    async function toggleMute(): Promise<void> {
        if (!roomId || !room) {
            return;
        }

        const next = !room.viewer_muted;
        setMutePending(true);

        try {
            await setMutedMutation.mutateAsync({ roomId, muted: next });
            setRoom(prev => (prev ? { ...prev, viewer_muted: next } : prev));
            patchRooms(prev => patchRoom(prev, roomId, { viewer_muted: next }));
            setToast(next ? MUTED_MESSAGE : UNMUTED_MESSAGE);
        } catch (err) {
            setToast(err instanceof Error ? err.message : MUTE_FAILED);
        } finally {
            setMutePending(false);
        }
    }

    const addReaction = useAddChatMessageReaction().mutateAsync;
    const removeReaction = useRemoveChatMessageReaction().mutateAsync;
    const toggleReaction = useCallback(
        async (message: ChatMessage, emoji: string) => {
            const existing = (message.reactions ?? []).find(reaction => reaction.emoji === emoji);

            try {
                if (existing && existing.viewer_reacted) {
                    await removeReaction({ messageId: message.id, emoji });
                } else {
                    await addReaction({ messageId: message.id, emoji });
                }
            } catch (err) {
                setToast(err instanceof Error ? err.message : REACTION_FAILED);
            }
        },
        [addReaction, removeReaction],
    );

    const pin = usePinChatMessage(roomId).mutateAsync;
    const unpin = useUnpinChatMessage(roomId).mutateAsync;
    const togglePin = useCallback(
        async (message: ChatMessage) => {
            try {
                if (message.pinned) {
                    await unpin(message.id);
                } else {
                    await pin(message.id);
                }
            } catch (err) {
                setToast(err instanceof Error ? err.message : PIN_FAILED);
            }
        },
        [pin, unpin],
    );

    function onSent(message: ChatMessage) {
        addMessage(message);
        toBottom({ force: true });
    }

    function notifyTyping() {
        if (!roomId) {
            return;
        }

        sendRealtime({ type: REALTIME_COMMANDS.TYPING, data: { room_id: roomId } });
    }

    return {
        thread: {
            data: room,
            id: roomId,
            loading,
            set: setRoom,
            toggleMute,
            mutePending,
            voice: { ...voice, enabled: voiceEnabled },
        },
        session: {
            viewer: user,
            status: session.status,
            messages: session.messages,
            hasMore: session.history.hasMore,
            loadingMore: session.history.loadingMore,
            containerRef: session.scroll.containerRef,
            contentRef: session.scroll.contentRef,
            endRef: session.scroll.endRef,
            onScroll: session.scroll.onScroll,
            toBottom: session.scroll.toBottom,
            editingMessageId: session.editing.messageId,
            startEditing: session.editing.start,
            cancelEditing: session.editing.cancel,
            deleteMessage: session.editing.remove,
            editMessage: session.editing.save,
            editLast: session.editing.editLast,
            replyingTo,
            setReplyingTo,
            typingUserIds: session.typing.userIds,
            notifyTyping,
            matchesViewerMention,
            readReceipts,
            onSent,
            toggleReaction,
            togglePin,
            history: session.history,
        },
        capabilities,
        toast: {
            message: toast,
            show: setToast,
        },
    };
}
