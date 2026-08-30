import { apiDelete, apiFetch, apiPost, apiPostFormData, apiPut, buildQueryString } from "../client";
import type { OCDetail, OCImage, OCListResponse } from "../../types/api";

export async function listOCs(params: {
    sort?: string;
    series?: string;
    custom?: string;
    user_id?: string;
    crack?: boolean;
    limit?: number;
    offset?: number;
}): Promise<OCListResponse> {
    const qs = buildQueryString({
        sort: params.sort,
        series: params.series,
        custom: params.custom,
        user_id: params.user_id,
        crack: params.crack ? "true" : undefined,
        limit: params.limit,
        offset: params.offset,
    });
    return apiFetch<OCListResponse>(`/ocs${qs}`);
}

export async function getOC(id: string): Promise<OCDetail> {
    return apiFetch<OCDetail>(`/ocs/${id}`);
}

export async function createOC(data: {
    name: string;
    description: string;
    series: string;
    custom_series_name: string;
}): Promise<{ id: string }> {
    return apiPost<{ id: string }, typeof data>("/ocs", data);
}

export async function updateOC(
    id: string,
    data: {
        name: string;
        description: string;
        series: string;
        custom_series_name: string;
    },
): Promise<void> {
    await apiPut<unknown, typeof data>(`/ocs/${id}`, data);
}

export async function deleteOC(id: string): Promise<void> {
    await apiDelete(`/ocs/${id}`);
}

export async function uploadOCImage(ocId: string, file: File): Promise<{ image_url: string }> {
    const formData = new FormData();
    formData.append("image", file);
    return apiPostFormData<{ image_url: string }>(`/ocs/${ocId}/image`, formData);
}

export async function addOCGalleryImage(ocId: string, file: File, caption: string): Promise<OCImage> {
    const formData = new FormData();
    formData.append("image", file);
    if (caption) {
        formData.append("caption", caption);
    }
    return apiPostFormData<OCImage>(`/ocs/${ocId}/gallery`, formData);
}

export async function deleteOCGalleryImage(ocId: string, imageId: number): Promise<void> {
    await apiDelete(`/ocs/${ocId}/gallery/${imageId}`);
}

export async function voteOC(ocId: string, value: number): Promise<void> {
    await apiPost<unknown, { value: number }>(`/ocs/${ocId}/vote`, { value });
}

export async function favouriteOC(ocId: string): Promise<{ favourited: boolean }> {
    return apiPost<{ favourited: boolean }, Record<string, never>>(`/ocs/${ocId}/favourite`, {});
}
