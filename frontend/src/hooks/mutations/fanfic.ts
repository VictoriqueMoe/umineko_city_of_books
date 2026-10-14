import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    createFanfic,
    createFanficChapter,
    deleteFanfic,
    deleteFanficChapter,
    deleteFanficCover,
    favouriteFanfic,
    unfavouriteFanfic,
    updateFanfic,
    updateFanficChapter,
    uploadFanficCover,
} from "../../api/endpoints/fanfic";
import {
    createFanficComment,
    deleteFanficComment,
    likeFanficComment,
    unlikeFanficComment,
    updateFanficComment,
    uploadFanficCommentMedia,
} from "../../api/endpoints/comments";
import { invalidateViews } from "../../api/cache/invalidate";
import { commentMutations } from "./comments";

export function useCreateFanfic() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: Parameters<typeof createFanfic>[0]) => createFanfic(data),
        onSuccess: () => invalidateViews(qc, "fanfic"),
    });
}

export function useUpdateFanfic(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: Parameters<typeof updateFanfic>[1]) => updateFanfic(id, data),
        onSuccess: () => invalidateViews(qc, "fanfic"),
    });
}

export function useDeleteFanfic() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteFanfic(id),
        onSuccess: () => invalidateViews(qc, "fanfic"),
    });
}

export function useUploadFanficCover(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (file: File) => uploadFanficCover(id, file),
        onSuccess: () => invalidateViews(qc, "fanfic"),
    });
}

export function useUploadFanficCoverFor() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, file }: { id: string; file: File }) => uploadFanficCover(id, file),
        onSuccess: () => invalidateViews(qc, "fanfic"),
    });
}

export function useDeleteFanficCover(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: () => deleteFanficCover(id),
        onSuccess: () => invalidateViews(qc, "fanfic"),
    });
}

export function useCreateFanficChapter(fanficId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ title, body }: { title: string; body: string }) => createFanficChapter(fanficId, title, body),
        onSuccess: () => invalidateViews(qc, "fanfic"),
    });
}

export function useUpdateFanficChapter(_fanficId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ chapterId, title, body }: { chapterId: string; title: string; body: string }) =>
            updateFanficChapter(chapterId, title, body),
        onSuccess: () => invalidateViews(qc, "fanfic"),
    });
}

export function useDeleteFanficChapter(_fanficId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (chapterId: string) => deleteFanficChapter(chapterId),
        onSuccess: () => invalidateViews(qc, "fanfic"),
    });
}

export function useFavouriteFanfic() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => favouriteFanfic(id),
        onSuccess: () => invalidateViews(qc, "fanfic"),
    });
}

export function useUnfavouriteFanfic() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unfavouriteFanfic(id),
        onSuccess: () => invalidateViews(qc, "fanfic"),
    });
}

export function useCreateFanficComment(fanficId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) =>
            createFanficComment(fanficId, body, parentId),
        onSuccess: () => invalidateViews(qc, "fanfic"),
    });
}

const fanficCommentMutations = commentMutations("fanfic", {
    update: (id, body) => updateFanficComment(id, body),
    remove: id => deleteFanficComment(id),
    like: id => likeFanficComment(id),
    unlike: id => unlikeFanficComment(id),
    uploadMedia: (commentId, file) => uploadFanficCommentMedia(commentId, file),
});

export const useUpdateFanficComment = fanficCommentMutations.useUpdate;
export const useDeleteFanficComment = fanficCommentMutations.useDelete;
export const useLikeFanficComment = fanficCommentMutations.useLike;
export const useUnlikeFanficComment = fanficCommentMutations.useUnlike;
export const useUploadFanficCommentMedia = fanficCommentMutations.useUploadMedia;
