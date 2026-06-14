import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { useAuth } from "./useAuth";
import { useNotifications } from "./useNotifications";
import { usePageTitle } from "./usePageTitle";
import { type ReplyTarget } from "../components/chat/ChatComposer/ChatComposer";
import { useVoiceChat } from "../components/chat/Voice/useVoiceChat";
import { useSiteInfo } from "./useSiteInfo";
import { useTypingIndicator } from "./useTypingIndicator";
import { buildMentionMatcher } from "../utils/mentions";
import { formatTimeOfDay } from "../utils/time";
import { fetchResolveDMRoom, fetchUserRooms } from "../api/queries/chat";
import { fetchMutualFollowers, fetchSearchUsers } from "../api/queries/misc";
import { useDeleteChatRoom, useMarkChatRoomRead } from "../api/mutations/chat";
import { applySharedChatWSBranch, handleIncomingChatMessage, maybePlayChatMessageSound } from "../utils/chatStream";
import { useChatMessageHandlers } from "./useChatMessageHandlers";
import { useMessageHistory } from "./useMessageHistory";
import type { ChatMessage, ChatRoom, User, WSMessage } from "../types/api";

export function getRoomDisplayName(room: ChatRoom, currentUser: User): string {
    if (room.type === "group") {
        return room.name || "Group Chat";
    }

    const other = room.members.find(m => m.id !== currentUser.id);
    if (other) {
        return other.display_name;
    }

    return "Direct Message";
}

export function getRoomAvatarUser(room: ChatRoom, currentUser: User): User | null {
    if (room.type === "dm") {
        return room.members.find(m => m.id !== currentUser.id) ?? null;
    }

    return null;
}

export function renderSeenLabel(
    msg: ChatMessage,
    idx: number,
    messages: ChatMessage[],
    room: ChatRoom | undefined,
    selfId: string,
    receipts: Record<string, Record<string, string>>,
): string | null {
    if (!room) {
        return null;
    }

    for (let j = idx + 1; j < messages.length; j++) {
        if (messages[j].sender.id === selfId) {
            return null;
        }
    }

    const roomReceipts = receipts[room.id];
    if (!roomReceipts) {
        return null;
    }

    let latestReadAt = "";
    let seenByName = "";
    for (let i = 0; i < room.members.length; i++) {
        const member = room.members[i];
        if (member.id === selfId) {
            continue;
        }

        const readAt = roomReceipts[member.id];
        if (!readAt) {
            continue;
        }

        if (readAt < msg.created_at) {
            continue;
        }

        if (readAt > latestReadAt) {
            latestReadAt = readAt;
            seenByName = room.type === "dm" ? "" : member.display_name;
        }
    }

    if (!latestReadAt) {
        return null;
    }

    const time = formatTimeOfDay(latestReadAt);
    if (room.type === "dm") {
        return `seen ${time}`;
    }

    return `seen by ${seenByName} ${time}`;
}

export function useDmController() {
    usePageTitle("Chat");

    const { roomId: urlRoomId } = useParams<{ roomId: string }>();
    const location = useLocation();
    const navigate = useNavigate();
    const { user } = useAuth();
    const matchesViewerMention = useMemo(() => buildMentionMatcher(user?.username), [user?.username]);
    const { addWSListener, sendWSMessage, wsEpoch } = useNotifications();
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
    const { typingUserIds, noteTyping, clearUser: clearTypingUser, reset: resetTyping } = useTypingIndicator();
    const mobileView: "list" | "room" = urlRoomId || draftRecipient ? "room" : "list";
    const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);
    const [editingMessageId, setEditingMessageId] = useState<string | null>(null);
    const [replyingTo, setReplyingTo] = useState<ReplyTarget | null>(null);
    const {
        messages,
        setMessages,
        hasMore,
        loadingMore,
        containerRef: messagesContainerRef,
        contentRef: messagesContentRef,
        endRef: messagesEndRef,
        scrollToBottom,
        handleScroll: handleDmScroll,
        addMessage,
        resync,
    } = useMessageHistory(activeRoomId ?? undefined);

    const didResyncMountRef = useRef(false);
    useEffect(() => {
        if (!didResyncMountRef.current) {
            didResyncMountRef.current = true;
            return;
        }

        resync().catch(() => {});
    }, [wsEpoch, resync]);

    const deleteChatRoomMutation = useDeleteChatRoom();
    const markChatRoomReadMutation = useMarkChatRoomRead();

    useEffect(() => {
        document.body.dataset.chatPage = "true";
        return () => {
            delete document.body.dataset.chatPage;
        };
    }, []);

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
                    navigate(`/chat/${resolved.room.id}`, { replace: true });
                } else {
                    setDraftRecipient(resolved.recipient);
                    setActiveRoomId(null);
                }
            })
            .catch(() => {});
    }, [location.state, location.pathname, navigate]);

    const dmDebounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
    const activeRoomIdRef = useRef(activeRoomId);
    const activeRoomMutedRef = useRef(false);

    useEffect(() => {
        const active = rooms.find(r => r.id === activeRoomId);
        activeRoomMutedRef.current = active?.viewer_muted ?? false;
    }, [rooms, activeRoomId]);

    useEffect(() => {
        activeRoomIdRef.current = activeRoomId;
    }, [activeRoomId]);

    useEffect(() => {
        if (!user) {
            return;
        }

        fetchUserRooms()
            .then(res => {
                setRooms((res.rooms ?? []).filter(r => r.type === "dm"));
            })
            .catch(() => {})
            .finally(() => setLoading(false));
    }, [user]);

    useEffect(() => {
        if (!user) {
            return;
        }

        return addWSListener((msg: WSMessage) => {
            if (msg.type === "chat_read_receipt") {
                const data = msg.data as { room_id: string; user_id: string; read_at: string };
                setReadReceipts(prev => {
                    const room = prev[data.room_id] ?? {};
                    if (room[data.user_id] && room[data.user_id] >= data.read_at) {
                        return prev;
                    }

                    return {
                        ...prev,
                        [data.room_id]: { ...room, [data.user_id]: data.read_at },
                    };
                });
                return;
            }

            if (
                applySharedChatWSBranch(msg, {
                    activeRoomId: activeRoomIdRef.current,
                    setMessages,
                    noteTyping,
                })
            ) {
                return;
            }

            if (msg.type !== "chat_message") {
                return;
            }

            const chatMsg = msg.data as ChatMessage;

            if (chatMsg.room_id === activeRoomIdRef.current) {
                clearTypingUser(chatMsg.sender.id);
            }

            const added = handleIncomingChatMessage(chatMsg, activeRoomIdRef.current, setMessages, scrollToBottom);
            if (added && user) {
                maybePlayChatMessageSound({
                    senderId: chatMsg.sender.id,
                    currentUserId: user.id,
                    roomMuted: activeRoomMutedRef.current,
                    enabled: user.private?.play_message_sound ?? true,
                });
            }

            setRooms(prev => {
                let foundIdx = -1;
                for (let i = 0; i < prev.length; i++) {
                    if (prev[i].id === chatMsg.room_id) {
                        foundIdx = i;
                        break;
                    }
                }

                if (foundIdx === -1) {
                    fetchUserRooms()
                        .then(res => setRooms((res.rooms ?? []).filter(r => r.type === "dm")))
                        .catch(() => {});
                    return prev;
                }

                const target = prev[foundIdx];
                const updated: ChatRoom = {
                    ...target,
                    last_message_at: chatMsg.created_at,
                    unread: chatMsg.room_id !== activeRoomIdRef.current && chatMsg.sender.id !== user.id,
                };
                const next = prev.slice();
                next.splice(foundIdx, 1);
                next.unshift(updated);
                return next;
            });
        });
    }, [user, addWSListener, scrollToBottom, setMessages, noteTyping, clearTypingUser]);

    useEffect(() => {
        resetTyping();
    }, [activeRoomId, resetTyping]);

    useEffect(() => {
        if (!activeRoomId) {
            return;
        }

        sendWSMessage({ type: "join_room", data: { room_id: activeRoomId } });

        return () => {
            sendWSMessage({ type: "leave_room", data: { room_id: activeRoomId } });
        };
    }, [activeRoomId, sendWSMessage, wsEpoch]);

    const markChatRoomReadAsync = markChatRoomReadMutation.mutateAsync;

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
        }, 200);
        return () => clearTimeout(dmDebounceRef.current);
    }, [dmSearch]);

    function handleRoomSelect(roomId: string) {
        setActiveRoomId(roomId);
        setReplyingTo(null);
        setRooms(prev => prev.map(r => (r.id === roomId ? { ...r, unread: false } : r)));
        navigate(`/chat/${roomId}`, { replace: true });
    }

    function handleMobileBack() {
        setActiveRoomId(null);
        setReplyingTo(null);
        setDraftRecipient(null);
        navigate("/chat", { replace: true });
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
            setMessages([message]);
            setActiveRoomId(room.id);
            setDraftRecipient(null);
            navigate(`/chat/${room.id}`, { replace: true });
            scrollToBottom({ force: true });
            return;
        }

        addMessage(message);

        setRooms(prev => {
            let foundIdx = -1;
            for (let i = 0; i < prev.length; i++) {
                if (prev[i].id === message.room_id) {
                    foundIdx = i;
                    break;
                }
            }

            if (foundIdx === -1) {
                return prev;
            }

            const target = prev[foundIdx];
            const updated: ChatRoom = {
                ...target,
                last_message_at: message.created_at,
                unread: false,
            };
            const next = prev.slice();
            next.splice(foundIdx, 1);
            next.unshift(updated);
            return next;
        });

        scrollToBottom({ force: true });
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
                navigate("/chat", { replace: true });
            }
        } catch (err) {
            setDmError(err instanceof Error ? err.message : "Failed to open conversation");
        } finally {
            setDmCreating(false);
        }
    }

    const { handleDeleteMessage, handleEditMessage, handleEditLast } = useChatMessageHandlers({
        user,
        messages,
        setMessages,
        setEditingMessageId,
    });

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
            navigate("/chat", { replace: true });
        } catch {
            return;
        }
    }

    const activeRoom = rooms.find(r => r.id === activeRoomId);

    const typingNames = typingUserIds
        .filter(id => id !== user?.id)
        .map(id => {
            const m = activeRoom?.members.find(mem => mem.id === id);
            if (!m) {
                return "Someone";
            }

            if (m.display_name && m.display_name.trim() !== "") {
                return m.display_name;
            }

            return m.username;
        });

    function notifyTyping() {
        sendWSMessage({ type: "typing", data: { room_id: activeRoomId } });
    }

    return {
        user,
        loading,
        mobileView,
        rooms,
        activeRoomId,
        activeRoom,
        draftRecipient,
        setDraftRecipient,
        messages,
        hasMore,
        loadingMore,
        messagesContainerRef,
        messagesContentRef,
        messagesEndRef,
        handleDmScroll,
        scrollToBottom,
        readReceipts,
        matchesViewerMention,
        typingNames,
        voice,
        voiceEnabled,
        replyingTo,
        setReplyingTo,
        editingMessageId,
        setEditingMessageId,
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
        handleRoomSelect,
        handleMobileBack,
        handleSentMessage,
        handleSelectUser,
        handleDeleteMessage,
        handleEditMessage,
        handleEditLast,
        handleDeleteChat,
        notifyTyping,
    };
}

export type DmController = ReturnType<typeof useDmController>;
