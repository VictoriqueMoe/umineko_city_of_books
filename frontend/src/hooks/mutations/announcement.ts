import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    createAnnouncementComment,
    deleteAnnouncementComment,
    likeAnnouncementComment,
    unlikeAnnouncementComment,
    updateAnnouncementComment,
} from "../../api/endpoints/comments";
import type { MediaUpload } from "../../types/api";
import { invalidateViews } from "../../api/cache/invalidate";
import { commentMutations } from "./comments";

export function useCreateAnnouncementComment(announcementId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId, media }: { body: string; parentId?: string; media?: readonly MediaUpload[] }) =>
            createAnnouncementComment(announcementId, body, parentId, media),
        onSuccess: () => invalidateViews(qc, "announcement"),
    });
}

const announcementCommentMutations = commentMutations("announcement", {
    update: (id, body) => updateAnnouncementComment(id, body),
    remove: id => deleteAnnouncementComment(id),
    like: id => likeAnnouncementComment(id),
    unlike: id => unlikeAnnouncementComment(id),
});

export const useUpdateAnnouncementComment = announcementCommentMutations.useUpdate;
export const useDeleteAnnouncementComment = announcementCommentMutations.useDelete;
export const useLikeAnnouncementComment = announcementCommentMutations.useLike;
export const useUnlikeAnnouncementComment = announcementCommentMutations.useUnlike;
