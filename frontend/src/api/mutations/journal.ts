import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    createJournal,
    createJournalComment,
    createJournalEntry,
    deleteJournal,
    deleteJournalComment,
    deleteJournalEntry,
    followJournal,
    likeJournalComment,
    unfollowJournal,
    unlikeJournalComment,
    updateJournal,
    updateJournalComment,
    updateJournalEntry,
    uploadJournalCommentMedia,
    uploadJournalEntryMedia,
} from "../endpoints";
import type { CreateJournalPayload, JournalEntryPayload } from "../../types/api";
import { queryKeys } from "../queryKeys";

export function useCreateJournal() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (payload: CreateJournalPayload) => createJournal(payload),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.journal.all }),
    });
}

export function useUpdateJournal(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (payload: CreateJournalPayload) => updateJournal(id, payload),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.journal.all }),
    });
}

export function useDeleteJournal() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteJournal(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.journal.all }),
    });
}

export function useFollowJournal() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => followJournal(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.journal.all }),
    });
}

export function useUnfollowJournal() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unfollowJournal(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.journal.all }),
    });
}

export function useCreateJournalComment(journalId: string, entryId?: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) =>
            createJournalComment(journalId, body, parentId, entryId),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.journal.all }),
    });
}

export function useCreateJournalEntry(journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (payload: JournalEntryPayload) => createJournalEntry(journalId, payload),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.journal.all }),
    });
}

export function useUpdateJournalEntry(_journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, payload }: { id: string; payload: JournalEntryPayload }) => updateJournalEntry(id, payload),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.journal.all }),
    });
}

export function useDeleteJournalEntry(_journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteJournalEntry(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.journal.all }),
    });
}

export function useUpdateJournalComment(_journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, body }: { id: string; body: string }) => updateJournalComment(id, body),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.journal.all }),
    });
}

export function useDeleteJournalComment(_journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteJournalComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.journal.all }),
    });
}

export function useLikeJournalComment(_journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => likeJournalComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.journal.all }),
    });
}

export function useUnlikeJournalComment(_journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unlikeJournalComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.journal.all }),
    });
}

export function useUploadJournalCommentMedia(_journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ commentId, file }: { commentId: string; file: File }) =>
            uploadJournalCommentMedia(commentId, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.journal.all }),
    });
}

export function useUploadJournalEntryMedia(_journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ entryId, file }: { entryId: string; file: File }) => uploadJournalEntryMedia(entryId, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.journal.all }),
    });
}
