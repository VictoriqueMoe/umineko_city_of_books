import { useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router";
import { marked } from "marked";
import DOMPurify from "dompurify";
import type { Announcement } from "../../types/api";
import { getAnnouncement } from "../../api/endpoints";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { relativeTime } from "../../utils/notifications";
import styles from "./AnnouncementsPage.module.css";

function renderMarkdown(md: string): string {
    const raw = marked.parse(md, { async: false }) as string;
    return DOMPurify.sanitize(raw);
}

export function AnnouncementDetailPage() {
    const { id } = useParams<{ id: string }>();
    const navigate = useNavigate();
    const [announcement, setAnnouncement] = useState<Announcement | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        if (!id) {
            return;
        }
        getAnnouncement(id)
            .then(setAnnouncement)
            .catch(() => setAnnouncement(null))
            .finally(() => setLoading(false));
    }, [id]);

    if (loading) {
        return <div className="loading">Loading announcement...</div>;
    }

    if (!announcement) {
        return <div className="empty-state">Announcement not found.</div>;
    }

    return (
        <div className={styles.page}>
            <span className={styles.back} onClick={() => navigate("/announcements")}>
                &larr; All Announcements
            </span>

            <div className={styles.detail}>
                <h1 className={styles.detailTitle}>{announcement.title}</h1>
                <div className={styles.detailMeta}>
                    <ProfileLink user={announcement.author} size="small" />
                    <span>{relativeTime(announcement.created_at)}</span>
                    {announcement.updated_at !== announcement.created_at && <span>(edited)</span>}
                </div>
                <div className={styles.body} dangerouslySetInnerHTML={{ __html: renderMarkdown(announcement.body) }} />
            </div>
        </div>
    );
}
