import { useCallback, useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { typingNames as resolveTypingNames } from "../domain/chat/memberRoster";
import { isTimeoutActive } from "../domain/chat/roomPolicy";
import { queryKeys } from "../api/queryKeys";
import { useDeleteChatRoom, useJoinChatRoom, useLeaveChatRoom } from "./mutations/chat";
import { useResetOnChange } from "./useResetOnChange";
import { useWatchParty } from "./useWatchParty";
import { REALTIME_EVENTS } from "../api/realtime/events";
import { useRealtimeEvent } from "../api/realtime/useRealtime";
import { useChatThread } from "./useChatThread";
import { useMessageAnchor } from "./chat/useMessageAnchor";
import { useRoomMembers } from "./chat/useRoomMembers";
import { useRoomModeration } from "./chat/useRoomModeration";
import { useRoomViewPrefs } from "./chat/useRoomViewPrefs";
import { useRoomWatchPartyInvite } from "./chat/useRoomWatchPartyInvite";
import type { RoomInfoTab } from "../components/chat/RoomInfoPanel/RoomInfoPanel";

const MAX_ROOM_MESSAGES = 300;
const LEAVE_DELAY_MS = 1500;
const TIMEOUT_TICK_MS = 30_000;
const ROOMS_PATH = "/rooms";
const DM_PATH = "/chat";

const KICKED_MESSAGE = "You were removed from this room";
const ROOM_DELETED_MESSAGE = "This room was deleted by the host";

const ROOM_LIFECYCLE_EVENTS = [
    REALTIME_EVENTS.CHAT_KICKED,
    REALTIME_EVENTS.CHAT_ROOM_DELETED,
    REALTIME_EVENTS.CHAT_ROOM_UPDATED,
] as const;

const PIN_EVENTS = [REALTIME_EVENTS.CHAT_MESSAGE_PINNED, REALTIME_EVENTS.CHAT_MESSAGE_UNPINNED] as const;

const ATTACHMENT_EVENTS = [
    REALTIME_EVENTS.CHAT_MESSAGE,
    REALTIME_EVENTS.CHAT_MESSAGE_EDITED,
    REALTIME_EVENTS.CHAT_MESSAGE_DELETED,
] as const;

export function useRoomController() {
    const { thread, session, capabilities, toast } = useChatThread({
        maxMessages: MAX_ROOM_MESSAGES,
        requireMembership: true,
    });

    const roomId = thread.id;
    const room = thread.data;
    const setRoom = thread.set;
    const user = session.viewer;
    const setToast = toast.show;

    const navigate = useNavigate();
    const qc = useQueryClient();

    const [joining, setJoining] = useState(false);
    const [mobileView, setMobileView] = useState<"members" | "chat">("chat");
    const [panelTab, setPanelTab] = useState<RoomInfoTab | null>(null);
    const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);
    const [editProfileOpen, setEditProfileOpen] = useState(false);
    const [inviteModalOpen, setInviteModalOpen] = useState(false);
    const [moderationDialogOpen, setModerationDialogOpen] = useState(false);

    useResetOnChange(roomId, () => {
        setJoining(false);
        setPanelTab(null);
        setLightboxSrc(null);
        setEditProfileOpen(false);
        setInviteModalOpen(false);
        setModerationDialogOpen(false);
    });

    const prefs = useRoomViewPrefs(roomId);
    const watchParty = useWatchParty(roomId ?? null, user?.id ?? null);

    const roomIsDM = room?.type === "dm";
    useEffect(() => {
        if (!roomId || !roomIsDM) {
            return;
        }

        navigate(`${DM_PATH}/${roomId}`, { replace: true });
    }, [roomId, roomIsDM, navigate]);

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
        voiceParticipantIds: thread.voice.participantIds,
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

    const { messages, history } = session;
    const { setMessages, loadUntilMessage } = history;

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

    const joinRoomMutation = useJoinChatRoom();
    const leaveRoomMutation = useLeaveChatRoom();
    const deleteRoomMutation = useDeleteChatRoom();

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

    useRealtimeEvent(ATTACHMENT_EVENTS, event => {
        if (!user || !roomId || event.data.room_id !== roomId) {
            return;
        }

        const message = event.data as { media?: unknown[]; body?: string };
        const carriesMedia = Array.isArray(message.media) && message.media.length > 0;
        const carriesLink = typeof message.body === "string" && message.body.includes("http");

        if (carriesMedia) {
            qc.invalidateQueries({ queryKey: queryKeys.chat.attachments(roomId, "media") });
        }
        if (carriesLink) {
            qc.invalidateQueries({ queryKey: queryKeys.chat.attachments(roomId, "links") });
        }
    });

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

    const editLastMessage = session.editLast;
    const handleEditLast = useCallback(() => {
        if (viewerTimedOut) {
            return;
        }

        editLastMessage();
    }, [viewerTimedOut, editLastMessage]);

    const typingNames = resolveTypingNames(session.typingUserIds, members, user?.id);

    return {
        room: {
            data: room,
            id: roomId,
            loading: thread.loading,
            joining,
            viewerTimeoutUntil,
            viewerTimedOut,
            set: setRoom,
            join: handleJoin,
            toggleMute: thread.toggleMute,
            leave: handleLeave,
            remove: handleDelete,
            backToRooms,
        },
        session: {
            viewer: user,
            messages,
            hasMore: session.hasMore,
            loadingMore: session.loadingMore,
            containerRef: session.containerRef,
            contentRef: session.contentRef,
            endRef: session.endRef,
            onScroll: session.onScroll,
            toBottom: session.toBottom,
            editingMessageId: session.editingMessageId,
            startEditing: session.startEditing,
            cancelEditing: session.cancelEditing,
            replyingTo: session.replyingTo,
            setReplyingTo: session.setReplyingTo,
            typingNames,
            notifyTyping: session.notifyTyping,
            matchesViewerMention: session.matchesViewerMention,
            onSent: session.onSent,
            deleteMessage: session.deleteMessage,
            editMessage: session.editMessage,
            editLast: handleEditLast,
            toggleReaction: session.toggleReaction,
            togglePin: session.togglePin,
        },
        members: {
            list: members,
            groups: memberGroups,
            presence: presenceMapMerged,
            onlineWeight: memberOnlineWeight,
            current: currentMember,
            set: setMembers,
        },
        moderation: {
            ...moderation,
            busy: thread.mutePending ? "mute" : moderation.busy,
        },
        capabilities,
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
        voice: thread.voice,
        watchParty: { ...watchParty, invitedPartyMissing },
        panels: {
            panelTab,
            openPanel: setPanelTab,
            closePanel: () => setPanelTab(null),
            lightboxSrc,
            setLightboxSrc,
            editProfileOpen,
            setEditProfileOpen,
            inviteModalOpen,
            setInviteModalOpen,
            moderationDialogOpen,
            setModerationDialogOpen,
        },
        toast,
    };
}

export type RoomController = ReturnType<typeof useRoomController>;
