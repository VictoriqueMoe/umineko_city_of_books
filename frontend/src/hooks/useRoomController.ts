import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router";
import { useAuth } from "./useAuth";
import { usePageTitle } from "./usePageTitle";
import type { ChatMessage, ChatRoom } from "../types/api";
import { buildMentionMatcher } from "../domain/mentions";
import { typingNames as resolveTypingNames } from "../domain/chat/memberRoster";
import { isTimeoutActive } from "../domain/chat/roomPolicy";
import { useUserRooms } from "./queries/chat";
import { queryKeys } from "../api/queryKeys";
import {
    useAddChatMessageReaction,
    useDeleteChatRoom,
    useJoinChatRoom,
    useLeaveChatRoom,
    useMarkChatRoomRead,
    usePinChatMessage,
    useRemoveChatMessageReaction,
    useSetChatRoomMuted,
    useUnpinChatMessage,
} from "./mutations/chat";
import { usePresenceReporter } from "./usePresenceReporter";
import { type ReplyTarget } from "../components/chat/ChatComposer/ChatComposer";
import { useWatchParty } from "./useWatchParty";
import { useVoiceChat } from "./useVoiceChat";
import { useSiteInfo } from "./useSiteInfo";
import { REALTIME_EVENTS } from "../api/realtime/events";
import { REALTIME_COMMANDS, sendRealtime } from "../api/realtime/outbound";
import { useRealtimeEvent, useRealtimeStatus } from "../api/realtime/useRealtime";
import { useChatSession } from "./chat/useChatSession";
import { useMessageAnchor } from "./chat/useMessageAnchor";
import { useRoomMembers } from "./chat/useRoomMembers";
import { useRoomModeration } from "./chat/useRoomModeration";
import { useRoomScopedOverride } from "./chat/useRoomScopedOverride";
import { useRoomViewPrefs } from "./chat/useRoomViewPrefs";
import { useRoomWatchPartyInvite } from "./chat/useRoomWatchPartyInvite";

const MAX_ROOM_MESSAGES = 300;
const LEAVE_DELAY_MS = 1500;
const TOAST_MS = 4000;
const TIMEOUT_TICK_MS = 30_000;
const ROOMS_PATH = "/rooms";

const KICKED_MESSAGE = "You were removed from this room";
const ROOM_DELETED_MESSAGE = "This room was deleted by the host";

const ROOM_LIFECYCLE_EVENTS = [
    REALTIME_EVENTS.CHAT_KICKED,
    REALTIME_EVENTS.CHAT_ROOM_DELETED,
    REALTIME_EVENTS.CHAT_ROOM_UPDATED,
] as const;

const PIN_EVENTS = [REALTIME_EVENTS.CHAT_MESSAGE_PINNED, REALTIME_EVENTS.CHAT_MESSAGE_UNPINNED] as const;

export function useRoomController() {
    const { roomId } = useParams<{ roomId: string }>();
    const navigate = useNavigate();
    const { user } = useAuth();
    const qc = useQueryClient();
    const matchesViewerMention = useMemo(() => buildMentionMatcher(user?.username), [user?.username]);
    const realtimeEpoch = useRealtimeStatus();

    const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);
    const [toast, setToast] = useState<string | null>(null);
    const [joining, setJoining] = useState(false);
    const [mobileView, setMobileView] = useState<"members" | "chat">("chat");
    const [replyingTo, setReplyingTo] = useState<ReplyTarget | null>(null);
    const [pinnedOpen, setPinnedOpen] = useState(false);
    const [searchOpen, setSearchOpen] = useState(false);
    const [editProfileOpen, setEditProfileOpen] = useState(false);
    const [inviteModalOpen, setInviteModalOpen] = useState(false);
    const [moderationDialogOpen, setModerationDialogOpen] = useState(false);

    const prefs = useRoomViewPrefs(roomId);

    usePresenceReporter(roomId);

    const watchParty = useWatchParty(roomId ?? null, user?.id ?? null);
    const voiceEnabled = useSiteInfo()?.voice_enabled ?? false;

    const userRoomsQuery = useUserRooms();
    const userRoomsLoading = userRoomsQuery.loading;
    const userRoomsList = userRoomsQuery.rooms;
    const baseRoom = roomId ? (userRoomsList.find(r => r.id === roomId) ?? null) : null;
    const [room, setRoom] = useRoomScopedOverride<ChatRoom | null>(roomId, baseRoom);
    const voiceParticipants = useMemo(() => room?.voice_participants ?? [], [room?.voice_participants]);
    const voice = useVoiceChat(roomId ?? "", voiceParticipants);
    const loading = !!roomId && userRoomsLoading;

    const changeMemberCount = useCallback(
        (delta: number) => {
            setRoom(prev => {
                if (!prev) {
                    return prev;
                }

                const current = prev.member_count ?? prev.members.length;

                return { ...prev, member_count: Math.max(0, current + delta) };
            });
        },
        [setRoom],
    );

    const { members, setMembers, memberGroups, presenceMapMerged, memberOnlineWeight, currentMember } = useRoomMembers({
        roomId,
        enabled: !!roomId && !!room,
        viewerId: user?.id,
        voiceParticipantIds: voice.participantIds,
        onMemberCountDelta: changeMemberCount,
    });

    const viewerTimeoutUntil = currentMember?.timeout_until ?? undefined;

    const [nowTick, setNowTick] = useState(() => Date.now());
    useEffect(() => {
        if (!viewerTimeoutUntil) {
            return;
        }

        const reseed = setTimeout(() => setNowTick(Date.now()), 0);
        const t = setInterval(() => setNowTick(Date.now()), TIMEOUT_TICK_MS);

        return () => {
            clearTimeout(reseed);
            clearInterval(t);
        };
    }, [viewerTimeoutUntil]);

    const viewerTimedOut = isTimeoutActive(viewerTimeoutUntil, nowTick);

    const session = useChatSession({
        roomId: room ? roomId : undefined,
        user,
        maxMessages: MAX_ROOM_MESSAGES,
        sound: {
            enabled: user?.private?.play_message_sound ?? true,
            muted: room?.viewer_muted ?? false,
        },
        onEditError: setToast,
        editLastBlocked: viewerTimedOut,
    });
    const { messages } = session;
    const { setMessages, addMessage, loadUntilMessage, resync } = session.history;

    const moderation = useRoomModeration({ roomId, setMembers, setMessages, notify: setToast });
    const { setBusy } = moderation;

    const { highlightedMsgId, handleJumpToMessage } = useMessageAnchor({
        messages,
        loadUntilMessage,
        onError: setToast,
    });

    const { invitedPartyMissing } = useRoomWatchPartyInvite({
        roomReady: !!room,
        loaded: watchParty.loaded,
        sessions: watchParty.sessions,
        join: watchParty.join,
        onError: setToast,
    });

    usePageTitle(room?.name ?? "Chat Room");

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

    const markReadMutation = useMarkChatRoomRead();
    const markRead = markReadMutation.mutate;
    const lastMarkedReadRef = useRef<string | null>(null);
    const joinRoomMutation = useJoinChatRoom();
    const leaveRoomMutation = useLeaveChatRoom();
    const deleteRoomMutation = useDeleteChatRoom();
    const setMutedMutation = useSetChatRoomMuted();
    const pinMutation = usePinChatMessage(roomId ?? undefined);
    const unpinMutation = useUnpinChatMessage(roomId ?? undefined);
    const addReactionMutation = useAddChatMessageReaction();
    const removeReactionMutation = useRemoveChatMessageReaction();

    const roomReadyForMarking = !!room;
    const roomLastMessageAt = room?.last_message_at;
    useEffect(() => {
        if (!roomId || !roomReadyForMarking) {
            return;
        }

        const signal = `${roomId}:${roomLastMessageAt ?? ""}`;
        if (lastMarkedReadRef.current === signal) {
            return;
        }

        lastMarkedReadRef.current = signal;
        markRead(roomId);
    }, [roomId, roomReadyForMarking, roomLastMessageAt, markRead]);

    const isRoomMember = !!room;

    useEffect(() => {
        if (!roomId || !isRoomMember) {
            return;
        }

        sendRealtime({ type: REALTIME_COMMANDS.JOIN_ROOM, data: { room_id: roomId } });

        return () => {
            sendRealtime({ type: REALTIME_COMMANDS.LEAVE_ROOM, data: { room_id: roomId } });
        };
    }, [roomId, isRoomMember, realtimeEpoch]);

    useRealtimeEvent(ROOM_LIFECYCLE_EVENTS, event => {
        if (!user || event.data.room_id !== roomId) {
            return;
        }

        if (event.type === REALTIME_EVENTS.CHAT_KICKED) {
            const { reason } = event.data;
            setToast(reason ? `${KICKED_MESSAGE}: ${reason}` : KICKED_MESSAGE);
            setTimeout(() => navigate(ROOMS_PATH), LEAVE_DELAY_MS);
            return;
        }

        if (event.type === REALTIME_EVENTS.CHAT_ROOM_DELETED) {
            setToast(ROOM_DELETED_MESSAGE);
            setTimeout(() => navigate(ROOMS_PATH), LEAVE_DELAY_MS);
            return;
        }

        const update = event.data;
        setRoom(prev => {
            if (!prev) {
                return prev;
            }

            return {
                ...prev,
                name: update.name,
                description: update.description,
                tags: update.tags ?? [],
                is_public: update.is_public,
                is_rp: update.is_rp,
            };
        });
        qc.invalidateQueries({ queryKey: queryKeys.chat.rooms() });
        qc.invalidateQueries({ queryKey: queryKeys.chat.roomsList() });
    });

    useRealtimeEvent(PIN_EVENTS, event => {
        if (!user || !roomId || event.data.room_id !== roomId) {
            return;
        }

        qc.invalidateQueries({ queryKey: queryKeys.chat.pinned(roomId) });
    });

    function handleSentMessage(message: ChatMessage) {
        addMessage(message);
        session.scroll.toBottom({ force: true });
    }

    function notifyTyping() {
        if (!roomId) {
            return;
        }

        sendRealtime({ type: REALTIME_COMMANDS.TYPING, data: { room_id: roomId } });
    }

    function backToRooms() {
        navigate(ROOMS_PATH);
    }

    async function handleJoin() {
        if (!roomId) {
            return;
        }
        setJoining(true);
        try {
            const joined = await joinRoomMutation.mutateAsync({ roomId });
            setRoom(joined);
            await watchParty.refresh();
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to join room");
        } finally {
            setJoining(false);
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
            navigate(ROOMS_PATH);
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
            navigate(ROOMS_PATH);
        } catch (err) {
            setToast(err instanceof Error ? err.message : "Failed to delete");
            setBusy(null);
        }
    }

    const addReaction = addReactionMutation.mutateAsync;
    const removeReaction = removeReactionMutation.mutateAsync;
    const handleReactionToggle = useCallback(
        async (message: ChatMessage, emoji: string) => {
            const existing = (message.reactions ?? []).find(r => r.emoji === emoji);
            try {
                if (existing && existing.viewer_reacted) {
                    await removeReaction({ messageId: message.id, emoji });
                } else {
                    await addReaction({ messageId: message.id, emoji });
                }
            } catch (err) {
                setToast(err instanceof Error ? err.message : "Failed to update reaction");
            }
        },
        [addReaction, removeReaction],
    );

    const pin = pinMutation.mutateAsync;
    const unpin = unpinMutation.mutateAsync;
    const handlePinToggle = useCallback(
        async (message: ChatMessage) => {
            try {
                if (message.pinned) {
                    await unpin(message.id);
                } else {
                    await pin(message.id);
                }
            } catch (err) {
                setToast(err instanceof Error ? err.message : "Failed to update pin");
            }
        },
        [pin, unpin],
    );

    const typingNames = resolveTypingNames(session.typing.userIds, members, user?.id);

    return {
        room: {
            data: room,
            id: roomId,
            loading,
            joining,
            viewerTimeoutUntil,
            viewerTimedOut,
            set: setRoom,
            join: handleJoin,
            toggleMute: handleToggleMute,
            leave: handleLeave,
            remove: handleDelete,
            backToRooms,
        },
        session: {
            viewer: user,
            messages,
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
            replyingTo,
            setReplyingTo,
            typingNames,
            notifyTyping,
            matchesViewerMention,
            onSent: handleSentMessage,
            deleteMessage: session.editing.remove,
            editMessage: session.editing.save,
            editLast: session.editing.editLast,
            toggleReaction: handleReactionToggle,
            togglePin: handlePinToggle,
        },
        members: {
            list: members,
            groups: memberGroups,
            presence: presenceMapMerged,
            onlineWeight: memberOnlineWeight,
            current: currentMember,
            set: setMembers,
        },
        moderation,
        prefs: {
            sidebarCollapsed: prefs.sidebarCollapsed,
            toggleSidebar: prefs.toggleSidebar,
            descExpanded: prefs.descExpanded,
            toggleDescExpanded: prefs.toggleDescExpanded,
            mobileView,
            setMobileView,
        },
        anchor: {
            highlightedMsgId,
            jumpTo: handleJumpToMessage,
        },
        voice: { ...voice, enabled: voiceEnabled },
        watchParty: { ...watchParty, invitedPartyMissing },
        panels: {
            pinnedOpen,
            setPinnedOpen,
            searchOpen,
            setSearchOpen,
            lightboxSrc,
            setLightboxSrc,
            editProfileOpen,
            setEditProfileOpen,
            inviteModalOpen,
            setInviteModalOpen,
            moderationDialogOpen,
            setModerationDialogOpen,
        },
        toast: {
            message: toast,
            show: setToast,
        },
    };
}

export type RoomController = ReturnType<typeof useRoomController>;
