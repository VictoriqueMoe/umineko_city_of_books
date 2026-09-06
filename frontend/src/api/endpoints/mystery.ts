import { apiDelete, apiFetch, apiPost, apiPostFormData, apiPut, buildQueryString } from "../client";
import { mediaFormData } from "./mediaFormData";
import type {
    GMLeaderboardResponse,
    KnoxContract,
    MysteryAttachment,
    MysteryDetail,
    MysteryLeaderboardResponse,
    MysteryListResponse,
    PostMedia,
} from "../../types/api";

export async function listMysteries(params: {
    sort?: string;
    solved?: string;
    limit?: number;
    offset?: number;
}): Promise<MysteryListResponse> {
    const qs = buildQueryString({
        sort: params.sort,
        solved: params.solved,
        limit: params.limit ?? 20,
        offset: params.offset,
    });
    return apiFetch<MysteryListResponse>(`/mysteries${qs}`);
}

export async function getMystery(id: string): Promise<MysteryDetail> {
    return apiFetch<MysteryDetail>(`/mysteries/${id}`);
}

export async function createMystery(data: {
    title: string;
    body: string;
    difficulty: string;
    free_for_all: boolean;
    keep_open_after_solve: boolean;
    knox_contract: KnoxContract;
    clues: { body: string; truth_type: string }[];
}): Promise<{ id: string }> {
    return apiPost<{ id: string }, typeof data>("/mysteries", data);
}

export async function updateMystery(
    id: string,
    data: {
        title: string;
        body: string;
        difficulty: string;
        free_for_all: boolean;
        keep_open_after_solve: boolean;
        knox_contract: KnoxContract;
        clues: { body: string; truth_type: string }[];
    },
): Promise<void> {
    await apiPut<unknown, typeof data>(`/mysteries/${id}`, data);
}

export async function deleteMystery(id: string): Promise<void> {
    await apiDelete(`/mysteries/${id}`);
}

export async function createMysteryAttempt(
    mysteryId: string,
    body: string,
    parentId?: string,
): Promise<{ id: string }> {
    return apiPost<{ id: string }, { body: string; parent_id?: string }>(`/mysteries/${mysteryId}/attempts`, {
        body,
        parent_id: parentId,
    });
}

export async function deleteMysteryAttempt(id: string): Promise<void> {
    await apiDelete(`/mystery-attempts/${id}`);
}

export async function voteMysteryAttempt(id: string, value: number): Promise<void> {
    await apiPost<unknown, { value: number }>(`/mystery-attempts/${id}/vote`, { value });
}

export async function markMysterySolved(mysteryId: string, attemptId: string): Promise<void> {
    await apiPost<unknown, { attempt_id: string }>(`/mysteries/${mysteryId}/solve`, { attempt_id: attemptId });
}

export async function closeMystery(mysteryId: string): Promise<void> {
    await apiPost<unknown, Record<string, never>>(`/mysteries/${mysteryId}/close`, {});
}

export async function setMysteryPaused(mysteryId: string, paused: boolean): Promise<void> {
    await apiPost<unknown, { paused: boolean }>(`/mysteries/${mysteryId}/pause`, { paused });
}

export async function setMysteryGmAway(mysteryId: string, away: boolean): Promise<void> {
    await apiPost<unknown, { away: boolean }>(`/mysteries/${mysteryId}/away`, { away });
}

export async function deleteMysteryClue(mysteryId: string, clueId: number): Promise<void> {
    await apiDelete(`/mysteries/${mysteryId}/clues/${clueId}`);
}

export async function updateMysteryClue(mysteryId: string, clueId: number, body: string): Promise<void> {
    await apiPut<unknown, { body: string }>(`/mysteries/${mysteryId}/clues/${clueId}`, { body });
}

export async function addMysteryClue(
    mysteryId: string,
    body: string,
    truthType: string,
    playerId?: string,
): Promise<void> {
    await apiPost<unknown, { body: string; truth_type: string; player_id?: string }>(`/mysteries/${mysteryId}/clues`, {
        body,
        truth_type: truthType,
        player_id: playerId,
    });
}

export async function uploadMysteryAttachment(mysteryId: string, file: File): Promise<MysteryAttachment> {
    const formData = new FormData();
    formData.append("file", file);
    return apiPostFormData<MysteryAttachment>(`/mysteries/${mysteryId}/attachments`, formData);
}

export async function deleteMysteryAttachment(mysteryId: string, attachmentId: number): Promise<void> {
    await apiDelete(`/mysteries/${mysteryId}/attachments/${attachmentId}`);
}

export async function uploadMysteryMedia(mysteryId: string, file: File, isSpoiler = false): Promise<PostMedia> {
    return apiPostFormData<PostMedia>(`/mysteries/${mysteryId}/media`, mediaFormData(file, isSpoiler));
}

export async function deleteMysteryMedia(mysteryId: string, mediaId: number): Promise<void> {
    await apiDelete(`/mysteries/${mysteryId}/media/${mediaId}`);
}

export async function getMysteryLeaderboard(limit?: number): Promise<MysteryLeaderboardResponse> {
    const qs = buildQueryString({ limit });
    return apiFetch<MysteryLeaderboardResponse>(`/mysteries/leaderboard${qs}`);
}

export async function getGMLeaderboard(limit?: number): Promise<GMLeaderboardResponse> {
    const qs = buildQueryString({ limit });
    return apiFetch<GMLeaderboardResponse>(`/mysteries/gm-leaderboard${qs}`);
}
