import { useCallback, useEffect, useState } from "react";
import { useLocation } from "react-router";
import type { Response as TheoryResponse } from "../../../types/api";
import { useAuth } from "../../../hooks/useAuth";
import { useVote } from "../../../hooks/useVote";
import { useDeleteResponse, useVoteResponse } from "../../../api/mutations/theory";
import type { Series } from "../../../api/endpoints";
import { Button } from "../../Button/Button";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { VoteButton } from "../VoteButton/VoteButton";
import { EvidenceList } from "../EvidenceList/EvidenceList";
import { ResponseEditor } from "../ResponseEditor/ResponseEditor";
import { ReportButton } from "../../ReportButton/ReportButton";
import { can } from "../../../utils/permissions";
import styles from "./ResponseCard.module.css";

interface ResponseCardProps {
    response: TheoryResponse;
    theoryId: string;
    series?: Series;
    onDeleted?: () => void;
    onReply?: (parentId: string, parentAuthor: string) => void;
    replyTarget?: { parentId: string; parentAuthor: string } | null;
    isThreadReply?: boolean;
    mentionedAuthor?: string;
}

function ResponseCard({
    response,
    theoryId,
    series = "umineko",
    onDeleted,
    onReply,
    replyTarget,
    isThreadReply,
    mentionedAuthor,
}: ResponseCardProps) {
    const { user } = useAuth();
    const location = useLocation();
    const isHighlighted = location.hash === `#response-${response.id}`;

    const voteMutation = useVoteResponse(theoryId);
    const deleteMutation = useDeleteResponse(theoryId);
    const voteFn = useCallback(
        async (value: number) => {
            await voteMutation.mutateAsync({ responseId: response.id, value });
        },
        [response.id, voteMutation],
    );

    const { score, userVote, vote } = useVote(response.vote_score, response.user_vote ?? 0, voteFn);

    async function handleDelete() {
        if (!window.confirm("Are you sure you want to delete this response?")) {
            return;
        }
        await deleteMutation.mutateAsync(response.id);
        onDeleted?.();
    }

    const showEditor = replyTarget?.parentId === response.id;
    const sideClass = response.side === "with_love" ? styles.withLove : styles.withoutLove;

    return (
        <div
            id={`response-${response.id}`}
            className={`${styles.card} ${sideClass}${isHighlighted ? ` ${styles.highlighted}` : ""}`}
        >
            <div className={styles.voteStrip}>
                <VoteButton score={score} userVote={userVote} onVote={vote} />
            </div>
            <div className={styles.content}>
                {mentionedAuthor && <div className={styles.mention}>@{mentionedAuthor}</div>}
                <div className={styles.body}>{response.body}</div>

                <EvidenceList evidence={response.evidence ?? []} series={series} />

                <div className={styles.meta}>
                    <ProfileLink user={response.author} size="small" />
                    <div className={styles.actionsInline}>
                        {user && onReply && (
                            <Button
                                variant="ghost"
                                size="small"
                                onClick={() => onReply(response.id, response.author.display_name)}
                            >
                                Reply
                            </Button>
                        )}
                        {user && user.id !== response.author.id && (
                            <ReportButton targetType="response" targetId={response.id} contextId={theoryId} />
                        )}
                        {user && (user.id === response.author.id || can(user.role, "delete_any_response")) && (
                            <Button variant="danger" size="small" onClick={handleDelete}>
                                Delete
                            </Button>
                        )}
                    </div>
                </div>

                {showEditor && !isThreadReply && (
                    <ResponseEditor
                        theoryId={theoryId}
                        parentId={response.id}
                        inheritedSide={response.side}
                        onCreated={() => onDeleted?.()}
                        series={series}
                    />
                )}
            </div>
        </div>
    );
}

function flattenThread(replies: TheoryResponse[]): Array<{ reply: TheoryResponse; mentionedAuthor?: string }> {
    const result: Array<{ reply: TheoryResponse; mentionedAuthor?: string }> = [];
    for (const r of replies) {
        result.push({ reply: r });
        if (r.replies && r.replies.length > 0) {
            for (const nested of flattenThreadRecursive(r.replies, r.author.display_name)) {
                result.push(nested);
            }
        }
    }
    return result;
}

function flattenThreadRecursive(
    replies: TheoryResponse[],
    parentAuthor: string,
): Array<{ reply: TheoryResponse; mentionedAuthor: string }> {
    const result: Array<{ reply: TheoryResponse; mentionedAuthor: string }> = [];
    for (const r of replies) {
        result.push({ reply: r, mentionedAuthor: parentAuthor });
        if (r.replies && r.replies.length > 0) {
            result.push(...flattenThreadRecursive(r.replies, r.author.display_name));
        }
    }
    return result;
}

export function ResponseList({
    responses,
    theoryId,
    series = "umineko",
    onDeleted,
}: {
    responses: TheoryResponse[];
    theoryId: string;
    series?: Series;
    onDeleted?: () => void;
}) {
    const location = useLocation();
    const [replyTarget, setReplyTarget] = useState<{ parentId: string; parentAuthor: string } | null>(null);
    const [expandedThreads, setExpandedThreads] = useState<Set<string>>(() => {
        const hash = window.location.hash.replace("#response-", "");
        if (!hash) {
            return new Set<string>();
        }
        for (const r of responses) {
            const inThread = (r.replies ?? []).some(function check(reply: TheoryResponse): boolean {
                if (reply.id === hash) {
                    return true;
                }
                return (reply.replies ?? []).some(check);
            });
            if (inThread) {
                return new Set([r.id]);
            }
        }
        return new Set<string>();
    });

    useEffect(() => {
        const hash = location.hash;
        if (!hash) {
            return;
        }
        const timer = setTimeout(() => {
            const el = document.querySelector(hash);
            if (el) {
                el.scrollIntoView({ behavior: "smooth", block: "center" });
            }
        }, 300);
        return () => clearTimeout(timer);
    }, [location.hash, expandedThreads]);

    function handleReply(parentId: string, parentAuthor: string) {
        if (replyTarget?.parentId === parentId) {
            setReplyTarget(null);
        } else {
            setReplyTarget({ parentId, parentAuthor });
        }
    }

    function handleCreated() {
        setReplyTarget(null);
        onDeleted?.();
    }

    function toggleThread(responseId: string) {
        setExpandedThreads(prev => {
            const next = new Set(prev);
            if (next.has(responseId)) {
                next.delete(responseId);
            } else {
                next.add(responseId);
            }
            return next;
        });
    }

    return (
        <div className={styles.list}>
            {responses.map(response => {
                const threadReplies = flattenThread(response.replies ?? []);
                const hasThread = threadReplies.length > 0;

                return (
                    <div key={response.id} className={styles.threadGroup}>
                        <ResponseCard
                            response={response}
                            theoryId={theoryId}
                            series={series}
                            onDeleted={handleCreated}
                            onReply={handleReply}
                            replyTarget={replyTarget}
                        />

                        {hasThread && (
                            <ThreadReplies
                                replies={threadReplies}
                                response={response}
                                theoryId={theoryId}
                                series={series}
                                expanded={expandedThreads.has(response.id)}
                                onToggle={() => toggleThread(response.id)}
                                onDeleted={handleCreated}
                                onReply={handleReply}
                                replyTarget={replyTarget}
                            />
                        )}
                    </div>
                );
            })}
        </div>
    );
}

function ThreadReplies({
    replies,
    response,
    theoryId,
    series = "umineko",
    expanded,
    onToggle,
    onDeleted,
    onReply,
    replyTarget,
}: {
    replies: Array<{ reply: TheoryResponse; mentionedAuthor?: string }>;
    response: TheoryResponse;
    theoryId: string;
    series?: Series;
    expanded: boolean;
    onToggle: () => void;
    onDeleted?: () => void;
    onReply: (parentId: string, parentAuthor: string) => void;
    replyTarget: { parentId: string; parentAuthor: string } | null;
}) {
    if (!expanded) {
        return (
            <Button variant="ghost" size="small" onClick={onToggle}>
                Show {replies.length} {replies.length === 1 ? "reply" : "replies"}
            </Button>
        );
    }

    return (
        <div className={styles.thread}>
            <div className={styles.threadLine} />
            <div className={styles.threadReplies}>
                {replies.map(({ reply, mentionedAuthor }) => (
                    <div key={reply.id}>
                        <ResponseCard
                            response={reply}
                            theoryId={theoryId}
                            series={series}
                            onDeleted={onDeleted}
                            onReply={onReply}
                            replyTarget={replyTarget}
                            isThreadReply
                            mentionedAuthor={mentionedAuthor}
                        />
                        {replyTarget?.parentId === reply.id && (
                            <ResponseEditor
                                theoryId={theoryId}
                                parentId={reply.id}
                                inheritedSide={response.side}
                                onCreated={() => onDeleted?.()}
                                series={series}
                            />
                        )}
                    </div>
                ))}
                <Button variant="ghost" size="small" onClick={onToggle}>
                    Hide replies
                </Button>
            </div>
        </div>
    );
}
