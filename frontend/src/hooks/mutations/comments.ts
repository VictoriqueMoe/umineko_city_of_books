import { useMutation, useQueryClient } from "@tanstack/react-query";
import { invalidateViews, type ViewDomain } from "../../api/cache/invalidate";

interface CommentApi {
    update: (id: string, body: string) => Promise<void>;
    remove: (id: string) => Promise<void>;
    like: (id: string) => Promise<void>;
    unlike: (id: string) => Promise<void>;
}

export function commentMutations(domain: ViewDomain, api: CommentApi) {
    return {
        useUpdate(_parentId?: string) {
            const qc = useQueryClient();
            return useMutation({
                mutationFn: ({ id, body }: { id: string; body: string }) => api.update(id, body),
                onSuccess: () => invalidateViews(qc, domain),
            });
        },
        useDelete(_parentId?: string) {
            const qc = useQueryClient();
            return useMutation({
                mutationFn: (id: string) => api.remove(id),
                onSuccess: () => invalidateViews(qc, domain),
            });
        },
        useLike(_parentId?: string) {
            const qc = useQueryClient();
            return useMutation({
                mutationFn: (id: string) => api.like(id),
                onSuccess: () => invalidateViews(qc, domain),
            });
        },
        useUnlike(_parentId?: string) {
            const qc = useQueryClient();
            return useMutation({
                mutationFn: (id: string) => api.unlike(id),
                onSuccess: () => invalidateViews(qc, domain),
            });
        },
    };
}
