import { Link } from "react-router";
import type { Journal } from "../../../types/api";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { workLabel } from "../../../utils/journalWorks";
import styles from "./JournalCard.module.css";

interface JournalCardProps {
    journal: Journal;
}

function relativeDate(dateStr: string): string {
    const diff = Date.now() - new Date(dateStr).getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 1) {
        return "just now";
    }
    if (mins < 60) {
        return `${mins}m ago`;
    }
    const hours = Math.floor(mins / 60);
    if (hours < 24) {
        return `${hours}h ago`;
    }
    const days = Math.floor(hours / 24);
    if (days < 30) {
        return `${days}d ago`;
    }
    return new Date(dateStr).toLocaleDateString();
}

export function JournalCard({ journal }: JournalCardProps) {
    return (
        <Link to={`/journals/${journal.id}`} className={styles.card}>
            <div className={styles.byline} onClick={e => e.stopPropagation()}>
                <ProfileLink user={journal.author} size="small" />
                's Reading Journal
            </div>
            <div className={styles.header}>
                <h3 className={styles.title}>{journal.title}</h3>
                <span className={styles.work}>{workLabel(journal.work)}</span>
                {journal.is_archived && <span className={styles.archived}>Archived</span>}
            </div>
            <p className={styles.body}>{journal.body}</p>
            <div className={styles.meta}>
                <span>
                    {"\u2605"} {journal.follower_count} follower{journal.follower_count === 1 ? "" : "s"}
                </span>
                <span>
                    {"\uD83D\uDCAC"} {journal.comment_count}
                </span>
                <span className={styles.activity}>Last update {relativeDate(journal.last_author_activity_at)}</span>
            </div>
        </Link>
    );
}
