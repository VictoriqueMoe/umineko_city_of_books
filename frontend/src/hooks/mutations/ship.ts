import { useMutation, useQueryClient } from "@tanstack/react-query";
import { createShip, deleteShip, updateShip, uploadShipImage, voteShip } from "../../api/endpoints/ship";
import {
    createShipComment,
    deleteShipComment,
    likeShipComment,
    unlikeShipComment,
    updateShipComment,
    uploadShipCommentMedia,
} from "../../api/endpoints/comments";
import type { ShipCharacter } from "../../types/api";
import { queryKeys } from "../../api/queryKeys";
import { commentMutations } from "./comments";

export function useCreateShip() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: { title: string; description: string; characters: ShipCharacter[] }) => createShip(data),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.ship.all }),
    });
}

export function useUpdateShip(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: { title: string; description: string; characters: ShipCharacter[] }) => updateShip(id, data),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.ship.all }),
    });
}

export function useDeleteShip() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteShip(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.ship.all }),
    });
}

export function useUploadShipImageById() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, file }: { id: string; file: File }) => uploadShipImage(id, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.ship.all }),
    });
}

export function useVoteShip(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (value: number) => voteShip(id, value),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.ship.all }),
    });
}

export function useCreateShipComment(shipId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) =>
            createShipComment(shipId, body, parentId),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.ship.all }),
    });
}

const shipCommentMutations = commentMutations(queryKeys.ship.all, {
    update: (id, body) => updateShipComment(id, body),
    remove: id => deleteShipComment(id),
    like: id => likeShipComment(id),
    unlike: id => unlikeShipComment(id),
    uploadMedia: (commentId, file) => uploadShipCommentMedia(commentId, file),
});

export const useUpdateShipComment = shipCommentMutations.useUpdate;
export const useDeleteShipComment = shipCommentMutations.useDelete;
export const useLikeShipComment = shipCommentMutations.useLike;
export const useUnlikeShipComment = shipCommentMutations.useUnlike;
export const useUploadShipCommentMedia = shipCommentMutations.useUploadMedia;
