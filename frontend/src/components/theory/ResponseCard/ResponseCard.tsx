import { useCallback, useState } from "react";
import type { Response as TheoryResponse } from "../../../types/api";
import { useAuth } from "../../../hooks/useAuth";
import { useVote } from "../../../hooks/useVote";
import { deleteResponse, voteResponse } from "../../../api/endpoints";
import { Button } from "../../Button/Button";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { VoteButton } from "../VoteButton/VoteButton";
import { EvidenceList } from "../EvidenceList/EvidenceList";
import { ResponseEditor } from "../ResponseEditor/ResponseEditor";
import { can } from "../../../utils/permissions";
import styles from "./ResponseCard.module.css";

interface ResponseCardProps {
    response: TheoryResponse;
    theoryId: string;
    onDeleted?: () => void;
    onReply?: (parentId: string, parentAuthor: string) => void;
    replyTarget?: { parentId: string; parentAuthor: string } | null;
    isThreadReply?: boolean;
    mentionedAuthor?: string;
}

function ResponseCard({
    response,
    theoryId,
    onDeleted,
    onReply,
    replyTarget,
    isThreadReply,
    mentionedAuthor,
}: ResponseCardProps) {
    const { user } = useAuth();

    const voteFn = useCallback(
        async (value: number) => {
            await voteResponse(response.id, value);
        },
        [response.id],
    );

    const { score, userVote, vote } = useVote(response.vote_score, response.user_vote ?? 0, voteFn);

    async function handleDelete() {
        await deleteResponse(response.id);
        onDeleted?.();
    }

    const showEditor = replyTarget?.parentId === response.id;
    const sideClass = response.side === "with_love" ? styles.withLove : styles.withoutLove;

    return (
        <div className={`${styles.card} ${sideClass}`}>
            <div className={styles.voteStrip}>
                <VoteButton score={score} userVote={userVote} onVote={vote} />
            </div>
            <div className={styles.content}>
                {mentionedAuthor && <div className={styles.mention}>@{mentionedAuthor}</div>}
                <div className={styles.body}>{response.body}</div>

                <EvidenceList evidence={response.evidence ?? []} />

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
    onDeleted,
}: {
    responses: TheoryResponse[];
    theoryId: string;
    onDeleted?: () => void;
}) {
    const [replyTarget, setReplyTarget] = useState<{ parentId: string; parentAuthor: string } | null>(null);
    const [expandedThreads, setExpandedThreads] = useState<Set<string>>(new Set());

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
                            onDeleted={handleCreated}
                            onReply={handleReply}
                            replyTarget={replyTarget}
                        />

                        {hasThread && (
                            <ThreadReplies
                                replies={threadReplies}
                                response={response}
                                theoryId={theoryId}
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
    expanded,
    onToggle,
    onDeleted,
    onReply,
    replyTarget,
}: {
    replies: Array<{ reply: TheoryResponse; mentionedAuthor?: string }>;
    response: TheoryResponse;
    theoryId: string;
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
