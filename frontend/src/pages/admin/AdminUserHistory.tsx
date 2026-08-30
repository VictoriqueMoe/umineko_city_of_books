import { useUserAuditLog } from "../../hooks/queries/admin";
import { hasNextPage, usePageOffset } from "../../hooks/usePageOffset";
import { Pagination } from "../../components/Pagination/Pagination";
import { auditActionLabel, parseAuditDetails } from "../../domain/audit";
import { formatFullDateTime } from "../../utils/time";
import styles from "./AdminUserDetail.module.css";

const HISTORY_LIMIT = 10;

interface AdminUserHistoryProps {
    userId: string;
}

export function AdminUserHistory({ userId }: AdminUserHistoryProps) {
    const page = usePageOffset({ limit: HISTORY_LIMIT });
    const history = useUserAuditLog(userId, true, page.limit, page.offset);

    return (
        <div className={styles.card}>
            <h2 className={styles.sectionTitle}>Account History</h2>
            <p className={styles.helpText}>
                Everything recorded against this account: staff actions, chat room bans, word filter kicks, watch party
                kicks, account creation and password resets.
            </p>
            {history.loading ? (
                <span className={styles.emptyState}>Loading...</span>
            ) : history.failed ? (
                <span className={styles.bannedBadge}>Could not load the account history.</span>
            ) : history.entries.length === 0 ? (
                <span className={styles.emptyState}>Nothing has been recorded against this account.</span>
            ) : (
                <>
                    <div className={styles.historyList}>
                        {history.entries.map(entry => (
                            <div key={entry.id} className={styles.historyRow}>
                                <div className={styles.historyMain}>
                                    <span className={styles.historyAction}>{auditActionLabel(entry.action)}</span>
                                    {parseAuditDetails(entry.details).map((part, i) => (
                                        <span key={i} className={styles.historyDetails}>
                                            {part.key && <span className={styles.historyDetailKey}>{part.key}</span>}
                                            {part.value}
                                        </span>
                                    ))}
                                </div>
                                <span className={styles.historyMeta}>
                                    {entry.actor_name || "system"}
                                    <span className={styles.historyTime}>{formatFullDateTime(entry.created_at)}</span>
                                </span>
                            </div>
                        ))}
                    </div>
                    <Pagination
                        offset={page.offset}
                        limit={page.limit}
                        total={history.total}
                        hasNext={hasNextPage(page, history.total)}
                        hasPrev={page.hasPrev}
                        onNext={page.goNext}
                        onPrev={page.goPrev}
                        size="small"
                        compact
                    />
                </>
            )}
        </div>
    );
}
