import { apiPost, apiPostFormData } from "../client";
import type { MediaUpload } from "../../types/api";

export interface CreateUploads {
    media?: readonly MediaUpload[];
    attachments?: readonly File[];
}

export function mediaFormData(file: File, isSpoiler: boolean): FormData {
    const formData = new FormData();
    formData.append("media", file);
    if (isSpoiler) {
        formData.append("is_spoiler", "true");
    }

    return formData;
}

export function appendMediaParts(formData: FormData, files?: readonly File[], spoilers?: readonly boolean[]): void {
    if (!files || files.length === 0) {
        return;
    }

    const spoilerIndexes: number[] = [];
    for (const [i, file] of files.entries()) {
        formData.append("media", file);
        if (spoilers?.[i] === true) {
            spoilerIndexes.push(i);
        }
    }

    if (spoilerIndexes.length > 0) {
        formData.append("spoiler_indexes", spoilerIndexes.join(","));
    }
}

export function createFormData(payload: unknown, uploads: CreateUploads): FormData {
    const formData = new FormData();
    formData.append("payload", JSON.stringify(payload));

    const media = uploads.media ?? [];
    appendMediaParts(
        formData,
        media.map(item => item.file),
        media.map(item => item.isSpoiler),
    );

    for (const file of uploads.attachments ?? []) {
        formData.append("attachments", file);
    }

    return formData;
}

export async function postCreate<T, B>(path: string, payload: B, uploads: CreateUploads = {}): Promise<T> {
    const hasMedia = (uploads.media?.length ?? 0) > 0;
    const hasAttachments = (uploads.attachments?.length ?? 0) > 0;
    if (!hasMedia && !hasAttachments) {
        return apiPost<T, B>(path, payload);
    }

    return apiPostFormData<T>(path, createFormData(payload, uploads));
}
