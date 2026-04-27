import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    createAnnouncementComment,
    deleteAnnouncementComment,
    likeAnnouncementComment,
    unlikeAnnouncementComment,
    updateAnnouncementComment,
    uploadAnnouncementCommentMedia,
} from "../endpoints";

const detail = (id: string) => ["announcements", "detail", id] as const;

export function useCreateAnnouncementComment(announcementId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) =>
            createAnnouncementComment(announcementId, body, parentId),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(announcementId) }),
    });
}

export function useUpdateAnnouncementComment(announcementId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, body }: { id: string; body: string }) => updateAnnouncementComment(id, body),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(announcementId) }),
    });
}

export function useDeleteAnnouncementComment(announcementId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteAnnouncementComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(announcementId) }),
    });
}

export function useLikeAnnouncementComment(announcementId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => likeAnnouncementComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(announcementId) }),
    });
}

export function useUnlikeAnnouncementComment(announcementId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unlikeAnnouncementComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(announcementId) }),
    });
}

export function useUploadAnnouncementCommentMedia(announcementId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ commentId, file }: { commentId: string; file: File }) =>
            uploadAnnouncementCommentMedia(commentId, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(announcementId) }),
    });
}
