import {useCallback, useState} from "react";
import type {Response as TheoryResponse} from "../../types/api";
import {useAuth} from "../../hooks/useAuth";
import {useVote} from "../../hooks/useVote";
import {deleteResponse, voteResponse} from "../../api/endpoints";
import {ProfileLink} from "../common/ProfileLink";
import {VoteButton} from "./VoteButton";
import {EvidenceList} from "./EvidenceList";
import {ResponseEditor} from "./ResponseEditor";

interface ResponseCardProps {
    response: TheoryResponse;
    theoryId: number;
    onDeleted?: () => void;
    onReply?: (parentId: number, parentAuthor: string) => void;
    replyTarget?: { parentId: number; parentAuthor: string } | null;
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

    return (
        <div className={`response-card ${response.side}`}>
            <div className="response-vote-strip">
                <VoteButton score={score} userVote={userVote} onVote={vote} />
            </div>
            <div className="response-content">
                {mentionedAuthor && <div className="response-mention">@{mentionedAuthor}</div>}
                <div className="response-body">{response.body}</div>

                <EvidenceList evidence={response.evidence ?? []} />

                <div className="response-meta">
                    <ProfileLink user={response.author} size="small" />
                    <div className="response-actions-inline">
                        {user && onReply && (
                            <button
                                className="response-action-btn"
                                onClick={() => onReply(response.id, response.author.display_name)}
                            >
                                Reply
                            </button>
                        )}
                        {user && user.id === response.author.id && (
                            <button className="response-action-btn danger" onClick={handleDelete}>
                                Delete
                            </button>
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
    theoryId: number;
    onDeleted?: () => void;
}) {
    const [replyTarget, setReplyTarget] = useState<{ parentId: number; parentAuthor: string } | null>(null);

    function handleReply(parentId: number, parentAuthor: string) {
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

    return (
        <div className="response-list">
            {responses.map(response => {
                const threadReplies = flattenThread(response.replies ?? []);
                const hasThread = threadReplies.length > 0;

                return (
                    <div key={response.id} className="response-thread-group">
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
    onDeleted,
    onReply,
    replyTarget,
}: {
    replies: Array<{ reply: TheoryResponse; mentionedAuthor?: string }>;
    response: TheoryResponse;
    theoryId: number;
    onDeleted?: () => void;
    onReply: (parentId: number, parentAuthor: string) => void;
    replyTarget: { parentId: number; parentAuthor: string } | null;
}) {
    const [expanded, setExpanded] = useState(false);

    if (!expanded) {
        return (
            <button className="thread-expand-btn" onClick={() => setExpanded(true)}>
                Show {replies.length} {replies.length === 1 ? "reply" : "replies"}
            </button>
        );
    }

    return (
        <div className="response-thread">
            <div className="thread-line" />
            <div className="thread-replies">
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
                <button className="thread-expand-btn" onClick={() => setExpanded(false)}>
                    Hide replies
                </button>
            </div>
        </div>
    );
}
