import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { useAuth } from "./useAuth";
import { usePageTitle } from "./usePageTitle";
import { type ReplyTarget } from "../components/chat/ChatComposer/ChatComposer";
import { useVoiceChat } from "./useVoiceChat";
import { useSiteInfo } from "./useSiteInfo";
import { buildMentionMatcher } from "../domain/mentions";
import { typingNames as resolveTypingNames } from "../domain/chat/memberRoster";
import { moveRoomToFront } from "../domain/chat/dmRoster";
import { fetchResolveDMRoom, fetchUserRooms } from "./queries/chat";
import { fetchMutualFollowers, fetchSearchUsers } from "./queries/user";
import { useDeleteChatRoom, useMarkChatRoomRead, useSetChatRoomMuted } from "./mutations/chat";
import { useChatSession } from "./chat/useChatSession";
import { REALTIME_EVENTS } from "../api/realtime/events";
import { REALTIME_COMMANDS, sendRealtime } from "../api/realtime/outbound";
import { useRealtimeEvent, useRealtimeStatus } from "../api/realtime/useRealtime";
import type { ChatMessage, ChatRoom, User } from "../types/api";

const MAX_DM_MESSAGES = 300;
const TOAST_MS = 4000;
const SEARCH_DEBOUNCE_MS = 200;
const CHAT_PATH = "/chat";

function dmRoomsOf(rooms: ChatRoom[] | undefined): ChatRoom[] {
    return (rooms ?? []).filter(r => r.type === "dm");
}

export function useDmController() {
    usePageTitle("Chat");

    const { roomId: urlRoomId } = useParams<{ roomId: string }>();
    const location = useLocation();
    const navigate = useNavigate();
    const { user } = useAuth();
    const matchesViewerMention = useMemo(() => buildMentionMatcher(user?.username), [user?.username]);
    const realtimeEpoch = useRealtimeStatus();

    const [rooms, setRooms] = useState<ChatRoom[]>([]);
    const [activeRoomId, setActiveRoomId] = useState<string | null>(urlRoomId ?? null);
    const voice = useVoiceChat(activeRoomId ?? "");
    const voiceEnabled = useSiteInfo()?.voice_enabled ?? false;
    const [readReceipts, setReadReceipts] = useState<Record<string, Record<string, string>>>({});
    const [loading, setLoading] = useState(true);
    const [showNewDm, setShowNewDm] = useState(false);
    const [dmSearch, setDmSearch] = useState("");
    const [dmResults, setDmResults] = useState<User[]>([]);
    const [dmMutuals, setDmMutuals] = useState<User[]>([]);
    const [dmError, setDmError] = useState("");
    const [dmCreating, setDmCreating] = useState(false);
    const [draftRecipient, setDraftRecipient] = useState<User | null>(null);
    const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);
    const [replyingTo, setReplyingTo] = useState<ReplyTarget | null>(null);
    const [toast, setToast] = useState<string | null>(null);

    const [syncedUrlRoomId, setSyncedUrlRoomId] = useState(urlRoomId);
    if (syncedUrlRoomId !== urlRoomId) {
        setSyncedUrlRoomId(urlRoomId);
        setActiveRoomId(urlRoomId ?? null);
        setReplyingTo(null);
    }

    const mobileView: "list" | "room" = urlRoomId || draftRecipient ? "room" : "list";
    const activeRoom = rooms.find(r => r.id === activeRoomId);

    const session = useChatSession({
        roomId: activeRoomId ?? undefined,
        user,
        maxMessages: MAX_DM_MESSAGES,
        sound: {
            enabled: user?.private?.play_message_sound ?? true,
            muted: activeRoom?.viewer_muted ?? false,
        },
        onEditError: setToast,
    });
    const { setMessages, seedMessages, addMessage, resync } = session.history;

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

    const deleteChatRoomMutation = useDeleteChatRoom();
    const markChatRoomReadMutation = useMarkChatRoomRead();
    const setMutedMutation = useSetChatRoomMuted();
    const [mutePending, setMutePending] = useState(false);

    useEffect(() => {
        const state = location.state as { dmUserId?: string } | null;
        if (!state?.dmUserId) {
            return;
        }

        const targetId = state.dmUserId;
        navigate(location.pathname, { replace: true, state: null });

        fetchResolveDMRoom(targetId)
            .then(resolved => {
                if (resolved.room) {
                    setRooms(prev => {
                        const exists = prev.find(r => r.id === resolved.room!.id);
                        if (exists) {
                            return prev;
                        }

                        return [resolved.room!, ...prev];
                    });
                    setActiveRoomId(resolved.room.id);
                    setDraftRecipient(null);
                    navigate(`${CHAT_PATH}/${resolved.room.id}`, { replace: true });
                } else {
                    setDraftRecipient(resolved.recipient);
                    setActiveRoomId(null);
                }
            })
            .catch(() => {});
    }, [location.state, location.pathname, navigate]);

    useEffect(() => {
        if (!user) {
            return;
        }

        fetchUserRooms()
            .then(res => {
                setRooms(dmRoomsOf(res.rooms));
            })
            .catch(() => {})
            .finally(() => setLoading(false));
    }, [user]);

    useRealtimeEvent(REALTIME_EVENTS.CHAT_READ_RECEIPT, event => {
        if (!user) {
            return;
        }

        const { room_id: receiptRoomId, user_id: readerId, read_at: readAt } = event.data;
        setReadReceipts(prev => {
            const room = prev[receiptRoomId] ?? {};
            if (room[readerId] && room[readerId] >= readAt) {
                return prev;
            }

            return {
                ...prev,
                [receiptRoomId]: { ...room, [readerId]: readAt },
            };
        });
    });

    useRealtimeEvent(REALTIME_EVENTS.CHAT_MESSAGE, event => {
        if (!user) {
            return;
        }

        const chatMsg = event.data;
        setRooms(prev => {
            const next = moveRoomToFront(prev, chatMsg.room_id, {
                last_message_at: chatMsg.created_at,
                unread: chatMsg.room_id !== activeRoomId && chatMsg.sender.id !== user.id,
            });

            if (next === prev) {
                fetchUserRooms()
                    .then(res => setRooms(dmRoomsOf(res.rooms)))
                    .catch(() => {});
            }

            return next;
        });
    });

    useEffect(() => {
        if (!activeRoomId) {
            return;
        }

        sendRealtime({ type: REALTIME_COMMANDS.JOIN_ROOM, data: { room_id: activeRoomId } });

        return () => {
            sendRealtime({ type: REALTIME_COMMANDS.LEAVE_ROOM, data: { room_id: activeRoomId } });
        };
    }, [activeRoomId, realtimeEpoch]);

    const markChatRoomReadAsync = markChatRoomReadMutation.mutateAsync;
    const activeRoomIdRef = useRef(activeRoomId);
    useEffect(() => {
        activeRoomIdRef.current = activeRoomId;
    }, [activeRoomId]);

    useEffect(() => {
        if (!activeRoomId) {
            return;
        }

        markChatRoomReadAsync(activeRoomId).catch(() => {});
    }, [activeRoomId, markChatRoomReadAsync]);

    useEffect(() => {
        if (!activeRoomId) {
            return;
        }

        function handleFocus() {
            if (activeRoomIdRef.current) {
                markChatRoomReadAsync(activeRoomIdRef.current).catch(() => {});
            }
        }

        window.addEventListener("focus", handleFocus);
        return () => {
            window.removeEventListener("focus", handleFocus);
        };
    }, [activeRoomId, markChatRoomReadAsync]);

    useEffect(() => {
        if (showNewDm) {
            fetchMutualFollowers()
                .then(setDmMutuals)
                .catch(() => setDmMutuals([]));
        }
    }, [showNewDm]);

    const dmDebounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
    useEffect(() => {
        clearTimeout(dmDebounceRef.current);
        if (!dmSearch.trim()) {
            dmDebounceRef.current = setTimeout(() => {
                setDmResults([]);
            }, 0);
            return () => clearTimeout(dmDebounceRef.current);
        }

        dmDebounceRef.current = setTimeout(() => {
            fetchSearchUsers(dmSearch)
                .then(setDmResults)
                .catch(() => setDmResults([]));
        }, SEARCH_DEBOUNCE_MS);
        return () => clearTimeout(dmDebounceRef.current);
    }, [dmSearch]);

    function handleRoomSelect(roomId: string) {
        setActiveRoomId(roomId);
        setReplyingTo(null);
        setRooms(prev => prev.map(r => (r.id === roomId ? { ...r, unread: false } : r)));
        navigate(`${CHAT_PATH}/${roomId}`, { replace: true });
    }

    function handleMobileBack() {
        setActiveRoomId(null);
        setReplyingTo(null);
        setDraftRecipient(null);
        navigate(CHAT_PATH, { replace: true });
    }

    function handleSentMessage(message: ChatMessage, room?: ChatRoom) {
        if (room) {
            setRooms(prev => {
                const exists = prev.find(r => r.id === room.id);
                if (exists) {
                    return prev;
                }

                return [room, ...prev];
            });
            seedMessages(room.id, [message]);
            setActiveRoomId(room.id);
            setDraftRecipient(null);
            navigate(`${CHAT_PATH}/${room.id}`, { replace: true });
            session.scroll.toBottom({ force: true });
            return;
        }

        addMessage(message);

        setRooms(prev =>
            moveRoomToFront(prev, message.room_id, { last_message_at: message.created_at, unread: false }),
        );

        session.scroll.toBottom({ force: true });
    }

    async function handleSelectUser(selectedUser: User) {
        setDmCreating(true);
        setDmError("");

        try {
            const resolved = await fetchResolveDMRoom(selectedUser.id);
            setShowNewDm(false);
            setDmSearch("");
            setDmResults([]);

            if (resolved.room) {
                setRooms(prev => {
                    const exists = prev.find(r => r.id === resolved.room!.id);
                    if (exists) {
                        return prev;
                    }

                    return [resolved.room!, ...prev];
                });
                handleRoomSelect(resolved.room.id);
                setDraftRecipient(null);
            } else {
                setDraftRecipient(resolved.recipient);
                setActiveRoomId(null);
                setMessages([]);
                navigate(CHAT_PATH, { replace: true });
            }
        } catch (err) {
            setDmError(err instanceof Error ? err.message : "Failed to open conversation");
        } finally {
            setDmCreating(false);
        }
    }

    async function handleDeleteChat() {
        if (!activeRoomId) {
            return;
        }

        if (!window.confirm("Remove this conversation from your chat list?")) {
            return;
        }

        try {
            await deleteChatRoomMutation.mutateAsync(activeRoomId);
            setRooms(prev => prev.filter(r => r.id !== activeRoomId));
            setMessages([]);
            setActiveRoomId(null);
            navigate(CHAT_PATH, { replace: true });
        } catch {
            return;
        }
    }

    async function handleToggleMute() {
        if (!activeRoomId || !activeRoom) {
            return;
        }

        const next = !activeRoom.viewer_muted;
        setMutePending(true);

        try {
            await setMutedMutation.mutateAsync({ roomId: activeRoomId, muted: next });
            setRooms(prev => prev.map(r => (r.id === activeRoomId ? { ...r, viewer_muted: next } : r)));
            setToast(next ? "Notifications muted" : "Notifications unmuted");
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to update mute");
        } finally {
            setMutePending(false);
        }
    }

    function notifyTyping() {
        if (!activeRoomId) {
            return;
        }

        sendRealtime({ type: REALTIME_COMMANDS.TYPING, data: { room_id: activeRoomId } });
    }

    const typingNames = resolveTypingNames(session.typing.userIds, activeRoom?.members, user?.id);

    return {
        user,
        loading,
        mobileView,
        rooms,
        activeRoomId,
        activeRoom,
        draftRecipient,
        setDraftRecipient,
        messages: session.messages,
        hasMore: session.history.hasMore,
        loadingMore: session.history.loadingMore,
        messagesContainerRef: session.scroll.containerRef,
        messagesContentRef: session.scroll.contentRef,
        messagesEndRef: session.scroll.endRef,
        handleDmScroll: session.scroll.onScroll,
        scrollToBottom: session.scroll.toBottom,
        readReceipts,
        matchesViewerMention,
        typingNames,
        voice,
        voiceEnabled,
        replyingTo,
        setReplyingTo,
        editingMessageId: session.editing.messageId,
        startEditing: session.editing.start,
        cancelEditing: session.editing.cancel,
        lightboxSrc,
        setLightboxSrc,
        showNewDm,
        setShowNewDm,
        dmSearch,
        setDmSearch,
        dmResults,
        dmMutuals,
        dmError,
        dmCreating,
        toast,
        showToast: setToast,
        handleRoomSelect,
        handleMobileBack,
        handleSentMessage,
        handleSelectUser,
        handleDeleteMessage: session.editing.remove,
        handleEditMessage: session.editing.save,
        handleEditLast: session.editing.editLast,
        handleDeleteChat,
        handleToggleMute,
        mutePending,
        notifyTyping,
    };
}

export type DmController = ReturnType<typeof useDmController>;
