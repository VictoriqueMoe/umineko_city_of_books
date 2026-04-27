import { useMutation, useQueryClient } from "@tanstack/react-query";
import { markSidebarVisited } from "../endpoints";
import type { SidebarLastVisitedResponse } from "../../types/api";

export function useMarkSidebarVisited() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (key: string) => markSidebarVisited(key),
        onMutate: key => {
            const visitedAt = new Date().toISOString();
            qc.setQueryData<SidebarLastVisitedResponse>(["sidebar", "last-visited"], prev => {
                const visited = { ...(prev?.visited ?? {}), [key]: visitedAt };
                return { visited };
            });
        },
    });
}
