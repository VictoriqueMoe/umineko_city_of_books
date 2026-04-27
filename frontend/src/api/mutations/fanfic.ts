import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    createFanfic,
    createFanficChapter,
    createFanficComment,
    deleteFanfic,
    deleteFanficChapter,
    deleteFanficComment,
    deleteFanficCover,
    favouriteFanfic,
    likeFanficComment,
    unfavouriteFanfic,
    unlikeFanficComment,
    updateFanfic,
    updateFanficChapter,
    updateFanficComment,
    uploadFanficCommentMedia,
    uploadFanficCover,
} from "../endpoints";
import { queryKeys } from "../queryKeys";

const detail = (id: string) => queryKeys.fanfic.detail(id);

export function useCreateFanfic() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: Parameters<typeof createFanfic>[0]) => createFanfic(data),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.fanfic.all }),
    });
}

export function useUpdateFanfic(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: Parameters<typeof updateFanfic>[1]) => updateFanfic(id, data),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(id) }),
    });
}

export function useDeleteFanfic() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteFanfic(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.fanfic.all }),
    });
}

export function useUploadFanficCover(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (file: File) => uploadFanficCover(id, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(id) }),
    });
}

export function useUploadFanficCoverFor() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, file }: { id: string; file: File }) => uploadFanficCover(id, file),
        onSuccess: (_d, vars) => qc.invalidateQueries({ queryKey: detail(vars.id) }),
    });
}

export function useDeleteFanficCover(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: () => deleteFanficCover(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(id) }),
    });
}

export function useCreateFanficChapter(fanficId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ title, body }: { title: string; body: string }) => createFanficChapter(fanficId, title, body),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(fanficId) }),
    });
}

export function useUpdateFanficChapter(fanficId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ chapterId, title, body }: { chapterId: string; title: string; body: string }) =>
            updateFanficChapter(chapterId, title, body),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(fanficId) }),
    });
}

export function useDeleteFanficChapter(fanficId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (chapterId: string) => deleteFanficChapter(chapterId),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(fanficId) }),
    });
}

export function useFavouriteFanfic() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => favouriteFanfic(id),
        onSuccess: (_d, id) => qc.invalidateQueries({ queryKey: detail(id) }),
    });
}

export function useUnfavouriteFanfic() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unfavouriteFanfic(id),
        onSuccess: (_d, id) => qc.invalidateQueries({ queryKey: detail(id) }),
    });
}

export function useCreateFanficComment(fanficId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) =>
            createFanficComment(fanficId, body, parentId),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(fanficId) }),
    });
}

export function useUpdateFanficComment(fanficId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, body }: { id: string; body: string }) => updateFanficComment(id, body),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(fanficId) }),
    });
}

export function useDeleteFanficComment(fanficId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteFanficComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(fanficId) }),
    });
}

export function useLikeFanficComment(fanficId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => likeFanficComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(fanficId) }),
    });
}

export function useUnlikeFanficComment(fanficId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unlikeFanficComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(fanficId) }),
    });
}

export function useUploadFanficCommentMedia(fanficId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ commentId, file }: { commentId: string; file: File }) =>
            uploadFanficCommentMedia(commentId, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(fanficId) }),
    });
}
