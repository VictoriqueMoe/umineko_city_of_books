import { apiDelete, apiFetch, apiPost, apiPostFormData, apiPut, buildQueryString } from "../client";
import { mediaFormData } from "./mediaFormData";
import type {
    CreateJournalPayload,
    JournalComment,
    JournalDetail,
    JournalEntry,
    JournalEntryPayload,
    JournalListResponse,
    JournalWork,
    PostMedia,
} from "../../types/api";

export async function listJournals(params: {
    sort?: string;
    work?: JournalWork | "";
    author?: string;
    search?: string;
    includeArchived?: boolean;
    limit?: number;
    offset?: number;
}): Promise<JournalListResponse> {
    const qs = buildQueryString({
        sort: params.sort,
        work: params.work || undefined,
        author: params.author,
        search: params.search,
        include_archived: params.includeArchived ? "true" : undefined,
        limit: params.limit ?? 20,
        offset: params.offset,
    });
    return apiFetch<JournalListResponse>(`/journals${qs}`);
}

export async function getJournal(id: string): Promise<JournalDetail> {
    return apiFetch<JournalDetail>(`/journals/${id}`);
}

export async function createJournal(payload: CreateJournalPayload): Promise<{ id: string }> {
    return apiPost<{ id: string }, CreateJournalPayload>("/journals", payload);
}

export async function updateJournal(id: string, payload: CreateJournalPayload): Promise<void> {
    await apiPut<unknown, CreateJournalPayload>(`/journals/${id}`, payload);
}

export async function deleteJournal(id: string): Promise<void> {
    await apiDelete<unknown>(`/journals/${id}`);
}

export async function setJournalPaused(id: string, paused: boolean): Promise<void> {
    await apiPut<unknown, { paused: boolean }>(`/journals/${id}/pause`, { paused });
}

export async function followJournal(id: string): Promise<void> {
    await apiPost<unknown, Record<string, never>>(`/journals/${id}/follow`, {});
}

export async function unfollowJournal(id: string): Promise<void> {
    await apiDelete(`/journals/${id}/follow`);
}

export async function createJournalComment(
    journalId: string,
    body: string,
    parentId?: string,
    entryId?: string,
): Promise<{ id: string }> {
    return apiPost<{ id: string }, { body: string; parent_id?: string; entry_id?: string }>(
        `/journals/${journalId}/comments`,
        { body, parent_id: parentId, entry_id: entryId },
    );
}

export async function getJournalEntry(
    journalId: string,
    entryNumber: number,
): Promise<{ entry: JournalEntry; comments: JournalComment[] }> {
    return apiFetch<{ entry: JournalEntry; comments: JournalComment[] }>(
        `/journals/${journalId}/entries/${entryNumber}`,
    );
}

export async function createJournalEntry(
    journalId: string,
    payload: JournalEntryPayload,
): Promise<{ id: string; entry_number: number }> {
    return apiPost<{ id: string; entry_number: number }, JournalEntryPayload>(
        `/journals/${journalId}/entries`,
        payload,
    );
}

export async function updateJournalEntry(entryId: string, payload: JournalEntryPayload): Promise<void> {
    await apiPut<unknown, JournalEntryPayload>(`/journal-entries/${entryId}`, payload);
}

export async function deleteJournalEntry(entryId: string): Promise<void> {
    await apiDelete(`/journal-entries/${entryId}`);
}

export async function uploadJournalEntryMedia(entryId: string, file: File, isSpoiler = false): Promise<PostMedia> {
    return apiPostFormData<PostMedia>(`/journal-entries/${entryId}/media`, mediaFormData(file, isSpoiler));
}

export async function deleteJournalEntryMedia(entryId: string, mediaId: number): Promise<void> {
    await apiDelete(`/journal-entries/${entryId}/media/${mediaId}`);
}
