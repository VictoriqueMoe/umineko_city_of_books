import { useCallback, useEffect, useState } from "react";
import { getAuditLog } from "../../api/endpoints";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Pagination } from "../../components/Pagination/Pagination";
import { Select } from "../../components/Select/Select";
import type { AuditLogEntry } from "../../types/api";
import styles from "./AdminAuditLog.module.css";

const LIMIT = 50;

export function AdminAuditLog() {
    usePageTitle("Admin - Audit Log");
    const [entries, setEntries] = useState<AuditLogEntry[]>([]);
    const [total, setTotal] = useState(0);
    const [offset, setOffset] = useState(0);
    const [actionFilter, setActionFilter] = useState("");
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState("");

    const fetchLog = useCallback(
        async (currentOffset: number) => {
            setLoading(true);
            setError("");
            try {
                const res = await getAuditLog({
                    action: actionFilter || undefined,
                    limit: LIMIT,
                    offset: currentOffset,
                });
                setEntries(res.entries);
                setTotal(res.total);
            } catch (e) {
                setError(e instanceof Error ? e.message : "Failed to load audit log");
            } finally {
                setLoading(false);
            }
        },
        [actionFilter],
    );

    useEffect(() => {
        fetchLog(offset);
    }, [fetchLog, offset]);

    function handleFilterChange(value: string) {
        setActionFilter(value);
        setOffset(0);
    }

    return (
        <div className={styles.page}>
            <h1 className={styles.title}>Audit Log</h1>

            <div className={styles.filterRow}>
                <span className={styles.filterLabel}>Filter by action:</span>
                <Select value={actionFilter} onChange={e => handleFilterChange(e.target.value)}>
                    <option value="">All Actions</option>
                    <option value="set_role">Set Role</option>
                    <option value="remove_role">Remove Role</option>
                    <option value="ban_user">Ban User</option>
                    <option value="unban_user">Unban User</option>
                    <option value="delete_user">Delete User</option>
                    <option value="delete_theory">Delete Theory</option>
                    <option value="delete_response">Delete Response</option>
                    <option value="update_settings">Update Settings</option>
                </Select>
            </div>

            {loading && <div className={styles.loading}>Loading audit log...</div>}
            {error && <div className={styles.error}>{error}</div>}

            {!loading && !error && (
                <>
                    {entries.length === 0 ? (
                        <div className={styles.empty}>No audit log entries found</div>
                    ) : (
                        <table className={styles.table}>
                            <thead>
                                <tr>
                                    <th>Timestamp</th>
                                    <th>Actor</th>
                                    <th>Action</th>
                                    <th>Target Type</th>
                                    <th>Target ID</th>
                                    <th>Details</th>
                                </tr>
                            </thead>
                            <tbody>
                                {entries.map(entry => (
                                    <tr key={entry.id}>
                                        <td>{new Date(entry.created_at).toLocaleString()}</td>
                                        <td>{entry.actor_name}</td>
                                        <td>{entry.action}</td>
                                        <td>{entry.target_type}</td>
                                        <td>{entry.target_id}</td>
                                        <td>
                                            <span className={styles.details} title={entry.details}>
                                                {entry.details}
                                            </span>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    )}

                    <Pagination
                        offset={offset}
                        limit={LIMIT}
                        total={total}
                        hasNext={offset + LIMIT < total}
                        hasPrev={offset > 0}
                        onNext={() => setOffset(prev => prev + LIMIT)}
                        onPrev={() => setOffset(prev => Math.max(0, prev - LIMIT))}
                    />
                </>
            )}
        </div>
    );
}
