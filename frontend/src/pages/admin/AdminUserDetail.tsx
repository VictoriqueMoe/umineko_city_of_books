import type { ReactNode } from "react";
import { useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useAdminUserActions, type AdminUserConfirmation } from "../../hooks/useAdminUserActions";
import { Button } from "../../components/Button/Button";
import { ConfirmDialog } from "../../components/ConfirmDialog/ConfirmDialog";
import { Input } from "../../components/Input/Input";
import { Modal } from "../../components/Modal/Modal";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { RolePill } from "../../components/RolePill/RolePill";
import { Select } from "../../components/Select/Select";
import { useAuth } from "../../hooks/useAuth";
import { can } from "../../domain/permissions";
import { formatDate } from "../../utils/time";
import { AdminUserHistory } from "./AdminUserHistory";
import { AdminUserIPMatches } from "./AdminUserIPMatches";
import styles from "./AdminUserDetail.module.css";

interface ConfirmationCopy {
    title: string;
    body: ReactNode;
    confirmLabel: string;
}

function confirmationCopy(kind: AdminUserConfirmation, displayName: string): ConfirmationCopy {
    switch (kind) {
        case "unverifyEmail":
            return {
                title: "Mark Email Unverified",
                body: "Mark this email unverified? The user will be blocked from posting, commenting and messaging until they verify it again.",
                confirmLabel: "Unverify",
            };
        case "forceLogout":
            return {
                title: "Revoke All Sessions",
                body: "Sign this user out of every device? Their password is unchanged.",
                confirmLabel: "Revoke Sessions",
            };
        case "resetPassword":
            return {
                title: "Reset Password",
                body: "Reset this user's password? Their current password will stop working and all their sessions will be logged out.",
                confirmLabel: "Reset",
            };
        case "deleteUser":
            return {
                title: "Confirm Delete",
                body: (
                    <>
                        Are you sure you want to delete{" "}
                        <strong>
                            <bdi>{displayName}</bdi>
                        </strong>
                        ? This action cannot be undone.
                    </>
                ),
                confirmLabel: "Delete",
            };
    }
}

export function AdminUserDetail() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { user: currentUser } = useAuth();
    const {
        user,
        loading,
        error,
        feedback,
        capabilities,
        pending,
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
        dismissResetPasswordResult,
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
        requestUnverifyEmail,
        requestForceLogout,
        requestResetPassword,
        requestDeleteUser,
        confirmPending,
        cancelPending,
    } = useAdminUserActions(id);
    usePageTitle(user ? `Admin - ${user.display_name}` : "Admin - User");

    if (loading) {
        return <div className={styles.loading}>Loading user...</div>;
    }

    if (!user) {
        return <div className={styles.error}>{error || "Could not load this user."}</div>;
    }

    const confirmCopy = confirmation === null ? null : confirmationCopy(confirmation, user.display_name);

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
                                {capabilities.canSetEmailVerified &&
                                    (user.email_verified ? (
                                        <Button
                                            variant="ghost"
                                            size="small"
                                            onClick={requestUnverifyEmail}
                                            disabled={pending.unverifyEmail}
                                        >
                                            Mark Unverified
                                        </Button>
                                    ) : (
                                        <Button
                                            variant="ghost"
                                            size="small"
                                            onClick={verifyEmail}
                                            disabled={pending.verifyEmail}
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
                    {user.restricted && (
                        <div className={styles.infoItem}>
                            <span className={styles.infoLabel}>New Account</span>
                            <span className={styles.bannedBadge}>Restricted</span>
                        </div>
                    )}
                    {user.approved_at && (
                        <div className={styles.infoItem}>
                            <span className={styles.infoLabel}>Approved At</span>
                            <span className={styles.infoValue}>{formatDate(user.approved_at)}</span>
                        </div>
                    )}
                    {user.approved_by && (
                        <div className={styles.infoItem}>
                            <span className={styles.infoLabel}>Approved By</span>
                            <span className={styles.infoValue}>{user.approved_by.display_name}</span>
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

            {capabilities.canManageEmail && (
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
                            onChange={e => setEmailInput(e.target.value)}
                            placeholder="user@example.com"
                        />
                        <Button
                            variant="primary"
                            size="small"
                            onClick={saveEmail}
                            disabled={!emailInput.trim() || emailInput.trim() === user.email || pending.saveEmail}
                        >
                            Save Email
                        </Button>
                    </div>
                </div>
            )}

            {capabilities.canManageAccount && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>Display Name</h2>

                    <p className={styles.helpText}>
                        A locked display name cannot be changed by the user. Their username is unaffected.
                    </p>
                    <div className={styles.editRow}>
                        <Input
                            type="text"
                            value={displayNameInput}
                            onChange={e => setDisplayNameInput(e.target.value)}
                            placeholder="Display name"
                            maxLength={40}
                        />
                        <Button
                            variant="primary"
                            size="small"
                            onClick={saveDisplayName}
                            disabled={
                                !displayNameInput.trim() ||
                                displayNameInput.trim() === user.display_name ||
                                pending.saveDisplayName
                            }
                        >
                            Save Name
                        </Button>
                        <Button
                            variant={user.display_name_locked ? "secondary" : "danger"}
                            size="small"
                            onClick={toggleDisplayNameLock}
                            disabled={pending.toggleDisplayNameLock}
                        >
                            {user.display_name_locked ? "Unlock Name" : "Lock Name"}
                        </Button>
                        <span className={user.display_name_locked ? styles.bannedBadge : styles.mutedBadge}>
                            {user.display_name_locked ? "Locked" : "Unlocked"}
                        </span>
                    </div>
                </div>
            )}

            {capabilities.canEditMysteryScore && (
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
                                    onChange={e => setDetectiveScoreInput(e.target.value)}
                                    style={{ width: "100px" }}
                                />
                                <Button variant="primary" size="small" onClick={saveDetectiveScore}>
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
                                    onChange={e => setGMScoreInput(e.target.value)}
                                    style={{ width: "100px" }}
                                />
                                <Button variant="primary" size="small" onClick={saveGMScore}>
                                    Save
                                </Button>
                            </div>
                        </div>
                    </div>
                </div>
            )}

            {capabilities.canManageRoles && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>Role</h2>
                    {user.role ? (
                        <div className={styles.roleDisplay}>
                            <span className={styles.currentRole}>
                                Current: <RolePill role={user.role} userId={user.id} />
                            </span>
                            <Button variant="danger" size="small" onClick={removeRole}>
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
                        <Button variant="primary" onClick={assignRole}>
                            {user.role ? "Change Role" : "Assign Role"}
                        </Button>
                    </div>
                </div>
            )}

            {capabilities.canBan && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>Ban Management</h2>
                    {user.banned ? (
                        <Button variant="primary" onClick={unbanUser}>
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
                            <Button variant="danger" onClick={banUser} disabled={!banReason.trim()}>
                                Ban User
                            </Button>
                        </div>
                    )}
                </div>
            )}

            {capabilities.canLock && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>Lock Management</h2>
                    <p className={styles.fieldLabel}>
                        A locked user can read the site and DM staff, but cannot post, comment, or message other users.
                    </p>
                    {user.locked ? (
                        <Button variant="primary" onClick={unlockUser}>
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
                            <Button variant="danger" onClick={lockUser} disabled={!lockReason.trim()}>
                                Lock User
                            </Button>
                        </div>
                    )}
                </div>
            )}

            {capabilities.canManageAccount && (user.restricted || user.approved_at) && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>New Account Restriction</h2>
                    <p className={styles.fieldLabel}>
                        For their first day, a new member cannot post attachments or links, and their posts show no link
                        previews. Approving lifts that immediately, for someone you recognise or have already vouched
                        for.
                    </p>
                    {user.approved_at ? (
                        <Button variant="danger" onClick={revokeApproval}>
                            Revoke Approval
                        </Button>
                    ) : (
                        <Button variant="primary" onClick={approveAccount}>
                            Approve Account
                        </Button>
                    )}
                </div>
            )}

            {capabilities.canRevokeSessions && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>Sessions</h2>
                    <p className={styles.helpText}>
                        Signs the user out everywhere without changing their password. Use this when a session may have
                        been hijacked.
                    </p>
                    <Button variant="danger" onClick={requestForceLogout} disabled={pending.forceLogout}>
                        Revoke All Sessions
                    </Button>
                </div>
            )}

            {can(currentUser, "view_users") && !!user.ip && <AdminUserIPMatches userId={user.id} />}

            {can(currentUser, "view_audit_log") && <AdminUserHistory userId={user.id} />}

            {capabilities.canResetPassword && (
                <div className={styles.card}>
                    <h2 className={styles.sectionTitle}>Password</h2>
                    <p className={styles.fieldLabel}>
                        Generates a new random password and logs the user out everywhere. Use this for users who are
                        locked out and have no email to self-reset.
                    </p>
                    <Button variant="danger" onClick={requestResetPassword} disabled={pending.resetPassword}>
                        Reset Password
                    </Button>
                </div>
            )}

            {capabilities.canDeleteUser && (
                <div className={`${styles.card} ${styles.dangerZone}`}>
                    <h2 className={styles.sectionTitle}>Danger Zone</h2>
                    <Button variant="danger" onClick={requestDeleteUser}>
                        Delete User
                    </Button>
                </div>
            )}

            <Modal isOpen={resetPasswordResult !== null} onClose={dismissResetPasswordResult} title="New Password">
                <div className={styles.modalBody}>
                    Share this new password with{" "}
                    <strong>
                        <bdi>{user.display_name}</bdi>
                    </strong>{" "}
                    securely. It will not be shown again.
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
                    <Button variant="primary" onClick={dismissResetPasswordResult}>
                        Done
                    </Button>
                </div>
            </Modal>

            {confirmCopy && (
                <ConfirmDialog
                    open
                    title={confirmCopy.title}
                    body={confirmCopy.body}
                    confirmLabel={confirmCopy.confirmLabel}
                    destructive
                    onConfirm={confirmPending}
                    onCancel={cancelPending}
                />
            )}
        </div>
    );
}
