import styles from "./VoteButton.module.css";

interface VoteButtonProps {
    score: number;
    userVote: number;
    onVote: (value: number) => void;
}

export function VoteButton({ score, userVote, onVote }: VoteButtonProps) {
    return (
        <div className={styles.container}>
            <button
                className={`${styles.arrow}${userVote === 1 ? ` ${styles.upActive}` : ""}`}
                onClick={() => onVote(1)}
                aria-label="Upvote"
            >
                {"\u25B2"}
            </button>
            <span className={styles.score}>{score}</span>
            <button
                className={`${styles.arrow}${userVote === -1 ? ` ${styles.downActive}` : ""}`}
                onClick={() => onVote(-1)}
                aria-label="Downvote"
            >
                {"\u25BC"}
            </button>
        </div>
    );
}
