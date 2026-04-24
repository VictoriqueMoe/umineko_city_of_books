import { useState } from "react";
import { Link } from "react-router";
import type { Theory } from "../../../types/api";
import type { Series } from "../../../api/endpoints";
import { useAuth } from "../../../hooks/useAuth";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { CredibilityBadge } from "../CredibilityBadge/CredibilityBadge";
import { formatSeriesEpisode, userProgressForSeries } from "../../../utils/seriesConfig";
import { formatFullDateTime } from "../../../utils/time";
import styles from "./TheoryCard.module.css";

interface TheoryCardProps {
    theory: Theory;
}

export function TheoryCard({ theory }: TheoryCardProps) {
    const { user } = useAuth();
    const [spoilerRevealed, setSpoilerRevealed] = useState(false);

    const seriesKey = (theory.series || "umineko") as Series;
    const userProgress = userProgressForSeries(user, seriesKey);
    const isSpoiler = !spoilerRevealed && userProgress > 0 && theory.episode > 0 && theory.episode >= userProgress;

    return (
        <Link
            to={`/theory/${theory.id}`}
            className={styles.card}
            onClick={e => {
                if (isSpoiler) {
                    e.preventDefault();
                }
            }}
        >
            {isSpoiler && (
                <div className={styles.spoilerOverlay}>
                    <span>Spoiler: {formatSeriesEpisode(seriesKey, theory.episode)}</span>
                    <button
                        onClick={e => {
                            e.stopPropagation();
                            setSpoilerRevealed(true);
                        }}
                    >
                        Show anyway
                    </button>
                </div>
            )}
            <div className={isSpoiler ? styles.blurred : undefined}>
                <div className={styles.byline} onClick={e => e.stopPropagation()}>
                    <ProfileLink user={theory.author} size="small" />
                    's Blue Truth
                </div>
                <div className={styles.header}>
                    <h3 className={styles.title}>{theory.title}</h3>
                    {theory.episode > 0 && (
                        <span className={styles.episode}>{formatSeriesEpisode(seriesKey, theory.episode)}</span>
                    )}
                </div>
                <p className={styles.body}>{theory.body}</p>
                <div className={styles.meta}>
                    <CredibilityBadge score={theory.credibility_score} />
                    <span>{theory.vote_score} votes</span>
                    <span className={styles.withLove}>
                        {"\u2764"} {theory.with_love_count}
                    </span>
                    <span className={styles.withoutLove}>
                        {"\u2718"} {theory.without_love_count}
                    </span>
                    <span className={styles.timestamp}>{formatFullDateTime(theory.created_at)}</span>
                </div>
            </div>
        </Link>
    );
}
