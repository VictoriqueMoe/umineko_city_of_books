import { useState } from "react";
import { Link } from "react-router";
import { useAuditLog } from "../../api/queries/admin";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Pagination } from "../../components/Pagination/Pagination";
import { Select } from "../../components/Select/Select";
import {
    AUDIT_ACTION_LABELS,
    auditActionLabel,
    auditTargetLabel,
    parseAuditDetails,
    shortId,
} from "../../utils/auditLog";
import { formatFullDateTime } from "../../utils/time";
import styles from "./AdminAuditLog.module.css";

const LIMIT = 50;

const FILTERABLE_ACTIONS = Object.keys(AUDIT_ACTION_LABELS).sort((a, b) =>
    AUDIT_ACTION_LABELS[a].localeCompare(AUDIT_ACTION_LABELS[b]),
);

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
                    {FILTERABLE_ACTIONS.map(action => (
                        <option key={action} value={action}>
                            {auditActionLabel(action)}
                        </option>
                    ))}
                </Select>
            </div>

            {loading && <div className={styles.loading}>Loading audit log...</div>}
            {error && <div className={styles.error}>{error}</div>}

            {!loading && !error && (
                <>
                    {entries.length === 0 ? (
                        <div className={styles.empty}>No audit log entries found</div>
                    ) : (
                        <div className={styles.tableWrap}>
                            <table className={styles.table}>
                                <thead>
                                    <tr>
                                        <th>When</th>
                                        <th>Action</th>
                                        <th>Subject</th>
                                        <th>By</th>
                                        <th>Details</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {entries.map(entry => (
                                        <tr key={entry.id}>
                                            <td className={styles.timestampCell}>
                                                {formatFullDateTime(entry.created_at)}
                                            </td>
                                            <td>
                                                <span className={styles.action}>{auditActionLabel(entry.action)}</span>
                                                <span className={styles.targetType}>
                                                    {auditTargetLabel(entry.target_type)}
                                                    {entry.target_id && (
                                                        <span className={styles.targetId} title={entry.target_id}>
                                                            {" "}
                                                            {shortId(entry.target_id)}
                                                        </span>
                                                    )}
                                                </span>
                                            </td>
                                            <td>
                                                {entry.subject_id ? (
                                                    <Link
                                                        to={`/admin/users/${entry.subject_id}`}
                                                        className={styles.subjectLink}
                                                    >
                                                        {entry.subject_name || entry.subject_username}
                                                    </Link>
                                                ) : entry.target_type === "user" && entry.target_id ? (
                                                    <Link
                                                        to={`/admin/users/${entry.target_id}`}
                                                        className={styles.subjectLink}
                                                    >
                                                        {shortId(entry.target_id)}
                                                    </Link>
                                                ) : (
                                                    <span className={styles.muted}>&mdash;</span>
                                                )}
                                            </td>
                                            <td>{entry.actor_name || <span className={styles.muted}>system</span>}</td>
                                            <td>
                                                <span className={styles.details} title={entry.details}>
                                                    {parseAuditDetails(entry.details).map((part, i) => (
                                                        <span key={i} className={styles.detailPart}>
                                                            {part.key && (
                                                                <span className={styles.detailKey}>{part.key}</span>
                                                            )}
                                                            <span className={styles.detailValue}>{part.value}</span>
                                                        </span>
                                                    ))}
                                                </span>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        </div>
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
