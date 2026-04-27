import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
    addBannedGif,
    adminDeleteUser,
    assignVanityRole,
    banUser,
    createAnnouncement,
    createGlobalBannedWord,
    createInvite,
    createVanityRole,
    deleteAnnouncement,
    deleteGlobalBannedWord,
    deleteInvite,
    deleteVanityRole,
    lockUser,
    pinAnnouncement,
    removeBannedGif,
    removeUserRole,
    resolveReport,
    setUserRole,
    unassignVanityRole,
    unbanUser,
    unlockUser,
    updateAdminSettings,
    updateAnnouncement,
    updateDetectiveScore,
    updateGlobalBannedWord,
    updateGMScore,
    updateVanityRole,
} from "../endpoints";
import type { CreateBannedWordRequest, SiteSettings } from "../../types/api";
import { queryKeys } from "../queryKeys";

export function useSetUserRole() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, role }: { id: string; role: string }) => setUserRole(id, role),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin"] }),
    });
}

export function useUpdateDetectiveScore() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, desiredScore }: { id: string; desiredScore: number }) =>
            updateDetectiveScore(id, desiredScore),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "user"] }),
    });
}

export function useUpdateGMScore() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, desiredScore }: { id: string; desiredScore: number }) => updateGMScore(id, desiredScore),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "user"] }),
    });
}

export function useRemoveUserRole() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, role }: { id: string; role: string }) => removeUserRole(id, role),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin"] }),
    });
}

export function useBanUser() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, reason }: { id: string; reason: string }) => banUser(id, reason),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin"] }),
    });
}

export function useUnbanUser() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unbanUser(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin"] }),
    });
}

export function useLockUser() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, reason }: { id: string; reason: string }) => lockUser(id, reason),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin"] }),
    });
}

export function useUnlockUser() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unlockUser(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin"] }),
    });
}

export function useAdminDeleteUser() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => adminDeleteUser(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin"] }),
    });
}

export function useUpdateAdminSettings() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (settings: SiteSettings) => updateAdminSettings(settings),
        onSuccess: () => {
            void qc.invalidateQueries({ queryKey: ["admin", "settings"] });
            void qc.invalidateQueries({ queryKey: ["site-info"] });
        },
    });
}

export function useCreateInvite() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: () => createInvite(),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.invites() }),
    });
}

export function useDeleteInvite() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (code: string) => deleteInvite(code),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.invites() }),
    });
}

export function useResolveReport() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, comment }: { id: number; comment: string }) => resolveReport(id, comment),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "reports"] }),
    });
}

export function useCreateGlobalBannedWord() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (req: CreateBannedWordRequest) => createGlobalBannedWord(req),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.bannedWords("global") }),
    });
}

export function useUpdateGlobalBannedWord() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ ruleId, req }: { ruleId: string; req: CreateBannedWordRequest }) =>
            updateGlobalBannedWord(ruleId, req),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.bannedWords("global") }),
    });
}

export function useDeleteGlobalBannedWord() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (ruleId: string) => deleteGlobalBannedWord(ruleId),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.bannedWords("global") }),
    });
}

export function useAddBannedGif() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: { input: string; reason?: string }) => addBannedGif(data),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.bannedGifs() }),
    });
}

export function useRemoveBannedGif() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ kind, value }: { kind: string; value: string }) => removeBannedGif(kind, value),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.bannedGifs() }),
    });
}

export function useCreateAnnouncement() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ title, body }: { title: string; body: string }) => createAnnouncement(title, body),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.announcements() }),
    });
}

export function useUpdateAnnouncement() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, title, body }: { id: string; title: string; body: string }) =>
            updateAnnouncement(id, title, body),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.announcements() }),
    });
}

export function useDeleteAnnouncement() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteAnnouncement(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.announcements() }),
    });
}

export function usePinAnnouncement() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, pinned }: { id: string; pinned: boolean }) => pinAnnouncement(id, pinned),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.announcements() }),
    });
}

export function useCreateVanityRole() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: { label: string; color: string; sort_order: number }) => createVanityRole(data),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.vanityRoles() }),
    });
}

export function useUpdateVanityRole() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: { label: string; color: string; sort_order: number } }) =>
            updateVanityRole(id, data),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.vanityRoles() }),
    });
}

export function useDeleteVanityRole() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteVanityRole(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.vanityRoles() }),
    });
}

export function useAssignVanityRole() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ roleId, userId }: { roleId: string; userId: string }) => assignVanityRole(roleId, userId),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "vanity-role-users"] }),
    });
}

export function useUnassignVanityRole() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ roleId, userId }: { roleId: string; userId: string }) => unassignVanityRole(roleId, userId),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "vanity-role-users"] }),
    });
}
