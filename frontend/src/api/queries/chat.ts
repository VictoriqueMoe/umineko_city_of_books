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
    resolveDMRoom,
} from "../endpoints";
import { queryClient } from "../queryClient";
import { queryKeys } from "../queryKeys";
import { useAuth } from "../../hooks/useAuth";

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
        queryKey: ["chat", "dm-resolve", recipientId],
        queryFn: () => resolveDMRoom(recipientId),
    });
}

export function useUserRooms() {
    const query = useQuery({
        queryKey: ["chat", "rooms", "user"],
        queryFn: () => getUserRooms(),
    });
    return { rooms: query.data?.rooms ?? [], loading: query.isLoading, refresh: query.refetch };
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
        queryKey: ["chat", "unread-count"],
        queryFn: () => getChatUnreadCount(),
        enabled: !!user,
    });
    return { count: query.data?.count ?? 0, refresh: query.refetch };
}

export function useChatRoomBans(roomId: string, enabled = true) {
    const query = useQuery({
        queryKey: ["chat", "rooms", roomId, "bans"],
        queryFn: () => listChatRoomBans(roomId),
        enabled: enabled && !!roomId,
    });
    return { bans: query.data?.bans ?? [], loading: query.isLoading, refresh: query.refetch };
}

export function useChatRoomBannedWords(roomId: string, enabled = true) {
    const query = useQuery({
        queryKey: ["chat", "rooms", roomId, "banned-words"],
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
