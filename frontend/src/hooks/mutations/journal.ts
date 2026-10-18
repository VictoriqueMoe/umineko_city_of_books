import { useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import {
    createJournal,
    createJournalComment,
    createJournalEntry,
    deleteJournal,
    deleteJournalEntry,
    deleteJournalEntryMedia,
    followJournal,
    setJournalPaused,
    unfollowJournal,
    updateJournal,
    updateJournalEntry,
    uploadJournalEntryMedia,
} from "../../api/endpoints/journal";
import {
    deleteJournalComment,
    likeJournalComment,
    unlikeJournalComment,
    updateJournalComment,
} from "../../api/endpoints/comments";
import type { CreateJournalPayload, JournalDetail, JournalEntryPayload, MediaUpload } from "../../types/api";
import { invalidateViews } from "../../api/cache/invalidate";
import { queryKeys } from "../../api/queryKeys";
import { commentMutations } from "./comments";

function patchFollowing(qc: QueryClient, id: string, following: boolean): void {
    qc.setQueryData<JournalDetail>(queryKeys.journal.detail(id), prev =>
        prev ? { ...prev, is_following: following } : prev,
    );
}

function patchPaused(qc: QueryClient, id: string, paused: boolean): void {
    qc.setQueryData<JournalDetail>(queryKeys.journal.detail(id), prev =>
        prev ? { ...prev, is_paused: paused } : prev,
    );
}

export function useCreateJournal() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (payload: CreateJournalPayload) => createJournal(payload),
        onSuccess: () => invalidateViews(qc, "journal"),
    });
}

export function useUpdateJournal(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (payload: CreateJournalPayload) => updateJournal(id, payload),
        onSuccess: () => invalidateViews(qc, "journal"),
    });
}

export function useDeleteJournal() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteJournal(id),
        onSuccess: () => invalidateViews(qc, "journal"),
    });
}

export function useSetJournalPaused() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, paused }: { id: string; paused: boolean }) => setJournalPaused(id, paused),
        onMutate: ({ id, paused }) => {
            patchPaused(qc, id, paused);
        },
        onError: (_error, { id, paused }) => {
            patchPaused(qc, id, !paused);
        },
        onSuccess: () => invalidateViews(qc, "journal"),
    });
}

export function useFollowJournal() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => followJournal(id),
        onMutate: (id: string) => {
            patchFollowing(qc, id, true);
        },
        onError: (_error, id) => {
            patchFollowing(qc, id, false);
        },
        onSuccess: () => invalidateViews(qc, "journal"),
    });
}

export function useUnfollowJournal() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unfollowJournal(id),
        onMutate: (id: string) => {
            patchFollowing(qc, id, false);
        },
        onError: (_error, id) => {
            patchFollowing(qc, id, true);
        },
        onSuccess: () => invalidateViews(qc, "journal"),
    });
}

export function useCreateJournalComment(journalId: string, entryId?: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId, media }: { body: string; parentId?: string; media?: readonly MediaUpload[] }) =>
            createJournalComment(journalId, body, parentId, entryId, media),
        onSuccess: () => invalidateViews(qc, "journal"),
    });
}

export function useCreateJournalEntry(journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ payload, media }: { payload: JournalEntryPayload; media?: readonly MediaUpload[] }) =>
            createJournalEntry(journalId, payload, media),
        onSuccess: () => invalidateViews(qc, "journal"),
    });
}

export function useUpdateJournalEntry(_journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: JournalEntryPayload }) => updateJournalEntry(id, payload),
        onSuccess: () => invalidateViews(qc, "journal"),
    });
}

export function useDeleteJournalEntry(_journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteJournalEntry(id),
        onSuccess: () => invalidateViews(qc, "journal"),
    });
}

const journalCommentMutations = commentMutations("journal", {
    update: (id, body) => updateJournalComment(id, body),
    remove: id => deleteJournalComment(id),
    like: id => likeJournalComment(id),
    unlike: id => unlikeJournalComment(id),
});

export const useUpdateJournalComment = journalCommentMutations.useUpdate;
export const useDeleteJournalComment = journalCommentMutations.useDelete;
export const useLikeJournalComment = journalCommentMutations.useLike;
export const useUnlikeJournalComment = journalCommentMutations.useUnlike;

export function useUploadJournalEntryMedia(_journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ entryId, file, isSpoiler }: { entryId: string; file: File; isSpoiler?: boolean }) =>
            uploadJournalEntryMedia(entryId, file, isSpoiler ?? false),
        onSuccess: () => invalidateViews(qc, "journal"),
    });
}

export function useDeleteJournalEntryMedia(_journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ entryId, mediaId }: { entryId: string; mediaId: number }) =>
            deleteJournalEntryMedia(entryId, mediaId),
        onSuccess: () => invalidateViews(qc, "journal"),
    });
}
