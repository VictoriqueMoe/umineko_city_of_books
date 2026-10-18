import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    addOCGalleryImage,
    createOC,
    deleteOC,
    deleteOCGalleryImage,
    favouriteOC,
    updateOC,
    uploadOCImage,
    voteOC,
} from "../../api/endpoints/oc";
import {
    createOCComment,
    deleteOCComment,
    likeOCComment,
    unlikeOCComment,
    updateOCComment,
} from "../../api/endpoints/comments";
import type { MediaUpload } from "../../types/api";
import { invalidateViews } from "../../api/cache/invalidate";
import { commentMutations } from "./comments";

export function useCreateOC() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: { name: string; description: string; series: string; custom_series_name: string }) =>
            createOC(data),
        onSuccess: () => invalidateViews(qc, "oc"),
    });
}

export function useUpdateOC(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: { name: string; description: string; series: string; custom_series_name: string }) =>
            updateOC(id, data),
        onSuccess: () => invalidateViews(qc, "oc"),
    });
}

export function useDeleteOC() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteOC(id),
        onSuccess: () => invalidateViews(qc, "oc"),
    });
}

export function useUploadOCImageById() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, file }: { id: string; file: File }) => uploadOCImage(id, file),
        onSuccess: () => invalidateViews(qc, "oc"),
    });
}

export function useAddOCGalleryImage() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, file, caption }: { id: string; file: File; caption: string }) =>
            addOCGalleryImage(id, file, caption),
        onSuccess: () => invalidateViews(qc, "oc"),
    });
}

export function useDeleteOCGalleryImage() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ ocId, imageId }: { ocId: string; imageId: number }) => deleteOCGalleryImage(ocId, imageId),
        onSuccess: () => invalidateViews(qc, "oc"),
    });
}

export function useVoteOC(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (value: number) => voteOC(id, value),
        onSuccess: () => invalidateViews(qc, "oc"),
    });
}

export function useFavouriteOC() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => favouriteOC(id),
        onSuccess: () => invalidateViews(qc, "oc"),
    });
}

export function useCreateOCComment(ocId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId, media }: { body: string; parentId?: string; media?: readonly MediaUpload[] }) =>
            createOCComment(ocId, body, parentId, media),
        onSuccess: () => invalidateViews(qc, "oc"),
    });
}

const ocCommentMutations = commentMutations("oc", {
    update: (id, body) => updateOCComment(id, body),
    remove: id => deleteOCComment(id),
    like: id => likeOCComment(id),
    unlike: id => unlikeOCComment(id),
});

export const useUpdateOCComment = ocCommentMutations.useUpdate;
export const useDeleteOCComment = ocCommentMutations.useDelete;
export const useLikeOCComment = ocCommentMutations.useLike;
export const useUnlikeOCComment = ocCommentMutations.useUnlike;
