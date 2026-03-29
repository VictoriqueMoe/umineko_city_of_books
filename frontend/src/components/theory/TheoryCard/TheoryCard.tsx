import { useNavigate } from "react-router";
import type { Theory } from "../../../types/api";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import styles from "./TheoryCard.module.css";

interface TheoryCardProps {
    theory: Theory;
}

export function TheoryCard({ theory }: TheoryCardProps) {
    const navigate = useNavigate();

    return (
        <div
            className={styles.card}
            onClick={() => navigate(`/theory/${theory.id}`)}
            role="button"
            tabIndex={0}
            onKeyDown={e => {
                if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    navigate(`/theory/${theory.id}`);
                }
            }}
        >
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
                <span>{theory.vote_score} votes</span>
                <span className={styles.withLove}>
                    {"\u2764"} {theory.with_love_count}
                </span>
                <span className={styles.withoutLove}>
                    {"\u2718"} {theory.without_love_count}
                </span>
            </div>
        </div>
    );
}
