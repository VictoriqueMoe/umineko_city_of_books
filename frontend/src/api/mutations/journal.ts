import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    createJournal,
    createJournalComment,
    deleteJournal,
    deleteJournalComment,
    followJournal,
    likeJournalComment,
    unfollowJournal,
    unlikeJournalComment,
    updateJournal,
    updateJournalComment,
    uploadJournalCommentMedia,
} from "../endpoints";
import type { CreateJournalPayload } from "../../types/api";
import { queryKeys } from "../queryKeys";

const detail = (id: string) => queryKeys.journal.detail(id);

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
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(id) }),
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
        onSuccess: (_d, id) => qc.invalidateQueries({ queryKey: detail(id) }),
    });
}

export function useUnfollowJournal() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unfollowJournal(id),
        onSuccess: (_d, id) => qc.invalidateQueries({ queryKey: detail(id) }),
    });
}

export function useCreateJournalComment(journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) =>
            createJournalComment(journalId, body, parentId),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(journalId) }),
    });
}

export function useUpdateJournalComment(journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, body }: { id: string; body: string }) => updateJournalComment(id, body),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(journalId) }),
    });
}

export function useDeleteJournalComment(journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteJournalComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(journalId) }),
    });
}

export function useLikeJournalComment(journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => likeJournalComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(journalId) }),
    });
}

export function useUnlikeJournalComment(journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unlikeJournalComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(journalId) }),
    });
}

export function useUploadJournalCommentMedia(journalId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ commentId, file }: { commentId: string; file: File }) =>
            uploadJournalCommentMedia(commentId, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(journalId) }),
    });
}
