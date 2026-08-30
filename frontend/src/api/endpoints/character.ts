import { apiFetch } from "../client";
import type { CharacterListResponse } from "../../types/api";

export async function listCharacters(series: string): Promise<CharacterListResponse> {
    return apiFetch<CharacterListResponse>(`/characters/${series}`);
}
