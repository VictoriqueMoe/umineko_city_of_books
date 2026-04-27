import { useQuery } from "@tanstack/react-query";
import { getNotifications, getPushPublicKey, getPushStatus, getUnreadCount } from "../endpoints";
import { queryKeys } from "../queryKeys";

export function useNotifications(limit = 20, offset = 0) {
    const query = useQuery({
        queryKey: queryKeys.notifications.list({ limit, offset }),
        queryFn: () => getNotifications({ limit, offset }),
    });
    return {
        notifications: query.data?.notifications ?? [],
        total: query.data?.total ?? 0,
        loading: query.isPending,
        refresh: query.refetch,
    };
}

export function useUnreadCount() {
    const query = useQuery({
        queryKey: queryKeys.notifications.unreadCount(),
        queryFn: () => getUnreadCount(),
    });
    return { count: query.data?.count ?? 0, refresh: query.refetch };
}

export function usePushPublicKey() {
    const query = useQuery({
        queryKey: ["push", "public-key"],
        queryFn: () => getPushPublicKey(),
        staleTime: Infinity,
    });
    return { publicKey: query.data?.public_key ?? "", loading: query.isPending };
}

export function usePushStatus(endpoint: string) {
    const query = useQuery({
        queryKey: ["push", "status", endpoint],
        queryFn: () => getPushStatus(endpoint),
        enabled: !!endpoint,
    });
    return { subscribed: query.data?.subscribed ?? false, loading: query.isPending };
}
