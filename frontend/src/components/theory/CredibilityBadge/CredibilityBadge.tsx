import styles from "./CredibilityBadge.module.css";

interface CredibilityBadgeProps {
    score: number;
}

function scoreClass(score: number): string {
    if (score >= 70) {
        return styles.high;
    }
    if (score >= 40) {
        return styles.mid;
    }
    return styles.low;
}

export function CredibilityBadge({ score }: CredibilityBadgeProps) {
    const rounded = Math.round(score);

    return (
        <span className={styles.badge}>
            <span className={styles.prefix}>Credibility</span>
            <span className={`${styles.score} ${scoreClass(score)}`}>{rounded}</span>
            <span className={styles.infoIcon}>
                ?
                <span className={styles.tooltip}>
                    A 0-100 score based on the strength of debate responses.<br /><br />
                    Responses backed by red or gold truth evidence carry more weight.<br /><br />
                    50 is neutral, higher means stronger community support.
                </span>
            </span>
        </span>
    );
}
