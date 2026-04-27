import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    createArt,
    createArtComment,
    createGallery,
    deleteArt,
    deleteArtComment,
    deleteGallery,
    likeArt,
    likeArtComment,
    setArtGallery,
    setGalleryCover,
    unlikeArt,
    unlikeArtComment,
    updateArt,
    updateArtComment,
    updateGallery,
    uploadArtCommentMedia,
} from "../endpoints";
import { queryKeys } from "../queryKeys";

type CreateArtInput = {
    metadata: {
        title: string;
        description: string;
        corner: string;
        art_type: string;
        tags: string[];
        is_spoiler: boolean;
        gallery_id?: string;
    };
    imageFile: File;
};

type UpdateArtInput = { title: string; description: string; tags: string[]; is_spoiler: boolean };

export function useCreateArt() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (input: CreateArtInput) => createArt(input.metadata, input.imageFile),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.art.all });
        },
    });
}

export function useUpdateArt(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: UpdateArtInput) => updateArt(id, data),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.art.all });
        },
    });
}

export function useDeleteArt() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteArt(id),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.art.all });
        },
    });
}

export function useLikeArt() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => likeArt(id),
        onSuccess: (_d, id) => {
            void qc.invalidateQueries({ queryKey: queryKeys.art.detail(id) });
        },
    });
}

export function useUnlikeArt() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unlikeArt(id),
        onSuccess: (_d, id) => {
            void qc.invalidateQueries({ queryKey: queryKeys.art.detail(id) });
        },
    });
}

export function useCreateArtComment(artId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) =>
            createArtComment(artId, body, parentId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.art.detail(artId) });
        },
    });
}

export function useUpdateArtComment(artId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ commentId, body }: { commentId: string; body: string }) => updateArtComment(commentId, body),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.art.detail(artId) });
        },
    });
}

export function useDeleteArtComment(artId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (commentId: string) => deleteArtComment(commentId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.art.detail(artId) });
        },
    });
}

export function useLikeArtComment(artId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (commentId: string) => likeArtComment(commentId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.art.detail(artId) });
        },
    });
}

export function useUnlikeArtComment(artId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (commentId: string) => unlikeArtComment(commentId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.art.detail(artId) });
        },
    });
}

export function useUploadArtCommentMedia(artId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ commentId, file }: { commentId: string; file: File }) => uploadArtCommentMedia(commentId, file),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.art.detail(artId) });
        },
    });
}

export function useCreateGallery() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ name, description }: { name: string; description?: string }) =>
            createGallery(name, description ?? ""),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.art.all });
        },
    });
}

export function useUpdateGallery(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ name, description }: { name: string; description?: string }) =>
            updateGallery(id, name, description ?? ""),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.art.all });
        },
    });
}

export function useDeleteGallery() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteGallery(id),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.art.all });
        },
    });
}

export function useSetGalleryCover(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (artId: string) => setGalleryCover(id, artId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.art.all });
        },
    });
}

export function useSetArtGallery() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ artId, galleryId }: { artId: string; galleryId: string | null }) =>
            setArtGallery(artId, galleryId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.art.all });
        },
    });
}
