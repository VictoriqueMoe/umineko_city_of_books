import { useRef, useState } from "react";
import { useNavigate } from "react-router";
import { useReports } from "../../api/queries/admin";
import { useResolveReport } from "../../api/mutations/admin";
import { usePageTitle } from "../../hooks/usePageTitle";
import { Button } from "../../components/Button/Button";
import { Modal } from "../../components/Modal/Modal";
import { Select } from "../../components/Select/Select";
import { formatFullDateTime } from "../../utils/time";
import styles from "./AdminReports.module.css";

function reportTargetPath(report: import("../../api/endpoints").ReportItem): string | null {
    if (report.target_type === "theory") {
        return `/theory/${report.target_id}`;
    }
    if (report.target_type === "response" && report.context_id) {
        return `/theory/${report.context_id}#response-${report.target_id}`;
    }
    if (report.target_type === "post") {
        return `/game-board/${report.target_id}`;
    }
    if (report.target_type === "comment" && report.context_id) {
        return `/game-board/${report.context_id}#comment-${report.target_id}`;
    }
    if (report.target_type === "art") {
        return `/gallery/art/${report.target_id}`;
    }
    if (report.target_type === "art_comment" && report.context_id) {
        return `/gallery/art/${report.context_id}#comment-${report.target_id}`;
    }
    if (report.target_type === "mystery") {
        return `/mystery/${report.target_id}`;
    }
    if (report.target_type === "ship_comment" && report.context_id) {
        return `/ships/${report.context_id}#comment-${report.target_id}`;
    }
    if (report.target_type === "fanfic_comment" && report.context_id) {
        return `/fanfiction/${report.context_id}#comment-${report.target_id}`;
    }
    if (report.target_type === "announcement_comment" && report.context_id) {
        return `/announcements/${report.context_id}#comment-${report.target_id}`;
    }
    if (report.target_type === "mystery_comment" && report.context_id) {
        return `/mystery/${report.context_id}#comment-${report.target_id}`;
    }
    if (report.target_type === "journal_comment" && report.context_id) {
        return `/journals/${report.context_id}#comment-${report.target_id}`;
    }
    if (report.target_type === "journal") {
        return `/journals/${report.target_id}`;
    }
    if (report.target_type === "mystery_attempt" && report.context_id) {
        return `/mystery/${report.context_id}#attempt-${report.target_id}`;
    }
    if (report.target_type === "oc_comment" && report.context_id) {
        return `/oc/${report.context_id}#comment-${report.target_id}`;
    }
    if (report.target_type === "secret_comment" && report.context_id) {
        return `/secrets/${report.context_id}#comment-${report.target_id}`;
    }

    return null;
}

export function AdminReports() {
    usePageTitle("Admin - Reports");
    const navigate = useNavigate();
    const [status, setStatus] = useState("open");
    const { reports, loading } = useReports(status);
    const resolveReportMutation = useResolveReport();
    const [resolvingId, setResolvingId] = useState<number | null>(null);
    const [comment, setComment] = useState("");
    const textareaRef = useRef<HTMLTextAreaElement>(null);

    function openResolveModal(id: number) {
        setResolvingId(id);
        setComment("");
        setTimeout(() => textareaRef.current?.focus(), 50);
    }

    async function handleResolve() {
        if (resolvingId === null) {
            return;
        }
        try {
            await resolveReportMutation.mutateAsync({ id: resolvingId, comment });
        } catch {
            // ignore
        }
        setResolvingId(null);
        setComment("");
    }

    return (
        <div className={styles.page}>
            <h1 className={styles.title}>Reports</h1>

            <div className={styles.filterRow}>
                <span className={styles.filterLabel}>Status:</span>
                <Select value={status} onChange={e => setStatus(e.target.value)}>
                    <option value="open">Open</option>
                    <option value="resolved">Resolved</option>
                    <option value="">All</option>
                </Select>
            </div>

            {loading ? (
                <div className={styles.loading}>Loading reports...</div>
            ) : reports.length === 0 ? (
                <div className={styles.empty}>No reports found</div>
            ) : (
                <table className={styles.table}>
                    <thead>
                        <tr>
                            <th>Reporter</th>
                            <th>Type</th>
                            <th>Reason</th>
                            <th>Date</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        {reports.map(report => {
                            const targetPath = reportTargetPath(report);

                            return (
                                <tr key={report.id}>
                                    <td className={styles.reporter}>
                                        {report.reporter_avatar ? (
                                            <img className={styles.avatar} src={report.reporter_avatar} alt="" />
                                        ) : (
                                            <span className={styles.avatarPlaceholder}>
                                                {report.reporter_name.charAt(0).toUpperCase()}
                                            </span>
                                        )}
                                        {report.reporter_name}
                                    </td>
                                    <td className={styles.type}>{report.target_type}</td>
                                    <td className={styles.reason}>{report.reason}</td>
                                    <td>{formatFullDateTime(report.created_at)}</td>
                                    <td className={styles.actions}>
                                        {targetPath && (
                                            <Button variant="ghost" size="small" onClick={() => navigate(targetPath)}>
                                                View
                                            </Button>
                                        )}
                                        {report.status === "open" && (
                                            <Button
                                                variant="primary"
                                                size="small"
                                                onClick={() => openResolveModal(report.id)}
                                            >
                                                Resolve
                                            </Button>
                                        )}
                                    </td>
                                </tr>
                            );
                        })}
                    </tbody>
                </table>
            )}

            <Modal isOpen={resolvingId !== null} onClose={() => setResolvingId(null)} title="Resolve Report">
                <div className={styles.resolveModal}>
                    <label className={styles.resolveLabel}>Message to the reporter (optional):</label>
                    <textarea
                        ref={textareaRef}
                        className={styles.resolveTextarea}
                        value={comment}
                        onChange={e => setComment(e.target.value)}
                        placeholder="Let them know what action was taken..."
                        rows={4}
                        maxLength={500}
                    />
                    <div className={styles.resolveActions}>
                        <Button variant="ghost" size="small" onClick={() => setResolvingId(null)}>
                            Cancel
                        </Button>
                        <Button variant="primary" size="small" onClick={handleResolve}>
                            Resolve
                        </Button>
                    </div>
                </div>
            </Modal>
        </div>
    );
}
