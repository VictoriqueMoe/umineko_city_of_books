import { useState } from "react";
import { useInvites } from "../../hooks/queries/admin";
import { useCreateInvite, useDeleteInvite } from "../../hooks/mutations/admin";
import { usePageTitle } from "../../hooks/usePageTitle";
import { errorMessage } from "../../utils/errorMessage";
import { Button } from "../../components/Button/Button";
import { ConfirmDialog } from "../../components/ConfirmDialog/ConfirmDialog";
import { formatDate } from "../../utils/time";
import styles from "./AdminInvites.module.css";

export function AdminInvites() {
    usePageTitle("Admin - Invites");
    const { invites, loading } = useInvites(50, 0);
    const createInviteMutation = useCreateInvite();
    const deleteInviteMutation = useDeleteInvite();
    const [error, setError] = useState("");
    const [pendingDeleteCode, setPendingDeleteCode] = useState<string | null>(null);

    async function handleCreate() {
        try {
            await createInviteMutation.mutateAsync();
        } catch (e) {
            setError(errorMessage(e, "Failed to create invite"));
        }
    }

    async function confirmDelete() {
        const code = pendingDeleteCode;
        if (!code) {
            return;
        }

        setPendingDeleteCode(null);

        try {
            await deleteInviteMutation.mutateAsync(code);
        } catch (e) {
            setError(errorMessage(e, "Failed to delete invite"));
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
                                <td>{formatDate(inv.created_at)}</td>
                                <td>
                                    {!inv.used_by && (
                                        <Button
                                            variant="danger"
                                            size="small"
                                            onClick={() => setPendingDeleteCode(inv.code)}
                                        >
                                            Delete
                                        </Button>
                                    )}
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            )}

            <ConfirmDialog
                open={pendingDeleteCode !== null}
                title="Delete Invite"
                body="Are you sure you want to delete this invite?"
                confirmLabel="Delete"
                destructive
                onConfirm={confirmDelete}
                onCancel={() => setPendingDeleteCode(null)}
            />
        </div>
    );
}
