import { useCallback, useEffect, useRef, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { useNotifications } from "../../hooks/useNotifications";
import { usePageTitle } from "../../hooks/usePageTitle";
import type { ChatMessage, ChatRoom, ChatRoomMember, User, WSMessage } from "../../types/api";
import {
    deleteChatRoom,
    getChatRoomMembers,
    getUserRooms,
    joinChatRoom,
    kickChatRoomMember,
    leaveChatRoom,
    markChatRoomRead,
    setChatRoomMuted,
} from "../../api/endpoints";
import { useMessageHistory } from "../../hooks/useMessageHistory";
import { Button } from "../../components/Button/Button";
import { ChatComposer, type ReplyTarget } from "../../components/chat/ChatComposer/ChatComposer";
import { MessageBubble } from "../../components/chat/MessageBubble/MessageBubble";
import { Lightbox } from "../../components/Lightbox/Lightbox";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import {
    ChatMessageMediaAddedPayload,
    handleIncomingChatMessage,
    handleIncomingChatMessageMedia,
} from "../../utils/chatStream";
import styles from "./RoomPage.module.css";

export function RoomPage() {
    const { roomId } = useParams<{ roomId: string }>();
    const navigate = useNavigate();
    const location = useLocation();
    const { user } = useAuth();
    const { addWSListener, sendWSMessage } = useNotifications();
    const [room, setRoom] = useState<ChatRoom | null>(null);
    const [members, setMembers] = useState<ChatRoomMember[]>([]);
    const [loading, setLoading] = useState(true);
    const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);
    const [toast, setToast] = useState<string | null>(null);
    const [joining, setJoining] = useState(false);
    const [busy, setBusy] = useState<string | null>(null);
    const [mobileView, setMobileView] = useState<"members" | "chat">("chat");
    const [replyingTo, setReplyingTo] = useState<ReplyTarget | null>(null);
    const [highlightedMsgId, setHighlightedMsgId] = useState<string | null>(null);
    const [descExpanded, setDescExpanded] = useState(false);
    const roomIdRef = useRef(roomId);
    const handledHashRef = useRef<string | null>(null);
    const {
        messages,
        setMessages,
        hasMore,
        loadingMore,
        containerRef: messagesContainerRef,
        endRef: messagesEndRef,
        scrollToBottom,
        handleScroll: handleMessagesScroll,
        addMessage,
    } = useMessageHistory(room ? roomId : undefined);

    const targetMsgId = location.hash.startsWith("#msg-") ? location.hash.slice(5) : null;
    const pendingTargetMsgId = targetMsgId && handledHashRef.current !== targetMsgId ? targetMsgId : null;

    usePageTitle(room?.name ?? "Chat Room");

    useEffect(() => {
        roomIdRef.current = roomId;
    }, [roomId]);

    useEffect(() => {
        document.body.dataset.chatPage = "true";
        return () => {
            delete document.body.dataset.chatPage;
        };
    }, []);

    useEffect(() => {
        if (!toast) {
            return;
        }
        const t = setTimeout(() => setToast(null), 4000);
        return () => clearTimeout(t);
    }, [toast]);

    useEffect(() => {
        if (!pendingTargetMsgId || messages.length === 0) {
            return;
        }
        if (!messages.some(m => m.id === pendingTargetMsgId)) {
            return;
        }
        const t = setTimeout(() => {
            const el = document.getElementById(`chat-msg-${pendingTargetMsgId}`);
            if (el) {
                el.scrollIntoView({ behavior: "smooth", block: "center" });
                setHighlightedMsgId(pendingTargetMsgId);
                handledHashRef.current = pendingTargetMsgId;
            }
        }, 300);
        return () => clearTimeout(t);
    }, [pendingTargetMsgId, messages]);

    useEffect(() => {
        if (!highlightedMsgId) {
            return;
        }
        const t = setTimeout(() => setHighlightedMsgId(null), 3000);
        return () => clearTimeout(t);
    }, [highlightedMsgId]);

    const loadRoom = useCallback(async () => {
        if (!roomId) {
            return;
        }
        setLoading(true);
        try {
            const res = await getUserRooms();
            const found = res.rooms?.find(r => r.id === roomId);
            setRoom(found ?? null);
        } catch {
            setRoom(null);
        } finally {
            setLoading(false);
        }
    }, [roomId]);

    const loadMembers = useCallback(async () => {
        if (!roomId) {
            return;
        }
        try {
            const res = await getChatRoomMembers(roomId);
            setMembers(res.members ?? []);
        } catch {
            setMembers([]);
        }
    }, [roomId]);

    useEffect(() => {
        loadRoom();
    }, [loadRoom]);

    useEffect(() => {
        if (!room) {
            return;
        }
        loadMembers();
    }, [room, loadMembers]);

    useEffect(() => {
        if (!roomId || !room) {
            return;
        }
        markChatRoomRead(roomId).catch(() => {});
    }, [roomId, room]);

    useEffect(() => {
        if (!roomId) {
            return;
        }
        sendWSMessage({ type: "join_room", data: { room_id: roomId } });
        return () => {
            sendWSMessage({ type: "leave_room", data: { room_id: roomId } });
        };
    }, [roomId, sendWSMessage]);

    useEffect(() => {
        if (!user) {
            return;
        }
        return addWSListener((msg: WSMessage) => {
            if (msg.type === "chat_message") {
                const chatMsg = msg.data as ChatMessage;
                handleIncomingChatMessage(chatMsg, roomIdRef.current ?? null, setMessages, scrollToBottom);
                return;
            }
            if (msg.type === "chat_message_media_added") {
                const payload = msg.data as ChatMessageMediaAddedPayload;
                handleIncomingChatMessageMedia(payload, roomIdRef.current ?? null, setMessages);
                return;
            }
            if (msg.type === "chat_member_joined") {
                const data = msg.data as { room_id: string; user: User };
                if (data.room_id !== roomIdRef.current) {
                    return;
                }
                loadMembers();
                setRoom(prev => {
                    if (!prev) {
                        return prev;
                    }
                    return {
                        ...prev,
                        member_count: (prev.member_count ?? prev.members.length) + 1,
                    };
                });
                return;
            }
            if (msg.type === "chat_member_left") {
                const data = msg.data as { room_id: string; user_id: string };
                if (data.room_id !== roomIdRef.current) {
                    return;
                }
                setMembers(prev => prev.filter(m => m.user.id !== data.user_id));
                setRoom(prev => {
                    if (!prev) {
                        return prev;
                    }
                    return {
                        ...prev,
                        member_count: Math.max(0, (prev.member_count ?? prev.members.length) - 1),
                    };
                });
                return;
            }
            if (msg.type === "chat_kicked") {
                const data = msg.data as { room_id: string };
                if (data.room_id !== roomIdRef.current) {
                    return;
                }
                setToast("You were removed from this room");
                setTimeout(() => navigate("/rooms"), 1500);
                return;
            }
            if (msg.type === "chat_room_deleted") {
                const data = msg.data as { room_id: string };
                if (data.room_id !== roomIdRef.current) {
                    return;
                }
                setToast("This room was deleted by the host");
                setTimeout(() => navigate("/rooms"), 1500);
                return;
            }
        });
    }, [user, addWSListener, scrollToBottom, setMessages, navigate, loadMembers]);

    function handleSentMessage(message: ChatMessage) {
        addMessage(message);
        scrollToBottom();
    }

    async function handleJoin() {
        if (!roomId) {
            return;
        }
        setJoining(true);
        try {
            const joined = await joinChatRoom(roomId);
            setRoom(joined);
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to join room");
        } finally {
            setJoining(false);
        }
    }

    async function handleKick(targetId: string) {
        if (!roomId || !window.confirm("Kick this member from the room?")) {
            return;
        }
        setBusy(targetId);
        try {
            await kickChatRoomMember(roomId, targetId);
            setMembers(prev => prev.filter(m => m.user.id !== targetId));
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to kick");
        } finally {
            setBusy(null);
        }
    }

    async function handleToggleMute() {
        if (!roomId || !room) {
            return;
        }
        setBusy("mute");
        const next = !room.viewer_muted;
        try {
            await setChatRoomMuted(roomId, next);
            setRoom(prev => {
                if (!prev) {
                    return prev;
                }
                return { ...prev, viewer_muted: next };
            });
            setToast(next ? "Notifications muted" : "Notifications unmuted");
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to update mute");
        } finally {
            setBusy(null);
        }
    }

    async function handleLeave() {
        if (!roomId || !window.confirm("Leave this room?")) {
            return;
        }
        setBusy("self");
        try {
            await leaveChatRoom(roomId);
            navigate("/rooms");
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to leave");
            setBusy(null);
        }
    }

    async function handleDelete() {
        if (!roomId || !window.confirm("Delete this room? Everyone will be removed and the messages will be lost.")) {
            return;
        }
        setBusy("delete");
        try {
            await deleteChatRoom(roomId);
            navigate("/rooms");
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to delete");
            setBusy(null);
        }
    }

    if (!user) {
        return null;
    }

    if (loading) {
        return <div className="loading">Loading room...</div>;
    }

    if (!room) {
        return (
            <div className={styles.notMember}>
                <p>You're not a member of this room.</p>
                {roomId && (
                    <Button variant="primary" size="small" onClick={handleJoin} disabled={joining}>
                        {joining ? "Joining..." : "Try to Join"}
                    </Button>
                )}
                <Button variant="ghost" size="small" onClick={() => navigate("/rooms")}>
                    Back to Rooms
                </Button>
                {toast && <div className={styles.toast}>{toast}</div>}
            </div>
        );
    }

    const isHost = room.viewer_role === "host";
    const isSystem = room.is_system;

    return (
        <div className={styles.roomWrapper}>
            <div className={styles.roomLayout} data-mobile-view={mobileView}>
                <aside className={styles.sidebar}>
                    <div className={styles.sidebarHeader}>
                        <button
                            type="button"
                            className={styles.backButton}
                            onClick={() => {
                                if (mobileView === "members") {
                                    setMobileView("chat");
                                } else {
                                    navigate("/rooms");
                                }
                            }}
                            aria-label={mobileView === "members" ? "Back to chat" : "Back to rooms"}
                        >
                            {"\u2190"}
                        </button>
                        <span className={styles.sidebarTitle}>Members</span>
                        <span className={styles.memberCount}>{members.length}</span>
                    </div>
                    <div className={styles.memberList}>
                        {members.map(m => (
                            <div key={m.user.id} className={styles.memberRow}>
                                <ProfileLink user={m.user} size="small" />
                                {m.role === "host" && <span className={styles.hostBadge}>Host</span>}
                                {isHost && !isSystem && m.user.id !== user.id && m.role !== "host" && (
                                    <button
                                        className={styles.kickBtn}
                                        onClick={() => handleKick(m.user.id)}
                                        disabled={busy === m.user.id}
                                    >
                                        ✕
                                    </button>
                                )}
                            </div>
                        ))}
                    </div>
                    <div className={styles.sidebarFooter}>
                        <Button
                            variant="secondary"
                            size="small"
                            onClick={handleToggleMute}
                            disabled={busy === "mute"}
                            title={room.viewer_muted ? "Unmute notifications" : "Mute notifications"}
                        >
                            {busy === "mute"
                                ? "..."
                                : room.viewer_muted
                                  ? "Unmute notifications"
                                  : "Mute notifications"}
                        </Button>
                        {isSystem ? null : isHost ? (
                            <Button variant="danger" size="small" onClick={handleDelete} disabled={busy === "delete"}>
                                {busy === "delete" ? "Deleting..." : "Delete Room"}
                            </Button>
                        ) : (
                            <Button variant="danger" size="small" onClick={handleLeave} disabled={busy === "self"}>
                                {busy === "self" ? "Leaving..." : "Leave Room"}
                            </Button>
                        )}
                    </div>
                </aside>

                <div className={styles.messageArea}>
                    <div className={styles.roomHeader}>
                        <button
                            type="button"
                            className={styles.mobileMembersBtn}
                            onClick={() => setMobileView("members")}
                            aria-label="Members"
                        >
                            {"\u2630"}
                        </button>
                        <div className={styles.roomHeaderInfo}>
                            <div className={styles.roomTitleRow}>
                                <span className={styles.roomTitle}>{room.name}</span>
                                {room.is_system && <span className={styles.rpBadge}>Staff</span>}
                                {room.is_rp && <span className={styles.rpBadge}>RP</span>}
                            </div>
                            <span className={styles.roomMeta}>
                                {room.member_count ?? room.members.length} members
                                {room.is_public ? " · public" : " · private"}
                            </span>
                        </div>
                    </div>
                    {(room.description || (room.tags && room.tags.length > 0)) && (
                        <div className={styles.roomInfoCollapsible} data-expanded={descExpanded}>
                            <button
                                type="button"
                                className={styles.roomInfoToggle}
                                onClick={() => setDescExpanded(prev => !prev)}
                            >
                                {descExpanded ? "Hide info \u25B2" : "Show info \u25BC"}
                            </button>
                            <div className={styles.roomInfoContent}>
                                {room.description && <div className={styles.roomDescription}>{room.description}</div>}
                                {room.tags && room.tags.length > 0 && (
                                    <div className={styles.roomTags}>
                                        {room.tags.map(t => (
                                            <span key={t} className={styles.roomTag}>
                                                #{t}
                                            </span>
                                        ))}
                                    </div>
                                )}
                            </div>
                        </div>
                    )}

                    <div className={styles.messages} ref={messagesContainerRef} onScroll={handleMessagesScroll}>
                        {hasMore && (
                            <div className={styles.loadMoreBar}>
                                {loadingMore ? "Loading older messages..." : "Scroll up for more"}
                            </div>
                        )}
                        {messages.length === 0 && !hasMore && (
                            <div className={styles.messagesEmpty}>No messages yet. Say hello!</div>
                        )}
                        {messages.map(msg => (
                            <MessageBubble
                                key={msg.id}
                                message={msg}
                                isOwn={msg.sender.id === user.id}
                                highlighted={msg.id === highlightedMsgId}
                                onLightbox={setLightboxSrc}
                                onReply={m =>
                                    setReplyingTo({
                                        id: m.id,
                                        senderName: m.sender.display_name,
                                        bodyPreview: m.body.length > 80 ? m.body.slice(0, 80) + "..." : m.body,
                                    })
                                }
                            />
                        ))}
                        <div ref={messagesEndRef} />
                    </div>
                    <ChatComposer
                        roomId={room.id}
                        draftRecipientId={null}
                        onSent={handleSentMessage}
                        mentionPool={members.map(m => m.user)}
                        replyingTo={replyingTo}
                        onCancelReply={() => setReplyingTo(null)}
                    />
                </div>
            </div>

            {mobileView === "members" && (
                <button
                    type="button"
                    className={styles.mobileBackToChat}
                    onClick={() => setMobileView("chat")}
                    aria-label="Back to chat"
                >
                    {"\u2190 Back to chat"}
                </button>
            )}

            {toast && <div className={styles.toast}>{toast}</div>}
            {lightboxSrc && <Lightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />}
        </div>
    );
}
