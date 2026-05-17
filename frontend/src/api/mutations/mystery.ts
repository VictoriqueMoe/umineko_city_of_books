import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    addMysteryClue,
    closeMystery,
    createMystery,
    createMysteryAttempt,
    createMysteryComment,
    deleteMystery,
    deleteMysteryAttachment,
    deleteMysteryAttempt,
    deleteMysteryClue,
    deleteMysteryComment,
    deleteMysteryMedia,
    likeMysteryComment,
    markMysterySolved,
    setMysteryGmAway,
    setMysteryPaused,
    unlikeMysteryComment,
    updateMystery,
    updateMysteryClue,
    updateMysteryComment,
    uploadMysteryAttachment,
    uploadMysteryCommentMedia,
    uploadMysteryMedia,
    voteMysteryAttempt,
} from "../endpoints";
import { queryKeys } from "../queryKeys";

export function useCreateMystery() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: Parameters<typeof createMystery>[0]) => createMystery(data),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useUpdateMystery(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: Parameters<typeof updateMystery>[1]) => updateMystery(id, data),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useDeleteMystery() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteMystery(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useCreateMysteryAttempt(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) =>
            createMysteryAttempt(mysteryId, body, parentId),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useDeleteMysteryAttempt(_mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteMysteryAttempt(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useVoteMysteryAttempt(_mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, value }: { id: string; value: number }) => voteMysteryAttempt(id, value),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useMarkMysterySolved(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (attemptId: string) => markMysterySolved(mysteryId, attemptId),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useCloseMystery(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: () => closeMystery(mysteryId),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useSetMysteryPaused(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (paused: boolean) => setMysteryPaused(mysteryId, paused),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useSetMysteryGmAway(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (away: boolean) => setMysteryGmAway(mysteryId, away),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useDeleteMysteryClue(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (clueId: number) => deleteMysteryClue(mysteryId, clueId),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useUpdateMysteryClue(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ clueId, body }: { clueId: number; body: string }) => updateMysteryClue(mysteryId, clueId, body),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useAddMysteryClue(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, truthType, playerId }: { body: string; truthType: string; playerId?: string }) =>
            addMysteryClue(mysteryId, body, truthType, playerId),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useCreateMysteryComment(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) =>
            createMysteryComment(mysteryId, body, parentId),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useUpdateMysteryComment(_mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, body }: { id: string; body: string }) => updateMysteryComment(id, body),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useDeleteMysteryComment(_mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteMysteryComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useLikeMysteryComment(_mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => likeMysteryComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useUnlikeMysteryComment(_mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unlikeMysteryComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useUploadMysteryCommentMedia(_mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ commentId, file }: { commentId: string; file: File }) =>
            uploadMysteryCommentMedia(commentId, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useUploadMysteryAttachment(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (file: File) => uploadMysteryAttachment(mysteryId, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useUploadMysteryAttachmentToAny() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ mysteryId, file }: { mysteryId: string; file: File }) =>
            uploadMysteryAttachment(mysteryId, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useDeleteMysteryAttachment(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (attachmentId: number) => deleteMysteryAttachment(mysteryId, attachmentId),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useUploadMysteryMedia(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (file: File) => uploadMysteryMedia(mysteryId, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useUploadMysteryMediaToAny() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ mysteryId, file }: { mysteryId: string; file: File }) => uploadMysteryMedia(mysteryId, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}

export function useDeleteMysteryMedia(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (mediaId: number) => deleteMysteryMedia(mysteryId, mediaId),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.mystery.all }),
    });
}
