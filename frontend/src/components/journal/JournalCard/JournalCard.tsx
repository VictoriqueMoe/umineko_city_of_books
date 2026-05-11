import { Link } from "react-router";
import type { Journal } from "../../../types/api";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { workLabel } from "../../../utils/journalWorks";
import { relativeTime } from "../../../utils/time";
import styles from "./JournalCard.module.css";

interface JournalCardProps {
    journal: Journal;
}

function entryHeading(number: number, title?: string | null): string {
    if (title && title.trim() !== "") {
        return `Entry ${number}: ${title}`;
    }
    return `Entry ${number}`;
}

export function JournalCard({ journal }: JournalCardProps) {
    const hasLatest = typeof journal.latest_entry_number === "number";
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
            {hasLatest && (
                <div className={styles.latest}>
                    <span className={styles.latestLabel}>Latest:</span>{" "}
                    <span className={styles.latestEntry}>
                        {entryHeading(journal.latest_entry_number!, journal.latest_entry_title)}
                    </span>
                    {journal.latest_entry_at && (
                        <span className={styles.latestWhen}>
                            {"·"} {relativeTime(journal.latest_entry_at)}
                        </span>
                    )}
                </div>
            )}
            {journal.latest_entry_excerpt && <p className={styles.body}>{journal.latest_entry_excerpt}</p>}
            <div className={styles.meta}>
                <span>
                    {"★"} {journal.follower_count} follower{journal.follower_count === 1 ? "" : "s"}
                </span>
                <span>
                    {"📖"} {journal.entry_count} {journal.entry_count === 1 ? "entry" : "entries"}
                </span>
                <span>
                    {"💬"} {journal.comment_count}
                </span>
                <span className={styles.activity}>Last update {relativeTime(journal.last_author_activity_at)}</span>
            </div>
        </Link>
    );
}
