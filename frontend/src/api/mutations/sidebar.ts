import { useMutation, useQueryClient } from "@tanstack/react-query";
import { markSidebarVisited } from "../endpoints";
import type { SidebarLastVisitedResponse } from "../../types/api";

const LAST_VISITED_KEY = ["sidebar", "last-visited"] as const;

export function useMarkSidebarVisited() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (key: string) => markSidebarVisited(key),
        onMutate: key => {
            const previous = qc.getQueryData<SidebarLastVisitedResponse>(LAST_VISITED_KEY)?.visited?.[key];

            const visitedAt = new Date().toISOString();
            qc.setQueryData<SidebarLastVisitedResponse>(LAST_VISITED_KEY, prev => {
                const visited = { ...(prev?.visited ?? {}), [key]: visitedAt };
                return { visited };
            });

            return { previous };
        },
        onError: (_error, key, context) => {
            qc.setQueryData<SidebarLastVisitedResponse>(LAST_VISITED_KEY, prev => {
                if (!prev) {
                    return prev;
                }

                const visited = { ...prev.visited };
                if (context?.previous === undefined) {
                    delete visited[key];
                } else {
                    visited[key] = context.previous;
                }

                return { visited };
            });
        },
    });
}
