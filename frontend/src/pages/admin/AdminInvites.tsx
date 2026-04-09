import { useCallback, useEffect, useState } from "react";
import { createInvite, deleteInvite, getInvites, type InviteItem } from "../../api/endpoints";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Button } from "../../components/Button/Button";
import styles from "./AdminInvites.module.css";

export function AdminInvites() {
    usePageTitle("Admin - Invites");
    const [invites, setInvites] = useState<InviteItem[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");

    const fetchInvites = useCallback(async () => {
        setLoading(true);
        try {
            const result = await getInvites({ limit: 50 });
            setInvites(result.invites ?? []);
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to load invites");
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        fetchInvites();
    }, [fetchInvites]);

    async function handleCreate() {
        try {
            await createInvite();
            await fetchInvites();
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to create invite");
        }
    }

    async function handleDelete(code: string) {
        if (!window.confirm("Are you sure you want to delete this invite?")) {
            return;
        }
        try {
            await deleteInvite(code);
            await fetchInvites();
        } catch (e) {
            setError(e instanceof Error ? e.message : "Failed to delete invite");
        }
    }

    if (loading) {
        return <div className={styles.loading}>Loading invites...</div>;
    }

    return (
        <div className={styles.page}>
            <div className={styles.header}>
                <h1 className={styles.title}>Invites</h1>
                <Button variant="primary" onClick={handleCreate}>
                    Create Invite
                </Button>
            </div>

            {error && <div className={styles.error}>{error}</div>}

            {invites.length === 0 ? (
                <div className={styles.empty}>No invites created yet.</div>
            ) : (
                <table className={styles.table}>
                    <thead>
                        <tr>
                            <th>Code</th>
                            <th>Status</th>
                            <th>Created</th>
                            <th></th>
                        </tr>
                    </thead>
                    <tbody>
                        {invites.map(inv => (
                            <tr key={inv.code}>
                                <td className={styles.code}>{inv.code}</td>
                                <td>
                                    {inv.used_by ? (
                                        <span className={styles.used}>Used</span>
                                    ) : (
                                        <span className={styles.available}>Available</span>
                                    )}
                                </td>
                                <td>{new Date(inv.created_at).toLocaleDateString()}</td>
                                <td>
                                    {!inv.used_by && (
                                        <Button variant="danger" size="small" onClick={() => handleDelete(inv.code)}>
                                            Delete
                                        </Button>
                                    )}
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            )}
        </div>
    );
}
