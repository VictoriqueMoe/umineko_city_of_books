import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    addOCGalleryImage,
    createOC,
    createOCComment,
    deleteOC,
    deleteOCComment,
    deleteOCGalleryImage,
    favouriteOC,
    likeOCComment,
    unlikeOCComment,
    updateOC,
    updateOCComment,
    updateOCGalleryImage,
    uploadOCCommentMedia,
    uploadOCImage,
    voteOC,
} from "../endpoints";
import { queryKeys } from "../queryKeys";

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

export function useUpdateOCGalleryImage() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({
            ocId,
            imageId,
            data,
        }: {
            ocId: string;
            imageId: number;
            data: { caption?: string; sort_order?: number };
        }) => updateOCGalleryImage(ocId, imageId, data),
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

export function useUpdateOCComment() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, body }: { id: string; body: string }) => updateOCComment(id, body),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.oc.all }),
    });
}

export function useDeleteOCComment() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteOCComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.oc.all }),
    });
}

export function useLikeOCComment() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => likeOCComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.oc.all }),
    });
}

export function useUnlikeOCComment() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unlikeOCComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.oc.all }),
    });
}

export function useUploadOCCommentMedia() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ commentId, file }: { commentId: string; file: File }) => uploadOCCommentMedia(commentId, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.oc.all }),
    });
}
