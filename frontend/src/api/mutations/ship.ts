import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    createShip,
    createShipComment,
    deleteShip,
    deleteShipComment,
    likeShipComment,
    unlikeShipComment,
    updateShip,
    updateShipComment,
    uploadShipCommentMedia,
    uploadShipImage,
    voteShip,
} from "../endpoints";
import type { ShipCharacter } from "../../types/api";
import { queryKeys } from "../queryKeys";

const detail = (id: string) => queryKeys.ship.detail(id);

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
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(id) }),
    });
}

export function useDeleteShip() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteShip(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.ship.all }),
    });
}

export function useUploadShipImage(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (file: File) => uploadShipImage(id, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(id) }),
    });
}

export function useUploadShipImageById() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, file }: { id: string; file: File }) => uploadShipImage(id, file),
        onSuccess: (_data, vars) => qc.invalidateQueries({ queryKey: detail(vars.id) }),
    });
}

export function useVoteShip(id: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (value: number) => voteShip(id, value),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(id) }),
    });
}

export function useCreateShipComment(shipId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ body, parentId }: { body: string; parentId?: string }) =>
            createShipComment(shipId, body, parentId),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(shipId) }),
    });
}

export function useUpdateShipComment(shipId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, body }: { id: string; body: string }) => updateShipComment(id, body),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(shipId) }),
    });
}

export function useDeleteShipComment(shipId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteShipComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(shipId) }),
    });
}

export function useLikeShipComment(shipId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => likeShipComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(shipId) }),
    });
}

export function useUnlikeShipComment(shipId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unlikeShipComment(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(shipId) }),
    });
}

export function useUploadShipCommentMedia(shipId: string) {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ commentId, file }: { commentId: string; file: File }) => uploadShipCommentMedia(commentId, file),
        onSuccess: () => qc.invalidateQueries({ queryKey: detail(shipId) }),
    });
}
