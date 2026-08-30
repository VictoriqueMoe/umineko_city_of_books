import { apiDelete, apiFetch, apiPost, apiPostFormData, apiPut, buildQueryString } from "../client";
import type { FanficChapter, FanficDetail, FanficListResponse } from "../../types/api";

export async function listFanfics(params: {
    sort?: string;
    series?: string;
    rating?: string;
    genre_a?: string;
    genre_b?: string;
    language?: string;
    status?: string;
    tag?: string;
    char_a?: string;
    char_b?: string;
    char_c?: string;
    char_d?: string;
    pairing?: boolean;
    lemons?: boolean;
    search?: string;
    limit?: number;
    offset?: number;
}): Promise<FanficListResponse> {
    const qs = buildQueryString({
        sort: params.sort,
        series: params.series,
        rating: params.rating,
        genre_a: params.genre_a,
        genre_b: params.genre_b,
        language: params.language,
        status: params.status,
        tag: params.tag,
        char_a: params.char_a,
        char_b: params.char_b,
        char_c: params.char_c,
        char_d: params.char_d,
        pairing: params.pairing ? "true" : undefined,
        lemons: params.lemons ? "true" : undefined,
        search: params.search,
        limit: params.limit ?? 25,
        offset: params.offset,
    });
    return apiFetch<FanficListResponse>(`/fanfics${qs}`);
}

export async function getFanfic(id: string): Promise<FanficDetail> {
    return apiFetch<FanficDetail>(`/fanfics/${id}`);
}

export async function createFanfic(data: {
    title: string;
    summary: string;
    series: string;
    rating: string;
    language: string;
    status?: string;
    is_oneshot: boolean;
    contains_lemons: boolean;
    genres: string[];
    tags: string[];
    characters: { series: string; character_id?: string; character_name: string; sort_order: number }[];
    is_pairing: boolean;
    body?: string;
}): Promise<{ id: string }> {
    return apiPost<{ id: string }, typeof data>("/fanfics", data);
}

export async function updateFanfic(
    id: string,
    data: {
        title: string;
        summary: string;
        series: string;
        rating: string;
        language: string;
        status: string;
        is_oneshot: boolean;
        contains_lemons: boolean;
        genres: string[];
        tags: string[];
        characters: { series: string; character_id?: string; character_name: string; sort_order: number }[];
        is_pairing: boolean;
    },
): Promise<void> {
    await apiPut<unknown, typeof data>(`/fanfics/${id}`, data);
}

export async function deleteFanfic(id: string): Promise<void> {
    await apiDelete(`/fanfics/${id}`);
}

export async function uploadFanficCover(fanficId: string, file: File): Promise<{ image_url: string }> {
    const formData = new FormData();
    formData.append("image", file);
    return apiPostFormData<{ image_url: string }>(`/fanfics/${fanficId}/cover`, formData);
}

export async function deleteFanficCover(fanficId: string): Promise<void> {
    await apiDelete(`/fanfics/${fanficId}/cover`);
}

export async function getFanficChapter(fanficId: string, chapterNumber: number): Promise<FanficChapter> {
    return apiFetch<FanficChapter>(`/fanfics/${fanficId}/chapters/${chapterNumber}`);
}

export async function createFanficChapter(fanficId: string, title: string, body: string): Promise<{ id: string }> {
    return apiPost<{ id: string }, { title: string; body: string }>(`/fanfics/${fanficId}/chapters`, { title, body });
}

export async function updateFanficChapter(chapterId: string, title: string, body: string): Promise<void> {
    await apiPut<unknown, { title: string; body: string }>(`/fanfic-chapters/${chapterId}`, { title, body });
}

export async function deleteFanficChapter(chapterId: string): Promise<void> {
    await apiDelete(`/fanfic-chapters/${chapterId}`);
}

export async function favouriteFanfic(id: string): Promise<void> {
    await apiPost<unknown, Record<string, never>>(`/fanfics/${id}/favourite`, {});
}

export async function unfavouriteFanfic(id: string): Promise<void> {
    await apiDelete(`/fanfics/${id}/favourite`);
}

export async function getFanficLanguages(): Promise<string[]> {
    const res = await apiFetch<{ languages: string[] }>("/fanfic-languages");
    return res.languages;
}

export async function getFanficSeries(): Promise<string[]> {
    const res = await apiFetch<{ series: string[] }>("/fanfic-series");
    return res.series;
}

export async function searchOCCharacters(query: string): Promise<string[]> {
    const qs = buildQueryString({ q: query });
    const res = await apiFetch<{ characters: string[] }>(`/fanfic-oc-characters${qs}`);
    return res.characters;
}
