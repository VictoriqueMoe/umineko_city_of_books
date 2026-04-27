import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { useNotifications } from "../../hooks/useNotifications";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Modal } from "../../components/Modal/Modal";
import { ChatComposer } from "../../components/chat/ChatComposer/ChatComposer";
import { TypingIndicator } from "../../components/chat/TypingIndicator/TypingIndicator";
import { useTypingIndicator } from "../../hooks/useTypingIndicator";
import { MessageBubble } from "../../components/chat/MessageBubble/MessageBubble";
import { Lightbox } from "../../components/Lightbox/Lightbox";
import { buildMentionMatcher } from "../../utils/mentions";
import { isSiteStaff } from "../../utils/permissions";
import { formatTimeOfDay } from "../../utils/time";
import { fetchResolveDMRoom, fetchUserRooms } from "../../api/queries/chat";
import { fetchMutualFollowers, fetchSearchUsers } from "../../api/queries/misc";
import { useDeleteChatRoom, useMarkChatRoomRead } from "../../api/mutations/chat";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { applySharedChatWSBranch, handleIncomingChatMessage, maybePlayChatMessageSound } from "../../utils/chatStream";
import { useChatMessageHandlers } from "../../hooks/useChatMessageHandlers";
import { useMessageHistory } from "../../hooks/useMessageHistory";
import type { ChatMessage, ChatRoom, User, WSMessage } from "../../types/api";
import styles from "./ChatPage.module.css";

function getRoomDisplayName(room: ChatRoom, currentUser: User): string {
    if (room.type === "group") {
        return room.name || "Group Chat";
    }
    const other = room.members.find(m => m.id !== currentUser.id);
    if (other) {
        return other.display_name;
    }
    return "Direct Message";
}

function getRoomAvatarUser(room: ChatRoom, currentUser: User): User | null {
    if (room.type === "dm") {
        return room.members.find(m => m.id !== currentUser.id) ?? null;
    }
    return null;
}

function renderSeenLabel(
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

export function ChatPage() {
    usePageTitle("Chat");
    const { roomId: urlRoomId } = useParams<{ roomId: string }>();
    const location = useLocation();
    const navigate = useNavigate();
    const { user } = useAuth();
    const matchesViewerMention = useMemo(() => buildMentionMatcher(user?.username), [user?.username]);
    const { addWSListener, sendWSMessage, wsEpoch } = useNotifications();
    const [rooms, setRooms] = useState<ChatRoom[]>([]);
    const [activeRoomId, setActiveRoomId] = useState<string | null>(urlRoomId ?? null);
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
    const {
        messages,
        setMessages,
        hasMore,
        loadingMore,
        containerRef: messagesContainerRef,
        endRef: messagesEndRef,
        scrollToBottom,
        handleScroll: handleDmScroll,
        addMessage,
    } = useMessageHistory(activeRoomId ?? undefined);

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
                    enabled: user.play_message_sound ?? true,
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

    useEffect(() => {
        if (!activeRoomId) {
            return;
        }
        markChatRoomReadMutation.mutateAsync(activeRoomId).catch(() => {});
    }, [activeRoomId, markChatRoomReadMutation]);

    useEffect(() => {
        if (!activeRoomId) {
            return;
        }
        function handleFocus() {
            if (activeRoomIdRef.current) {
                markChatRoomReadMutation.mutateAsync(activeRoomIdRef.current).catch(() => {});
            }
        }
        window.addEventListener("focus", handleFocus);
        return () => {
            window.removeEventListener("focus", handleFocus);
        };
    }, [activeRoomId, markChatRoomReadMutation]);

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
        setRooms(prev => prev.map(r => (r.id === roomId ? { ...r, unread: false } : r)));
        navigate(`/chat/${roomId}`, { replace: true });
    }

    function handleMobileBack() {
        setActiveRoomId(null);
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
            // ignore
        }
    }

    if (!user) {
        return null;
    }

    if (loading) {
        return <div className={styles.keysLoading}>Loading chat...</div>;
    }

    const activeRoom = rooms.find(r => r.id === activeRoomId);
    const isSiteMod = isSiteStaff(user.role);

    return (
        <div className={styles.chatWrapper}>
            <div className={styles.chatLayout} data-mobile-view={mobileView}>
                <div className={styles.roomList}>
                    <div className={styles.roomListHeader}>
                        <span className={styles.roomListTitle}>Messages</span>
                        <Button variant="ghost" size="small" onClick={() => setShowNewDm(true)}>
                            New DM
                        </Button>
                    </div>
                    <div className={styles.rooms}>
                        {rooms.length === 0 && <div className={styles.emptyRooms}>No conversations yet</div>}
                        {rooms.map(room => {
                            const avatarUser = getRoomAvatarUser(room, user);
                            return (
                                <button
                                    key={room.id}
                                    className={`${styles.roomItem}${room.id === activeRoomId ? ` ${styles.roomItemActive}` : ""}`}
                                    onClick={() => handleRoomSelect(room.id)}
                                >
                                    {avatarUser ? (
                                        <ProfileLink user={avatarUser} size="small" />
                                    ) : (
                                        <span className={styles.roomName}>{getRoomDisplayName(room, user)}</span>
                                    )}
                                    {room.unread && <span className={styles.unreadDot} aria-label="unread" />}
                                </button>
                            );
                        })}
                    </div>
                </div>

                <div className={styles.messageArea}>
                    {!activeRoom && draftRecipient ? (
                        <>
                            <div className={styles.messageHeader}>
                                <div className={styles.messageHeaderLeft}>
                                    <button
                                        type="button"
                                        className={styles.backButton}
                                        onClick={handleMobileBack}
                                        aria-label="Back to conversations"
                                    >
                                        {"\u2190"}
                                    </button>
                                    <ProfileLink user={draftRecipient} size="small" />
                                </div>
                                <Button variant="ghost" size="small" onClick={() => setDraftRecipient(null)}>
                                    Cancel
                                </Button>
                            </div>
                            <div className={styles.messages}>
                                <div className={styles.messageAreaEmpty}>
                                    Send your first message to {draftRecipient.display_name}.
                                </div>
                                <div ref={messagesEndRef} />
                            </div>
                            <ChatComposer
                                roomId={null}
                                draftRecipientId={draftRecipient.id}
                                onSent={handleSentMessage}
                            />
                        </>
                    ) : !activeRoom ? (
                        <div className={styles.messageAreaEmpty}>Select a conversation</div>
                    ) : (
                        <>
                            <div className={styles.messageHeader}>
                                <div className={styles.messageHeaderLeft}>
                                    <button
                                        type="button"
                                        className={styles.backButton}
                                        onClick={handleMobileBack}
                                        aria-label="Back to conversations"
                                    >
                                        {"\u2190"}
                                    </button>
                                    {getRoomAvatarUser(activeRoom, user) ? (
                                        <ProfileLink user={getRoomAvatarUser(activeRoom, user)!} size="small" />
                                    ) : (
                                        <span>{getRoomDisplayName(activeRoom, user)}</span>
                                    )}
                                </div>
                                <Button variant="danger" size="small" onClick={handleDeleteChat}>
                                    Delete Chat
                                </Button>
                            </div>
                            <div className={styles.messages} ref={messagesContainerRef} onScroll={handleDmScroll}>
                                {hasMore && (
                                    <div className={styles.loadMoreBar}>
                                        {loadingMore ? "Loading older messages..." : "Scroll up for more"}
                                    </div>
                                )}
                                {messages.map((msg, idx) => {
                                    const isOwn = msg.sender.id === user.id;
                                    const seenLabel = isOwn
                                        ? renderSeenLabel(msg, idx, messages, activeRoom, user.id, readReceipts)
                                        : null;
                                    return (
                                        <MessageBubble
                                            key={msg.id}
                                            message={msg}
                                            isOwn={isOwn}
                                            notifiesViewer={
                                                msg.reply_to?.sender_id === user.id ||
                                                (matchesViewerMention ? matchesViewerMention(msg.body) : false)
                                            }
                                            seenLabel={seenLabel}
                                            onLightbox={setLightboxSrc}
                                            onDelete={handleDeleteMessage}
                                            onEdit={handleEditMessage}
                                            onEditStart={m => setEditingMessageId(m.id)}
                                            onEditCancel={() => setEditingMessageId(null)}
                                            editing={editingMessageId === msg.id}
                                            canModerate={isSiteMod}
                                            senderIsStaff={isSiteStaff(msg.sender.role)}
                                        />
                                    );
                                })}
                                <div ref={messagesEndRef} />
                            </div>
                            <TypingIndicator
                                names={typingUserIds
                                    .filter(id => id !== user.id)
                                    .map(id => {
                                        const m = activeRoom.members.find(mem => mem.id === id);
                                        if (!m) {
                                            return "Someone";
                                        }
                                        if (m.display_name && m.display_name.trim() !== "") {
                                            return m.display_name;
                                        }
                                        return m.username;
                                    })}
                            />
                            <ChatComposer
                                roomId={activeRoomId}
                                draftRecipientId={null}
                                onSent={handleSentMessage}
                                onTyping={() => sendWSMessage({ type: "typing", data: { room_id: activeRoomId } })}
                                onEditLast={handleEditLast}
                            />
                        </>
                    )}
                </div>

                <Modal isOpen={showNewDm} onClose={() => setShowNewDm(false)} title="New Direct Message">
                    <div className={styles.modalBody}>
                        <Input
                            fullWidth
                            type="text"
                            placeholder="Search users..."
                            value={dmSearch}
                            onChange={e => setDmSearch(e.target.value)}
                        />
                        {dmError && <div className={styles.modalError}>{dmError}</div>}

                        <div className={styles.userList}>
                            {dmSearch.trim() ? (
                                dmResults.length === 0 ? (
                                    <div className={styles.emptyRooms}>No users found</div>
                                ) : (
                                    dmResults.map(u => (
                                        <button
                                            key={u.id}
                                            className={styles.userOption}
                                            onClick={() => handleSelectUser(u)}
                                            disabled={dmCreating}
                                        >
                                            <ProfileLink user={u} size="small" clickable={false} />
                                        </button>
                                    ))
                                )
                            ) : (
                                <>
                                    {dmMutuals.length > 0 && (
                                        <div className={styles.mutualsLabel}>Mutual followers</div>
                                    )}
                                    {dmMutuals.map(u => (
                                        <button
                                            key={u.id}
                                            className={styles.userOption}
                                            onClick={() => handleSelectUser(u)}
                                            disabled={dmCreating}
                                        >
                                            <ProfileLink user={u} size="small" clickable={false} />
                                        </button>
                                    ))}
                                    {dmMutuals.length === 0 && (
                                        <div className={styles.emptyRooms}>
                                            Search for a user to start a conversation
                                        </div>
                                    )}
                                </>
                            )}
                        </div>
                    </div>
                </Modal>
            </div>
            {lightboxSrc && <Lightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />}
        </div>
    );
}
