import { apiDelete, apiFetch, apiPost, apiPostFormData, apiPut, buildQueryString } from "../client";
import type { CreatePollPayload, Poll, PostDetail, PostListResponse, PostMedia } from "../../types/api";

export async function getCornerCounts(): Promise<Record<string, number>> {
    return apiFetch<Record<string, number>>("/posts/corner-counts");
}

export async function listPosts(params: {
    tab?: string;
    corner?: string;
    search?: string;
    sort?: string;
    seed?: number;
    limit?: number;
    offset?: number;
    resolved?: string;
}): Promise<PostListResponse> {
    const qs = buildQueryString(params);
    return apiFetch<PostListResponse>(`/posts${qs}`);
}

export async function getPost(id: string): Promise<PostDetail> {
    return apiFetch<PostDetail>(`/posts/${id}`);
}

export async function updatePost(id: string, body: string): Promise<void> {
    await apiPut<unknown, { body: string }>(`/posts/${id}`, { body });
}

export async function createPost(
    body: string,
    corner: string = "general",
    poll?: CreatePollPayload,
    sharedContentId?: string,
    sharedContentType?: string,
): Promise<{ id: string }> {
    return apiPost<
        { id: string },
        {
            body: string;
            corner: string;
            poll?: CreatePollPayload;
            shared_content_id?: string;
            shared_content_type?: string;
        }
    >("/posts", {
        body,
        corner,
        poll,
        shared_content_id: sharedContentId,
        shared_content_type: sharedContentType,
    });
}

export async function votePoll(postId: string, optionId: number): Promise<Poll> {
    return apiPost<Poll, { option_id: number }>(`/posts/${postId}/poll/vote`, { option_id: optionId });
}

export async function resolveSuggestion(postId: string, status: string = "done"): Promise<void> {
    await apiPost<unknown, { status: string }>(`/posts/${postId}/resolve`, { status });
}

export async function unresolveSuggestion(postId: string): Promise<void> {
    await apiDelete(`/posts/${postId}/resolve`);
}

export async function deletePost(id: string): Promise<void> {
    await apiDelete(`/posts/${id}`);
}

export async function uploadPostMedia(postId: string, file: File): Promise<PostMedia> {
    const formData = new FormData();
    formData.append("media", file);
    return apiPostFormData<PostMedia>(`/posts/${postId}/media`, formData);
}

export async function deletePostMedia(postId: string, mediaId: number): Promise<void> {
    await apiDelete(`/posts/${postId}/media/${mediaId}`);
}

export async function likePost(id: string): Promise<void> {
    await apiPost<unknown, undefined>(`/posts/${id}/like`, undefined);
}

export async function unlikePost(id: string): Promise<void> {
    await apiDelete(`/posts/${id}/like`);
}
