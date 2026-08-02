import { useState } from "react";
import { useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useAdminUser, useUserAuditLog, useUserIPMatches } from "../../api/queries/admin";
import {
    useAdminDeleteUser,
    useBanUser,
    useForceLogoutUser,
    useLockUser,
    useRemoveUserRole,
    useResetUserPassword,
    useSetDisplayNameLock,
    useSetUserDisplayName,
    useSetUserEmail,
    useSetUserRole,
    useUnbanUser,
    useUnlockUser,
    useUnverifyUserEmail,
    useUpdateDetectiveScore,
    useUpdateGMScore,
    useVerifyUserEmail,
} from "../../api/mutations/admin";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Modal } from "../../components/Modal/Modal";
import { Pagination } from "../../components/Pagination/Pagination";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { RolePill } from "../../components/RolePill/RolePill";
import { Select } from "../../components/Select/Select";
import { useAuth } from "../../hooks/useAuth";
import { auditActionLabel, parseAuditDetails } from "../../utils/auditLog";
import { can } from "../../utils/permissions";
import { formatDate, formatFullDateTime } from "../../utils/time";
import styles from "./AdminUserDetail.module.css";

const HISTORY_LIMIT = 10;

export function AdminUserDetail() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { user: currentUser } = useAuth();
    const { user, loading } = useAdminUser(id ?? "");
    usePageTitle(user ? `Admin - ${user.display_name}` : "Admin - User");
    const [error, setError] = useState("");
    const [feedback, setFeedback] = useState("");

    const setRoleMutation = useSetUserRole();
    const removeRoleMutation = useRemoveUserRole();
    const banUserMutation = useBanUser();
    const unbanUserMutation = useUnbanUser();
    const lockUserMutation = useLockUser();
    const unlockUserMutation = useUnlockUser();
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

    const isProtectedTarget = user?.role === "super_admin";
    const canManageAccount = can(currentUser?.role, "manage_user_account") && !isProtectedTarget;
    const canManageEmail = can(currentUser?.role, "manage_user_email") && !isProtectedTarget;
    const canSetEmailVerified = can(currentUser?.role, "set_email_verified") && !isProtectedTarget;
    const [historyOffset, setHistoryOffset] = useState(0);
    const ipMatches = useUserIPMatches(id ?? "", can(currentUser?.role, "view_users") && !!user?.ip);
    const moderationHistory = useUserAuditLog(
        id ?? "",
        can(currentUser?.role, "view_audit_log"),
        HISTORY_LIMIT,
        historyOffset,
    );

    const [selectedRole, setSelectedRole] = useState("admin");
    const [banReason, setBanReason] = useState("");
    const [lockReason, setLockReason] = useState("");
    const [deleteModalOpen, setDeleteModalOpen] = useState(false);
    const [resetPasswordResult, setResetPasswordResult] = useState<string | null>(null);
    const [accountDraft, setAccountDraft] = useState<{
        userId: string | null;
        email: string | null;
        displayName: string | null;
    }>({
        userId: null,
        email: null,
        displayName: null,
    });
    const [scoreDraft, setScoreDraft] = useState<{
        userId: string | null;
        detective: string | null;
        gm: string | null;
    }>({
        userId: null,
        detective: null,
        gm: null,
    });
    const activeScoreDraft =
        scoreDraft.userId === (user?.id ?? null) ? scoreDraft : { userId: user?.id ?? null, detective: null, gm: null };
    const detectiveScoreInput = activeScoreDraft.detective ?? (user ? String(user.detective_score) : "0");
    const gmScoreInput = activeScoreDraft.gm ?? (user ? String(user.gm_score) : "0");

    function setDetectiveScoreInput(value: string) {
        setScoreDraft(prev => {
            const base =
                prev.userId === (user?.id ?? null) ? prev : { userId: user?.id ?? null, detective: null, gm: null };
            return { ...base, detective: value };
        });
    }
    function setGMScoreInput(value: string) {
        setScoreDraft(prev => {
            const base =
                prev.userId === (user?.id ?? null) ? prev : { userId: user?.id ?? null, detective: null, gm: null };
            return { ...base, gm: value };
        });
    }

    const activeAccountDraft =
        accountDraft.userId === (user?.id ?? null)
            ? accountDraft
            : { userId: user?.id ?? null, email: null, displayName: null };
    const emailInput = activeAccountDraft.email ?? user?.email ?? "";
    const displayNameInput = activeAccountDraft.displayName ?? user?.display_name ?? "";

    function patchAccountDraft(update: { email?: string | null; displayName?: string | null }) {
        setAccountDraft(prev => {
            const base =
                prev.userId === (user?.id ?? null)
                    ? prev
                    : { userId: user?.id ?? null, email: null, displayName: null };
            return { ...base, ...update };
        });
    }

    async function handleSetEmail() {
        if (!id || !emailInput.trim()) {
            return;
        }
        try {
            await setEmailMutation.mutateAsync({ id, email: emailInput.trim() });
            patchAccountDraft({ email: null });
            setFeedback("Email updated. A verification link was sent to the new address.");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to change email");
        }
    }

    async function handleVerifyEmail() {
        if (!id) {
            return;
        }
        try {
            await verifyEmailMutation.mutateAsync(id);
            setFeedback("Email marked as verified");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to verify email");
        }
    }

    async function handleUnverifyEmail() {
        if (!id) {
            return;
        }
        if (
            !window.confirm(
                "Mark this email unverified? The user will be blocked from posting, commenting and messaging until they verify it again.",
            )
        ) {
            return;
        }
        try {
            await unverifyEmailMutation.mutateAsync(id);
            setFeedback("Email marked as unverified");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to unverify email");
        }
    }

    async function handleSetDisplayName() {
        if (!id || !displayNameInput.trim()) {
            return;
        }
        try {
            await setDisplayNameMutation.mutateAsync({ id, displayName: displayNameInput.trim() });
            patchAccountDraft({ displayName: null });
            setFeedback("Display name updated");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to change display name");
        }
    }

    async function handleToggleDisplayNameLock() {
        if (!id || !user) {
            return;
        }
        try {
            await setDisplayNameLockMutation.mutateAsync({ id, locked: !user.display_name_locked });
            setFeedback(user.display_name_locked ? "Display name unlocked" : "Display name locked");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to change display name lock");
        }
    }

    async function handleForceLogout() {
        if (!id) {
            return;
        }
        if (!window.confirm("Sign this user out of every device? Their password is unchanged.")) {
            return;
        }
        try {
            await forceLogoutMutation.mutateAsync(id);
            setFeedback("All sessions revoked");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to revoke sessions");
        }
    }

    async function handleSetRole() {
        if (!id) {
            return;
        }
        try {
            await setRoleMutation.mutateAsync({ id, role: selectedRole });
            setFeedback("Role assigned");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to set role");
        }
    }

    async function handleRemoveRole() {
        if (!id || !user?.role) {
            return;
        }
        try {
            await removeRoleMutation.mutateAsync({ id, role: user.role });
            setFeedback("Role removed");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to remove role");
        }
    }

    async function handleBan() {
        if (!id || !banReason.trim()) {
            return;
        }
        try {
            await banUserMutation.mutateAsync({ id, reason: banReason.trim() });
            setBanReason("");
            setFeedback("User banned");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to ban user");
        }
    }

    async function handleUnban() {
        if (!id) {
            return;
        }
        try {
            await unbanUserMutation.mutateAsync(id);
            setFeedback("User unbanned");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to unban user");
        }
    }

    async function handleLock() {
        if (!id || !lockReason.trim()) {
            return;
        }
        try {
            await lockUserMutation.mutateAsync({ id, reason: lockReason.trim() });
            setLockReason("");
            setFeedback("User locked");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to lock user");
        }
    }

    async function handleUnlock() {
        if (!id) {
            return;
        }
        try {
            await unlockUserMutation.mutateAsync(id);
            setFeedback("User unlocked");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to unlock user");
        }
    }

    async function handleDelete() {
        if (!id) {
            return;
        }
        try {
            await deleteUserMutation.mutateAsync(id);
            navigate("/admin/users");
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to delete user");
            setDeleteModalOpen(false);
        }
    }

    async function handleResetPassword() {
        if (!id) {
            return;
        }
        if (
            !window.confirm(
                "Reset this user's password? Their current password will stop working and all their sessions will be logged out.",
            )
        ) {
            return;
        }
        try {
            const result = await resetPasswordMutation.mutateAsync(id);
            setResetPasswordResult(result.password);
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to reset password");
        }
    }

    if (loading) {
        return <div className={styles.loading}>Loading user...</div>;
    }

    if (!user) {
        return <div className={styles.error}>{error || "Could not load this user."}</div>;
    }

    return (
        <div className={styles.page}>
            <span className={styles.backLink} onClick={() => navigate("/admin/users")}>
                &larr; Back to Users
            </span>

            <h1 className={styles.title}>User Details</h1>

            {error && <div className={styles.error}>{error}</div>}
            {feedback && <div className={styles.success}>{feedback}</div>}

            <div className={styles.card}>
                <div className={styles.userHeader}>
                    <ProfileLink
                        user={{
                            id: user.id,
                            username: user.username,
                            display_name: user.display_name,
                            avatar_url: user.avatar_url,
                            role: user.role,
                        }}
                        size="large"
                    />
                </div>

                <div className={styles.infoGrid}>
                    <div className={`${styles.infoItem} ${styles.wideInfoItem}`}>
                        <span className={styles.infoLabel}>Email</span>
                        {user.email ? (
                            <span className={styles.valueRow}>
                                <span className={styles.infoValue}>{user.email}</span>
                                <span
                                    className={`${styles.inlineBadge} ${
                                        user.email_verified ? styles.activeBadge : styles.bannedBadge
                                    }`}
                                >
                                    {user.email_verified ? "Verified" : "Unverified"}
                                </span>
                                {canSetEmailVerified &&
                                    (user.email_verified ? (
                                        <Button
                                            variant="ghost"
                                            size="small"
                                            onClick={handleUnverifyEmail}
                                            disabled={unverifyEmailMutation.isPending}
                                        >
                                            Mark Unverified
                                        </Button>
                                    ) : (
                                        <Button
                                            variant="ghost"
                                            size="small"
                                            onClick={handleVerifyEmail}
                                            disabled={verifyEmailMutation.isPending}
                                        >
                                            Mark Verified
                                        </Button>
                                    ))}
                            </span>
                        ) : (
                            <span className={styles.bannedBadge}>No email set</span>
                        )}
                    </div>
                    {user.ip && (
                        <div className={`${styles.infoItem} ${styles.wideInfoItem}`}>
                            <span className={styles.infoLabel}>IP Address</span>
                            <span className={`${styles.infoValue} ${styles.monoValue}`}>{user.ip}</span>
                        </div>
                    )}
                    <div className={styles.infoItem}>
                        <span className={styles.infoLabel}>Status</span>
                        <span className={user.banned ? styles.bannedBadge : styles.activeBadge}>
                            {user.banned ? "Banned" : "Active"}
                        </span>
                    </div>
                    {user.banned && user.ban_reason && (
                        <div className={styles.infoItem}>
                            <span className={styles.infoLabel}>Ban Reason</span>
                            <span className={styles.infoValue}>{user.ban_reason}</span>
                        </div>
                    )}
                    {user.banned && user.banned_by && (
                        <div className={styles.infoItem}>
                            <span className={styles.infoLabel}>Banned By</span>
                            <span className={styles.infoValue}>
                                <ProfileLink user={user.banned_by} size="small" />
                            </span>
                        </div>
                    )}
                    {user.banned && user.banned_at && (
                        <div className={styles.infoItem}>
                            <span className={styles.infoLabel}>Banned At</span>
                            <span className={styles.infoValue}>{formatDate(user.banned_at)}</span>
                        </div>
                    )}
                    {user.locked && (
                        <div className={styles.infoItem}>
                            <span className={styles.infoLabel}>Lock</span>
                            <span className={styles.bannedBadge}>Locked</span>
                        </div>
                    )}
                    {user.locked && user.lock_reason && (
                        <div className={styles.infoItem}>
                            <span className={styles.infoLabel}>Lock Reason</span>
                            <span className={styles.infoValue}>{user.lock_reason}</span>
                        </div>
                    )}
                    {user.locked && user.locked_at && (
                        <div className={styles.infoItem}>
                            <span className={styles.infoLabel}>Locked At</span>
                            <span className={styles.infoValue}>{formatDate(user.locked_at)}</span>
                        </div>
                    )}
                    <div className={styles.infoItem}>
                        <span className={styles.infoLabel}>Theories</span>
                        <span className={styles.infoValue}>{user.theory_count}</span>
                    </div>
                    <div className={styles.infoItem}>
                        <span className={styles.infoLabel}>Responses</span>
                        <span className={styles.infoValue}>{user.response_count}</span>
                    </div>
                    <div className={styles.infoItem}>
                        <span className={styles.infoLabel}>Joined</span>
                        <span className={styles.infoValue}>{formatDate(user.created_at)}</span>
                    </div>
                </div>
            </div>

            {canManageEmail && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>Change Email</h2>

                    <p className={styles.helpText}>
                        Marks the address unverified and sends a verification link to the new one. The old address is
                        told the email changed. Whoever controls the address can reset the password, so treat this as an
                        account-takeover capability.
                    </p>
                    <div className={styles.editRow}>
                        <Input
                            type="email"
                            value={emailInput}
                            onChange={e => patchAccountDraft({ email: e.target.value })}
                            placeholder="user@example.com"
                        />
                        <Button
                            variant="primary"
                            size="small"
                            onClick={handleSetEmail}
                            disabled={
                                !emailInput.trim() || emailInput.trim() === user.email || setEmailMutation.isPending
                            }
                        >
                            Save Email
                        </Button>
                    </div>
                </div>
            )}

            {canManageAccount && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>Display Name</h2>

                    <p className={styles.helpText}>
                        A locked display name cannot be changed by the user. Their username is unaffected.
                    </p>
                    <div className={styles.editRow}>
                        <Input
                            type="text"
                            value={displayNameInput}
                            onChange={e => patchAccountDraft({ displayName: e.target.value })}
                            placeholder="Display name"
                            maxLength={40}
                        />
                        <Button
                            variant="primary"
                            size="small"
                            onClick={handleSetDisplayName}
                            disabled={
                                !displayNameInput.trim() ||
                                displayNameInput.trim() === user.display_name ||
                                setDisplayNameMutation.isPending
                            }
                        >
                            Save Name
                        </Button>
                        <Button
                            variant={user.display_name_locked ? "secondary" : "danger"}
                            size="small"
                            onClick={handleToggleDisplayNameLock}
                            disabled={setDisplayNameLockMutation.isPending}
                        >
                            {user.display_name_locked ? "Unlock Name" : "Lock Name"}
                        </Button>
                        <span className={user.display_name_locked ? styles.bannedBadge : styles.mutedBadge}>
                            {user.display_name_locked ? "Locked" : "Unlocked"}
                        </span>
                    </div>
                </div>
            )}

            {can(currentUser?.role, "edit_mystery_score") && user.role !== "super_admin" && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>Mystery Scores</h2>
                    <div className={styles.fieldGroup}>
                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>Detective Score</span>
                            <div style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
                                <Input
                                    type="text"
                                    inputMode="numeric"
                                    value={detectiveScoreInput}
                                    onChange={e => {
                                        const val = e.target.value;
                                        if (/^-?\d*$/.test(val)) {
                                            setDetectiveScoreInput(val);
                                        }
                                    }}
                                    style={{ width: "100px" }}
                                />
                                <Button
                                    variant="primary"
                                    size="small"
                                    onClick={async () => {
                                        const num = parseInt(detectiveScoreInput, 10) || 0;
                                        try {
                                            await updateDetectiveScoreMutation.mutateAsync({
                                                id: user.id,
                                                desiredScore: num,
                                            });
                                        } catch {}
                                    }}
                                >
                                    Save
                                </Button>
                            </div>
                        </div>
                        <div className={styles.field}>
                            <span className={styles.fieldLabel}>Game Master Score</span>
                            <div style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
                                <Input
                                    type="text"
                                    inputMode="numeric"
                                    value={gmScoreInput}
                                    onChange={e => {
                                        const val = e.target.value;
                                        if (/^-?\d*$/.test(val)) {
                                            setGMScoreInput(val);
                                        }
                                    }}
                                    style={{ width: "100px" }}
                                />
                                <Button
                                    variant="primary"
                                    size="small"
                                    onClick={async () => {
                                        const num = parseInt(gmScoreInput, 10) || 0;
                                        try {
                                            await updateGMScoreMutation.mutateAsync({
                                                id: user.id,
                                                desiredScore: num,
                                            });
                                        } catch {}
                                    }}
                                >
                                    Save
                                </Button>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            {can(currentUser?.role, "manage_roles") && user.role !== "super_admin" && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>Role</h2>
                    {user.role ? (
                        <div className={styles.roleDisplay}>
                            <span className={styles.currentRole}>
                                Current: <RolePill role={user.role} userId={user.id} />
                            </span>
                            <Button variant="danger" size="small" onClick={handleRemoveRole}>
                                Remove Role
                            </Button>
                        </div>
                    ) : (
                        <span className={styles.noRole}>No role assigned</span>
                    )}
                    <div className={styles.roleAssign}>
                        <Select value={selectedRole} onChange={e => setSelectedRole(e.target.value)}>
                            <option value="admin">Admin</option>
                            <option value="moderator">Moderator</option>
                        </Select>
                        <Button variant="primary" onClick={handleSetRole}>
                            {user.role ? "Change Role" : "Assign Role"}
                        </Button>
                    </div>
                </div>
            )}

            {can(currentUser?.role, "ban_user") && user.role !== "super_admin" && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>Ban Management</h2>
                    {user.banned ? (
                        <Button variant="primary" onClick={handleUnban}>
                            Unban User
                        </Button>
                    ) : (
                        <div className={styles.actionRow}>
                            <div className={styles.actionField}>
                                <span className={styles.fieldLabel}>Ban Reason</span>
                                <Input
                                    value={banReason}
                                    onChange={e => setBanReason(e.target.value)}
                                    placeholder="Reason for ban..."
                                />
                            </div>
                            <Button variant="danger" onClick={handleBan} disabled={!banReason.trim()}>
                                Ban User
                            </Button>
                        </div>
                    )}
                </div>
            )}

            {can(currentUser?.role, "ban_user") && user.role !== "super_admin" && user.role !== "admin" && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>Lock Management</h2>
                    <p className={styles.fieldLabel}>
                        A locked user can read the site and DM staff, but cannot post, comment, or message other users.
                    </p>
                    {user.locked ? (
                        <Button variant="primary" onClick={handleUnlock}>
                            Unlock User
                        </Button>
                    ) : (
                        <div className={styles.actionRow}>
                            <div className={styles.actionField}>
                                <span className={styles.fieldLabel}>Lock Reason</span>
                                <Input
                                    value={lockReason}
                                    onChange={e => setLockReason(e.target.value)}
                                    placeholder="Reason for lock..."
                                />
                            </div>
                            <Button variant="danger" onClick={handleLock} disabled={!lockReason.trim()}>
                                Lock User
                            </Button>
                        </div>
                    )}
                </div>
            )}

            {can(currentUser?.role, "ban_user") && user.role !== "super_admin" && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>Sessions</h2>
                    <p className={styles.helpText}>
                        Signs the user out everywhere without changing their password. Use this when a session may have
                        been hijacked.
                    </p>
                    <Button variant="danger" onClick={handleForceLogout} disabled={forceLogoutMutation.isPending}>
                        Revoke All Sessions
                    </Button>
                </div>
            )}

            {can(currentUser?.role, "view_users") && !!user.ip && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>Other Accounts On This IP</h2>
                    {ipMatches.loading ? (
                        <span className={styles.emptyState}>Loading...</span>
                    ) : ipMatches.failed ? (
                        <span className={styles.bannedBadge}>Could not load accounts for this IP address.</span>
                    ) : ipMatches.users.length === 0 ? (
                        <span className={styles.emptyState}>No other accounts share this IP address.</span>
                    ) : (
                        <div className={styles.matchList}>
                            {ipMatches.users.map(match => (
                                <div key={match.id} className={styles.matchRow}>
                                    <ProfileLink
                                        user={{
                                            id: match.id,
                                            username: match.username,
                                            display_name: match.display_name,
                                            avatar_url: match.avatar_url,
                                            role: match.role,
                                        }}
                                        size="small"
                                    />
                                    <span className={styles.matchMeta}>
                                        {match.banned && <span className={styles.bannedBadge}>Banned</span>}
                                        {match.locked && <span className={styles.bannedBadge}>Locked</span>}
                                        <span>Joined {formatDate(match.created_at)}</span>
                                        <Button
                                            variant="ghost"
                                            size="small"
                                            onClick={() => navigate(`/admin/users/${match.id}`)}
                                        >
                                            Manage
                                        </Button>
                                    </span>
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            )}

            {can(currentUser?.role, "view_audit_log") && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>Account History</h2>
                    <p className={styles.helpText}>
                        Everything recorded against this account: staff actions, chat room bans, word filter kicks,
                        watch party kicks, account creation and password resets.
                    </p>
                    {moderationHistory.loading ? (
                        <span className={styles.emptyState}>Loading...</span>
                    ) : moderationHistory.failed ? (
                        <span className={styles.bannedBadge}>Could not load the account history.</span>
                    ) : moderationHistory.entries.length === 0 ? (
                        <span className={styles.emptyState}>Nothing has been recorded against this account.</span>
                    ) : (
                        <>
                            <div className={styles.historyList}>
                                {moderationHistory.entries.map(entry => (
                                    <div key={entry.id} className={styles.historyRow}>
                                        <div className={styles.historyMain}>
                                            <span className={styles.historyAction}>
                                                {auditActionLabel(entry.action)}
                                            </span>
                                            {parseAuditDetails(entry.details).map((part, i) => (
                                                <span key={i} className={styles.historyDetails}>
                                                    {part.key && (
                                                        <span className={styles.historyDetailKey}>{part.key}</span>
                                                    )}
                                                    {part.value}
                                                </span>
                                            ))}
                                        </div>
                                        <span className={styles.historyMeta}>
                                            {entry.actor_name || "system"}
                                            <span className={styles.historyTime}>
                                                {formatFullDateTime(entry.created_at)}
                                            </span>
                                        </span>
                                    </div>
                                ))}
                            </div>
                            <Pagination
                                offset={historyOffset}
                                limit={HISTORY_LIMIT}
                                total={moderationHistory.total}
                                hasNext={historyOffset + HISTORY_LIMIT < moderationHistory.total}
                                hasPrev={historyOffset > 0}
                                onNext={() => setHistoryOffset(historyOffset + HISTORY_LIMIT)}
                                onPrev={() => setHistoryOffset(Math.max(0, historyOffset - HISTORY_LIMIT))}
                                size="small"
                                compact
                            />
                        </>
                    )}
                </div>
            )}

            {can(currentUser?.role, "reset_password") && user.role !== "super_admin" && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>Password</h2>
                    <p className={styles.fieldLabel}>
                        Generates a new random password and logs the user out everywhere. Use this for users who are
                        locked out and have no email to self-reset.
                    </p>
                    <Button variant="danger" onClick={handleResetPassword} disabled={resetPasswordMutation.isPending}>
                        Reset Password
                    </Button>
                </div>
            )}

            {can(currentUser?.role, "delete_any_user") && user.role !== "super_admin" && (
                <div className={`${styles.card} ${styles.dangerZone}`}>
                    <h2 className={styles.sectionTitle}>Danger Zone</h2>
                    <Button variant="danger" onClick={() => setDeleteModalOpen(true)}>
                        Delete User
                    </Button>
                </div>
            )}

            <Modal
                isOpen={resetPasswordResult !== null}
                onClose={() => setResetPasswordResult(null)}
                title="New Password"
            >
                <div className={styles.modalBody}>
                    Share this new password with <strong>{user.display_name}</strong> securely. It will not be shown
                    again.
                    <div className={styles.infoItem} style={{ marginTop: "1rem" }}>
                        <span className={styles.infoLabel}>Password</span>
                        <code className={styles.infoValue}>{resetPasswordResult}</code>
                    </div>
                </div>
                <div className={styles.modalActions}>
                    <Button
                        variant="secondary"
                        onClick={() => {
                            if (resetPasswordResult) {
                                navigator.clipboard.writeText(resetPasswordResult);
                            }
                        }}
                    >
                        Copy
                    </Button>
                    <Button variant="primary" onClick={() => setResetPasswordResult(null)}>
                        Done
                    </Button>
                </div>
            </Modal>

            <Modal isOpen={deleteModalOpen} onClose={() => setDeleteModalOpen(false)} title="Confirm Delete">
                <div className={styles.modalBody}>
                    Are you sure you want to delete <strong>{user.display_name}</strong>? This action cannot be undone.
                </div>
                <div className={styles.modalActions}>
                    <Button variant="secondary" onClick={() => setDeleteModalOpen(false)}>
                        Cancel
                    </Button>
                    <Button variant="danger" onClick={handleDelete}>
                        Delete
                    </Button>
                </div>
            </Modal>
        </div>
    );
}
