import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import type { ReportItem } from "../../api/endpoints";
import { getReports, resolveReport } from "../../api/endpoints";
import { Button } from "../../components/Button/Button";
import { Modal } from "../../components/Modal/Modal";
import { Select } from "../../components/Select/Select";
import styles from "./AdminReports.module.css";

export function AdminReports() {
    const navigate = useNavigate();
    const [reports, setReports] = useState<ReportItem[]>([]);
    const [status, setStatus] = useState("open");
    const [loading, setLoading] = useState(true);
    const [resolvingId, setResolvingId] = useState<number | null>(null);
    const [comment, setComment] = useState("");
    const textareaRef = useRef<HTMLTextAreaElement>(null);

    const fetchReports = useCallback(async (filterStatus: string) => {
        try {
            const res = await getReports(filterStatus);
            setReports(res.reports);
        } catch {
            setReports([]);
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => {
        setLoading(true);
        fetchReports(status);
    }, [status, fetchReports]);

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
            await resolveReport(resolvingId, comment);
            setReports(prev => prev.filter(r => r.id !== resolvingId));
        } catch {
            // ignore
        }
        setResolvingId(null);
        setComment("");
    }

    function handleViewTarget(report: ReportItem) {
        if (report.target_type === "theory") {
            navigate(`/theory/${report.target_id}`);
        } else if (report.target_type === "response" && report.context_id) {
            navigate(`/theory/${report.context_id}#response-${report.target_id}`);
        } else if (report.target_type === "post") {
            navigate(`/game-board/${report.target_id}`);
        } else if (report.target_type === "comment" && report.context_id) {
            navigate(`/game-board/${report.context_id}#comment-${report.target_id}`);
        } else if (report.target_type === "art") {
            navigate(`/gallery/art/${report.target_id}`);
        } else if (report.target_type === "art_comment" && report.context_id) {
            navigate(`/gallery/art/${report.context_id}#comment-${report.target_id}`);
        }
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
                        {reports.map(report => (
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
                                <td>{new Date(report.created_at).toLocaleString()}</td>
                                <td className={styles.actions}>
                                    <Button variant="ghost" size="small" onClick={() => handleViewTarget(report)}>
                                        View
                                    </Button>
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
                        ))}
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
