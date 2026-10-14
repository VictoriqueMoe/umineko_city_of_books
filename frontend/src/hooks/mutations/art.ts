import { useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import type { ArtDetail } from "../../types/api";
import {
    createArt,
    createGallery,
    deleteArt,
    deleteGallery,
    likeArt,
    setArtGallery,
    setGalleryCover,
    unlikeArt,
    updateArt,
    updateGallery,
} from "../../api/endpoints/art";
import {
    createArtComment,
    deleteArtComment,
    likeArtComment,
    unlikeArtComment,
    updateArtComment,
    uploadArtCommentMedia,
} from "../../api/endpoints/comments";
import { invalidateViews } from "../../api/cache/invalidate";
import { queryKeys } from "../../api/queryKeys";

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

function overlayLike(qc: QueryClient, id: string, delta: number, liked: boolean) {
    qc.setQueryData<ArtDetail>(queryKeys.art.detail(id), prev =>
        prev ? { ...prev, user_liked: liked, like_count: prev.like_count + delta } : prev,
    );
}

export function useCreateArt() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (input: CreateArtInput) => createArt(input.metadata, input.imageFile),
        onSuccess: () => {
            invalidateViews(qc, "art");
        },
    });
}

export function useUpdateArt(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: UpdateArtInput) => updateArt(id, data),
        onSuccess: () => {
            invalidateViews(qc, "art");
        },
    });
}

export function useDeleteArt() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteArt(id),
        onSuccess: () => {
            invalidateViews(qc, "art");
        },
    });
}

export function useLikeArt() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => likeArt(id),
        onMutate: (id: string) => {
            overlayLike(qc, id, 1, true);
        },
        onError: (_e, id) => {
            overlayLike(qc, id, -1, false);
        },
        onSuccess: () => {
            invalidateViews(qc, "art");
        },
    });
}

export function useUnlikeArt() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unlikeArt(id),
        onMutate: (id: string) => {
            overlayLike(qc, id, -1, false);
        },
        onError: (_e, id) => {
            overlayLike(qc, id, 1, true);
        },
        onSuccess: () => {
            invalidateViews(qc, "art");
        },
    });
}

export function useCreateArtComment(artId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) =>
            createArtComment(artId, body, parentId),
        onSuccess: () => {
            invalidateViews(qc, "art");
        },
    });
}

export function useUpdateArtComment(artId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ commentId, body }: { commentId: string; body: string }) => updateArtComment(commentId, body),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.art.detail(artId) });
        },
    });
}

export function useDeleteArtComment(_artId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (commentId: string) => deleteArtComment(commentId),
        onSuccess: () => {
            invalidateViews(qc, "art");
        },
    });
}

export function useLikeArtComment(artId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (commentId: string) => likeArtComment(commentId),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.art.detail(artId) });
        },
    });
}

export function useUnlikeArtComment(artId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (commentId: string) => unlikeArtComment(commentId),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.art.detail(artId) });
        },
    });
}

export function useUploadArtCommentMedia(artId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ commentId, file }: { commentId: string; file: File }) => uploadArtCommentMedia(commentId, file),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.art.detail(artId) });
        },
    });
}

export function useCreateGallery() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ name, description }: { name: string; description?: string }) =>
            createGallery(name, description ?? ""),
        onSuccess: () => {
            invalidateViews(qc, "art");
        },
    });
}

export function useUpdateGallery(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ name, description }: { name: string; description?: string }) =>
            updateGallery(id, name, description ?? ""),
        onSuccess: () => {
            invalidateViews(qc, "art");
        },
    });
}

export function useDeleteGallery() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteGallery(id),
        onSuccess: () => {
            invalidateViews(qc, "art");
        },
    });
}

export function useSetGalleryCover(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (artId: string) => setGalleryCover(id, artId),
        onSuccess: () => {
            invalidateViews(qc, "art");
        },
    });
}

export function useSetArtGallery() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ artId, galleryId }: { artId: string; galleryId: string | null }) =>
            setArtGallery(artId, galleryId),
        onSuccess: () => {
            invalidateViews(qc, "art");
        },
    });
}
