import { useState } from "react";
import { useNavigate } from "react-router";
import { marked } from "marked";
import DOMPurify from "dompurify";
import { useLatestAnnouncement } from "../../api/queries/announcement";
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
    const { announcement } = useLatestAnnouncement();
    const [dismissed, setDismissed] = useState(false);

    const dismissedId = typeof localStorage !== "undefined" ? localStorage.getItem(DISMISSED_KEY) : null;
    const visible = announcement && !dismissed && dismissedId !== announcement.id;

    if (!visible || !announcement) {
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
                <span className={styles.title} onClick={() => navigate(`/announcements/${announcement.id}`)}>
                    {announcement.title}
                </span>
                <button className={styles.dismiss} onClick={handleDismiss} title="Dismiss">
                    {"✕"}
                </button>
            </div>
            <div className={styles.body} dangerouslySetInnerHTML={{ __html: renderMarkdown(announcement.body) }} />
            <span className={styles.readMore} onClick={() => navigate(`/announcements/${announcement.id}`)}>
                Read more &rarr;
            </span>
            <div className={styles.meta}>
                <ProfileLink user={announcement.author} size="small" />
                <span>{relativeTime(announcement.created_at)}</span>
            </div>
        </div>
    );
}
