import { useState } from "react";
import { useAuditLog } from "../../api/queries/admin";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Pagination } from "../../components/Pagination/Pagination";
import { Select } from "../../components/Select/Select";
import { formatFullDateTime } from "../../utils/time";
import styles from "./AdminAuditLog.module.css";

const LIMIT = 50;

export function AdminAuditLog() {
    usePageTitle("Admin - Audit Log");
    const [offset, setOffset] = useState(0);
    const [actionFilter, setActionFilter] = useState("");
    const { entries, total, loading } = useAuditLog(actionFilter, LIMIT, offset);
    const error = "";

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
                                        <td>{formatFullDateTime(entry.created_at)}</td>
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
