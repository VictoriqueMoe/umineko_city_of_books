import { useEffect, useReducer, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { usePageTitle } from "../../hooks/usePageTitle";
import {
    adminDeleteUser,
    banUser,
    getAdminUser,
    removeUserRole,
    setUserRole,
    unbanUser,
    updateDetectiveScore,
    updateGMScore,
} from "../../api/endpoints";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Modal } from "../../components/Modal/Modal";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { RolePill } from "../../components/RolePill/RolePill";
import { Select } from "../../components/Select/Select";
import type { AdminUserDetail as AdminUserDetailType } from "../../types/api";
import { useAuth } from "../../hooks/useAuth";
import { can } from "../../utils/permissions";
import styles from "./AdminUserDetail.module.css";

export function AdminUserDetail() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const { user: currentUser } = useAuth();
    const [user, setUser] = useState<AdminUserDetailType | null>(null);
    usePageTitle(user ? `Admin - ${user.display_name}` : "Admin - User");
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");
    const [feedback, setFeedback] = useState("");
    const [refreshKey, refresh] = useReducer((x: number) => x + 1, 0);

    const [selectedRole, setSelectedRole] = useState("admin");
    const [banReason, setBanReason] = useState("");
    const [deleteModalOpen, setDeleteModalOpen] = useState(false);
    const [detectiveScoreInput, setDetectiveScoreInput] = useState("0");
    const [gmScoreInput, setGMScoreInput] = useState("0");

    useEffect(() => {
        if (!id) {
            return;
        }
        let cancelled = false;
        async function load() {
            setLoading(true);
            setError("");
            try {
                const data = await getAdminUser(id!);
                if (!cancelled) {
                    setUser(data);
                    setDetectiveScoreInput(String(data.detective_score));
                    setGMScoreInput(String(data.gm_score));
                }
            } catch (e) {
                if (!cancelled) {
                    setError(e instanceof Error ? e.message : "Failed to load user");
                }
            } finally {
                if (!cancelled) {
                    setLoading(false);
                }
            }
        }
        load();
        return () => {
            cancelled = true;
        };
    }, [id, refreshKey]);

    async function handleSetRole() {
        if (!id) {
            return;
        }
        try {
            await setUserRole(id, selectedRole);
            setFeedback("Role assigned");
            refresh();
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to set role");
        }
    }

    async function handleRemoveRole() {
        if (!id || !user?.role) {
            return;
        }
        try {
            await removeUserRole(id, user.role);
            setFeedback("Role removed");
            refresh();
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to remove role");
        }
    }

    async function handleBan() {
        if (!id || !banReason.trim()) {
            return;
        }
        try {
            await banUser(id, banReason.trim());
            setBanReason("");
            setFeedback("User banned");
            refresh();
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to ban user");
        }
    }

    async function handleUnban() {
        if (!id) {
            return;
        }
        try {
            await unbanUser(id);
            setFeedback("User unbanned");
            refresh();
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to unban user");
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
            await adminDeleteUser(id);
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
                            <span className={styles.infoValue}>{new Date(user.banned_at).toLocaleDateString()}</span>
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
                        <span className={styles.infoValue}>{new Date(user.created_at).toLocaleDateString()}</span>
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
                                            await updateDetectiveScore(user.id, num);
                                            refresh();
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
                                            await updateGMScore(user.id, num);
                                            refresh();
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
