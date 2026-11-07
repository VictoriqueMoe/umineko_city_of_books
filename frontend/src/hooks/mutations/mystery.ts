import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    addMysteryClue,
    closeMystery,
    createMystery,
    createMysteryAttempt,
    deleteMystery,
    deleteMysteryAttachment,
    deleteMysteryAttempt,
    deleteMysteryClue,
    deleteMysteryMedia,
    markMysterySolved,
    setMysteryGmAway,
    setMysteryPaused,
    updateMystery,
    updateMysteryClue,
    attachMysteryFile,
    attachMysteryMedia,
    voteMysteryAttempt,
} from "../../api/endpoints/mystery";
import {
    createMysteryComment,
    deleteMysteryComment,
    likeMysteryComment,
    unlikeMysteryComment,
    updateMysteryComment,
} from "../../api/endpoints/comments";
import type { UploadRef } from "../../types/api";
import { invalidateViews } from "../../api/cache/invalidate";
import { commentMutations } from "./comments";

type CreateMysteryInput = {
    data: Parameters<typeof createMystery>[0];
    media?: readonly UploadRef[];
    attachments?: readonly UploadRef[];
};

export function useCreateMystery() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ data, media, attachments }: CreateMysteryInput) => createMystery(data, { media, attachments }),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useUpdateMystery(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: Parameters<typeof updateMystery>[1]) => updateMystery(id, data),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useDeleteMystery() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteMystery(id),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useCreateMysteryAttempt(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) =>
            createMysteryAttempt(mysteryId, body, parentId),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useDeleteMysteryAttempt(_mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteMysteryAttempt(id),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useVoteMysteryAttempt(_mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, value }: { id: string; value: number }) => voteMysteryAttempt(id, value),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useMarkMysterySolved(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (attemptId: string) => markMysterySolved(mysteryId, attemptId),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useCloseMystery(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: () => closeMystery(mysteryId),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useSetMysteryPaused(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (paused: boolean) => setMysteryPaused(mysteryId, paused),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useSetMysteryGmAway(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (away: boolean) => setMysteryGmAway(mysteryId, away),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useDeleteMysteryClue(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (clueId: number) => deleteMysteryClue(mysteryId, clueId),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useUpdateMysteryClue(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ clueId, body }: { clueId: number; body: string }) => updateMysteryClue(mysteryId, clueId, body),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useAddMysteryClue(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, truthType, playerId }: { body: string; truthType: string; playerId?: string }) =>
            addMysteryClue(mysteryId, body, truthType, playerId),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useCreateMysteryComment(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId, media }: { body: string; parentId?: string; media?: readonly UploadRef[] }) =>
            createMysteryComment(mysteryId, body, parentId, media),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

const mysteryCommentMutations = commentMutations("mystery", {
    update: (id, body) => updateMysteryComment(id, body),
    remove: id => deleteMysteryComment(id),
    like: id => likeMysteryComment(id),
    unlike: id => unlikeMysteryComment(id),
});

export const useUpdateMysteryComment = mysteryCommentMutations.useUpdate;
export const useDeleteMysteryComment = mysteryCommentMutations.useDelete;
export const useLikeMysteryComment = mysteryCommentMutations.useLike;
export const useUnlikeMysteryComment = mysteryCommentMutations.useUnlike;

export function useAttachMysteryFile(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (attachment: UploadRef) => attachMysteryFile(mysteryId, attachment),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useAttachMysteryFileToAny() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ mysteryId, attachment }: { mysteryId: string; attachment: UploadRef }) =>
            attachMysteryFile(mysteryId, attachment),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useDeleteMysteryAttachment(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (attachmentId: number) => deleteMysteryAttachment(mysteryId, attachmentId),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useAttachMysteryMedia(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (media: UploadRef) => attachMysteryMedia(mysteryId, media),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useAttachMysteryMediaToAny() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ mysteryId, media }: { mysteryId: string; media: UploadRef }) =>
            attachMysteryMedia(mysteryId, media),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}

export function useDeleteMysteryMedia(mysteryId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (mediaId: number) => deleteMysteryMedia(mysteryId, mediaId),
        onSuccess: () => invalidateViews(qc, "mystery"),
    });
}
