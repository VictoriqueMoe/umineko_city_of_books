import { useMutation, useQueryClient } from "@tanstack/react-query";
import { unlockSecret } from "../../api/endpoints/secret";
import {
    createSecretComment,
    deleteSecretComment,
    likeSecretComment,
    unlikeSecretComment,
    updateSecretComment,
} from "../../api/endpoints/comments";
import type { MediaUpload } from "../../types/api";
import { invalidateViews } from "../../api/cache/invalidate";
import { commentMutations } from "./comments";

export function useCreateSecretComment(secretId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId, media }: { body: string; parentId?: string; media?: readonly MediaUpload[] }) =>
            createSecretComment(secretId, body, parentId, media),
        onSuccess: () => invalidateViews(qc, "secret"),
    });
}

const secretCommentMutations = commentMutations("secret", {
    update: (id, body) => updateSecretComment(id, body),
    remove: id => deleteSecretComment(id),
    like: id => likeSecretComment(id),
    unlike: id => unlikeSecretComment(id),
});

export const useUpdateSecretComment = secretCommentMutations.useUpdate;
export const useDeleteSecretComment = secretCommentMutations.useDelete;
export const useLikeSecretComment = secretCommentMutations.useLike;
export const useUnlikeSecretComment = secretCommentMutations.useUnlike;

export function useUnlockSecret() {
    return useMutation({
        mutationFn: ({ id, phrase }: { id: string; phrase: string }) => unlockSecret(id, phrase),
    });
}
