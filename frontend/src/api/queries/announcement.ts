import { useQuery } from "@tanstack/react-query";
import { getAnnouncement, getLatestAnnouncement, listAnnouncements } from "../endpoints";
import { queryKeys } from "../queryKeys";

export function useAnnouncementList(limit = 20, offset = 0) {
    const q = useQuery({
        queryKey: queryKeys.announcements.list({ limit, offset }),
        queryFn: () => listAnnouncements(limit, offset),
    });
    return {
        announcements: q.data?.announcements ?? [],
        total: q.data?.total ?? 0,
        loading: q.isLoading,
        refresh: q.refetch,
    };
}

export function useAnnouncement(id: string) {
    const q = useQuery({
        queryKey: queryKeys.announcements.detail(id),
        queryFn: () => getAnnouncement(id),
        enabled: !!id,
    });
    return {
        announcement: q.data ?? null,
        loading: q.isLoading,
        refresh: q.refetch,
    };
}

export function useLatestAnnouncement() {
    const q = useQuery({
        queryKey: queryKeys.announcements.latest(),
        queryFn: () => getLatestAnnouncement(),
    });
    return { announcement: q.data?.announcement ?? null, loading: q.isLoading };
}
