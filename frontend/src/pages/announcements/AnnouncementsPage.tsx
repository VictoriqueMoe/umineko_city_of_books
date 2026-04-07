import { useEffect, useState } from "react";
import { Link } from "react-router";
import type { Announcement } from "../../types/api";
import { listAnnouncements } from "../../api/endpoints";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { Pagination } from "../../components/Pagination/Pagination";
import { relativeTime } from "../../utils/notifications";
import styles from "./AnnouncementsPage.module.css";

export function AnnouncementsPage() {
    const [announcements, setAnnouncements] = useState<Announcement[]>([]);
    const [total, setTotal] = useState(0);
    const [offset, setOffset] = useState(0);
    const [loading, setLoading] = useState(true);
    const limit = 20;

    useEffect(() => {
        let cancelled = false;
        listAnnouncements(limit, offset)
            .then(data => {
                if (!cancelled) {
                    setAnnouncements(data.announcements);
                    setTotal(data.total);
                    setLoading(false);
                }
            })
            .catch(() => {
                if (!cancelled) {
                    setAnnouncements([]);
                    setLoading(false);
                }
            });
        return () => {
            cancelled = true;
        };
    }, [offset]);

    function preview(body: string): string {
        const plain = body.replace(/[#*_~`>[\]()!-]/g, "").replace(/\n+/g, " ");
        if (plain.length > 200) {
            return plain.slice(0, 200) + "...";
        }
        return plain;
    }

    if (loading) {
        return <div className="loading">Loading announcements...</div>;
    }

    return (
        <div className={styles.page}>
            <h1 className={styles.heading}>Announcements</h1>

            {announcements.length === 0 && <div className="empty-state">No announcements yet.</div>}

            <div className={styles.list}>
                {announcements.map(a => (
                    <Link
                        key={a.id}
                        to={`/announcements/${a.id}`}
                        className={`${styles.card}${a.pinned ? ` ${styles.cardPinned}` : ""}`}
                    >
                        <div className={styles.cardHeader}>
                            <span className={styles.cardTitle}>{a.title}</span>
                            {a.pinned && <span className={styles.pinnedBadge}>Pinned</span>}
                        </div>
                        <div className={styles.cardMeta}>
                            <ProfileLink user={a.author} size="small" clickable={false} />
                            <span>{relativeTime(a.created_at)}</span>
                        </div>
                        <p className={styles.cardPreview}>{preview(a.body)}</p>
                    </Link>
                ))}
            </div>

            <Pagination
                offset={offset}
                limit={limit}
                total={total}
                hasNext={offset + limit < total}
                hasPrev={offset > 0}
                onNext={() => setOffset(offset + limit)}
                onPrev={() => setOffset(Math.max(0, offset - limit))}
            />
        </div>
    );
}
