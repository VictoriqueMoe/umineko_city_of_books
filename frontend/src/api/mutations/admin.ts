import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { QueryClient } from "@tanstack/react-query";
import {
    addBannedGif,
    adminDeleteUser,
    assignVanityRole,
    banUser,
    createAnnouncement,
    createChatbot,
    createChatbotBasePrompt,
    deleteChatbotBasePrompt,
    updateChatbotBasePrompt,
    createGlobalBannedWord,
    createInvite,
    createVanityRole,
    deleteAnnouncement,
    deleteChatbot,
    deleteGlobalBannedWord,
    deleteInvite,
    deleteVanityRole,
    forceLogoutUser,
    lockUser,
    pinAnnouncement,
    removeBannedGif,
    removeUserRole,
    resetUserPassword,
    resolveReport,
    sendTestEmail,
    setDisplayNameLock,
    setUserDisplayName,
    setUserEmail,
    setUserRole,
    testChatbotModel,
    unassignVanityRole,
    unbanUser,
    unlockUser,
    unverifyUserEmail,
    updateAdminSettings,
    updateAnnouncement,
    updateChatbot,
    updateDetectiveScore,
    updateGlobalBannedWord,
    updateGMScore,
    updateRolePermissions,
    updateVanityRole,
    updateVanityRolePermissions,
    uploadOGDefaultImage,
    verifyUserEmail,
} from "../endpoints";
import type { ChatbotBasePromptPayload, ChatbotPayload, CreateBannedWordRequest, SiteSettings } from "../../types/api";
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

export function useResetUserPassword() {
    return useMutation({
        mutationFn: (id: string) => resetUserPassword(id),
    });
}

export function useSetUserEmail() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, email }: { id: string; email: string }) => setUserEmail(id, email),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "user"] }),
    });
}

export function useVerifyUserEmail() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => verifyUserEmail(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "user"] }),
    });
}

export function useUnverifyUserEmail() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => unverifyUserEmail(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "user"] }),
    });
}

export function useSetUserDisplayName() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, displayName }: { id: string; displayName: string }) => setUserDisplayName(id, displayName),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ["admin"] });
            qc.invalidateQueries({ queryKey: ["profile"] });
        },
    });
}

export function useSetDisplayNameLock() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, locked }: { id: string; locked: boolean }) => setDisplayNameLock(id, locked),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "user"] }),
    });
}

export function useForceLogoutUser() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => forceLogoutUser(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: ["admin", "user"] }),
    });
}

export function useUpdateAdminSettings() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (settings: SiteSettings) => updateAdminSettings(settings),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ["admin", "settings"] });
            qc.invalidateQueries({ queryKey: queryKeys.admin.chatbotModels() });
            qc.invalidateQueries({ queryKey: ["site-info"] });
        },
    });
}

export function useSendTestEmail() {
    return useMutation({
        mutationFn: () => sendTestEmail(),
    });
}

export function useUploadOGDefaultImage() {
    return useMutation({
        mutationFn: (file: File) => uploadOGDefaultImage(file),
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
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.admin.announcements() });
            qc.invalidateQueries({ queryKey: ["announcements"] });
        },
    });
}

export function useUpdateAnnouncement() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, title, body }: { id: string; title: string; body: string }) =>
            updateAnnouncement(id, title, body),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.admin.announcements() });
            qc.invalidateQueries({ queryKey: ["announcements"] });
        },
    });
}

export function useDeleteAnnouncement() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteAnnouncement(id),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.admin.announcements() });
            qc.invalidateQueries({ queryKey: ["announcements"] });
        },
    });
}

export function usePinAnnouncement() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, pinned }: { id: string; pinned: boolean }) => pinAnnouncement(id, pinned),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: queryKeys.admin.announcements() });
            qc.invalidateQueries({ queryKey: ["announcements"] });
        },
    });
}

async function invalidateVanityRoleViews(qc: QueryClient) {
    await Promise.all([
        qc.invalidateQueries({ queryKey: queryKeys.admin.vanityRoles() }),
        qc.invalidateQueries({ queryKey: queryKeys.admin.permissions() }),
    ]);
}

export function useCreateVanityRole() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: { label: string; color: string; sort_order: number }) => createVanityRole(data),
        onSuccess: () => invalidateVanityRoleViews(qc),
    });
}

export function useUpdateVanityRole() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: { label: string; color: string; sort_order: number } }) =>
            updateVanityRole(id, data),
        onSuccess: () => invalidateVanityRoleViews(qc),
    });
}

export function useDeleteVanityRole() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteVanityRole(id),
        onSuccess: () => invalidateVanityRoleViews(qc),
    });
}

export function useUpdateRolePermissions() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ role, permissions }: { role: string; permissions: string[] }) =>
            updateRolePermissions(role, permissions),
        onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.admin.permissions() }),
    });
}

export function useUpdateVanityRolePermissions() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, permissions }: { id: string; permissions: string[] }) =>
            updateVanityRolePermissions(id, permissions),
        onSuccess: () => invalidateVanityRoleViews(qc),
    });
}

async function invalidateChatbotViews(qc: QueryClient) {
    await Promise.all([
        qc.invalidateQueries({ queryKey: queryKeys.admin.chatbots() }),
        qc.invalidateQueries({ queryKey: queryKeys.admin.chatbotBasePrompts() }),
        qc.invalidateQueries({ queryKey: queryKeys.chatbots.all }),
    ]);
}

async function invalidateBasePromptViews(qc: QueryClient) {
    await qc.invalidateQueries({ queryKey: queryKeys.admin.chatbotBasePrompts() });
}

export function useCreateChatbotBasePrompt() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: ChatbotBasePromptPayload) => createChatbotBasePrompt(data),
        onSuccess: () => invalidateBasePromptViews(qc),
    });
}

export function useUpdateChatbotBasePrompt() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: ChatbotBasePromptPayload }) => updateChatbotBasePrompt(id, data),
        onSuccess: () => invalidateBasePromptViews(qc),
    });
}

export function useDeleteChatbotBasePrompt() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteChatbotBasePrompt(id),
        onSuccess: () => invalidateBasePromptViews(qc),
    });
}

export function useCreateChatbot() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (data: ChatbotPayload) => createChatbot(data),
        onSuccess: () => invalidateChatbotViews(qc),
    });
}

export function useUpdateChatbot() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: ({ id, data }: { id: string; data: ChatbotPayload }) => updateChatbot(id, data),
        onSuccess: () => invalidateChatbotViews(qc),
    });
}

export function useDeleteChatbot() {
    const qc = useQueryClient();
    return useMutation({
        mutationFn: (id: string) => deleteChatbot(id),
        onSuccess: () => invalidateChatbotViews(qc),
    });
}

export function useTestChatbotModel() {
    return useMutation({
        mutationFn: (model: string) => testChatbotModel(model),
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
