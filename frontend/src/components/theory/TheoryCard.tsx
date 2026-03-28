import {useNavigate} from "react-router";
import type {Theory} from "../../types/api";
import {ProfileLink} from "../common/ProfileLink";

interface TheoryCardProps {
    theory: Theory;
}

export function TheoryCard({ theory }: TheoryCardProps) {
    const navigate = useNavigate();

    return (
        <div
            className="theory-card"
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
            <div className="theory-card-byline" onClick={e => e.stopPropagation()}>
                <ProfileLink user={theory.author} size="small" />
                's Blue Truth
            </div>
            <div className="theory-card-header">
                <h3 className="theory-card-title">{theory.title}</h3>
                {theory.episode > 0 && <span className="truth-episode">Episode {theory.episode}</span>}
            </div>
            <p className="theory-card-body">{theory.body}</p>
            <div className="theory-card-meta">
                <span className="theory-card-score">{theory.vote_score} votes</span>
                <span className="theory-card-responses with-love">
                    {"\u2764"} {theory.with_love_count}
                </span>
                <span className="theory-card-responses without-love">
                    {"\u2718"} {theory.without_love_count}
                </span>
            </div>
        </div>
    );
}
