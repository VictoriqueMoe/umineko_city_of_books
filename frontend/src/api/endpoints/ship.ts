import { apiDelete, apiFetch, apiPost, apiPostFormData, apiPut, buildQueryString } from "../client";
import type { ShipCharacter, ShipDetail, ShipListResponse } from "../../types/api";

export async function listShips(params: {
    sort?: string;
    series?: string;
    character?: string;
    crackships?: boolean;
    limit?: number;
    offset?: number;
}): Promise<ShipListResponse> {
    const qs = buildQueryString({
        sort: params.sort,
        series: params.series,
        character: params.character,
        crackships: params.crackships ? "true" : undefined,
        limit: params.limit,
        offset: params.offset,
    });
    return apiFetch<ShipListResponse>(`/ships${qs}`);
}

export async function getShip(id: string): Promise<ShipDetail> {
    return apiFetch<ShipDetail>(`/ships/${id}`);
}

export async function createShip(data: {
    title: string;
    description: string;
    characters: ShipCharacter[];
}): Promise<{ id: string }> {
    return apiPost<{ id: string }, typeof data>("/ships", data);
}

export async function updateShip(
    id: string,
    data: {
        title: string;
        description: string;
        characters: ShipCharacter[];
    },
): Promise<void> {
    await apiPut<unknown, typeof data>(`/ships/${id}`, data);
}

export async function deleteShip(id: string): Promise<void> {
    await apiDelete(`/ships/${id}`);
}

export async function uploadShipImage(shipId: string, file: File): Promise<{ image_url: string }> {
    const formData = new FormData();
    formData.append("image", file);
    return apiPostFormData<{ image_url: string }>(`/ships/${shipId}/image`, formData);
}

export async function voteShip(shipId: string, value: number): Promise<void> {
    await apiPost<unknown, { value: number }>(`/ships/${shipId}/vote`, { value });
}
