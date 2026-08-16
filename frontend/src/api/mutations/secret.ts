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
import { queryKeys } from "../queryKeys";
import { commentMutations } from "./commentMutations";

export function useCreateSecretComment(secretId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) =>
            createSecretComment(secretId, body, parentId),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.secrets.all }),
    });
}

const secretCommentMutations = commentMutations(queryKeys.secrets.all, {
    update: (id, body) => updateSecretComment(id, body),
    remove: id => deleteSecretComment(id),
    like: id => likeSecretComment(id),
    unlike: id => unlikeSecretComment(id),
    uploadMedia: (commentId, file) => uploadSecretCommentMedia(commentId, file),
});

export const useUpdateSecretComment = secretCommentMutations.useUpdate;
export const useDeleteSecretComment = secretCommentMutations.useDelete;
export const useLikeSecretComment = secretCommentMutations.useLike;
export const useUnlikeSecretComment = secretCommentMutations.useUnlike;
export const useUploadSecretCommentMedia = secretCommentMutations.useUploadMedia;

export function useUnlockSecret() {
    return useMutation({
        mutationFn: ({ id, phrase }: { id: string; phrase: string }) => unlockSecret(id, phrase),
    });
}
