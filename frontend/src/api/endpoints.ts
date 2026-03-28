import {apiDelete, apiFetch, apiPost, apiPostFormData, apiPut, buildQueryString} from "./client";
import type {
    CreateResponsePayload,
    CreateTheoryPayload,
    NotificationListResponse,
    QuoteBrowseResponse,
    QuoteSearchResponse,
    TheoryDetail,
    TheoryListResponse,
    UpdateProfilePayload,
    User,
    UserProfile,
    VotePayload,
} from "../types/api";

const QUOTE_API = "https://quotes.auaurora.moe/api/v1";

export async function register(username: string, password: string, displayName: string): Promise<User> {
    return apiPost<User, { username: string; password: string; display_name: string }>("/auth/register", {
        username,
        password,
        display_name: displayName,
    });
}

export async function login(username: string, password: string): Promise<User> {
    return apiPost<User, { username: string; password: string }>("/auth/login", { username, password });
}

export async function logout(): Promise<void> {
    await apiPost<unknown, undefined>("/auth/logout", undefined);
}

export async function getMe(): Promise<User> {
    return apiFetch<User>("/auth/me");
}

export async function searchQuotes(params: {
    query?: string;
    character?: string;
    episode?: number;
    truth?: string;
    limit?: number;
    offset?: number;
}): Promise<QuoteSearchResponse> {
    const qs = buildQueryString({
        q: params.query,
        character: params.character,
        episode: params.episode,
        truth: params.truth,
        limit: params.limit ?? 30,
        offset: params.offset,
    });
    const response = await fetch(`${QUOTE_API}/search${qs}`);
    if (!response.ok) {
        throw new Error(`Quote API error: ${response.status}`);
    }
    return response.json();
}

export async function browseQuotes(params: {
    character?: string;
    episode?: number;
    truth?: string;
    limit?: number;
    offset?: number;
}): Promise<QuoteBrowseResponse> {
    const qs = buildQueryString({
        character: params.character,
        episode: params.episode,
        truth: params.truth,
        limit: params.limit ?? 30,
        offset: params.offset,
    });
    const response = await fetch(`${QUOTE_API}/browse${qs}`);
    if (!response.ok) {
        throw new Error(`Quote API error: ${response.status}`);
    }
    return response.json();
}

export async function getCharacters(): Promise<Record<string, string>> {
    const response = await fetch(`${QUOTE_API}/characters`);
    if (!response.ok) {
        throw new Error(`Quote API error: ${response.status}`);
    }
    return response.json();
}

export async function listTheories(params: {
    sort?: string;
    episode?: number;
    author?: number;
    limit?: number;
    offset?: number;
}): Promise<TheoryListResponse> {
    const qs = buildQueryString({
        sort: params.sort,
        episode: params.episode,
        author: params.author,
        limit: params.limit ?? 20,
        offset: params.offset,
    });
    return apiFetch<TheoryListResponse>(`/theories${qs}`);
}

export async function updateTheory(id: number, payload: CreateTheoryPayload): Promise<{ status: string }> {
    return apiPut<{ status: string }, CreateTheoryPayload>(`/theories/${id}`, payload);
}

export async function getTheory(id: number): Promise<TheoryDetail> {
    return apiFetch<TheoryDetail>(`/theories/${id}`);
}

export async function createTheory(payload: CreateTheoryPayload): Promise<{ id: number }> {
    return apiPost<{ id: number }, CreateTheoryPayload>("/theories", payload);
}

export async function deleteTheory(id: number): Promise<void> {
    await apiDelete<unknown>(`/theories/${id}`);
}

export async function createResponse(theoryId: number, payload: CreateResponsePayload): Promise<{ id: number }> {
    return apiPost<{ id: number }, CreateResponsePayload>(`/theories/${theoryId}/responses`, payload);
}

export async function deleteResponse(id: number): Promise<void> {
    await apiDelete<unknown>(`/responses/${id}`);
}

export async function voteTheory(id: number, value: number): Promise<void> {
    await apiPost<unknown, VotePayload>(`/theories/${id}/vote`, { value });
}

export async function voteResponse(id: number, value: number): Promise<void> {
    await apiPost<unknown, VotePayload>(`/responses/${id}/vote`, { value });
}

export async function getUserProfile(username: string): Promise<UserProfile> {
    return apiFetch<UserProfile>(`/users/${username}`);
}

export async function updateProfile(payload: UpdateProfilePayload): Promise<{ status: string }> {
    return apiPut<{ status: string }, UpdateProfilePayload>("/auth/profile", payload);
}

export async function uploadAvatar(file: File): Promise<{ avatar_url: string }> {
    const formData = new FormData();
    formData.append("avatar", file);
    return apiPostFormData<{ avatar_url: string }>("/auth/avatar", formData);
}

export async function getNotifications(params: { limit?: number; offset?: number }): Promise<NotificationListResponse> {
    const qs = buildQueryString({ limit: params.limit ?? 20, offset: params.offset });
    return apiFetch<NotificationListResponse>(`/notifications${qs}`);
}

export async function markNotificationRead(id: number): Promise<void> {
    await apiPost<unknown, undefined>(`/notifications/${id}/read`, undefined);
}

export async function markAllNotificationsRead(): Promise<void> {
    await apiPost<unknown, undefined>("/notifications/read", undefined);
}

export async function getUnreadCount(): Promise<{ count: number }> {
    return apiFetch<{ count: number }>("/notifications/unread-count");
}
