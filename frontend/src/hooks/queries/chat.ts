import type { AttachmentKind } from "../../api/endpoints/chat";
import { useCallback } from "react";
import { useInfiniteQuery, useQuery, useQueryClient } from "@tanstack/react-query";
import {
    getChatRoomMembers,
    getChatRoomAttachments,
    getChatRoomPinnedMessages,
    getChatUnreadCount,
    getUserRooms,
    listChatRoomBans,
    listChatRoomBannedWords,
    listMyChatRooms,
    listPublicChatRooms,
    resolveDMRoom,
} from "../../api/endpoints/chat";
import { queryClient } from "../../api/queryClient";
import { queryKeys, type RoomsListParams } from "../../api/queryKeys";
import { dmRoomsOf } from "../../domain/chat/dmRoster";
import { beforeCursor } from "../../domain/chat/messageStore";
import type { ChatRoom } from "../../types/api";
import { useAuth } from "../useAuth";

export const ROOMS_LIST_PAGE_SIZE = 20;

const EMPTY_ROOMS: ChatRoom[] = [];

export interface RoomsListResult {
    rooms: ChatRoom[];
    total: number;
    loading: boolean;
}

interface UserRoomsResponse {
    rooms: ChatRoom[];
}

interface DmRoster {
    dms: ChatRoom[];
    all: ChatRoom[];
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

export {
    getRoomMessages as fetchRoomMessages,
    getRoomMessagesBefore as fetchRoomMessagesBefore,
} from "../../api/endpoints/chat";

export function fetchResolveDMRoom(recipientId: string) {
    return queryClient.fetchQuery({
        queryKey: queryKeys.chat.dmResolve(recipientId),
        queryFn: () => resolveDMRoom(recipientId),
    });
}

function userRoomsOptions(enabled: boolean) {
    return {
        queryKey: queryKeys.chat.userRooms(),
        queryFn: () => getUserRooms(),
        enabled,
    };
}

function selectDmRoster(data: UserRoomsResponse): DmRoster {
    return { dms: dmRoomsOf(data.rooms), all: data.rooms };
}

export function useUserRooms(enabled = true) {
    const query = useQuery(userRoomsOptions(enabled));
    return { rooms: query.data?.rooms ?? EMPTY_ROOMS, loading: query.isLoading, refresh: query.refetch };
}

export function useDmRooms(enabled = true) {
    const query = useQuery({ ...userRoomsOptions(enabled), select: selectDmRoster });

    return {
        rooms: query.data?.dms ?? EMPTY_ROOMS,
        allRooms: query.data?.all ?? EMPTY_ROOMS,
        loading: query.isLoading,
    };
}

export function useUserRoomsCache() {
    const qc = useQueryClient();

    const patchRooms = useCallback(
        (update: (rooms: ChatRoom[]) => ChatRoom[]) => {
            qc.setQueryData<UserRoomsResponse>(queryKeys.chat.userRooms(), prev =>
                prev ? { ...prev, rooms: update(prev.rooms) } : prev,
            );
        },
        [qc],
    );

    const refreshRooms = useCallback(() => {
        qc.invalidateQueries({ queryKey: queryKeys.chat.userRooms() });
    }, [qc]);

    return { patchRooms, refreshRooms };
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

const ATTACHMENT_PAGE_SIZE = 50;

export function useChatRoomAttachments(roomId: string, kind: AttachmentKind, enabled = true) {
    const query = useInfiniteQuery({
        queryKey: queryKeys.chat.attachments(roomId, kind),
        queryFn: ({ pageParam }) => getChatRoomAttachments(roomId, kind, pageParam, ATTACHMENT_PAGE_SIZE),
        initialPageParam: undefined as string | undefined,
        getNextPageParam: lastPage => {
            if (lastPage.messages.length < ATTACHMENT_PAGE_SIZE) {
                return undefined;
            }

            const oldest = lastPage.messages[lastPage.messages.length - 1];

            return beforeCursor(oldest);
        },
        enabled: enabled && !!roomId,
    });

    return {
        messages: query.data?.pages.flatMap(page => page.messages) ?? [],
        loading: query.isLoading,
        loadingMore: query.isFetchingNextPage,
        hasMore: query.hasNextPage,
        loadMore: query.fetchNextPage,
        refresh: query.refetch,
    };
}

export function useChatRoomPinnedMessages(roomId: string, enabled = true) {
    const query = useQuery({
        queryKey: queryKeys.chat.pinned(roomId),
        queryFn: () => getChatRoomPinnedMessages(roomId),
        enabled: enabled && !!roomId,
    });
    return { messages: query.data?.messages ?? [], loading: query.isLoading, refresh: query.refetch };
}
