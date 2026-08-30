import { type Dispatch, type SetStateAction, useCallback, useMemo } from "react";
import { REALTIME_EVENTS } from "../../api/realtime/events";
import { useRealtimeEvent } from "../../api/realtime/useRealtime";
import {
    groupMembers,
    memberOnlineWeight as resolveMemberOnlineWeight,
    mergePresence,
    sortMembers,
    type MemberGroup,
    type PresenceMap,
} from "../../domain/chat/memberRoster";
import { applyChatMemberUpdateToMembers, applySiteRoleChangeToMembers } from "../../domain/chat/messagePatches";
import type { ChatRoomMember } from "../../types/api";
import { useChatRoomMembers } from "../queries/chat";
import { useRoomScopedOverride } from "./useRoomScopedOverride";

const EMPTY_PRESENCE: PresenceMap = {};

const ROOM_MEMBER_EVENTS = [
    REALTIME_EVENTS.CHAT_MEMBER_JOINED,
    REALTIME_EVENTS.CHAT_MEMBER_LEFT,
    REALTIME_EVENTS.CHAT_MEMBER_UPDATED,
    REALTIME_EVENTS.CHAT_PRESENCE_CHANGED,
] as const;

export interface RoomMembersOptions {
    roomId: string | undefined;
    enabled: boolean;
    viewerId: string | undefined;
    voiceParticipantIds: readonly string[];
    onMemberCountDelta: (delta: number) => void;
}

export interface RoomMembers {
    members: ChatRoomMember[];
    setMembers: Dispatch<SetStateAction<ChatRoomMember[]>>;
    memberGroups: MemberGroup[];
    presenceMapMerged: PresenceMap;
    memberOnlineWeight: (id: string) => number;
    currentMember: ChatRoomMember | null;
}

export function useRoomMembers(options: RoomMembersOptions): RoomMembers {
    const { roomId, enabled, viewerId, voiceParticipantIds, onMemberCountDelta } = options;

    const membersQuery = useChatRoomMembers(roomId ?? "", enabled);
    const membersRefresh = membersQuery.refresh;
    const baseMembers = membersQuery.members;

    const [members, setMembers] = useRoomScopedOverride<ChatRoomMember[]>(roomId, baseMembers);
    const [presenceMap, setPresenceMap] = useRoomScopedOverride<PresenceMap>(roomId, EMPTY_PRESENCE);

    const loadMembers = useCallback(() => {
        membersRefresh();
    }, [membersRefresh]);

    const presenceMapMerged = useMemo(() => mergePresence(baseMembers, presenceMap), [baseMembers, presenceMap]);

    const memberOnlineWeight = useCallback(
        (id: string) => resolveMemberOnlineWeight(presenceMapMerged, id),
        [presenceMapMerged],
    );

    const voiceIdSet = useMemo(() => new Set(voiceParticipantIds), [voiceParticipantIds]);
    const sortedMembers = useMemo(() => sortMembers(members, presenceMapMerged), [members, presenceMapMerged]);
    const memberGroups = useMemo(() => groupMembers(sortedMembers, voiceIdSet), [sortedMembers, voiceIdSet]);

    const currentMember = members.find(m => m.user.id === viewerId) ?? null;

    useRealtimeEvent(ROOM_MEMBER_EVENTS, event => {
        if (!viewerId || event.data.room_id !== roomId) {
            return;
        }

        if (event.type === REALTIME_EVENTS.CHAT_MEMBER_JOINED) {
            loadMembers();
            onMemberCountDelta(1);
            return;
        }

        if (event.type === REALTIME_EVENTS.CHAT_MEMBER_LEFT) {
            const leaverId = event.data.user_id;
            setMembers(prev => prev.filter(m => m.user.id !== leaverId));
            onMemberCountDelta(-1);
            return;
        }

        if (event.type === REALTIME_EVENTS.CHAT_MEMBER_UPDATED) {
            const update = event.data;
            setMembers(prev => applyChatMemberUpdateToMembers(prev, update));
            return;
        }

        const { user_id: presenceUserId, state } = event.data;
        setPresenceMap(prev => {
            const next = { ...prev };
            if (state === "active" || state === "idle") {
                next[presenceUserId] = state;
            } else {
                delete next[presenceUserId];
            }

            return next;
        });
    });

    useRealtimeEvent(REALTIME_EVENTS.ROLE_CHANGED, event => {
        if (!viewerId || !event.data.user_id) {
            return;
        }

        setMembers(prev => applySiteRoleChangeToMembers(prev, event.data));
    });

    return { members, setMembers, memberGroups, presenceMapMerged, memberOnlineWeight, currentMember };
}
