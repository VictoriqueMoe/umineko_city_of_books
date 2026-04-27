import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    createComment,
    createPost,
    deleteComment,
    deletePost,
    deletePostMedia,
    likeComment,
    likePost,
    resolveSuggestion,
    unlikeComment,
    unlikePost,
    unresolveSuggestion,
    updateComment,
    updatePost,
    uploadCommentMedia,
    uploadPostMedia,
    votePoll,
    type CreatePollPayload,
} from "../endpoints";
import { queryKeys } from "../queryKeys";

type CreatePostInput = {
    body: string;
    corner?: string;
    poll?: CreatePollPayload;
    sharedContentId?: string;
    sharedContentType?: string;
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
            ),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.all });
        },
    });
}

export function useUpdatePost(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (body: string) => updatePost(id, body),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.detail(id) });
            void qc.invalidateQueries({ queryKey: queryKeys.post.all });
        },
    });
}

export function useDeletePost() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deletePost(id),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.all });
        },
    });
}

export function useUploadPostMedia(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (file: File) => uploadPostMedia(id, file),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.detail(id) });
        },
    });
}

export function useUploadPostMediaById() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, file }: { id: string; file: File }) => uploadPostMedia(id, file),
        onSuccess: (_d, vars) => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.detail(vars.id) });
        },
    });
}

export function useDeletePostMedia(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (mediaId: number) => deletePostMedia(id, mediaId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.detail(id) });
        },
    });
}

export function useLikePost() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => likePost(id),
        onSuccess: (_d, id) => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.detail(id) });
        },
    });
}

export function useUnlikePost() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unlikePost(id),
        onSuccess: (_d, id) => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.detail(id) });
        },
    });
}

export function useVotePoll() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ postId, optionIdx }: { postId: string; optionIdx: number }) => votePoll(postId, optionIdx),
        onSuccess: (_d, vars) => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.detail(vars.postId) });
        },
    });
}

export function useResolveSuggestion() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, status }: { id: string; status?: string }) => resolveSuggestion(id, status),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.all });
        },
    });
}

export function useUnresolveSuggestion() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unresolveSuggestion(id),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.all });
        },
    });
}

export function useCreateComment(postId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) => createComment(postId, body, parentId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.detail(postId) });
        },
    });
}

export function useUpdateComment(postId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ commentId, body }: { commentId: string; body: string }) => updateComment(commentId, body),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.detail(postId) });
        },
    });
}

export function useDeleteComment(postId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (commentId: string) => deleteComment(commentId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.detail(postId) });
        },
    });
}

export function useLikeComment(postId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (commentId: string) => likeComment(commentId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.detail(postId) });
        },
    });
}

export function useUnlikeComment(postId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (commentId: string) => unlikeComment(commentId),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.detail(postId) });
        },
    });
}

export function useUploadCommentMedia(postId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ commentId, file }: { commentId: string; file: File }) => uploadCommentMedia(commentId, file),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: queryKeys.post.detail(postId) });
        },
    });
}
