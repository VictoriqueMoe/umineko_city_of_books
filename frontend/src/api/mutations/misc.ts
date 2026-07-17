import { useMutation, useQueryClient } from "@tanstack/react-query";
import { blockUser, createReport, followUser, unblockUser, unfollowUser } from "../endpoints";
import { queryKeys } from "../queryKeys";
import { useAuth } from "../../hooks/useAuth";

export function useFollowUser() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => followUser(id),
        onSuccess: (_d, id) => {
            qc.invalidateQueries({ queryKey: ["follow-stats", id] });
            qc.invalidateQueries({ queryKey: ["users", id] });
        },
    });
}

export function useUnfollowUser() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unfollowUser(id),
        onSuccess: (_d, id) => {
            qc.invalidateQueries({ queryKey: ["follow-stats", id] });
            qc.invalidateQueries({ queryKey: ["users", id] });
        },
    });
}

export function useBlockUser() {
    const qc = useQueryClient();
    const { user } = useAuth();
    return useMutation({
        mutationFn: (id: string) => blockUser(id),
        onSuccess: (_d, id) => {
            qc.invalidateQueries({ queryKey: ["block-status", id] });
            qc.invalidateQueries({ queryKey: queryKeys.profile.blockedUsers(user?.id ?? "") });
        },
    });
}

export function useUnblockUser() {
    const qc = useQueryClient();
    const { user } = useAuth();
    return useMutation({
        mutationFn: (id: string) => unblockUser(id),
        onSuccess: (_d, id) => {
            qc.invalidateQueries({ queryKey: ["block-status", id] });
            qc.invalidateQueries({ queryKey: queryKeys.profile.blockedUsers(user?.id ?? "") });
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
