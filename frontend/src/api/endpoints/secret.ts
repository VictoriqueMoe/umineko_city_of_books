import { apiFetch, apiPut } from "../client";
import type { SecretDetailResponse, SecretListResponse } from "../../types/api";

export async function unlockSecret(secret: string, phrase: string): Promise<void> {
    await apiPut<unknown, { secret: string; phrase: string }>("/preferences/secret-unlock", { secret, phrase });
}

export async function listSecrets(): Promise<SecretListResponse> {
    return apiFetch<SecretListResponse>("/secrets");
}

export async function getSecret(id: string): Promise<SecretDetailResponse> {
    return apiFetch<SecretDetailResponse>(`/secrets/${id}`);
}
