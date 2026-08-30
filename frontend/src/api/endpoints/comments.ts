import { apiDelete, apiPost, apiPostFormData, apiPut } from "../client";
import type { PostMedia } from "../../types/api";

type CommentLikeBody = undefined | Record<string, never>;

function commentEndpoints(parentPath: string, commentPath: string, likeBody: CommentLikeBody) {
    return {
        async create(parentId: string, body: string, replyToId?: string): Promise<{ id: string }> {
            return apiPost<{ id: string }, { body: string; parent_id?: string }>(`${parentPath}/${parentId}/comments`, {
                body,
                parent_id: replyToId,
            });
        },
        async update(id: string, body: string): Promise<void> {
            await apiPut<unknown, { body: string }>(`${commentPath}/${id}`, { body });
        },
        async remove(id: string): Promise<void> {
            await apiDelete(`${commentPath}/${id}`);
        },
        async like(id: string): Promise<void> {
            await apiPost<unknown, CommentLikeBody>(`${commentPath}/${id}/like`, likeBody);
        },
        async unlike(id: string): Promise<void> {
            await apiDelete(`${commentPath}/${id}/like`);
        },
        async uploadMedia(commentId: string, file: File): Promise<PostMedia> {
            const formData = new FormData();
            formData.append("media", file);
            return apiPostFormData<PostMedia>(`${commentPath}/${commentId}/media`, formData);
        },
    };
}

const postComments = commentEndpoints("/posts", "/comments", undefined);

export const createComment = postComments.create;
export const updateComment = postComments.update;
export const deleteComment = postComments.remove;
export const likeComment = postComments.like;
export const unlikeComment = postComments.unlike;
export const uploadCommentMedia = postComments.uploadMedia;

const artComments = commentEndpoints("/art", "/art-comments", undefined);

export const createArtComment = artComments.create;
export const updateArtComment = artComments.update;
export const deleteArtComment = artComments.remove;
export const likeArtComment = artComments.like;
export const unlikeArtComment = artComments.unlike;
export const uploadArtCommentMedia = artComments.uploadMedia;

const mysteryComments = commentEndpoints("/mysteries", "/mystery-comments", {});

export const createMysteryComment = mysteryComments.create;
export const updateMysteryComment = mysteryComments.update;
export const deleteMysteryComment = mysteryComments.remove;
export const likeMysteryComment = mysteryComments.like;
export const unlikeMysteryComment = mysteryComments.unlike;
export const uploadMysteryCommentMedia = mysteryComments.uploadMedia;

const secretComments = commentEndpoints("/secrets", "/secret-comments", {});

export const createSecretComment = secretComments.create;
export const updateSecretComment = secretComments.update;
export const deleteSecretComment = secretComments.remove;
export const likeSecretComment = secretComments.like;
export const unlikeSecretComment = secretComments.unlike;
export const uploadSecretCommentMedia = secretComments.uploadMedia;

const fanficComments = commentEndpoints("/fanfics", "/fanfic-comments", {});

export const createFanficComment = fanficComments.create;
export const updateFanficComment = fanficComments.update;
export const deleteFanficComment = fanficComments.remove;
export const likeFanficComment = fanficComments.like;
export const unlikeFanficComment = fanficComments.unlike;
export const uploadFanficCommentMedia = fanficComments.uploadMedia;

const announcementComments = commentEndpoints("/announcements", "/announcement-comments", {});

export const createAnnouncementComment = announcementComments.create;
export const updateAnnouncementComment = announcementComments.update;
export const deleteAnnouncementComment = announcementComments.remove;
export const likeAnnouncementComment = announcementComments.like;
export const unlikeAnnouncementComment = announcementComments.unlike;
export const uploadAnnouncementCommentMedia = announcementComments.uploadMedia;

const journalComments = commentEndpoints("/journals", "/journal-comments", {});

export const updateJournalComment = journalComments.update;
export const deleteJournalComment = journalComments.remove;
export const likeJournalComment = journalComments.like;
export const unlikeJournalComment = journalComments.unlike;
export const uploadJournalCommentMedia = journalComments.uploadMedia;

const shipComments = commentEndpoints("/ships", "/ship-comments", {});

export const createShipComment = shipComments.create;
export const updateShipComment = shipComments.update;
export const deleteShipComment = shipComments.remove;
export const likeShipComment = shipComments.like;
export const unlikeShipComment = shipComments.unlike;
export const uploadShipCommentMedia = shipComments.uploadMedia;

const ocComments = commentEndpoints("/ocs", "/oc-comments", {});

export const createOCComment = ocComments.create;
export const updateOCComment = ocComments.update;
export const deleteOCComment = ocComments.remove;
export const likeOCComment = ocComments.like;
export const unlikeOCComment = ocComments.unlike;
export const uploadOCCommentMedia = ocComments.uploadMedia;
