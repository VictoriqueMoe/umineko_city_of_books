import { type Dispatch, type SetStateAction, useCallback, useEffect, useRef, useState, useMemo } from "react";
import { useLocation, useNavigate, useParams } from "react-router";
import { useAuth } from "./useAuth";
import { useNotifications } from "./useNotifications";
import { usePageTitle } from "./usePageTitle";
import type { ChatMessage, ChatRoom, ChatRoomMember, User, WSMessage } from "../types/api";
import { buildMentionMatcher } from "../utils/mentions";
import { ROLE_GROUPS, type SiteRole } from "../utils/permissions";
import { parseServerDate } from "../utils/time";
import { useChatRoomMembers, useUserRooms } from "../api/queries/chat";
import {
    useAddChatMessageReaction,
    useBanChatRoomMember,
    useClearChatRoomMemberTimeout,
    useDeleteChatRoom,
    useJoinChatRoom,
    useKickChatRoomMember,
    useLeaveChatRoom,
    useMarkChatRoomRead,
    usePinChatMessage,
    useRemoveChatMessageReaction,
    useSetChatRoomMemberNickname,
    useSetChatRoomMemberTimeout,
    useSetChatRoomMuted,
    useUnlockChatRoomMemberNickname,
    useUnpinChatMessage,
} from "../api/mutations/chat";
import { useChatMessageHandlers } from "./useChatMessageHandlers";
import { useMessageHistory } from "./useMessageHistory";
import { usePresenceReporter } from "./usePresenceReporter";
import { useTypingIndicator } from "./useTypingIndicator";
import { type ReplyTarget } from "../components/chat/ChatComposer/ChatComposer";
import { useWatchParty } from "../components/chat/WatchParty/useWatchParty";
import { useVoiceChat } from "../components/chat/Voice/useVoiceChat";
import { useSiteInfo } from "./useSiteInfo";
import {
    applyChatMemberUpdate,
    applyChatMessagePinned,
    applyChatMessageUnpinned,
    applyLocalMemberChange,
    applyReactionAdded,
    applyReactionRemoved,
    applySharedChatWSBranch,
    ChatMemberUpdatedPayload,
    ChatMessagePinnedPayload,
    ChatMessageUnpinnedPayload,
    ChatReactionPayload,
    handleIncomingChatMessage,
    maybePlayChatMessageSound,
} from "../utils/chatStream";

export function useRoomController() {
    const { roomId } = useParams<{ roomId: string }>();
    const navigate = useNavigate();
    const location = useLocation();
    const { user } = useAuth();
    const matchesViewerMention = useMemo(() => buildMentionMatcher(user?.username), [user?.username]);
    const { addWSListener, sendWSMessage, wsEpoch } = useNotifications();
    const [roomOverride, setRoomOverride] = useState<{ roomId: string | null; room: ChatRoom | null }>({
        roomId: null,
        room: null,
    });
    const [membersOverride, setMembersOverride] = useState<{ roomId: string | null; members: ChatRoomMember[] | null }>(
        {
            roomId: null,
            members: null,
        },
    );
    const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);
    const [toast, setToast] = useState<string | null>(null);
    const [joining, setJoining] = useState(false);
    const [busy, setBusy] = useState<string | null>(null);
    const [mobileView, setMobileView] = useState<"members" | "chat">("chat");
    const [replyingTo, setReplyingTo] = useState<ReplyTarget | null>(null);
    const [editingMessageId, setEditingMessageId] = useState<string | null>(null);
    const [sidebarCollapsedOverride, setSidebarCollapsedOverride] = useState<{
        roomId: string | null;
        value: boolean | null;
    }>({
        roomId: null,
        value: null,
    });
    const sidebarCollapsed = (() => {
        if (!roomId) {
            return false;
        }
        if (sidebarCollapsedOverride.roomId === roomId && sidebarCollapsedOverride.value !== null) {
            return sidebarCollapsedOverride.value;
        }
        return localStorage.getItem(`ut-room-sidebar-collapsed-${roomId}`) === "1";
    })();

    function toggleSidebar() {
        if (!roomId) {
            return;
        }
        const next = !sidebarCollapsed;
        try {
            if (next) {
                localStorage.setItem(`ut-room-sidebar-collapsed-${roomId}`, "1");
            } else {
                localStorage.removeItem(`ut-room-sidebar-collapsed-${roomId}`);
            }
        } catch {
            setSidebarCollapsedOverride({ roomId, value: next });
            return;
        }
        setSidebarCollapsedOverride({ roomId, value: next });
    }
    const [highlightedMsgId, setHighlightedMsgId] = useState<string | null>(null);
    const [presenceState, setPresenceState] = useState<{
        roomId: string | null;
        map: Record<string, "active" | "idle">;
    }>({
        roomId: null,
        map: {},
    });
    const presenceMap = presenceState.roomId === roomId ? presenceState.map : {};
    const setPresenceMap = useCallback(
        (updater: (prev: Record<string, "active" | "idle">) => Record<string, "active" | "idle">) => {
            setPresenceState(prev => {
                const base = prev.roomId === roomId ? prev.map : {};
                return { roomId: roomId ?? null, map: updater(base) };
            });
        },
        [roomId],
    );
    usePresenceReporter({ roomId, sendWSMessage, wsEpoch });
    const { typingUserIds, noteTyping, clearUser: clearTypingUser } = useTypingIndicator(roomId);

    const roomInfoStorageKey = roomId ? `roomInfoExpanded:${roomId}` : null;
    const [descExpandedOverride, setDescExpandedOverride] = useState<{ key: string | null; value: boolean | null }>({
        key: null,
        value: null,
    });
    const descExpanded = (() => {
        if (typeof window === "undefined") {
            return true;
        }
        if (descExpandedOverride.key === roomInfoStorageKey && descExpandedOverride.value !== null) {
            return descExpandedOverride.value;
        }
        if (roomInfoStorageKey) {
            const stored = window.localStorage.getItem(roomInfoStorageKey);
            if (stored !== null) {
                return stored === "true";
            }
        }
        return window.matchMedia("(min-width: 769px)").matches;
    })();

    function toggleDescExpanded() {
        const next = !descExpanded;
        if (roomInfoStorageKey) {
            window.localStorage.setItem(roomInfoStorageKey, next ? "true" : "false");
        }
        setDescExpandedOverride({ key: roomInfoStorageKey, value: next });
    }
    const [pinnedOpen, setPinnedOpen] = useState(false);
    const [searchOpen, setSearchOpen] = useState(false);
    const [pinnedRefreshKey, setPinnedRefreshKey] = useState(0);
    const watchParty = useWatchParty(roomId ?? null, user?.id ?? null);
    const voiceEnabled = useSiteInfo()?.voice_enabled ?? false;
    const [editProfileOpen, setEditProfileOpen] = useState(false);
    const [inviteModalOpen, setInviteModalOpen] = useState(false);
    const [moderationDialogOpen, setModerationDialogOpen] = useState(false);
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
    const roomMutedRef = useRef(false);

    const userRoomsQuery = useUserRooms();
    const userRoomsLoading = userRoomsQuery.loading;
    const userRoomsList = userRoomsQuery.rooms;
    const baseRoom = roomId ? (userRoomsList.find(r => r.id === roomId) ?? null) : null;
    const room = roomOverride.roomId === roomId && roomOverride.room ? roomOverride.room : baseRoom;
    const voice = useVoiceChat(roomId ?? "", room?.voice_participants ?? []);
    const voiceIdSet = new Set(voice.participantIds);
    const loading = !!roomId && userRoomsLoading;
    const setRoom: Dispatch<SetStateAction<ChatRoom | null>> = useCallback(
        updater => {
            setRoomOverride(prev => {
                const baseValue = prev.roomId === roomId && prev.room ? prev.room : null;
                const next =
                    typeof updater === "function"
                        ? (updater as (p: ChatRoom | null) => ChatRoom | null)(baseValue)
                        : updater;
                return { roomId: roomId ?? null, room: next };
            });
        },
        [roomId],
    );

    const membersQuery = useChatRoomMembers(roomId ?? "", !!roomId && !!room);
    const membersRefresh = membersQuery.refresh;
    const baseMembers = membersQuery.members;
    const members =
        membersOverride.roomId === roomId && membersOverride.members ? membersOverride.members : baseMembers;
    const setMembers: Dispatch<SetStateAction<ChatRoomMember[]>> = useCallback(
        updater => {
            setMembersOverride(prev => {
                const baseValue = prev.roomId === roomId && prev.members ? prev.members : baseMembers;
                const next =
                    typeof updater === "function"
                        ? (updater as (p: ChatRoomMember[]) => ChatRoomMember[])(baseValue)
                        : updater;
                return { roomId: roomId ?? null, members: next };
            });
        },
        [roomId, baseMembers],
    );

    const presenceSeed: Record<string, "active" | "idle"> = {};
    for (const m of baseMembers) {
        if (m.presence === "active" || m.presence === "idle") {
            presenceSeed[m.user.id] = m.presence;
        }
    }
    const presenceMapMerged = { ...presenceSeed, ...presenceMap };

    const memberOnlineWeight = (id: string) => {
        const p = presenceMapMerged[id];
        if (p === "active" || p === "idle") {
            return 0;
        }
        return 1;
    };
    const memberRankWeight = (m: ChatRoomMember) => {
        if (m.user.role === "super_admin") {
            return 0;
        }

        if (m.role === "host") {
            return 1;
        }

        const idx = ROLE_GROUPS.findIndex(g => g.role === m.user.role);
        if (idx >= 0) {
            return idx + 1;
        }

        return ROLE_GROUPS.length + 1;
    };
    const memberSortName = (m: ChatRoomMember) => {
        const nickname = m.nickname?.trim();
        if (nickname) {
            return nickname.toLowerCase();
        }
        const displayName = m.user.display_name?.trim();
        if (displayName) {
            return displayName.toLowerCase();
        }
        return m.user.username.toLowerCase();
    };
    const memberRankLabel = (m: ChatRoomMember) => {
        const group = ROLE_GROUPS.find(g => g.role === m.user.role);
        if (m.user.role === "super_admin" && group) {
            return group.label;
        }

        if (m.role === "host") {
            return "Host";
        }

        if (group) {
            return group.label;
        }

        return "Members";
    };
    const sortedMembers = [...members].sort((a, b) => {
        const rank = memberRankWeight(a) - memberRankWeight(b);
        if (rank !== 0) {
            return rank;
        }

        const online = memberOnlineWeight(a.user.id) - memberOnlineWeight(b.user.id);
        if (online !== 0) {
            return online;
        }

        return memberSortName(a).localeCompare(memberSortName(b));
    });
    const memberGroups: { label: string; members: ChatRoomMember[] }[] = [];
    for (const m of sortedMembers) {
        const label = memberRankLabel(m);
        const last = memberGroups[memberGroups.length - 1];
        if (last && last.label === label) {
            last.members.push(m);
        } else {
            memberGroups.push({ label, members: [m] });
        }
    }

    const inVoiceMembers = sortedMembers.filter(m => voiceIdSet.has(m.user.id));
    if (inVoiceMembers.length > 0) {
        memberGroups.unshift({ label: "In Voice", members: inVoiceMembers });
    }

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
    const targetMsgCreatedAt = new URLSearchParams(location.search).get("at") ?? undefined;
    const [handledHash, setHandledHash] = useState<string | null>(null);
    const hashLoadRef = useRef<string | null>(null);
    const pendingTargetMsgId = targetMsgId && handledHash !== targetMsgId ? targetMsgId : null;

    const [nowTick, setNowTick] = useState(() => Date.now());
    useEffect(() => {
        const t = setInterval(() => setNowTick(Date.now()), 30_000);
        return () => clearInterval(t);
    }, []);

    const currentMember = members.find(m => m.user.id === user?.id) ?? null;
    const viewerTimeoutUntil = currentMember?.timeout_until ?? undefined;
    const viewerTimedOutDate = parseServerDate(viewerTimeoutUntil);
    const viewerTimedOut = viewerTimedOutDate ? viewerTimedOutDate.getTime() > nowTick : false;

    const { handleDeleteMessage, handleEditMessage, handleEditLast } = useChatMessageHandlers({
        user,
        messages,
        setMessages,
        setEditingMessageId,
        onError: (msg: string) => setToast(msg),
        editLastBlocked: viewerTimedOut,
    });

    usePageTitle(room?.name ?? "Chat Room");

    useEffect(() => {
        roomIdRef.current = roomId;
    }, [roomId]);

    useEffect(() => {
        roomMutedRef.current = room?.viewer_muted ?? false;
    }, [room?.viewer_muted]);

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
            if (hashLoadRef.current !== pendingTargetMsgId) {
                hashLoadRef.current = pendingTargetMsgId;
                loadUntilMessage(pendingTargetMsgId, targetMsgCreatedAt).then(found => {
                    if (!found) {
                        setHandledHash(pendingTargetMsgId);
                    }
                });
            }
            return;
        }
        const t = setTimeout(() => {
            const el = document.getElementById(`chat-msg-${pendingTargetMsgId}`);
            if (el) {
                el.scrollIntoView({ behavior: "smooth", block: "center" });
                setHighlightedMsgId(pendingTargetMsgId);
                setHandledHash(pendingTargetMsgId);
            }
        }, 300);
        return () => clearTimeout(t);
    }, [pendingTargetMsgId, targetMsgCreatedAt, messages, loadUntilMessage]);

    useEffect(() => {
        if (!highlightedMsgId) {
            return;
        }
        const t = setTimeout(() => setHighlightedMsgId(null), 3000);
        return () => clearTimeout(t);
    }, [highlightedMsgId]);

    const markReadMutation = useMarkChatRoomRead();
    const markRead = markReadMutation.mutate;
    const joinRoomMutation = useJoinChatRoom();
    const leaveRoomMutation = useLeaveChatRoom();
    const deleteRoomMutation = useDeleteChatRoom();
    const setMutedMutation = useSetChatRoomMuted();
    const kickMutation = useKickChatRoomMember(roomId ?? "");
    const banMutation = useBanChatRoomMember(roomId ?? "");
    const setNicknameMutation = useSetChatRoomMemberNickname(roomId ?? "");
    const unlockNicknameMutation = useUnlockChatRoomMemberNickname(roomId ?? "");
    const setTimeoutMutation = useSetChatRoomMemberTimeout(roomId ?? "");
    const clearTimeoutMutation = useClearChatRoomMemberTimeout(roomId ?? "");
    const pinMutation = usePinChatMessage(roomId ?? undefined);
    const unpinMutation = useUnpinChatMessage(roomId ?? undefined);
    const addReactionMutation = useAddChatMessageReaction();
    const removeReactionMutation = useRemoveChatMessageReaction();

    const loadMembers = useCallback(() => {
        membersRefresh();
    }, [membersRefresh]);

    useEffect(() => {
        if (!roomId || !room) {
            return;
        }
        markRead(roomId);
    }, [roomId, room, markRead]);

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
                const added = handleIncomingChatMessage(
                    chatMsg,
                    roomIdRef.current ?? null,
                    setMessages,
                    scrollToBottom,
                );
                if (added && user) {
                    maybePlayChatMessageSound({
                        senderId: chatMsg.sender.id,
                        currentUserId: user.id,
                        roomMuted: roomMutedRef.current,
                        enabled: user.private?.play_message_sound ?? true,
                    });
                }
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
                const data = msg.data as { room_id: string; reason?: string };
                if (data.room_id !== roomIdRef.current) {
                    return;
                }
                const baseMessage = "You were removed from this room";
                setToast(data.reason ? `${baseMessage}: ${data.reason}` : baseMessage);
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
            if (
                applySharedChatWSBranch(msg, {
                    activeRoomId: roomIdRef.current ?? null,
                    setMessages,
                    noteTyping,
                })
            ) {
                return;
            }
            if (msg.type === "chat_message") {
                const chatMsg = msg.data as ChatMessage;
                if (chatMsg.room_id === roomIdRef.current) {
                    clearTypingUser(chatMsg.sender.id);
                }
            }
            if (msg.type === "role_changed") {
                const data = msg.data as { user_id?: string; role?: string };
                if (!data.user_id) {
                    return;
                }
                const newRole = (data.role ?? "") as SiteRole;
                setMembers(prev =>
                    prev.map(m => {
                        if (m.user.id !== data.user_id) {
                            return m;
                        }
                        return { ...m, user: { ...m.user, role: newRole || undefined } };
                    }),
                );
                setMessages(prev =>
                    prev.map(m => {
                        if (m.sender.id !== data.user_id) {
                            return m;
                        }
                        return { ...m, sender: { ...m.sender, role: newRole || undefined } };
                    }),
                );
            }
        });
    }, [
        user,
        addWSListener,
        scrollToBottom,
        setMessages,
        navigate,
        loadMembers,
        noteTyping,
        clearTypingUser,
        setMembers,
        setPresenceMap,
        setRoom,
    ]);

    function handleSentMessage(message: ChatMessage) {
        addMessage(message);
        scrollToBottom({ force: true });
    }

    async function handleJoin() {
        if (!roomId) {
            return;
        }
        setJoining(true);
        try {
            const joined = await joinRoomMutation.mutateAsync({ roomId });
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
        const parsed = parseServerDate(value);
        if (!parsed) {
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
            const updated = await setNicknameMutation.mutateAsync({
                userId: nicknameDialogTarget.user.id,
                nickname: nicknameDialogValue.trim(),
            });
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
            const updated = await unlockNicknameMutation.mutateAsync(targetId);
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
            const updated = await setTimeoutMutation.mutateAsync({
                userId: timeoutDialogTarget.user.id,
                amount,
                unit: timeoutDialogUnit,
            });
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
            const updated = await clearTimeoutMutation.mutateAsync(targetId);
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
            await kickMutation.mutateAsync(targetId);
            setMembers(prev => prev.filter(m => m.user.id !== targetId));
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to kick");
        } finally {
            setBusy(null);
        }
    }

    async function handleBan(targetId: string) {
        if (!roomId) {
            return;
        }
        const reason = window.prompt(
            "Ban this member from the room? They will not be able to rejoin or see the room. Optional reason:",
            "",
        );
        if (reason === null) {
            return;
        }
        setBusy(targetId);
        try {
            await banMutation.mutateAsync({ userId: targetId, reason });
            setMembers(prev => prev.filter(m => m.user.id !== targetId));
            setToast("Member banned from the room.");
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to ban");
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
            await setMutedMutation.mutateAsync({ roomId, muted: next });
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
            await leaveRoomMutation.mutateAsync(roomId);
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
            await deleteRoomMutation.mutateAsync(roomId);
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
                await removeReactionMutation.mutateAsync({ messageId: message.id, emoji });
            } else {
                await addReactionMutation.mutateAsync({ messageId: message.id, emoji });
            }
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to update reaction");
        }
    }

    async function handlePinToggle(message: ChatMessage) {
        try {
            if (message.pinned) {
                await unpinMutation.mutateAsync(message.id);
            } else {
                await pinMutation.mutateAsync(message.id);
            }
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to update pin");
        }
    }

    async function handleJumpToMessage(messageId: string, targetCreatedAt?: string) {
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
        const found = await loadUntilMessage(messageId, targetCreatedAt);
        if (!found) {
            setToast("Couldn't locate that message.");
            return;
        }
        requestAnimationFrame(() => scrollToEl(false));
        setTimeout(() => scrollToEl(false), 300);
        setTimeout(() => scrollToEl(true), 600);
    }

    const typingNames = typingUserIds
        .filter(id => id !== user?.id)
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
        });

    return {
        user,
        navigate,
        loading,
        room,
        roomId,
        members,
        memberGroups,
        presenceMapMerged,
        memberOnlineWeight,
        currentMember,
        mobileView,
        setMobileView,
        sidebarCollapsed,
        toggleSidebar,
        descExpanded,
        toggleDescExpanded,
        messages,
        hasMore,
        loadingMore,
        messagesContainerRef,
        messagesEndRef,
        handleMessagesScroll,
        scrollToBottom,
        highlightedMsgId,
        matchesViewerMention,
        typingNames,
        voice,
        voiceEnabled,
        watchParty,
        replyingTo,
        setReplyingTo,
        editingMessageId,
        setEditingMessageId,
        viewerTimeoutUntil,
        viewerTimedOut,
        lightboxSrc,
        setLightboxSrc,
        toast,
        setToast,
        busy,
        joining,
        sendWSMessage,
        pinnedOpen,
        setPinnedOpen,
        searchOpen,
        setSearchOpen,
        pinnedRefreshKey,
        editProfileOpen,
        setEditProfileOpen,
        inviteModalOpen,
        setInviteModalOpen,
        moderationDialogOpen,
        setModerationDialogOpen,
        openMemberMenu,
        setOpenMemberMenu,
        setMembers,
        nicknameDialogTarget,
        setNicknameDialogTarget,
        nicknameDialogValue,
        setNicknameDialogValue,
        nicknameDialogError,
        nicknameDialogSaving,
        timeoutDialogTarget,
        setTimeoutDialogTarget,
        timeoutDialogAmount,
        setTimeoutDialogAmount,
        timeoutDialogUnit,
        setTimeoutDialogUnit,
        timeoutDialogError,
        timeoutDialogSaving,
        formatTimeoutUntil,
        openNicknameDialog,
        openTimeoutDialog,
        handleSentMessage,
        handleJoin,
        handleModSetNickname,
        handleModUnlockNickname,
        handleSetTimeout,
        handleClearTimeout,
        handleKick,
        handleBan,
        handleToggleMute,
        handleLeave,
        handleDelete,
        handleReactionToggle,
        handlePinToggle,
        handleJumpToMessage,
        handleDeleteMessage,
        handleEditMessage,
        handleEditLast,
    };
}

export type RoomController = ReturnType<typeof useRoomController>;
