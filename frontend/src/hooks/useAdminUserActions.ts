import { useState } from "react";
import { useNavigate } from "react-router";
import { useAdminUser } from "./queries/admin";
import {
    useAdminDeleteUser,
    useApproveUser,
    useBanUser,
    useForceLogoutUser,
    useLockUser,
    useRemoveUserRole,
    useResetUserPassword,
    useSetDisplayNameLock,
    useSetUserDisplayName,
    useSetUserEmail,
    useSetUserRole,
    useUnapproveUser,
    useUnbanUser,
    useUnlockUser,
    useUnverifyUserEmail,
    useUpdateDetectiveScore,
    useUpdateGMScore,
    useVerifyUserEmail,
} from "./mutations/admin";
import { useAuth } from "./useAuth";
import { adminUserCapabilities, type AdminUserCapabilities } from "../domain/adminTargets";
import { errorMessage } from "../utils/errorMessage";
import type { AdminUserDetail } from "../types/api";

const SCORE_PATTERN = /^-?\d*$/;

export type AdminUserConfirmation = "unverifyEmail" | "forceLogout" | "resetPassword" | "deleteUser";

interface ActionSpec<T> {
    ready: boolean;
    failure: string;
    success?: string;
    onDone?: (result: T) => void;
    onFailed?: () => void;
}

interface AccountDraft {
    userId: string | null;
    email: string | null;
    displayName: string | null;
}

interface ScoreDraft {
    userId: string | null;
    detective: string | null;
    gm: string | null;
}

export interface AdminUserPending {
    saveEmail: boolean;
    verifyEmail: boolean;
    unverifyEmail: boolean;
    saveDisplayName: boolean;
    toggleDisplayNameLock: boolean;
    forceLogout: boolean;
    resetPassword: boolean;
}

export interface AdminUserActions {
    user: AdminUserDetail | null;
    loading: boolean;
    error: string;
    feedback: string;
    capabilities: AdminUserCapabilities;
    pending: AdminUserPending;
    confirmation: AdminUserConfirmation | null;
    resetPasswordResult: string | null;
    selectedRole: string;
    banReason: string;
    lockReason: string;
    emailInput: string;
    displayNameInput: string;
    detectiveScoreInput: string;
    gmScoreInput: string;
    setSelectedRole: (value: string) => void;
    setBanReason: (value: string) => void;
    setLockReason: (value: string) => void;
    setEmailInput: (value: string) => void;
    setDisplayNameInput: (value: string) => void;
    setDetectiveScoreInput: (value: string) => void;
    setGMScoreInput: (value: string) => void;
    dismissResetPasswordResult: () => void;
    saveEmail: () => Promise<void>;
    verifyEmail: () => Promise<void>;
    saveDisplayName: () => Promise<void>;
    toggleDisplayNameLock: () => Promise<void>;
    assignRole: () => Promise<void>;
    removeRole: () => Promise<void>;
    banUser: () => Promise<void>;
    unbanUser: () => Promise<void>;
    lockUser: () => Promise<void>;
    unlockUser: () => Promise<void>;
    approveAccount: () => Promise<void>;
    revokeApproval: () => Promise<void>;
    saveDetectiveScore: () => Promise<void>;
    saveGMScore: () => Promise<void>;
    requestUnverifyEmail: () => void;
    requestForceLogout: () => void;
    requestResetPassword: () => void;
    requestDeleteUser: () => void;
    confirmPending: () => Promise<void>;
    cancelPending: () => void;
}

export function useAdminUserActions(id: string | undefined): AdminUserActions {
    const navigate = useNavigate();
    const { user: currentUser } = useAuth();

    const targetId = id ?? "";
    const hasTarget = targetId !== "";
    const { user, loading } = useAdminUser(targetId);

    const [error, setError] = useState("");
    const [feedback, setFeedback] = useState("");
    const [selectedRole, setSelectedRole] = useState("admin");
    const [banReason, setBanReason] = useState("");
    const [lockReason, setLockReason] = useState("");
    const [confirmation, setConfirmation] = useState<AdminUserConfirmation | null>(null);
    const [resetPasswordResult, setResetPasswordResult] = useState<string | null>(null);
    const [accountDraft, setAccountDraft] = useState<AccountDraft>({ userId: null, email: null, displayName: null });
    const [scoreDraft, setScoreDraft] = useState<ScoreDraft>({ userId: null, detective: null, gm: null });

    const setRoleMutation = useSetUserRole();
    const removeRoleMutation = useRemoveUserRole();
    const banUserMutation = useBanUser();
    const unbanUserMutation = useUnbanUser();
    const lockUserMutation = useLockUser();
    const unlockUserMutation = useUnlockUser();
    const approveUserMutation = useApproveUser();
    const unapproveUserMutation = useUnapproveUser();
    const deleteUserMutation = useAdminDeleteUser();
    const updateDetectiveScoreMutation = useUpdateDetectiveScore();
    const updateGMScoreMutation = useUpdateGMScore();
    const resetPasswordMutation = useResetUserPassword();
    const setEmailMutation = useSetUserEmail();
    const verifyEmailMutation = useVerifyUserEmail();
    const unverifyEmailMutation = useUnverifyUserEmail();
    const setDisplayNameMutation = useSetUserDisplayName();
    const setDisplayNameLockMutation = useSetDisplayNameLock();
    const forceLogoutMutation = useForceLogoutUser();

    const targetKey = user?.id ?? null;
    const activeAccountDraft: AccountDraft =
        accountDraft.userId === targetKey ? accountDraft : { userId: targetKey, email: null, displayName: null };
    const activeScoreDraft: ScoreDraft =
        scoreDraft.userId === targetKey ? scoreDraft : { userId: targetKey, detective: null, gm: null };

    const emailInput = activeAccountDraft.email ?? user?.email ?? "";
    const displayNameInput = activeAccountDraft.displayName ?? user?.display_name ?? "";
    const detectiveScoreInput = activeScoreDraft.detective ?? (user ? String(user.detective_score) : "0");
    const gmScoreInput = activeScoreDraft.gm ?? (user ? String(user.gm_score) : "0");

    function patchAccountDraft(update: { email?: string | null; displayName?: string | null }) {
        setAccountDraft(prev => {
            const base = prev.userId === targetKey ? prev : { userId: targetKey, email: null, displayName: null };

            return { ...base, ...update };
        });
    }

    function patchScoreDraft(update: { detective?: string | null; gm?: string | null }) {
        setScoreDraft(prev => {
            const base = prev.userId === targetKey ? prev : { userId: targetKey, detective: null, gm: null };

            return { ...base, ...update };
        });
    }

    async function perform<T>(spec: ActionSpec<T>, run: () => Promise<T>): Promise<void> {
        if (!spec.ready) {
            return;
        }

        try {
            const result = await run();

            spec.onDone?.(result);

            if (spec.success !== undefined) {
                setFeedback(spec.success);
            }
        } catch (e) {
            setError(errorMessage(e, spec.failure));
            spec.onFailed?.();
        }
    }

    function setEmailInput(value: string) {
        patchAccountDraft({ email: value });
    }

    function setDisplayNameInput(value: string) {
        patchAccountDraft({ displayName: value });
    }

    function setDetectiveScoreInput(value: string) {
        if (!SCORE_PATTERN.test(value)) {
            return;
        }

        patchScoreDraft({ detective: value });
    }

    function setGMScoreInput(value: string) {
        if (!SCORE_PATTERN.test(value)) {
            return;
        }

        patchScoreDraft({ gm: value });
    }

    function saveEmail(): Promise<void> {
        const email = emailInput.trim();

        return perform(
            {
                ready: hasTarget && email !== "",
                success: "Email updated. A verification link was sent to the new address.",
                failure: "Failed to change email",
                onDone: () => patchAccountDraft({ email: null }),
            },
            () => setEmailMutation.mutateAsync({ id: targetId, email }),
        );
    }

    function verifyEmail(): Promise<void> {
        return perform(
            {
                ready: hasTarget,
                success: "Email marked as verified",
                failure: "Failed to verify email",
            },
            () => verifyEmailMutation.mutateAsync(targetId),
        );
    }

    function unverifyEmail(): Promise<void> {
        return perform(
            {
                ready: hasTarget,
                success: "Email marked as unverified",
                failure: "Failed to unverify email",
            },
            () => unverifyEmailMutation.mutateAsync(targetId),
        );
    }

    function saveDisplayName(): Promise<void> {
        const displayName = displayNameInput.trim();

        return perform(
            {
                ready: hasTarget && displayName !== "",
                success: "Display name updated",
                failure: "Failed to change display name",
                onDone: () => patchAccountDraft({ displayName: null }),
            },
            () => setDisplayNameMutation.mutateAsync({ id: targetId, displayName }),
        );
    }

    function toggleDisplayNameLock(): Promise<void> {
        const locked = user?.display_name_locked ?? false;

        return perform(
            {
                ready: hasTarget && user !== null,
                success: locked ? "Display name unlocked" : "Display name locked",
                failure: "Failed to change display name lock",
            },
            () => setDisplayNameLockMutation.mutateAsync({ id: targetId, locked: !locked }),
        );
    }

    function forceLogout(): Promise<void> {
        return perform(
            {
                ready: hasTarget,
                success: "All sessions revoked",
                failure: "Failed to revoke sessions",
            },
            () => forceLogoutMutation.mutateAsync(targetId),
        );
    }

    function assignRole(): Promise<void> {
        return perform(
            {
                ready: hasTarget,
                success: "Role assigned",
                failure: "Failed to set role",
            },
            () => setRoleMutation.mutateAsync({ id: targetId, role: selectedRole }),
        );
    }

    function removeRole(): Promise<void> {
        const role = user?.role ?? "";

        return perform(
            {
                ready: hasTarget && role !== "",
                success: "Role removed",
                failure: "Failed to remove role",
            },
            () => removeRoleMutation.mutateAsync({ id: targetId, role }),
        );
    }

    function banUser(): Promise<void> {
        const reason = banReason.trim();

        return perform(
            {
                ready: hasTarget && reason !== "",
                success: "User banned",
                failure: "Failed to ban user",
                onDone: () => setBanReason(""),
            },
            () => banUserMutation.mutateAsync({ id: targetId, reason }),
        );
    }

    function unbanUser(): Promise<void> {
        return perform(
            {
                ready: hasTarget,
                success: "User unbanned",
                failure: "Failed to unban user",
            },
            () => unbanUserMutation.mutateAsync(targetId),
        );
    }

    function lockUser(): Promise<void> {
        const reason = lockReason.trim();

        return perform(
            {
                ready: hasTarget && reason !== "",
                success: "User locked",
                failure: "Failed to lock user",
                onDone: () => setLockReason(""),
            },
            () => lockUserMutation.mutateAsync({ id: targetId, reason }),
        );
    }

    function unlockUser(): Promise<void> {
        return perform(
            {
                ready: hasTarget,
                success: "User unlocked",
                failure: "Failed to unlock user",
            },
            () => unlockUserMutation.mutateAsync(targetId),
        );
    }

    function approveAccount(): Promise<void> {
        return perform(
            {
                ready: hasTarget,
                success: "Account approved, restrictions lifted",
                failure: "Failed to approve account",
            },
            () => approveUserMutation.mutateAsync(targetId),
        );
    }

    function revokeApproval(): Promise<void> {
        return perform(
            {
                ready: hasTarget,
                success: "Approval revoked",
                failure: "Failed to revoke approval",
            },
            () => unapproveUserMutation.mutateAsync(targetId),
        );
    }

    function saveDetectiveScore(): Promise<void> {
        const desiredScore = parseInt(detectiveScoreInput, 10) || 0;

        return perform(
            {
                ready: user !== null,
                failure: "Failed to update detective score",
            },
            () => updateDetectiveScoreMutation.mutateAsync({ id: user?.id ?? "", desiredScore }),
        );
    }

    function saveGMScore(): Promise<void> {
        const desiredScore = parseInt(gmScoreInput, 10) || 0;

        return perform(
            {
                ready: user !== null,
                failure: "Failed to update game master score",
            },
            () => updateGMScoreMutation.mutateAsync({ id: user?.id ?? "", desiredScore }),
        );
    }

    function deleteUser(): Promise<void> {
        return perform(
            {
                ready: hasTarget,
                failure: "Failed to delete user",
                onDone: () => navigate("/admin/users"),
                onFailed: () => setConfirmation(null),
            },
            () => deleteUserMutation.mutateAsync(targetId),
        );
    }

    function resetPassword(): Promise<void> {
        return perform(
            {
                ready: hasTarget,
                failure: "Failed to reset password",
                onDone: result => setResetPasswordResult(result.password),
            },
            () => resetPasswordMutation.mutateAsync(targetId),
        );
    }

    async function confirmPending(): Promise<void> {
        switch (confirmation) {
            case "unverifyEmail":
                setConfirmation(null);
                await unverifyEmail();
                return;
            case "forceLogout":
                setConfirmation(null);
                await forceLogout();
                return;
            case "resetPassword":
                setConfirmation(null);
                await resetPassword();
                return;
            case "deleteUser":
                await deleteUser();
                return;
            default:
                return;
        }
    }

    return {
        user,
        loading,
        error,
        feedback,
        capabilities: adminUserCapabilities(currentUser, user),
        pending: {
            saveEmail: setEmailMutation.isPending,
            verifyEmail: verifyEmailMutation.isPending,
            unverifyEmail: unverifyEmailMutation.isPending,
            saveDisplayName: setDisplayNameMutation.isPending,
            toggleDisplayNameLock: setDisplayNameLockMutation.isPending,
            forceLogout: forceLogoutMutation.isPending,
            resetPassword: resetPasswordMutation.isPending,
        },
        confirmation,
        resetPasswordResult,
        selectedRole,
        banReason,
        lockReason,
        emailInput,
        displayNameInput,
        detectiveScoreInput,
        gmScoreInput,
        setSelectedRole,
        setBanReason,
        setLockReason,
        setEmailInput,
        setDisplayNameInput,
        setDetectiveScoreInput,
        setGMScoreInput,
        dismissResetPasswordResult: () => setResetPasswordResult(null),
        saveEmail,
        verifyEmail,
        saveDisplayName,
        toggleDisplayNameLock,
        assignRole,
        removeRole,
        banUser,
        unbanUser,
        lockUser,
        unlockUser,
        approveAccount,
        revokeApproval,
        saveDetectiveScore,
        saveGMScore,
        requestUnverifyEmail: () => setConfirmation("unverifyEmail"),
        requestForceLogout: () => setConfirmation("forceLogout"),
        requestResetPassword: () => setConfirmation("resetPassword"),
        requestDeleteUser: () => setConfirmation("deleteUser"),
        confirmPending,
        cancelPending: () => setConfirmation(null),
    };
}
