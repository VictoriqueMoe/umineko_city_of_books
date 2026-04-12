import { useCallback, useEffect, useRef, useState } from "react";
import {
    assignVanityRole,
    createVanityRole,
    deleteVanityRole,
    getVanityRoles,
    getVanityRoleUsers,
    searchUsers,
    unassignVanityRole,
    updateVanityRole,
    type VanityRoleDefinition,
    type VanityRoleUsersResponse,
} from "../../api/endpoints";
import type { User } from "../../types/api";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Modal } from "../../components/Modal/Modal";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import styles from "./AdminVanityRoles.module.css";

function hexToRgba(hex: string, alpha: number): string {
    const r = parseInt(hex.slice(1, 3), 16);
    const g = parseInt(hex.slice(3, 5), 16);
    const b = parseInt(hex.slice(5, 7), 16);
    return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}

export function AdminVanityRoles() {
    usePageTitle("Admin - Vanity Roles");
    const [roles, setRoles] = useState<VanityRoleDefinition[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");

    const [editingRole, setEditingRole] = useState<VanityRoleDefinition | null>(null);
    const [showCreate, setShowCreate] = useState(false);
    const [formLabel, setFormLabel] = useState("");
    const [formColor, setFormColor] = useState("#888888");
    const [formOrder, setFormOrder] = useState(0);
    const [saving, setSaving] = useState(false);

    const [managingRole, setManagingRole] = useState<VanityRoleDefinition | null>(null);
    const [assignedUsers, setAssignedUsers] = useState<VanityRoleUsersResponse | null>(null);
    const [userSearch, setUserSearch] = useState("");
    const [searchResults, setSearchResults] = useState<User[]>([]);
    const [assigning, setAssigning] = useState<string | null>(null);
    const searchDebounce = useRef<ReturnType<typeof setTimeout>>(undefined);

    const fetchRoles = useCallback(async () => {
        setLoading(true);
        try {
            const result = await getVanityRoles();
            setRoles(result ?? []);
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to load vanity roles");
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchRoles();
    }, [fetchRoles]);

    function openCreate() {
        setEditingRole(null);
        setFormLabel("");
        setFormColor("#888888");
        setFormOrder(roles.length > 0 ? roles[roles.length - 1].sort_order + 1 : 0);
        setShowCreate(true);
    }

    function openEdit(role: VanityRoleDefinition) {
        setEditingRole(role);
        setFormLabel(role.label);
        setFormColor(role.color);
        setFormOrder(role.sort_order);
        setShowCreate(true);
    }

    function closeForm() {
        setShowCreate(false);
        setEditingRole(null);
    }

    async function handleSave() {
        setSaving(true);
        setError("");
        try {
            if (editingRole) {
                await updateVanityRole(editingRole.id, {
                    label: formLabel,
                    color: formColor,
                    sort_order: formOrder,
                });
            } else {
                await createVanityRole({
                    label: formLabel,
                    color: formColor,
                    sort_order: formOrder,
                });
            }
            closeForm();
            await fetchRoles();
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to save");
        } finally {
            setSaving(false);
        }
    }

    async function handleDelete(id: string) {
        if (!window.confirm("Delete this vanity role? It will be removed from all users.")) {
            return;
        }
        try {
            await deleteVanityRole(id);
            await fetchRoles();
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to delete");
        }
    }

    async function openManageUsers(role: VanityRoleDefinition) {
        setManagingRole(role);
        setUserSearch("");
        setSearchResults([]);
        try {
            const result = await getVanityRoleUsers(role.id, { limit: 50 });
            setAssignedUsers(result);
        } catch {
            setAssignedUsers({ users: [], total: 0, limit: 50, offset: 0 });
        }
    }

    function closeManage() {
        setManagingRole(null);
        setAssignedUsers(null);
        setUserSearch("");
        setSearchResults([]);
    }

    useEffect(() => {
        if (!managingRole || managingRole.is_system) {
            return;
        }
        clearTimeout(searchDebounce.current);
        if (userSearch.length < 2) {
            setSearchResults([]);
            return;
        }
        searchDebounce.current = setTimeout(async () => {
            try {
                const results = await searchUsers(userSearch);
                setSearchResults(results ?? []);
            } catch {
                setSearchResults([]);
            }
        }, 250);
        return () => clearTimeout(searchDebounce.current);
    }, [userSearch, managingRole]);

    async function handleAssign(userId: string) {
        if (!managingRole) {
            return;
        }
        setAssigning(userId);
        try {
            await assignVanityRole(managingRole.id, userId);
            const result = await getVanityRoleUsers(managingRole.id, { limit: 50 });
            setAssignedUsers(result);
            setUserSearch("");
            setSearchResults([]);
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to assign");
        } finally {
            setAssigning(null);
        }
    }

    async function handleUnassign(userId: string) {
        if (!managingRole) {
            return;
        }
        setAssigning(userId);
        try {
            await unassignVanityRole(managingRole.id, userId);
            const result = await getVanityRoleUsers(managingRole.id, { limit: 50 });
            setAssignedUsers(result);
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to remove");
        } finally {
            setAssigning(null);
        }
    }

    if (loading) {
        return <div className={styles.loading}>Loading vanity roles...</div>;
    }

    const assignedIds = new Set((assignedUsers?.users ?? []).map(u => u.id));

    return (
        <div className={styles.page}>
            <div className={styles.header}>
                <h1 className={styles.title}>Vanity Roles</h1>
                <Button variant="primary" onClick={openCreate}>
                    Create Role
                </Button>
            </div>

            {error && <div className={styles.error}>{error}</div>}

            {roles.length === 0 ? (
                <div className={styles.empty}>No vanity roles yet.</div>
            ) : (
                <table className={styles.table}>
                    <thead>
                        <tr>
                            <th>Preview</th>
                            <th>Label</th>
                            <th>Color</th>
                            <th>Order</th>
                            <th>Type</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        {roles.map(role => (
                            <tr key={role.id}>
                                <td>
                                    <span
                                        className={styles.preview}
                                        style={{
                                            backgroundColor: hexToRgba(role.color, 0.15),
                                            color: role.color,
                                            border: `1px solid ${hexToRgba(role.color, 0.4)}`,
                                        }}
                                    >
                                        {role.label}
                                    </span>
                                </td>
                                <td>{role.label}</td>
                                <td>
                                    <span className={styles.colorCell}>
                                        <span className={styles.colorDot} style={{ backgroundColor: role.color }} />
                                        {role.color}
                                    </span>
                                </td>
                                <td>{role.sort_order}</td>
                                <td>
                                    {role.is_system ? (
                                        <span className={styles.systemBadge}>System</span>
                                    ) : (
                                        <span className={styles.customBadge}>Custom</span>
                                    )}
                                </td>
                                <td className={styles.actions}>
                                    <Button variant="secondary" size="small" onClick={() => openEdit(role)}>
                                        Edit
                                    </Button>
                                    <Button variant="secondary" size="small" onClick={() => openManageUsers(role)}>
                                        Users
                                    </Button>
                                    {!role.is_system && (
                                        <Button variant="danger" size="small" onClick={() => handleDelete(role.id)}>
                                            Delete
                                        </Button>
                                    )}
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            )}

            <Modal
                isOpen={showCreate}
                onClose={closeForm}
                title={editingRole ? "Edit Vanity Role" : "Create Vanity Role"}
            >
                <div className={styles.form}>
                    <label className={styles.fieldLabel}>
                        Label
                        <Input
                            type="text"
                            value={formLabel}
                            onChange={e => setFormLabel(e.target.value)}
                            placeholder="e.g. Beta Tester"
                        />
                    </label>
                    <label className={styles.fieldLabel}>
                        Color (hex)
                        <div className={styles.colorInput}>
                            <input
                                type="color"
                                value={formColor}
                                onChange={e => setFormColor(e.target.value)}
                                className={styles.colorPicker}
                            />
                            <Input
                                type="text"
                                value={formColor}
                                onChange={e => setFormColor(e.target.value)}
                                placeholder="#ff0000"
                            />
                        </div>
                    </label>
                    <label className={styles.fieldLabel}>
                        Sort Order
                        <Input
                            type="number"
                            value={String(formOrder)}
                            onChange={e => setFormOrder(Number(e.target.value))}
                        />
                    </label>
                    <div className={styles.previewRow}>
                        Preview:
                        <span
                            className={styles.preview}
                            style={{
                                backgroundColor: hexToRgba(formColor, 0.15),
                                color: formColor,
                                border: `1px solid ${hexToRgba(formColor, 0.4)}`,
                            }}
                        >
                            {formLabel || "Label"}
                        </span>
                    </div>
                    <div className={styles.formActions}>
                        <Button variant="ghost" size="small" onClick={closeForm}>
                            Cancel
                        </Button>
                        <Button
                            variant="primary"
                            size="small"
                            onClick={handleSave}
                            disabled={saving || !formLabel.trim()}
                        >
                            {saving ? "Saving..." : "Save"}
                        </Button>
                    </div>
                </div>
            </Modal>

            <Modal isOpen={!!managingRole} onClose={closeManage} title={`Users - ${managingRole?.label ?? ""}`}>
                <div className={styles.manageBody}>
                    {managingRole?.is_system ? (
                        <div className={styles.systemNotice}>
                            This is a system role. Users are automatically assigned based on mystery leaderboard scores
                            and cannot be manually changed.
                        </div>
                    ) : (
                        <>
                            <Input
                                type="text"
                                value={userSearch}
                                onChange={e => setUserSearch(e.target.value)}
                                placeholder="Search users to assign..."
                            />
                            {searchResults.length > 0 && (
                                <div className={styles.searchResults}>
                                    {searchResults.map(u => {
                                        if (assignedIds.has(u.id)) {
                                            return null;
                                        }
                                        return (
                                            <div key={u.id} className={styles.userRow}>
                                                <ProfileLink user={u} size="small" />
                                                <Button
                                                    variant="primary"
                                                    size="small"
                                                    onClick={() => handleAssign(u.id)}
                                                    disabled={assigning === u.id}
                                                >
                                                    {assigning === u.id ? "..." : "Assign"}
                                                </Button>
                                            </div>
                                        );
                                    })}
                                </div>
                            )}
                        </>
                    )}

                    <div className={styles.assignedSection}>
                        <div className={styles.assignedLabel}>Assigned ({assignedUsers?.total ?? 0})</div>
                        {(!assignedUsers || assignedUsers.users.length === 0) && (
                            <div className={styles.empty}>No users assigned.</div>
                        )}
                        {assignedUsers && assignedUsers.users.length > 0 && (
                            <div className={styles.assignedList}>
                                {assignedUsers.users.map(u => (
                                    <div key={u.id} className={styles.userRow}>
                                        <span className={styles.userName}>
                                            {u.display_name} (@{u.username})
                                        </span>
                                        {!managingRole?.is_system && (
                                            <Button
                                                variant="danger"
                                                size="small"
                                                onClick={() => handleUnassign(u.id)}
                                                disabled={assigning === u.id}
                                            >
                                                {assigning === u.id ? "..." : "Remove"}
                                            </Button>
                                        )}
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                </div>
            </Modal>
        </div>
    );
}
