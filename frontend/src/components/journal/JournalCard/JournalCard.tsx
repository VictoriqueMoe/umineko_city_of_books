import { Link } from "react-router";
import type { Journal } from "../../../types/api";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { workLabel } from "../../../utils/journalWorks";
import { relativeTime } from "../../../utils/time";
import styles from "./JournalCard.module.css";

interface JournalCardProps {
    journal: Journal;
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
                <span className={styles.activity}>Last update {relativeTime(journal.last_author_activity_at)}</span>
            </div>
        </Link>
    );
}
