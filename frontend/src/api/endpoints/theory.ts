import { apiDelete, apiFetch, apiPost, apiPut, buildQueryString } from "../client";
import type {
    CreateResponsePayload,
    CreateTheoryPayload,
    Series,
    TheoryDetail,
    TheoryListResponse,
    VotePayload,
} from "../../types/api";

export async function listTheories(params: {
    sort?: string;
    episode?: number;
    author?: string;
    search?: string;
    series?: Series;
    limit?: number;
    offset?: number;
}): Promise<TheoryListResponse> {
    const qs = buildQueryString({
        sort: params.sort,
        episode: params.episode,
        author: params.author,
        search: params.search,
        series: params.series ?? "umineko",
        limit: params.limit ?? 20,
        offset: params.offset,
    });
    return apiFetch<TheoryListResponse>(`/theories${qs}`);
}

export async function updateTheory(id: string, payload: CreateTheoryPayload): Promise<{ status: string }> {
    return apiPut<{ status: string }, CreateTheoryPayload>(`/theories/${id}`, payload);
}

export async function getTheory(id: string): Promise<TheoryDetail> {
    return apiFetch<TheoryDetail>(`/theories/${encodeURIComponent(id)}`);
}

export async function createTheory(payload: CreateTheoryPayload): Promise<{ id: string }> {
    return apiPost<{ id: string }, CreateTheoryPayload>("/theories", payload);
}

export async function deleteTheory(id: string): Promise<void> {
    await apiDelete<unknown>(`/theories/${id}`);
}

export async function createResponse(theoryId: string, payload: CreateResponsePayload): Promise<{ id: string }> {
    return apiPost<{ id: string }, CreateResponsePayload>(`/theories/${theoryId}/responses`, payload);
}

export async function deleteResponse(id: string): Promise<void> {
    await apiDelete<unknown>(`/responses/${id}`);
}

export async function voteTheory(id: string, value: number): Promise<void> {
    await apiPost<unknown, VotePayload>(`/theories/${id}/vote`, { value });
}

export async function refuteTheory(theoryId: string, responseId: string): Promise<void> {
    await apiPost<unknown, { response_id: string }>(`/theories/${theoryId}/refute`, { response_id: responseId });
}

export async function voteResponse(id: string, value: number): Promise<void> {
    await apiPost<unknown, VotePayload>(`/responses/${id}/vote`, { value });
}
