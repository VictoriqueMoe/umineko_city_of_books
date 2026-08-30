import { useQuery } from "@tanstack/react-query";
import { getNotifications, getUnreadCount } from "../../api/endpoints/notification";
import { queryKeys } from "../../api/queryKeys";
import { useAuth } from "../useAuth";

export function useNotifications(limit = 20, offset = 0) {
    const query = useQuery({
        queryKey: queryKeys.notifications.list({ limit, offset }),
        queryFn: () => getNotifications({ limit, offset }),
    });
    return {
        notifications: query.data?.notifications ?? [],
        total: query.data?.total ?? 0,
        loading: query.isLoading,
        refresh: query.refetch,
    };
}

export function useUnreadCount() {
    const { user } = useAuth();
    const query = useQuery({
        queryKey: queryKeys.notifications.unreadCount(),
        queryFn: () => getUnreadCount(),
        enabled: !!user,
    });
    return { count: query.data?.count ?? 0, refresh: query.refetch };
}
