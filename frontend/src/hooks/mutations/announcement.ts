import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    createAnnouncementComment,
    deleteAnnouncementComment,
    likeAnnouncementComment,
    unlikeAnnouncementComment,
    updateAnnouncementComment,
    uploadAnnouncementCommentMedia,
} from "../../api/endpoints/comments";
import { invalidateViews } from "../../api/cache/invalidate";
import { commentMutations } from "./comments";

export function useCreateAnnouncementComment(announcementId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) =>
            createAnnouncementComment(announcementId, body, parentId),
        onSuccess: () => invalidateViews(qc, "announcement"),
    });
}

const announcementCommentMutations = commentMutations("announcement", {
    update: (id, body) => updateAnnouncementComment(id, body),
    remove: id => deleteAnnouncementComment(id),
    like: id => likeAnnouncementComment(id),
    unlike: id => unlikeAnnouncementComment(id),
    uploadMedia: (commentId, file) => uploadAnnouncementCommentMedia(commentId, file),
});

export const useUpdateAnnouncementComment = announcementCommentMutations.useUpdate;
export const useDeleteAnnouncementComment = announcementCommentMutations.useDelete;
export const useLikeAnnouncementComment = announcementCommentMutations.useLike;
export const useUnlikeAnnouncementComment = announcementCommentMutations.useUnlike;
export const useUploadAnnouncementCommentMedia = announcementCommentMutations.useUploadMedia;
