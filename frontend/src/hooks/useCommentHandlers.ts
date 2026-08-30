import { useCallback } from "react";
import {
    useCreateAnnouncementComment,
    useDeleteAnnouncementComment,
    useLikeAnnouncementComment,
    useUnlikeAnnouncementComment,
    useUpdateAnnouncementComment,
    useUploadAnnouncementCommentMedia,
} from "./mutations/announcement";
import {
    useCreateArtComment,
    useDeleteArtComment,
    useLikeArtComment,
    useUnlikeArtComment,
    useUpdateArtComment,
    useUploadArtCommentMedia,
} from "./mutations/art";
import {
    useCreateFanficComment,
    useDeleteFanficComment,
    useLikeFanficComment,
    useUnlikeFanficComment,
    useUpdateFanficComment,
    useUploadFanficCommentMedia,
} from "./mutations/fanfic";
import {
    useCreateJournalComment,
    useDeleteJournalComment,
    useLikeJournalComment,
    useUnlikeJournalComment,
    useUpdateJournalComment,
    useUploadJournalCommentMedia,
} from "./mutations/journal";
import {
    useCreateMysteryComment,
    useDeleteMysteryComment,
    useLikeMysteryComment,
    useUnlikeMysteryComment,
    useUpdateMysteryComment,
    useUploadMysteryCommentMedia,
} from "./mutations/mystery";
import {
    useCreateOCComment,
    useDeleteOCComment,
    useLikeOCComment,
    useUnlikeOCComment,
    useUpdateOCComment,
    useUploadOCCommentMedia,
} from "./mutations/oc";
import {
    useCreateComment,
    useDeleteComment,
    useLikeComment,
    useUnlikeComment,
    useUpdateComment,
    useUploadCommentMedia,
} from "./mutations/post";
import {
    useCreateSecretComment,
    useDeleteSecretComment,
    useLikeSecretComment,
    useUnlikeSecretComment,
    useUpdateSecretComment,
    useUploadSecretCommentMedia,
} from "./mutations/secret";
import {
    useCreateShipComment,
    useDeleteShipComment,
    useLikeShipComment,
    useUnlikeShipComment,
    useUpdateShipComment,
    useUploadShipCommentMedia,
} from "./mutations/ship";

export type CommentFamily =
    "announcement" | "art" | "fanfic" | "journal" | "mystery" | "oc" | "post" | "secret" | "ship";

export type CommentOperation = "create" | "update" | "delete" | "like" | "unlike" | "uploadMedia";

type CreateCommentHandler = (targetId: string, body: string, parentId?: string) => Promise<{ id: string }>;
type UpdateCommentHandler = (commentId: string, body: string) => Promise<void>;
type CommentIdHandler = (commentId: string) => Promise<void>;
type UploadCommentMediaHandler = (commentId: string, file: File) => Promise<unknown>;

interface CommentHandlerShape {
    create: CreateCommentHandler;
    update: UpdateCommentHandler;
    delete: CommentIdHandler;
    like: CommentIdHandler;
    unlike: CommentIdHandler;
    uploadMedia: UploadCommentMediaHandler;
}

interface CommentHandlerName {
    create: "createCommentFn";
    update: "updateFn";
    delete: "deleteFn";
    like: "likeFn";
    unlike: "unlikeFn";
    uploadMedia: "uploadMediaFn";
}

export type CommentHandlers<TEnabled extends CommentOperation> = {
    [K in CommentOperation as CommentHandlerName[K]]: K extends TEnabled ? CommentHandlerShape[K] : undefined;
};

export interface CommentHandlerOptions<TEnabled extends readonly CommentOperation[]> {
    enabled: TEnabled;
    entryId?: string;
}

interface CreateCommentMutation {
    mutateAsync: (variables: { body: string; parentId?: string }) => Promise<{ id: string }>;
}

interface UpdateCommentMutation {
    mutateAsync: (variables: { id: string; commentId: string; body: string }) => Promise<unknown>;
}

interface CommentIdMutation {
    mutateAsync: (commentId: string) => Promise<unknown>;
}

interface UploadCommentMediaMutation {
    mutateAsync: (variables: { commentId: string; file: File }) => Promise<unknown>;
}

interface CommentFamilyMutations {
    useCreate: (targetId: string, entryId?: string) => CreateCommentMutation;
    useUpdate: (targetId: string) => UpdateCommentMutation;
    useDelete: (targetId: string) => CommentIdMutation;
    useLike: (targetId: string) => CommentIdMutation;
    useUnlike: (targetId: string) => CommentIdMutation;
    useUploadMedia: (targetId: string) => UploadCommentMediaMutation;
}

const commentFamilies: Record<CommentFamily, CommentFamilyMutations> = {
    announcement: {
        useCreate: useCreateAnnouncementComment,
        useUpdate: useUpdateAnnouncementComment,
        useDelete: useDeleteAnnouncementComment,
        useLike: useLikeAnnouncementComment,
        useUnlike: useUnlikeAnnouncementComment,
        useUploadMedia: useUploadAnnouncementCommentMedia,
    },
    art: {
        useCreate: useCreateArtComment,
        useUpdate: useUpdateArtComment,
        useDelete: useDeleteArtComment,
        useLike: useLikeArtComment,
        useUnlike: useUnlikeArtComment,
        useUploadMedia: useUploadArtCommentMedia,
    },
    fanfic: {
        useCreate: useCreateFanficComment,
        useUpdate: useUpdateFanficComment,
        useDelete: useDeleteFanficComment,
        useLike: useLikeFanficComment,
        useUnlike: useUnlikeFanficComment,
        useUploadMedia: useUploadFanficCommentMedia,
    },
    journal: {
        useCreate: useCreateJournalComment,
        useUpdate: useUpdateJournalComment,
        useDelete: useDeleteJournalComment,
        useLike: useLikeJournalComment,
        useUnlike: useUnlikeJournalComment,
        useUploadMedia: useUploadJournalCommentMedia,
    },
    mystery: {
        useCreate: useCreateMysteryComment,
        useUpdate: useUpdateMysteryComment,
        useDelete: useDeleteMysteryComment,
        useLike: useLikeMysteryComment,
        useUnlike: useUnlikeMysteryComment,
        useUploadMedia: useUploadMysteryCommentMedia,
    },
    oc: {
        useCreate: useCreateOCComment,
        useUpdate: useUpdateOCComment,
        useDelete: useDeleteOCComment,
        useLike: useLikeOCComment,
        useUnlike: useUnlikeOCComment,
        useUploadMedia: useUploadOCCommentMedia,
    },
    post: {
        useCreate: useCreateComment,
        useUpdate: useUpdateComment,
        useDelete: useDeleteComment,
        useLike: useLikeComment,
        useUnlike: useUnlikeComment,
        useUploadMedia: useUploadCommentMedia,
    },
    secret: {
        useCreate: useCreateSecretComment,
        useUpdate: useUpdateSecretComment,
        useDelete: useDeleteSecretComment,
        useLike: useLikeSecretComment,
        useUnlike: useUnlikeSecretComment,
        useUploadMedia: useUploadSecretCommentMedia,
    },
    ship: {
        useCreate: useCreateShipComment,
        useUpdate: useUpdateShipComment,
        useDelete: useDeleteShipComment,
        useLike: useLikeShipComment,
        useUnlike: useUnlikeShipComment,
        useUploadMedia: useUploadShipCommentMedia,
    },
};

function isEnabled(enabled: readonly CommentOperation[], operation: CommentOperation): boolean {
    return enabled.includes(operation);
}

export function useCommentHandlers<const TEnabled extends readonly CommentOperation[]>(
    family: CommentFamily,
    targetId: string,
    options: CommentHandlerOptions<TEnabled>,
): CommentHandlers<TEnabled[number]> {
    const mutations = commentFamilies[family];

    const { mutateAsync: createComment } = mutations.useCreate(targetId, options.entryId);
    const { mutateAsync: updateComment } = mutations.useUpdate(targetId);
    const { mutateAsync: deleteComment } = mutations.useDelete(targetId);
    const { mutateAsync: likeComment } = mutations.useLike(targetId);
    const { mutateAsync: unlikeComment } = mutations.useUnlike(targetId);
    const { mutateAsync: uploadCommentMedia } = mutations.useUploadMedia(targetId);

    const createCommentFn = useCallback<CreateCommentHandler>(
        (_targetId, body, parentId) => createComment({ body, parentId }),
        [createComment],
    );

    const updateFn = useCallback<UpdateCommentHandler>(
        async (commentId, body) => {
            await updateComment({ id: commentId, commentId, body });
        },
        [updateComment],
    );

    const deleteFn = useCallback<CommentIdHandler>(
        async commentId => {
            await deleteComment(commentId);
        },
        [deleteComment],
    );

    const likeFn = useCallback<CommentIdHandler>(
        async commentId => {
            await likeComment(commentId);
        },
        [likeComment],
    );

    const unlikeFn = useCallback<CommentIdHandler>(
        async commentId => {
            await unlikeComment(commentId);
        },
        [unlikeComment],
    );

    const uploadMediaFn = useCallback<UploadCommentMediaHandler>(
        (commentId, file) => uploadCommentMedia({ commentId, file }),
        [uploadCommentMedia],
    );

    const { enabled } = options;

    return {
        createCommentFn: isEnabled(enabled, "create") ? createCommentFn : undefined,
        updateFn: isEnabled(enabled, "update") ? updateFn : undefined,
        deleteFn: isEnabled(enabled, "delete") ? deleteFn : undefined,
        likeFn: isEnabled(enabled, "like") ? likeFn : undefined,
        unlikeFn: isEnabled(enabled, "unlike") ? unlikeFn : undefined,
        uploadMediaFn: isEnabled(enabled, "uploadMedia") ? uploadMediaFn : undefined,
    } as CommentHandlers<TEnabled[number]>;
}
