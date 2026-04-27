import { useState } from "react";
import { useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useAdminUser } from "../../api/queries/admin";
import {
    useAdminDeleteUser,
    useBanUser,
    useLockUser,
    useRemoveUserRole,
    useSetUserRole,
    useUnbanUser,
    useUnlockUser,
    useUpdateDetectiveScore,
    useUpdateGMScore,
} from "../../api/mutations/admin";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Modal } from "../../components/Modal/Modal";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { RolePill } from "../../components/RolePill/RolePill";
import { Select } from "../../components/Select/Select";
import { useAuth } from "../../hooks/useAuth";
import { can } from "../../utils/permissions";
import { formatDate } from "../../utils/time";
import styles from "./AdminUserDetail.module.css";

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

    const [selectedRole, setSelectedRole] = useState("admin");
    const [banReason, setBanReason] = useState("");
    const [lockReason, setLockReason] = useState("");
    const [deleteModalOpen, setDeleteModalOpen] = useState(false);
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
        if (!window.confirm("Are you sure you want to delete this user? This cannot be undone.")) {
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

    if (loading) {
        return <div className={styles.loading}>Loading user...</div>;
    }

    if (error && !user) {
        return <div className={styles.error}>{error}</div>;
    }

    if (!user) {
        return null;
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
                    {user.ip && (
                        <div className={styles.infoItem}>
                            <span className={styles.infoLabel}>IP Address</span>
                            <span className={styles.infoValue}>{user.ip}</span>
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

            {can(currentUser?.role, "edit_mystery_score") && (
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

            {can(currentUser?.role, "delete_any_user") && user.role !== "super_admin" && (
                <div className={`${styles.card} ${styles.dangerZone}`}>
                    <h2 className={styles.sectionTitle}>Danger Zone</h2>
                    <Button variant="danger" onClick={() => setDeleteModalOpen(true)}>
                        Delete User
                    </Button>
                </div>
            )}

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
