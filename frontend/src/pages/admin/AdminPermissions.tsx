import { useId, useMemo, useState } from "react";
import { useAdminPermissions } from "../../api/queries/admin";
import { useUpdateRolePermissions, useUpdateVanityRolePermissions } from "../../api/mutations/admin";
import type { PermissionCatalogueItem } from "../../api/endpoints";
import { usePageTitle } from "../../hooks/usePageTitle";
import { useAuth } from "../../hooks/useAuth";
import { Button } from "../../components/Button/Button";
import { Select } from "../../components/Select/Select";
import { ToggleSwitch } from "../../components/ToggleSwitch/ToggleSwitch";
import { can } from "../../utils/permissions";
import styles from "./AdminPermissions.module.css";

function toggle(list: string[], value: string, on: boolean): string[] {
    if (!on) {
        return list.filter(item => item !== value);
    }

    if (list.includes(value)) {
        return list;
    }

    return [...list, value];
}

function sameSet(a: string[], b: string[]): boolean {
    if (a.length !== b.length) {
        return false;
    }

    for (const value of a) {
        if (!b.includes(value)) {
            return false;
        }
    }

    return true;
}

export function AdminPermissions() {
    usePageTitle("Admin - Permissions");
    const { user } = useAuth();
    const allowed = can(user, "manage_roles");
    const { catalogue, roles, vanityRoles, loading } = useAdminPermissions(allowed);
    const roleMutation = useUpdateRolePermissions();
    const vanityMutation = useUpdateVanityRolePermissions();
    const baseID = useId();

    const [error, setError] = useState("");
    const [roleDraft, setRoleDraft] = useState<Record<string, string[]>>({});
    const [vanityDraft, setVanityDraft] = useState<Record<string, string[]>>({});
    const [selectedVanityID, setSelectedVanityID] = useState("");

    const assignable = useMemo(() => catalogue.filter(item => item.vanity_assignable), [catalogue]);
    const selectedVanity = vanityRoles.find(role => role.id === selectedVanityID) ?? null;

    function fieldID(name: string): string {
        return `${baseID}-${name}`;
    }

    function roleValue(role: string, stored: string[]): string[] {
        return roleDraft[role] ?? stored;
    }

    function vanityValue(id: string, stored: string[]): string[] {
        return vanityDraft[id] ?? stored;
    }

    function setRolePermission(role: string, stored: string[], perm: string, on: boolean) {
        setRoleDraft(prev => ({ ...prev, [role]: toggle(roleValue(role, stored), perm, on) }));
    }

    function setVanityPermission(id: string, stored: string[], perm: string, on: boolean) {
        setVanityDraft(prev => ({ ...prev, [id]: toggle(vanityValue(id, stored), perm, on) }));
    }

    function saveRole(role: string, stored: string[]) {
        setError("");
        roleMutation.mutate(
            { role, permissions: roleValue(role, stored) },
            {
                onSuccess: () => {
                    setRoleDraft(prev => {
                        const next = { ...prev };
                        delete next[role];
                        return next;
                    });
                },
                onError: (e: Error) => setError(e.message),
            },
        );
    }

    function saveVanity(id: string, stored: string[]) {
        setError("");
        vanityMutation.mutate(
            { id, permissions: vanityValue(id, stored) },
            {
                onSuccess: () => {
                    setVanityDraft(prev => {
                        const next = { ...prev };
                        delete next[id];
                        return next;
                    });
                },
                onError: (e: Error) => setError(e.message),
            },
        );
    }

    if (!allowed) {
        return <div className={styles.empty}>You do not have permission to manage permissions.</div>;
    }

    if (loading) {
        return <div className={styles.loading}>Loading...</div>;
    }

    return (
        <div className={styles.page}>
            <div className={styles.header}>
                <h1 className={styles.title}>Permissions</h1>
            </div>

            <p className={styles.intro}>
                Voyager Witches and the Reality Author always hold every permission and are deliberately absent from
                this page, so no edit here can ever lock an administrator out. Vanity roles may only be granted the
                non-staff permissions listed below.
            </p>

            {error && <div className={styles.error}>{error}</div>}

            {roles.map(role => {
                const current = roleValue(role.role, role.permissions);
                const dirty = !sameSet(current, role.permissions);

                return (
                    <div key={role.role} className={styles.card}>
                        <h2 className={styles.sectionTitle}>{role.label}</h2>
                        <p className={styles.cardHint} id={fieldID(`role-${role.role}-hint`)}>
                            Untick anything this role should no longer be able to do. Changes apply the moment you save.
                        </p>
                        <div className={styles.toggleGrid} aria-describedby={fieldID(`role-${role.role}-hint`)}>
                            {catalogue.map((item: PermissionCatalogueItem) => (
                                <ToggleSwitch
                                    key={item.permission}
                                    enabled={current.includes(item.permission)}
                                    onChange={on => setRolePermission(role.role, role.permissions, item.permission, on)}
                                    label={item.label}
                                    description={item.permission}
                                />
                            ))}
                        </div>
                        <div className={styles.formActions}>
                            <Button
                                onClick={() => saveRole(role.role, role.permissions)}
                                disabled={!dirty || roleMutation.isPending}
                            >
                                {roleMutation.isPending ? "Saving..." : "Save"}
                            </Button>
                        </div>
                    </div>
                );
            })}

            <div className={styles.card}>
                <h2 className={styles.sectionTitle}>Vanity Roles</h2>

                {vanityRoles.length === 0 ? (
                    <div className={styles.empty}>
                        There are no custom vanity roles yet. Create one on the Vanity Roles page first.
                    </div>
                ) : (
                    <>
                        <div className={styles.fieldLabel}>
                            <label htmlFor={fieldID("vanity-role")}>Vanity role</label>
                            <Select
                                id={fieldID("vanity-role")}
                                value={selectedVanityID}
                                onChange={e => setSelectedVanityID(e.target.value)}
                                aria-describedby={fieldID("vanity-role-hint")}
                            >
                                <option value="">Select a vanity role...</option>
                                {vanityRoles.map(role => (
                                    <option key={role.id} value={role.id}>
                                        {role.label}
                                    </option>
                                ))}
                            </Select>
                            <span id={fieldID("vanity-role-hint")} className={styles.fieldHint}>
                                Only custom vanity roles can carry permissions. Leaderboard badges are excluded.
                            </span>
                        </div>

                        {selectedVanity && (
                            <>
                                <div className={styles.toggleGrid}>
                                    {assignable.map(item => (
                                        <ToggleSwitch
                                            key={item.permission}
                                            enabled={vanityValue(
                                                selectedVanity.id,
                                                selectedVanity.permissions,
                                            ).includes(item.permission)}
                                            onChange={on =>
                                                setVanityPermission(
                                                    selectedVanity.id,
                                                    selectedVanity.permissions,
                                                    item.permission,
                                                    on,
                                                )
                                            }
                                            label={item.label}
                                            description={item.permission}
                                        />
                                    ))}
                                </div>
                                <div className={styles.formActions}>
                                    <Button
                                        onClick={() => saveVanity(selectedVanity.id, selectedVanity.permissions)}
                                        disabled={
                                            sameSet(
                                                vanityValue(selectedVanity.id, selectedVanity.permissions),
                                                selectedVanity.permissions,
                                            ) || vanityMutation.isPending
                                        }
                                    >
                                        {vanityMutation.isPending ? "Saving..." : "Save"}
                                    </Button>
                                </div>
                            </>
                        )}
                    </>
                )}
            </div>
        </div>
    );
}
