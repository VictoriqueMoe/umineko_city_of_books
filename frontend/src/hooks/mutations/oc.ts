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
    uploadOCCommentMedia,
} from "../../api/endpoints/comments";
import { queryKeys } from "../../api/queryKeys";
import { commentMutations } from "./comments";

export function useCreateOC() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: { name: string; description: string; series: string; custom_series_name: string }) =>
            createOC(data),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.oc.all }),
    });
}

export function useUpdateOC(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: { name: string; description: string; series: string; custom_series_name: string }) =>
            updateOC(id, data),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.oc.all }),
    });
}

export function useDeleteOC() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteOC(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.oc.all }),
    });
}

export function useUploadOCImageById() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, file }: { id: string; file: File }) => uploadOCImage(id, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.oc.all }),
    });
}

export function useAddOCGalleryImage() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, file, caption }: { id: string; file: File; caption: string }) =>
            addOCGalleryImage(id, file, caption),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.oc.all }),
    });
}

export function useDeleteOCGalleryImage() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ ocId, imageId }: { ocId: string; imageId: number }) => deleteOCGalleryImage(ocId, imageId),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.oc.all }),
    });
}

export function useVoteOC(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (value: number) => voteOC(id, value),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.oc.all }),
    });
}

export function useFavouriteOC() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => favouriteOC(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.oc.all }),
    });
}

export function useCreateOCComment(ocId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) => createOCComment(ocId, body, parentId),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.oc.all }),
    });
}

const ocCommentMutations = commentMutations(queryKeys.oc.all, {
    update: (id, body) => updateOCComment(id, body),
    remove: id => deleteOCComment(id),
    like: id => likeOCComment(id),
    unlike: id => unlikeOCComment(id),
    uploadMedia: (commentId, file) => uploadOCCommentMedia(commentId, file),
});

export const useUpdateOCComment = ocCommentMutations.useUpdate;
export const useDeleteOCComment = ocCommentMutations.useDelete;
export const useLikeOCComment = ocCommentMutations.useLike;
export const useUnlikeOCComment = ocCommentMutations.useUnlike;
export const useUploadOCCommentMedia = ocCommentMutations.useUploadMedia;
