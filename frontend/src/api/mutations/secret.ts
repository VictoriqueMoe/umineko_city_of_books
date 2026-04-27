import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    createSecretComment,
    deleteSecretComment,
    likeSecretComment,
    unlikeSecretComment,
    unlockSecret,
    updateSecretComment,
    uploadSecretCommentMedia,
} from "../endpoints";

const detail = (id: string) => ["secrets", "detail", id] as const;

export function useCreateSecretComment(secretId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) =>
            createSecretComment(secretId, body, parentId),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(secretId) }),
    });
}

export function useUpdateSecretComment(secretId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, body }: { id: string; body: string }) => updateSecretComment(id, body),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(secretId) }),
    });
}

export function useDeleteSecretComment(secretId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteSecretComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(secretId) }),
    });
}

export function useLikeSecretComment(secretId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => likeSecretComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(secretId) }),
    });
}

export function useUnlikeSecretComment(secretId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unlikeSecretComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(secretId) }),
    });
}

export function useUnlockSecret() {
    return useMutation({
        mutationFn: ({ id, phrase }: { id: string; phrase: string }) => unlockSecret(id, phrase),
    });
}

export function useUploadSecretCommentMedia(secretId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ commentId, file }: { commentId: string; file: File }) =>
            uploadSecretCommentMedia(commentId, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(secretId) }),
    });
}
