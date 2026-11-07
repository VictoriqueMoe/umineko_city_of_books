import { apiDelete, apiFetch, apiPostFormData, type UploadProgress } from "../client";
import type { PendingUpload, UploadKind } from "../../types/api";

const UPLOAD_PATHS: Record<UploadKind, string> = {
    media: "/uploads/media",
    document: "/uploads/files",
};

export async function beginUpload(
    kind: UploadKind,
    file: File,
    onProgress: UploadProgress,
    signal?: AbortSignal,
): Promise<PendingUpload> {
    const formData = new FormData();
    formData.append("file", file);

    return apiPostFormData<PendingUpload>(UPLOAD_PATHS[kind], formData, onProgress, signal);
}

export async function getUploadStatuses(ids: readonly string[]): Promise<PendingUpload[]> {
    const { uploads } = await apiFetch<{ uploads: PendingUpload[] | null }>(`/uploads?ids=${ids.join(",")}`);

    return uploads ?? [];
}

export async function removeUpload(id: string): Promise<void> {
    await apiDelete<void>(`/uploads/${id}`);
}
