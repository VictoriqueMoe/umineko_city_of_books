import { useState } from "react";
import { useNavigate } from "react-router";
import type { Theory } from "../../../types/api";
import { useAuth } from "../../../hooks/useAuth";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { CredibilityBadge } from "../CredibilityBadge/CredibilityBadge";
import styles from "./TheoryCard.module.css";

interface TheoryCardProps {
    theory: Theory;
}

export function TheoryCard({ theory }: TheoryCardProps) {
    const navigate = useNavigate();
    const { user } = useAuth();
    const [spoilerRevealed, setSpoilerRevealed] = useState(false);

    const isSpoiler =
        !spoilerRevealed &&
        (user?.episode_progress ?? 0) > 0 &&
        theory.episode > 0 &&
        theory.episode >= (user?.episode_progress ?? 0);

    return (
        <div
            className={styles.card}
            onClick={() => {
                if (!isSpoiler) {
                    navigate(`/theory/${theory.id}`);
                }
            }}
            role="button"
            tabIndex={0}
            onKeyDown={e => {
                if ((e.key === "Enter" || e.key === " ") && !isSpoiler) {
                    e.preventDefault();
                    navigate(`/theory/${theory.id}`);
                }
            }}
        >
            {isSpoiler && (
                <div className={styles.spoilerOverlay}>
                    <span>Spoiler: Episode {theory.episode}</span>
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
                    {theory.episode > 0 && <span className={styles.episode}>Episode {theory.episode}</span>}
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
                </div>
            </div>
        </div>
    );
}
