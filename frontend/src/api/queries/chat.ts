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
} from "../endpoints";
import { queryClient } from "../queryClient";
import { queryKeys } from "../queryKeys";

export function fetchRoomMessages(roomId: string, limit?: number, offset?: number) {
    return queryClient.fetchQuery({
        queryKey: [...queryKeys.chat.roomMessages(roomId), { limit, offset }],
        queryFn: () => getRoomMessages(roomId, limit, offset),
    });
}

export function fetchRoomMessagesBefore(roomId: string, beforeCursor: string, limit?: number) {
    return queryClient.fetchQuery({
        queryKey: [...queryKeys.chat.roomMessages(roomId), "before", beforeCursor, { limit }],
        queryFn: () => getRoomMessagesBefore(roomId, beforeCursor, limit),
    });
}

export function fetchUserRooms() {
    return queryClient.fetchQuery({
        queryKey: ["chat", "rooms", "user"],
        queryFn: () => getUserRooms(),
    });
}

export function fetchResolveDMRoom(recipientId: string) {
    return queryClient.fetchQuery({
        queryKey: ["chat", "dm-resolve", recipientId],
        queryFn: () => resolveDMRoom(recipientId),
    });
}

export function fetchChatRoomMembers(roomId: string) {
    return queryClient.fetchQuery({
        queryKey: queryKeys.chat.roomMembers(roomId),
        queryFn: () => getChatRoomMembers(roomId),
    });
}

export function fetchMyChatRooms(params: {
    role?: "host" | "member";
    search?: string;
    rp?: boolean;
    tag?: string;
    includeArchived?: boolean;
    limit?: number;
    offset?: number;
}) {
    return queryClient.fetchQuery({
        queryKey: ["chat", "rooms", "mine", params],
        queryFn: () => listMyChatRooms(params),
    });
}

export function fetchPublicChatRooms(params: {
    search?: string;
    rp?: boolean;
    tag?: string;
    includeArchived?: boolean;
    limit?: number;
    offset?: number;
}) {
    return queryClient.fetchQuery({
        queryKey: ["chat", "rooms", "public", params],
        queryFn: () => listPublicChatRooms(params),
    });
}

export function useResolveDMRoom(recipientId: string, enabled = true) {
    const query = useQuery({
        queryKey: ["chat", "dm-resolve", recipientId],
        queryFn: () => resolveDMRoom(recipientId),
        enabled: enabled && !!recipientId,
    });
    return { data: query.data ?? null, loading: query.isPending };
}

export function usePublicChatRooms(params: {
    search?: string;
    rp?: boolean;
    tag?: string;
    includeArchived?: boolean;
    limit?: number;
    offset?: number;
}) {
    const query = useQuery({
        queryKey: ["chat", "rooms", "public", params],
        queryFn: () => listPublicChatRooms(params),
    });
    return {
        rooms: query.data?.rooms ?? [],
        total: query.data?.total ?? 0,
        loading: query.isPending,
        refresh: query.refetch,
    };
}

export function useMyChatRooms(params: {
    role?: "host" | "member";
    search?: string;
    rp?: boolean;
    tag?: string;
    includeArchived?: boolean;
    limit?: number;
    offset?: number;
}) {
    const query = useQuery({
        queryKey: ["chat", "rooms", "mine", params],
        queryFn: () => listMyChatRooms(params),
    });
    return {
        rooms: query.data?.rooms ?? [],
        total: query.data?.total ?? 0,
        loading: query.isPending,
        refresh: query.refetch,
    };
}

export function useUserRooms() {
    const query = useQuery({
        queryKey: ["chat", "rooms", "user"],
        queryFn: () => getUserRooms(),
    });
    return { rooms: query.data?.rooms ?? [], loading: query.isPending, refresh: query.refetch };
}

export function useChatRoomMembers(roomId: string, enabled = true) {
    const query = useQuery({
        queryKey: queryKeys.chat.roomMembers(roomId),
        queryFn: () => getChatRoomMembers(roomId),
        enabled: enabled && !!roomId,
    });
    return { members: query.data?.members ?? [], loading: query.isPending, refresh: query.refetch };
}

export function useRoomMessages(roomId: string, limit?: number, offset?: number, enabled = true) {
    const query = useQuery({
        queryKey: [...queryKeys.chat.roomMessages(roomId), { limit, offset }],
        queryFn: () => getRoomMessages(roomId, limit, offset),
        enabled: enabled && !!roomId,
    });
    return {
        messages: query.data?.messages ?? [],
        total: query.data?.total ?? 0,
        loading: query.isPending,
        refresh: query.refetch,
    };
}

export function useChatUnreadCount() {
    const query = useQuery({
        queryKey: ["chat", "unread-count"],
        queryFn: () => getChatUnreadCount(),
    });
    return { count: query.data?.count ?? 0, refresh: query.refetch };
}

export function useChatRoomBans(roomId: string, enabled = true) {
    const query = useQuery({
        queryKey: ["chat", "rooms", roomId, "bans"],
        queryFn: () => listChatRoomBans(roomId),
        enabled: enabled && !!roomId,
    });
    return { bans: query.data?.bans ?? [], loading: query.isPending, refresh: query.refetch };
}

export function useChatRoomBannedWords(roomId: string, enabled = true) {
    const query = useQuery({
        queryKey: ["chat", "rooms", roomId, "banned-words"],
        queryFn: () => listChatRoomBannedWords(roomId),
        enabled: enabled && !!roomId,
    });
    return { rules: query.data?.rules ?? [], loading: query.isPending, refresh: query.refetch };
}

export function useChatRoomPinnedMessages(roomId: string, enabled = true) {
    const query = useQuery({
        queryKey: queryKeys.chat.pinned(roomId),
        queryFn: () => getChatRoomPinnedMessages(roomId),
        enabled: enabled && !!roomId,
    });
    return { messages: query.data?.messages ?? [], loading: query.isPending, refresh: query.refetch };
}
