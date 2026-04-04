import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { marked } from "marked";
import DOMPurify from "dompurify";
import type { Announcement } from "../../types/api";
import { getLatestAnnouncement } from "../../api/endpoints";
import { ProfileLink } from "../ProfileLink/ProfileLink";
import { relativeTime } from "../../utils/notifications";
import styles from "./AnnouncementCard.module.css";

const DISMISSED_KEY = "dismissed_announcement";

function renderMarkdown(md: string): string {
    const raw = marked.parse(md, { async: false }) as string;
    return DOMPurify.sanitize(raw);
}

export function AnnouncementCard() {
    const navigate = useNavigate();
    const [announcement, setAnnouncement] = useState<Announcement | null>(null);
    const [dismissed, setDismissed] = useState(false);

    useEffect(() => {
        getLatestAnnouncement()
            .then(data => {
                if (!data.announcement) {
                    return;
                }
                const dismissedId = localStorage.getItem(DISMISSED_KEY);
                if (dismissedId === data.announcement.id) {
                    setDismissed(true);
                } else {
                    setAnnouncement(data.announcement);
                }
            })
            .catch(() => {});
    }, []);

    if (!announcement || dismissed) {
        return null;
    }

    function handleDismiss() {
        if (announcement) {
            localStorage.setItem(DISMISSED_KEY, announcement.id);
        }
        setDismissed(true);
    }

    return (
        <div className={styles.card}>
            <div className={styles.header}>
                <span className={styles.badge}>Announcement</span>
                <span
                    className={styles.title}
                    onClick={() => navigate(`/announcements/${announcement.id}`)}
                >
                    {announcement.title}
                </span>
                <button className={styles.dismiss} onClick={handleDismiss} title="Dismiss">
                    {"\u2715"}
                </button>
            </div>
            <div
                className={styles.body}
                dangerouslySetInnerHTML={{ __html: renderMarkdown(announcement.body) }}
            />
            <span
                className={styles.readMore}
                onClick={() => navigate(`/announcements/${announcement.id}`)}
            >
                Read more &rarr;
            </span>
            <div className={styles.meta}>
                <ProfileLink user={announcement.author} size="small" />
                <span>{relativeTime(announcement.created_at)}</span>
            </div>
        </div>
    );
}
