import { useMutation, useQueryClient } from "@tanstack/react-query";
import { blockUser, createReport, followUser, unblockUser, unfollowUser } from "../endpoints";

export function useFollowUser() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => followUser(id),
        onSuccess: (_d, id) => {
            void qc.invalidateQueries({ queryKey: ["follow-stats", id] });
            void qc.invalidateQueries({ queryKey: ["users", id] });
        },
    });
}

export function useUnfollowUser() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unfollowUser(id),
        onSuccess: (_d, id) => {
            void qc.invalidateQueries({ queryKey: ["follow-stats", id] });
            void qc.invalidateQueries({ queryKey: ["users", id] });
        },
    });
}

export function useBlockUser() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => blockUser(id),
        onSuccess: (_d, id) => {
            void qc.invalidateQueries({ queryKey: ["block-status", id] });
            void qc.invalidateQueries({ queryKey: ["blocked-users"] });
        },
    });
}

export function useUnblockUser() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unblockUser(id),
        onSuccess: (_d, id) => {
            void qc.invalidateQueries({ queryKey: ["block-status", id] });
            void qc.invalidateQueries({ queryKey: ["blocked-users"] });
        },
    });
}

export function useCreateReport() {
    return useMutation({
        mutationFn: ({
            targetType,
            targetId,
            reason,
            contextId,
        }: {
            targetType: string;
            targetId: string;
            reason: string;
            contextId?: string;
        }) => createReport(targetType, targetId, reason, contextId),
    });
}
