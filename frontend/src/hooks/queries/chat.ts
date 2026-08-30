import { useQuery } from "@tanstack/react-query";
import {
    getChatRoomMembers,
    getChatRoomPinnedMessages,
    getChatUnreadCount,
    getRoomMessages,
    getRoomMessagesBefore,
    getUserRooms,
    listChatRoomBans,
    listChatRoomBannedWords,
    listMyChatRooms,
    listPublicChatRooms,
    resolveDMRoom,
} from "../../api/endpoints/chat";
import { queryClient } from "../../api/queryClient";
import { queryKeys, type RoomsListParams } from "../../api/queryKeys";
import type { ChatRoom } from "../../types/api";
import { useAuth } from "../useAuth";

export const ROOMS_LIST_PAGE_SIZE = 20;

export interface RoomsListResult {
    rooms: ChatRoom[];
    total: number;
    loading: boolean;
}

function roomsListRequest(params: RoomsListParams) {
    return {
        search: params.search,
        rp: params.rpOnly,
        tag: params.tagFilter || undefined,
        includeArchived: params.includeArchived,
        limit: ROOMS_LIST_PAGE_SIZE * params.pages,
        offset: 0,
    };
}

export function fetchRoomMessages(roomId: string, limit?: number, offset?: number) {
    return getRoomMessages(roomId, limit, offset);
}

export function fetchRoomMessagesBefore(roomId: string, beforeCursor: string, limit?: number) {
    return getRoomMessagesBefore(roomId, beforeCursor, limit);
}

export function fetchUserRooms() {
    return getUserRooms();
}

export function fetchResolveDMRoom(recipientId: string) {
    return queryClient.fetchQuery({
        queryKey: queryKeys.chat.dmResolve(recipientId),
        queryFn: () => resolveDMRoom(recipientId),
    });
}

export function useUserRooms(enabled = true) {
    const query = useQuery({
        queryKey: queryKeys.chat.userRooms(),
        queryFn: () => getUserRooms(),
        enabled,
    });
    return { rooms: query.data?.rooms ?? [], loading: query.isLoading, refresh: query.refetch };
}

export function useHostedRooms(params: RoomsListParams, enabled = true): RoomsListResult {
    const query = useQuery({
        queryKey: queryKeys.chat.roomsListHosted(params),
        queryFn: () => listMyChatRooms({ role: "host", ...roomsListRequest(params) }),
        enabled,
    });

    return { rooms: query.data?.rooms ?? [], total: query.data?.total ?? 0, loading: query.isFetching };
}

export function useJoinedRooms(params: RoomsListParams, enabled = true): RoomsListResult {
    const query = useQuery({
        queryKey: queryKeys.chat.roomsListJoined(params),
        queryFn: () => listMyChatRooms({ role: "member", ...roomsListRequest(params) }),
        enabled,
    });

    return { rooms: query.data?.rooms ?? [], total: query.data?.total ?? 0, loading: query.isFetching };
}

export function usePublicRooms(params: RoomsListParams, enabled = true): RoomsListResult {
    const query = useQuery({
        queryKey: queryKeys.chat.roomsListDiscover(params),
        queryFn: () => listPublicChatRooms(roomsListRequest(params)),
        enabled,
    });

    return { rooms: query.data?.rooms ?? [], total: query.data?.total ?? 0, loading: query.isFetching };
}

export function useChatRoomMembers(roomId: string, enabled = true) {
    const query = useQuery({
        queryKey: queryKeys.chat.roomMembers(roomId),
        queryFn: () => getChatRoomMembers(roomId),
        enabled: enabled && !!roomId,
    });
    return { members: query.data?.members ?? [], loading: query.isLoading, refresh: query.refetch };
}

export function useChatUnreadCount() {
    const { user } = useAuth();
    const query = useQuery({
        queryKey: queryKeys.chat.unreadCount(),
        queryFn: () => getChatUnreadCount(),
        enabled: !!user,
    });
    return { count: query.data?.count ?? 0, refresh: query.refetch };
}

export function useChatRoomBans(roomId: string, enabled = true) {
    const query = useQuery({
        queryKey: queryKeys.chat.roomBans(roomId),
        queryFn: () => listChatRoomBans(roomId),
        enabled: enabled && !!roomId,
    });
    return { bans: query.data?.bans ?? [], loading: query.isLoading, refresh: query.refetch };
}

export function useChatRoomBannedWords(roomId: string, enabled = true) {
    const query = useQuery({
        queryKey: queryKeys.chat.roomBannedWords(roomId),
        queryFn: () => listChatRoomBannedWords(roomId),
        enabled: enabled && !!roomId,
    });
    return { rules: query.data?.rules ?? [], loading: query.isLoading, refresh: query.refetch };
}

export function useChatRoomPinnedMessages(roomId: string, enabled = true) {
    const query = useQuery({
        queryKey: queryKeys.chat.pinned(roomId),
        queryFn: () => getChatRoomPinnedMessages(roomId),
        enabled: enabled && !!roomId,
    });
    return { messages: query.data?.messages ?? [], loading: query.isLoading, refresh: query.refetch };
}
