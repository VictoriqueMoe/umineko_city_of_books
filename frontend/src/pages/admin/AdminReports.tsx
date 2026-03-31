import {useCallback, useEffect, useState} from "react";
import {useNavigate} from "react-router";
import type {ReportItem} from "../../api/endpoints";
import {getReports, resolveReport} from "../../api/endpoints";
import {Button} from "../../components/Button/Button";
import {Select} from "../../components/Select/Select";
import styles from "./AdminReports.module.css";

export function AdminReports() {
    const navigate = useNavigate();
    const [reports, setReports] = useState<ReportItem[]>([]);
    const [status, setStatus] = useState("open");
    const [loading, setLoading] = useState(true);

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

    async function handleResolve(id: number) {
        try {
            await resolveReport(id);
            setReports(prev => prev.filter(r => r.id !== id));
        } catch {
            // ignore
        }
    }

    function handleViewTarget(report: ReportItem) {
        if (report.target_type === "theory") {
            navigate(`/theory/${report.target_id}`);
        } else if (report.context_id) {
            navigate(`/theory/${report.context_id}#response-${report.target_id}`);
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
                                        <Button variant="primary" size="small" onClick={() => handleResolve(report.id)}>
                                            Resolve
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
