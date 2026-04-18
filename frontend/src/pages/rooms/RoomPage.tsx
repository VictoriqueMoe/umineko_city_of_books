import { useCallback, useEffect, useRef, useState } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { useNotifications } from "../../hooks/useNotifications";
import { usePageTitle } from "../../hooks/usePageTitle";
import type { ChatMessage, ChatRoom, ChatRoomMember, User, WSMessage } from "../../types/api";
import { isSiteStaff } from "../../utils/permissions";
import {
    addChatMessageReaction,
    clearChatRoomMemberTimeout,
    deleteChatMessage,
    deleteChatRoom,
    editChatMessage,
    getChatRoomMembers,
    getUserRooms,
    joinChatRoom,
    kickChatRoomMember,
    leaveChatRoom,
    markChatRoomRead,
    pinChatMessage,
    removeChatMessageReaction,
    setChatRoomMemberNickname,
    setChatRoomMemberTimeout,
    setChatRoomMuted,
    unlockChatRoomMemberNickname,
    unpinChatMessage,
} from "../../api/endpoints";
import { useMessageHistory } from "../../hooks/useMessageHistory";
import { usePresenceReporter } from "../../hooks/usePresenceReporter";
import { useTypingIndicator } from "../../hooks/useTypingIndicator";
import { TypingIndicator } from "../../components/chat/TypingIndicator/TypingIndicator";
import { Button } from "../../components/Button/Button";
import { ChatComposer, type ReplyTarget } from "../../components/chat/ChatComposer/ChatComposer";
import { EditRoomProfileDialog } from "../../components/chat/EditRoomProfileDialog/EditRoomProfileDialog";
import { InviteMembersModal } from "../../components/chat/InviteMembersModal/InviteMembersModal";
import { MessageBubble } from "../../components/chat/MessageBubble/MessageBubble";
import { PinnedMessagesPanel } from "../../components/chat/PinnedMessagesPanel/PinnedMessagesPanel";
import { Lightbox } from "../../components/Lightbox/Lightbox";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import {
    applyChatMemberUpdate,
    applyChatMessageDeleted,
    applyChatMessageEdited,
    applyChatMessagePinned,
    applyChatMessageUnpinned,
    applyLocalMemberChange,
    applyReactionAdded,
    applyReactionRemoved,
    ChatMemberUpdatedPayload,
    ChatMessageDeletedPayload,
    ChatMessagePinnedPayload,
    ChatMessageUnpinnedPayload,
    ChatReactionPayload,
    handleIncomingChatMessage,
} from "../../utils/chatStream";
import styles from "./RoomPage.module.css";

export function RoomPage() {
    const { roomId } = useParams<{ roomId: string }>();
    const navigate = useNavigate();
    const location = useLocation();
    const { user } = useAuth();
    const { addWSListener, sendWSMessage, wsEpoch } = useNotifications();
    const [room, setRoom] = useState<ChatRoom | null>(null);
    const [members, setMembers] = useState<ChatRoomMember[]>([]);
    const [loading, setLoading] = useState(true);
    const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);
    const [toast, setToast] = useState<string | null>(null);
    const [joining, setJoining] = useState(false);
    const [busy, setBusy] = useState<string | null>(null);
    const [mobileView, setMobileView] = useState<"members" | "chat">("chat");
    const [replyingTo, setReplyingTo] = useState<ReplyTarget | null>(null);
    const [editingMessageId, setEditingMessageId] = useState<string | null>(null);
    const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

    useEffect(() => {
        if (!roomId) {
            setSidebarCollapsed(false);
            return;
        }
        const stored = localStorage.getItem(`ut-room-sidebar-collapsed-${roomId}`);
        setSidebarCollapsed(stored === "1");
    }, [roomId]);

    function toggleSidebar() {
        if (!roomId) {
            return;
        }
        setSidebarCollapsed(prev => {
            const next = !prev;
            try {
                if (next) {
                    localStorage.setItem(`ut-room-sidebar-collapsed-${roomId}`, "1");
                } else {
                    localStorage.removeItem(`ut-room-sidebar-collapsed-${roomId}`);
                }
            } catch {
                // storage unavailable, in-memory state still works for the session
            }
            return next;
        });
    }
    const [highlightedMsgId, setHighlightedMsgId] = useState<string | null>(null);
    const [presenceMap, setPresenceMap] = useState<Record<string, "active" | "idle">>({});
    usePresenceReporter({ roomId, sendWSMessage, wsEpoch });
    const { typingUserIds, noteTyping, clearUser: clearTypingUser, reset: resetTyping } = useTypingIndicator();

    useEffect(() => {
        setPresenceMap({});
        resetTyping();
    }, [roomId, resetTyping]);
    const [descExpanded, setDescExpanded] = useState(false);
    const [pinnedOpen, setPinnedOpen] = useState(false);
    const [pinnedRefreshKey, setPinnedRefreshKey] = useState(0);
    const [editProfileOpen, setEditProfileOpen] = useState(false);
    const [inviteModalOpen, setInviteModalOpen] = useState(false);
    const [openMemberMenu, setOpenMemberMenu] = useState<string | null>(null);
    const [nicknameDialogTarget, setNicknameDialogTarget] = useState<ChatRoomMember | null>(null);
    const [nicknameDialogValue, setNicknameDialogValue] = useState("");
    const [nicknameDialogError, setNicknameDialogError] = useState<string>("");
    const [nicknameDialogSaving, setNicknameDialogSaving] = useState(false);
    const [timeoutDialogTarget, setTimeoutDialogTarget] = useState<ChatRoomMember | null>(null);
    const [timeoutDialogAmount, setTimeoutDialogAmount] = useState("1");
    const [timeoutDialogUnit, setTimeoutDialogUnit] = useState("hours");
    const [timeoutDialogError, setTimeoutDialogError] = useState("");
    const [timeoutDialogSaving, setTimeoutDialogSaving] = useState(false);
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
        loadUntilMessage,
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
            const list = res.members ?? [];
            setMembers(list);
            setPresenceMap(prev => {
                const next = { ...prev };
                for (const m of list) {
                    if (m.presence === "active" || m.presence === "idle") {
                        next[m.user.id] = m.presence;
                    }
                }
                return next;
            });
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
    }, [roomId, sendWSMessage, wsEpoch]);

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
            if (msg.type === "chat_member_updated") {
                const data = msg.data as ChatMemberUpdatedPayload;
                if (data.room_id !== roomIdRef.current) {
                    return;
                }
                applyChatMemberUpdate(data, setMembers, setMessages);
                return;
            }
            if (msg.type === "chat_message_pinned") {
                const data = msg.data as ChatMessagePinnedPayload;
                if (data.room_id !== roomIdRef.current) {
                    return;
                }
                applyChatMessagePinned(data, setMessages);
                setPinnedRefreshKey(k => k + 1);
                return;
            }
            if (msg.type === "chat_message_unpinned") {
                const data = msg.data as ChatMessageUnpinnedPayload;
                if (data.room_id !== roomIdRef.current) {
                    return;
                }
                applyChatMessageUnpinned(data, setMessages);
                setPinnedRefreshKey(k => k + 1);
                return;
            }
            if (msg.type === "chat_reaction_added") {
                const data = msg.data as ChatReactionPayload;
                if (data.room_id !== roomIdRef.current) {
                    return;
                }
                applyReactionAdded(data, user.id, setMessages);
                return;
            }
            if (msg.type === "chat_reaction_removed") {
                const data = msg.data as ChatReactionPayload;
                if (data.room_id !== roomIdRef.current) {
                    return;
                }
                applyReactionRemoved(data, user.id, setMessages);
                return;
            }
            if (msg.type === "chat_message_deleted") {
                const data = msg.data as ChatMessageDeletedPayload;
                if (data.room_id !== roomIdRef.current) {
                    return;
                }
                applyChatMessageDeleted(data, setMessages);
                return;
            }
            if (msg.type === "chat_message_edited") {
                const updated = msg.data as ChatMessage;
                if (updated.room_id !== roomIdRef.current) {
                    return;
                }
                applyChatMessageEdited(updated, setMessages);
                return;
            }
            if (msg.type === "chat_presence_changed") {
                const data = msg.data as { room_id: string; user_id: string; state: string };
                if (data.room_id !== roomIdRef.current) {
                    return;
                }
                setPresenceMap(prev => {
                    const next = { ...prev };
                    if (data.state === "active" || data.state === "idle") {
                        next[data.user_id] = data.state;
                    } else {
                        delete next[data.user_id];
                    }
                    return next;
                });
                return;
            }
            if (msg.type === "typing") {
                const data = msg.data as { room_id: string; user_id: string };
                if (data.room_id !== roomIdRef.current) {
                    return;
                }
                noteTyping(data.user_id);
                return;
            }
            if (msg.type === "chat_message") {
                const chatMsg = msg.data as ChatMessage;
                if (chatMsg.room_id === roomIdRef.current) {
                    clearTypingUser(chatMsg.sender.id);
                }
            }
        });
    }, [user, addWSListener, scrollToBottom, setMessages, navigate, loadMembers, noteTyping, clearTypingUser]);

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

    function openNicknameDialog(member: ChatRoomMember) {
        setNicknameDialogTarget(member);
        setNicknameDialogValue(member.nickname ?? "");
        setNicknameDialogError("");
        setOpenMemberMenu(null);
    }

    function openTimeoutDialog(member: ChatRoomMember) {
        setTimeoutDialogTarget(member);
        setTimeoutDialogAmount("1");
        setTimeoutDialogUnit("hours");
        setTimeoutDialogError("");
        setOpenMemberMenu(null);
    }

    function formatTimeoutUntil(value?: string): string {
        if (!value) {
            return "";
        }
        const parsed = new Date(value);
        if (Number.isNaN(parsed.getTime())) {
            return value;
        }
        return parsed.toLocaleString();
    }

    async function handleModSetNickname() {
        if (!roomId || !nicknameDialogTarget) {
            return;
        }
        setNicknameDialogSaving(true);
        setNicknameDialogError("");
        try {
            const updated = await setChatRoomMemberNickname(
                roomId,
                nicknameDialogTarget.user.id,
                nicknameDialogValue.trim(),
            );
            applyLocalMemberChange(updated, setMembers, setMessages);
            setNicknameDialogTarget(null);
        } catch (err) {
            setNicknameDialogError(err instanceof Error ? err.message : "Failed to set nickname");
        } finally {
            setNicknameDialogSaving(false);
        }
    }

    async function handleModUnlockNickname(targetId: string) {
        if (!roomId) {
            return;
        }
        setBusy(targetId);
        setOpenMemberMenu(null);
        try {
            const updated = await unlockChatRoomMemberNickname(roomId, targetId);
            applyLocalMemberChange(updated, setMembers, setMessages);
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to unlock nickname");
        } finally {
            setBusy(null);
        }
    }

    async function handleSetTimeout() {
        if (!roomId || !timeoutDialogTarget) {
            return;
        }
        const amount = Number(timeoutDialogAmount);
        if (!Number.isInteger(amount) || amount <= 0) {
            setTimeoutDialogError("Enter a whole number greater than zero");
            return;
        }

        setTimeoutDialogSaving(true);
        setTimeoutDialogError("");
        try {
            const updated = await setChatRoomMemberTimeout(
                roomId,
                timeoutDialogTarget.user.id,
                amount,
                timeoutDialogUnit,
            );
            setMembers(prev => prev.map(m => (m.user.id === updated.user.id ? updated : m)));
            setTimeoutDialogTarget(null);
        } catch (err) {
            setTimeoutDialogError(err instanceof Error ? err.message : "Failed to set timeout");
        } finally {
            setTimeoutDialogSaving(false);
        }
    }

    async function handleClearTimeout(targetId: string) {
        if (!roomId) {
            return;
        }
        setBusy(`timeout:${targetId}`);
        setOpenMemberMenu(null);
        try {
            const updated = await clearChatRoomMemberTimeout(roomId, targetId);
            setMembers(prev => prev.map(m => (m.user.id === updated.user.id ? updated : m)));
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to clear timeout");
        } finally {
            setBusy(null);
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

    async function handleReactionToggle(message: ChatMessage, emoji: string) {
        const existing = (message.reactions ?? []).find(r => r.emoji === emoji);
        try {
            if (existing && existing.viewer_reacted) {
                await removeChatMessageReaction(message.id, emoji);
            } else {
                await addChatMessageReaction(message.id, emoji);
            }
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to update reaction");
        }
    }

    async function handleDeleteMessage(message: ChatMessage) {
        try {
            await deleteChatMessage(message.id);
            setMessages(prev => prev.filter(m => m.id !== message.id));
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to delete message");
        }
    }

    async function handleEditMessage(message: ChatMessage, newBody: string) {
        try {
            const updated = await editChatMessage(message.id, newBody);
            applyChatMessageEdited(updated, setMessages);
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to edit message");
            throw err;
        }
    }

    function handleEditLast() {
        if (!user || viewerTimedOut) {
            return;
        }
        for (let i = messages.length - 1; i >= 0; i--) {
            const candidate = messages[i];
            if (candidate.sender.id === user.id && !candidate.is_system) {
                setEditingMessageId(candidate.id);
                requestAnimationFrame(() => {
                    const el = document.getElementById(`chat-msg-${candidate.id}`);
                    if (el) {
                        el.scrollIntoView({ behavior: "smooth", block: "center" });
                    }
                });
                return;
            }
        }
    }

    async function handlePinToggle(message: ChatMessage) {
        try {
            if (message.pinned) {
                await unpinChatMessage(message.id);
            } else {
                await pinChatMessage(message.id);
            }
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to update pin");
        }
    }

    async function handleJumpToMessage(messageId: string) {
        const scrollToEl = (smooth: boolean) => {
            const el = document.getElementById(`chat-msg-${messageId}`);
            if (el) {
                el.scrollIntoView({ behavior: smooth ? "smooth" : "auto", block: "center" });
                setHighlightedMsgId(messageId);
            }
        };
        if (messages.some(m => m.id === messageId)) {
            scrollToEl(true);
            return;
        }
        const found = await loadUntilMessage(messageId);
        if (!found) {
            setToast("Couldn't locate that message.");
            return;
        }
        requestAnimationFrame(() => scrollToEl(false));
        setTimeout(() => scrollToEl(false), 300);
        setTimeout(() => scrollToEl(true), 600);
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
    const isSiteMod = isSiteStaff(user.role);
    const canModerateRoom = isHost || isSiteMod;
    const currentMember = members.find(m => m.user.id === user.id) ?? null;
    const viewerTimeoutUntil = currentMember?.timeout_until ?? undefined;
    const viewerTimedOut = viewerTimeoutUntil ? new Date(viewerTimeoutUntil).getTime() > Date.now() : false;

    return (
        <div className={styles.roomWrapper}>
            <div
                className={styles.roomLayout}
                data-mobile-view={mobileView}
                data-sidebar-collapsed={sidebarCollapsed ? "true" : "false"}
            >
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
                        {isHost && !isSystem && (
                            <button
                                type="button"
                                className={styles.inviteButton}
                                onClick={() => setInviteModalOpen(true)}
                            >
                                + Invite
                            </button>
                        )}
                        <button
                            type="button"
                            className={styles.sidebarCollapseBtn}
                            onClick={toggleSidebar}
                            aria-label="Hide members"
                            data-tooltip="Hide members"
                        >
                            {"\u25C0"}
                        </button>
                    </div>
                    <div className={styles.memberList}>
                        {members.map(m => {
                            const effectiveUser: User = {
                                ...m.user,
                                display_name:
                                    m.nickname && m.nickname.trim() !== ""
                                        ? m.nickname
                                        : m.user.display_name && m.user.display_name.trim() !== ""
                                          ? m.user.display_name
                                          : m.user.username,
                                avatar_url:
                                    m.member_avatar_url && m.member_avatar_url.trim() !== ""
                                        ? m.member_avatar_url
                                        : m.user.avatar_url,
                            };
                            const isSelf = m.user.id === user.id;
                            const targetIsSiteMod = isSiteStaff(m.user.role);
                            const targetIsHost = m.role === "host";
                            const timeoutIsActive = Boolean(m.timeout_until);
                            const canKickTarget =
                                canModerateRoom && !isSystem && !isSelf && !targetIsHost && !targetIsSiteMod;
                            const canEditTargetNickname = isSiteMod && !targetIsSiteMod && !isSelf && !isSystem;
                            let canTimeoutTarget = false;
                            if (canModerateRoom && !isSystem && !isSelf && !targetIsSiteMod) {
                                if (isSiteMod) {
                                    canTimeoutTarget = true;
                                } else {
                                    canTimeoutTarget = !targetIsHost;
                                }
                            }
                            if (canTimeoutTarget && timeoutIsActive && m.timeout_set_by_staff && !isSiteMod) {
                                canTimeoutTarget = false;
                            }
                            const canClearTimeoutTarget =
                                canModerateRoom &&
                                !isSystem &&
                                timeoutIsActive &&
                                (isSiteMod || !m.timeout_set_by_staff);
                            const canActOnMember =
                                canKickTarget || canEditTargetNickname || canTimeoutTarget || canClearTimeoutTarget;
                            const menuOpen = openMemberMenu === m.user.id;
                            const presence = presenceMap[m.user.id];
                            const presenceClass =
                                presence === "active"
                                    ? styles.presenceActive
                                    : presence === "idle"
                                      ? styles.presenceIdle
                                      : styles.presenceAway;
                            const presenceTitle =
                                presence === "active"
                                    ? "Active in this room"
                                    : presence === "idle"
                                      ? "Idle or tab in background"
                                      : "Not currently viewing";
                            return (
                                <div key={m.user.id} className={styles.memberRow}>
                                    <span
                                        className={`${styles.presenceDot} ${presenceClass}`}
                                        title={presenceTitle}
                                        aria-label={presenceTitle}
                                    />
                                    <ProfileLink user={effectiveUser} size="small" />
                                    {m.role === "host" && <span className={styles.hostBadge}>Host</span>}
                                    {m.ghost && (
                                        <span
                                            className={styles.ghostBadge}
                                            title="Ghost member — not visible to non-staff"
                                        >
                                            {"\u{1F47B}"}
                                        </span>
                                    )}
                                    {timeoutIsActive && (
                                        <span
                                            className={styles.timeoutIcon}
                                            title={`Timed out until ${formatTimeoutUntil(m.timeout_until)}`}
                                            aria-label={`Timed out until ${formatTimeoutUntil(m.timeout_until)}`}
                                        >
                                            {"\u23F1"}
                                        </span>
                                    )}
                                    {isSelf && (
                                        <button
                                            type="button"
                                            className={styles.editSelfBtn}
                                            onClick={() => setEditProfileOpen(true)}
                                            title="Edit profile in this room"
                                            aria-label="Edit profile in this room"
                                        >
                                            {"\u270E"}
                                        </button>
                                    )}
                                    {canActOnMember && (
                                        <div className={styles.memberActions}>
                                            <button
                                                type="button"
                                                className={styles.modActionsBtn}
                                                onClick={() =>
                                                    setOpenMemberMenu(prev => (prev === m.user.id ? null : m.user.id))
                                                }
                                                aria-label="Moderator actions"
                                                title="Moderator actions"
                                            >
                                                {"\u22EE"}
                                            </button>
                                            {menuOpen && (
                                                <div
                                                    className={styles.modActionsMenu}
                                                    onMouseLeave={() => setOpenMemberMenu(null)}
                                                >
                                                    {canEditTargetNickname && (
                                                        <button type="button" onClick={() => openNicknameDialog(m)}>
                                                            Change nickname
                                                        </button>
                                                    )}
                                                    {canEditTargetNickname && m.nickname_locked && (
                                                        <button
                                                            type="button"
                                                            onClick={() => handleModUnlockNickname(m.user.id)}
                                                            disabled={busy === m.user.id}
                                                        >
                                                            Reset/unlock nickname
                                                        </button>
                                                    )}
                                                    {canKickTarget && (
                                                        <button
                                                            type="button"
                                                            className={styles.danger}
                                                            onClick={() => {
                                                                setOpenMemberMenu(null);
                                                                handleKick(m.user.id);
                                                            }}
                                                            disabled={busy === m.user.id}
                                                        >
                                                            Kick member
                                                        </button>
                                                    )}
                                                    {canTimeoutTarget && (
                                                        <button type="button" onClick={() => openTimeoutDialog(m)}>
                                                            Set timeout
                                                        </button>
                                                    )}
                                                    {canClearTimeoutTarget && (
                                                        <button
                                                            type="button"
                                                            onClick={() => handleClearTimeout(m.user.id)}
                                                            disabled={busy === `timeout:${m.user.id}`}
                                                        >
                                                            Remove timeout
                                                        </button>
                                                    )}
                                                </div>
                                            )}
                                        </div>
                                    )}
                                </div>
                            );
                        })}
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
                        {!isSystem && canModerateRoom && (
                            <Button variant="danger" size="small" onClick={handleDelete} disabled={busy === "delete"}>
                                {busy === "delete" ? "Deleting..." : "Delete Room"}
                            </Button>
                        )}
                        {!isSystem && !isHost && (
                            <Button variant="danger" size="small" onClick={handleLeave} disabled={busy === "self"}>
                                {busy === "self" ? "Leaving..." : "Leave Room"}
                            </Button>
                        )}
                    </div>
                </aside>

                {sidebarCollapsed && (
                    <button
                        type="button"
                        className={styles.sidebarExpandRail}
                        onClick={toggleSidebar}
                        aria-label="Show members"
                        title="Show members"
                    >
                        {"\u25B6"}
                    </button>
                )}
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
                        <button
                            type="button"
                            className={styles.pinHeaderBtn}
                            onClick={() => setPinnedOpen(true)}
                            aria-label="Pinned messages"
                            title="Pinned messages"
                        >
                            {"\u{1F4CC}"}
                        </button>
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
                                onReactionToggle={handleReactionToggle}
                                onPinToggle={canModerateRoom ? handlePinToggle : undefined}
                                onDelete={handleDeleteMessage}
                                onEdit={handleEditMessage}
                                onEditStart={m => setEditingMessageId(m.id)}
                                onEditCancel={() => setEditingMessageId(null)}
                                editing={editingMessageId === msg.id}
                                canPin={canModerateRoom}
                                canModerate={canModerateRoom}
                                canReact={!viewerTimedOut}
                                canEdit={!viewerTimedOut}
                                senderIsStaff={isSiteStaff(msg.sender.role)}
                            />
                        ))}
                        <div ref={messagesEndRef} />
                    </div>
                    <TypingIndicator
                        names={typingUserIds
                            .filter(id => id !== user.id)
                            .map(id => {
                                const m = members.find(mem => mem.user.id === id);
                                if (!m) {
                                    return "Someone";
                                }
                                if (m.nickname && m.nickname.trim() !== "") {
                                    return m.nickname;
                                }
                                if (m.user.display_name && m.user.display_name.trim() !== "") {
                                    return m.user.display_name;
                                }
                                return m.user.username;
                            })}
                    />
                    <ChatComposer
                        roomId={room.id}
                        draftRecipientId={null}
                        onSent={handleSentMessage}
                        mentionPool={members.map(m => m.user)}
                        replyingTo={replyingTo}
                        onCancelReply={() => setReplyingTo(null)}
                        onTyping={() => sendWSMessage({ type: "typing", data: { room_id: room.id } })}
                        onEditLast={handleEditLast}
                        timeoutUntil={viewerTimeoutUntil}
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

            <PinnedMessagesPanel
                roomId={room.id}
                isOpen={pinnedOpen}
                onClose={() => setPinnedOpen(false)}
                onJump={handleJumpToMessage}
                canUnpin={canModerateRoom}
                refreshKey={pinnedRefreshKey}
            />

            <EditRoomProfileDialog
                isOpen={editProfileOpen}
                roomId={room.id}
                currentMember={currentMember}
                onClose={() => setEditProfileOpen(false)}
                onSaved={updated => {
                    setMembers(prev => prev.map(m => (m.user.id === updated.user.id ? updated : m)));
                }}
            />

            <InviteMembersModal
                isOpen={inviteModalOpen}
                roomId={room.id}
                existingMemberIds={new Set(members.map(m => m.user.id))}
                onClose={() => setInviteModalOpen(false)}
                onInvited={result => {
                    if (result.invited_count > 0) {
                        setToast(
                            result.invited_count === 1 ? "1 member invited" : `${result.invited_count} members invited`,
                        );
                    } else if (result.skipped_count > 0) {
                        setToast("No one invited (all were already members or blocked)");
                    }
                }}
            />

            {nicknameDialogTarget && (
                <div className={styles.nicknameDialogOverlay} onClick={() => setNicknameDialogTarget(null)}>
                    <div className={styles.nicknameDialog} onClick={e => e.stopPropagation()}>
                        <h3>Change nickname for {nicknameDialogTarget.user.display_name}</h3>
                        <input
                            type="text"
                            value={nicknameDialogValue}
                            maxLength={32}
                            onChange={e => setNicknameDialogValue(e.target.value)}
                            placeholder="Nickname (leave blank to clear)"
                            autoFocus
                        />
                        {nicknameDialogError && <div className={styles.dialogError}>{nicknameDialogError}</div>}
                        <div className={styles.nicknameDialogActions}>
                            <Button
                                variant="ghost"
                                size="small"
                                onClick={() => setNicknameDialogTarget(null)}
                                disabled={nicknameDialogSaving}
                            >
                                Cancel
                            </Button>
                            <Button
                                variant="primary"
                                size="small"
                                onClick={handleModSetNickname}
                                disabled={nicknameDialogSaving}
                            >
                                {nicknameDialogSaving ? "Saving..." : "Save"}
                            </Button>
                        </div>
                    </div>
                </div>
            )}

            {timeoutDialogTarget && (
                <div className={styles.nicknameDialogOverlay} onClick={() => setTimeoutDialogTarget(null)}>
                    <div className={styles.nicknameDialog} onClick={e => e.stopPropagation()}>
                        <h3>Set timeout for {timeoutDialogTarget.user.display_name}</h3>
                        <div className={styles.timeoutDialogRow}>
                            <input
                                type="number"
                                min={1}
                                step={1}
                                value={timeoutDialogAmount}
                                onChange={e => setTimeoutDialogAmount(e.target.value)}
                                autoFocus
                            />
                            <select value={timeoutDialogUnit} onChange={e => setTimeoutDialogUnit(e.target.value)}>
                                <option value="seconds">seconds</option>
                                <option value="hours">hours</option>
                                <option value="weeks">weeks</option>
                                <option value="years">years</option>
                                <option value="decades">decades</option>
                                <option value="centuries">centuries</option>
                            </select>
                        </div>
                        {timeoutDialogError && <div className={styles.dialogError}>{timeoutDialogError}</div>}
                        <div className={styles.nicknameDialogActions}>
                            <Button
                                variant="ghost"
                                size="small"
                                onClick={() => setTimeoutDialogTarget(null)}
                                disabled={timeoutDialogSaving}
                            >
                                Cancel
                            </Button>
                            <Button
                                variant="danger"
                                size="small"
                                onClick={handleSetTimeout}
                                disabled={timeoutDialogSaving}
                            >
                                {timeoutDialogSaving ? "Saving..." : "Set timeout"}
                            </Button>
                        </div>
                    </div>
                </div>
            )}

            {toast && <div className={styles.toast}>{toast}</div>}
            {lightboxSrc && <Lightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />}
        </div>
    );
}
