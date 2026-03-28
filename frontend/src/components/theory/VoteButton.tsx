interface VoteButtonProps {
    score: number;
    userVote: number;
    onVote: (value: number) => void;
}

export function VoteButton({ score, userVote, onVote }: VoteButtonProps) {
    return (
        <div className="vote-button">
            <button
                className={`vote-arrow vote-up${userVote === 1 ? " active" : ""}`}
                onClick={() => onVote(1)}
                aria-label="Upvote"
            >
                {"\u25B2"}
            </button>
            <span className="vote-score">{score}</span>
            <button
                className={`vote-arrow vote-down${userVote === -1 ? " active" : ""}`}
                onClick={() => onVote(-1)}
                aria-label="Downvote"
            >
                {"\u25BC"}
            </button>
        </div>
    );
}
