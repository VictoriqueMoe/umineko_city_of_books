import { apiDelete, apiFetch, apiPatch, apiPost, apiPostFormData } from "../client";
import type {
    LiveStream,
    LiveStreamListResponse,
    StreamCredentials,
    StreamDefaultMode,
    StreamOwner,
    VoiceTokenResponse,
} from "../../types/api";

export async function listLiveStreams(): Promise<LiveStreamListResponse> {
    return apiFetch<LiveStreamListResponse>("/streams/live");
}

export async function getStream(id: string): Promise<LiveStream> {
    return apiFetch<LiveStream>(`/streams/${id}`);
}

export async function getMyStream(): Promise<StreamOwner | null> {
    return apiFetch<StreamOwner | null>("/streams/mine");
}

export async function getStreamCredentials(): Promise<StreamCredentials> {
    return apiFetch<StreamCredentials>("/streams/credentials");
}

export async function resetStreamCredentials(): Promise<StreamCredentials> {
    return apiPost<StreamCredentials, Record<string, never>>("/streams/credentials/reset", {});
}

export async function startStream(
    title: string,
    defaultMode: StreamDefaultMode,
    bitrate: number,
): Promise<StreamOwner> {
    return apiPost<StreamOwner, { title: string; defaultMode: StreamDefaultMode; bitrate: number }>("/streams", {
        title,
        defaultMode,
        bitrate,
    });
}

export async function stopStream(id: string): Promise<void> {
    await apiDelete<unknown>(`/streams/${id}`);
}

export async function updateStreamTitle(id: string, title: string): Promise<LiveStream> {
    return apiPatch<LiveStream, { title: string }>(`/streams/${id}`, { title });
}

export async function getStreamViewerToken(id: string): Promise<VoiceTokenResponse> {
    return apiPost<VoiceTokenResponse, Record<string, never>>(`/streams/${id}/token`, {});
}

export async function joinStreamChat(id: string): Promise<void> {
    await apiPost<unknown, Record<string, never>>(`/streams/${id}/join-chat`, {});
}

export async function uploadStreamThumbnail(id: string, blob: Blob): Promise<void> {
    const formData = new FormData();
    formData.append("thumbnail", blob, "thumb.webp");
    await apiPostFormData<unknown>(`/streams/${id}/thumbnail`, formData);
}
