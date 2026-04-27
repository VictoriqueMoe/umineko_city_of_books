import React, { useState } from "react";
import type { Poll } from "../../../types/api";
import { useVotePoll } from "../../../api/mutations/post";
import { useAuth } from "../../../hooks/useAuth";
import { parseServerDate } from "../../../utils/time";
import { Button } from "../../Button/Button";
import styles from "./PollDisplay.module.css";

interface PollDisplayProps {
    poll: Poll;
    postId: string;
    onVoted?: () => void;
}

function timeRemaining(expiresAt: string): string {
    const d = parseServerDate(expiresAt);
    if (!d) {
        return "";
    }
    const diff = d.getTime() - Date.now();
    if (diff <= 0) {
        return "Poll ended";
    }
    const mins = Math.floor(diff / 60000);
    if (mins < 60) {
        return `${mins}m remaining`;
    }
    const hours = Math.floor(mins / 60);
    if (hours < 24) {
        return `${hours}h remaining`;
    }
    const days = Math.floor(hours / 24);
    return `${days}d remaining`;
}

export function PollDisplay({ poll: initialPoll, postId, onVoted }: PollDisplayProps) {
    const { user } = useAuth();
    const [poll, setPoll] = useState(initialPoll);
    const [selected, setSelected] = useState<number | null>(null);
    const [submitting, setSubmitting] = useState(false);
    const voteMutation = useVotePoll();

    const hasVoted = poll.user_voted_option !== null;
    const showResults = hasVoted || poll.expired;
    const maxPercent = Math.max(...poll.options.map(o => o.percent), 1);

    async function handleVote() {
        if (selected === null || submitting) {
            return;
        }
        setSubmitting(true);
        try {
            const updatedPoll = await voteMutation.mutateAsync({ postId, optionIdx: selected });
            setPoll(updatedPoll);
            onVoted?.();
        } catch {
            // vote failed
        } finally {
            setSubmitting(false);
        }
    }

    function handleOptionClick(e: React.MouseEvent, optionId: number) {
        e.stopPropagation();
        if (showResults || !user) {
            return;
        }
        setSelected(prev => (prev === optionId ? null : optionId));
    }

    return (
        <div className={styles.poll} onClick={e => e.stopPropagation()}>
            <div className={styles.options}>
                {poll.options.map(option => {
                    const isSelected = selected === option.id;
                    const isVotedOption = poll.user_voted_option === option.id;
                    const isWinner = showResults && option.percent === maxPercent && option.percent > 0;

                    let className = styles.option;
                    if (isSelected) {
                        className += ` ${styles.optionSelected}`;
                    }
                    if (showResults) {
                        className += ` ${styles.optionDisabled}`;
                    }

                    return (
                        <div key={option.id} className={className} onClick={e => handleOptionClick(e, option.id)}>
                            {showResults && (
                                <div
                                    className={`${styles.resultBar}${isWinner ? ` ${styles.resultBarWinner}` : ""}`}
                                    style={{ width: `${option.percent}%` }}
                                />
                            )}
                            <div className={styles.optionContent}>
                                <span className={styles.optionLabel}>
                                    {isVotedOption && <span className={styles.checkmark}>&#10003;</span>}
                                    {option.label}
                                </span>
                                {showResults && (
                                    <span className={styles.optionPercent}>{Math.round(option.percent)}%</span>
                                )}
                            </div>
                        </div>
                    );
                })}
            </div>

            {!showResults && user && (
                <div className={styles.submitRow}>
                    <Button
                        variant="primary"
                        size="small"
                        onClick={handleVote}
                        disabled={selected === null || submitting}
                    >
                        {submitting ? "Submitting..." : "Submit Vote"}
                    </Button>
                </div>
            )}

            <div className={styles.footer}>
                <span>
                    {poll.total_votes} {poll.total_votes === 1 ? "vote" : "votes"}
                </span>
                <span>{timeRemaining(poll.expires_at)}</span>
            </div>
        </div>
    );
}
