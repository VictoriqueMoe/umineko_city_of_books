import { type QueryKey, useMutation, useQueryClient } from "@tanstack/react-query";

interface CommentApi<TMedia> {
    update: (id: string, body: string) => Promise<void>;
    remove: (id: string) => Promise<void>;
    like: (id: string) => Promise<void>;
    unlike: (id: string) => Promise<void>;
    uploadMedia: (commentId: string, file: File) => Promise<TMedia>;
}

export function commentMutations<TMedia>(invalidateKey: QueryKey, api: CommentApi<TMedia>) {
    return {
        useUpdate(_parentId?: string) {
            const qc = useQueryClient();
            return useMutation({
                mutationFn: ({ id, body }: { id: string; body: string }) => api.update(id, body),
                onSuccess: () => qc.invalidateQueries({ queryKey: invalidateKey }),
            });
        },
        useDelete(_parentId?: string) {
            const qc = useQueryClient();
            return useMutation({
                mutationFn: (id: string) => api.remove(id),
                onSuccess: () => qc.invalidateQueries({ queryKey: invalidateKey }),
            });
        },
        useLike(_parentId?: string) {
            const qc = useQueryClient();
            return useMutation({
                mutationFn: (id: string) => api.like(id),
                onSuccess: () => qc.invalidateQueries({ queryKey: invalidateKey }),
            });
        },
        useUnlike(_parentId?: string) {
            const qc = useQueryClient();
            return useMutation({
                mutationFn: (id: string) => api.unlike(id),
                onSuccess: () => qc.invalidateQueries({ queryKey: invalidateKey }),
            });
        },
        useUploadMedia(_parentId?: string) {
            const qc = useQueryClient();
            return useMutation({
                mutationFn: ({ commentId, file }: { commentId: string; file: File }) =>
                    api.uploadMedia(commentId, file),
                onSuccess: () => qc.invalidateQueries({ queryKey: invalidateKey }),
            });
        },
    };
}
