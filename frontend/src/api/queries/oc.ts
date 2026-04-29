import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { getOC, listOCs, listUserOCs, listUserOCSummaries } from "../endpoints";
import { queryKeys } from "../queryKeys";
import { useNotifications } from "../../hooks/useNotifications";
import type { OCSummary, WSMessage } from "../../types/api";

export function useOCList(params: {
    sort?: string;
    series?: string;
    custom?: string;
    user_id?: string;
    crack?: boolean;
    limit?: number;
    offset?: number;
}) {
    const q = useQuery({
        queryKey: queryKeys.oc.feed(params),
        queryFn: () => listOCs(params),
    });
    return { ocs: q.data?.ocs ?? [], total: q.data?.total ?? 0, loading: q.isLoading };
}

export function useOC(id: string) {
    const q = useQuery({
        queryKey: queryKeys.oc.detail(id),
        queryFn: () => getOC(id),
        enabled: !!id,
    });
    return { oc: q.data ?? null, loading: q.isLoading, refresh: q.refetch };
}

export function useUserOCs(userId: string) {
    const q = useQuery({
        queryKey: queryKeys.oc.userList(userId),
        queryFn: () => listUserOCs(userId),
        enabled: !!userId,
    });
    return { ocs: q.data?.ocs ?? [], total: q.data?.total ?? 0, loading: q.isLoading };
}

export function useUserOCSummaries(userId: string, currentUserId?: string) {
    const qc = useQueryClient();
    const { addWSListener } = useNotifications();

    const q = useQuery({
        queryKey: queryKeys.oc.userSummaries(userId),
        queryFn: () => listUserOCSummaries(userId),
        enabled: !!userId,
    });

    useEffect(() => {
        if (!userId || userId !== currentUserId) {
            return;
        }
        return addWSListener((msg: WSMessage) => {
            if (msg.type !== "user_ocs_changed") {
                return;
            }
            const data = msg.data as { action: string; oc: OCSummary };
            qc.setQueryData<OCSummary[]>(queryKeys.oc.userSummaries(userId), prev => {
                const existing = prev ?? [];
                if (data.action === "deleted") {
                    return existing.filter(item => item.id !== data.oc.id);
                }
                if (data.action === "updated") {
                    return existing.map(item => (item.id === data.oc.id ? data.oc : item));
                }
                if (data.action === "created") {
                    if (existing.some(item => item.id === data.oc.id)) {
                        return existing;
                    }
                    return [...existing, data.oc].sort((a, b) => a.name.localeCompare(b.name));
                }
                return existing;
            });
        });
    }, [userId, currentUserId, addWSListener, qc]);

    return { summaries: q.data ?? [], loading: q.isLoading };
}
