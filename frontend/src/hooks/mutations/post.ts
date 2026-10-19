import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    createPost,
    deletePost,
    deletePostMedia,
    likePost,
    resolveSuggestion,
    unlikePost,
    unresolveSuggestion,
    updatePost,
    uploadPostMedia,
    votePoll,
} from "../../api/endpoints/post";
import { createComment, deleteComment, likeComment, unlikeComment, updateComment } from "../../api/endpoints/comments";
import type { CreatePollPayload, MediaUpload } from "../../types/api";
import { ApiError } from "../../api/client";
import { invalidateViews } from "../../api/cache/invalidate";
import { queryKeys } from "../../api/queryKeys";

type CreatePostInput = {
    body: string;
    corner?: string;
    poll?: CreatePollPayload;
    sharedContentId?: string;
    sharedContentType?: string;
    media?: readonly MediaUpload[];
};

export function useCreatePost() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (input: CreatePostInput) =>
            createPost(
                input.body,
                input.corner ?? "general",
                input.poll,
                input.sharedContentId,
                input.sharedContentType,
                input.media,
            ),
        onSuccess: () => {
            invalidateViews(qc, "post");
        },
    });
}

export function useUpdatePost(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (body: string) => updatePost(id, body),
        onSuccess: () => {
            invalidateViews(qc, "post");
        },
    });
}

export function useDeletePost() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: async (id: string) => {
            try {
                await deletePost(id);
            } catch (err) {
                if (err instanceof ApiError && err.status === 404) {
                    return;
                }
                throw err;
            }
        },
        onSuccess: (_data, id) => {
            qc.removeQueries({ queryKey: queryKeys.post.detail(id) });
            invalidateViews(qc, "post");
        },
    });
}

export function useUploadPostMedia(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ file, isSpoiler }: { file: File; isSpoiler?: boolean }) =>
            uploadPostMedia(id, file, isSpoiler ?? false),
        onSuccess: () => {
            invalidateViews(qc, "post");
        },
    });
}

export function useDeletePostMedia(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (mediaId: number) => deletePostMedia(id, mediaId),
        onSuccess: () => {
            invalidateViews(qc, "post");
        },
    });
}

export function useLikePost() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => likePost(id),
        onSuccess: () => {
            invalidateViews(qc, "post");
        },
    });
}

export function useUnlikePost() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unlikePost(id),
        onSuccess: () => {
            invalidateViews(qc, "post");
        },
    });
}

export function useVotePoll() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ postId, optionIdx }: { postId: string; optionIdx: number }) => votePoll(postId, optionIdx),
        onSuccess: () => {
            invalidateViews(qc, "post");
        },
    });
}

export function useResolveSuggestion() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, status }: { id: string; status?: string }) => resolveSuggestion(id, status),
        onSuccess: () => {
            invalidateViews(qc, "post");
        },
    });
}

export function useUnresolveSuggestion() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unresolveSuggestion(id),
        onSuccess: () => {
            invalidateViews(qc, "post");
        },
    });
}

export function useCreateComment(postId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId, media }: { body: string; parentId?: string; media?: readonly MediaUpload[] }) =>
            createComment(postId, body, parentId, media),
        onSuccess: () => {
            invalidateViews(qc, "post");
        },
    });
}

export function useUpdateComment(postId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ commentId, body }: { commentId: string; body: string }) => updateComment(commentId, body),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.post.detail(postId) });
        },
    });
}

export function useDeleteComment(_postId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (commentId: string) => deleteComment(commentId),
        onSuccess: () => {
            invalidateViews(qc, "post");
        },
    });
}

export function useLikeComment(postId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (commentId: string) => likeComment(commentId),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.post.detail(postId) });
        },
    });
}

export function useUnlikeComment(postId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (commentId: string) => unlikeComment(commentId),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.post.detail(postId) });
        },
    });
}
