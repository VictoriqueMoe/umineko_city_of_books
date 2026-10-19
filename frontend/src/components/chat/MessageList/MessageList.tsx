import { memo, useCallback, useMemo, type ReactNode, type Ref } from "react";
import { seenLabel } from "../../../domain/chat/dm";
import { messagePermalink } from "../../../domain/chat/permalink";
import {
    isPendingSend,
    pendingPreview,
    threadEntryKey,
    type SenderIdentity,
    type ThreadEntry,
} from "../../../domain/chat/optimisticSend";
import { roomCapabilities } from "../../../domain/chat/roomPolicy";
import { isSiteStaff } from "../../../domain/permissions";
import type { ChatMessage, ChatRoom, UserProfile } from "../../../types/api";
import { useBlockedUserIds } from "../../../hooks/useBlockedUserIds";
import { siteUrl } from "../../../platform/siteOrigin";
import { errorMessage } from "../../../utils/errorMessage";
import { type ReplyTarget } from "../ChatComposer/ChatComposer";
import { MessageBubble } from "../MessageBubble/MessageBubble";

const NO_READ_RECEIPTS: Record<string, Record<string, string>> = {};
const LINK_COPIED = "Link copied";
const LINK_COPY_FAILED = "Couldn't copy the link";
const DEFAULT_EMPTY_LABEL = "No messages yet. Say hello!";

export interface MessageListClasses {
    messages: string;
    loadMoreBar: string;
    empty: string;
}

export interface MessageListProps {
    viewer: UserProfile;
    viewerIdentity?: SenderIdentity | null;
    room: ChatRoom | null;
    emptyLabel?: ReactNode;
    entries: ThreadEntry[];
    hasMore: boolean;
    loadingMore: boolean;
    editingMessageId: string | null;
    highlightedMessageId?: string | null;
    viewerTimedOut?: boolean;
    readReceipts?: Record<string, Record<string, string>>;
    matchesViewerMention: ((body: string) => boolean) | null;
    containerRef: Ref<HTMLDivElement>;
    contentRef: Ref<HTMLDivElement>;
    endRef: Ref<HTMLDivElement>;
    onScroll: () => void;
    onLightbox: (src: string) => void;
    onReply: (target: ReplyTarget) => void;
    onStartEditing: (message: ChatMessage) => void;
    onCancelEditing: () => void;
    onToggleReaction: (message: ChatMessage, emoji: string) => void;
    onTogglePin?: (message: ChatMessage) => void;
    onDelete: (message: ChatMessage) => void;
    onEdit: (message: ChatMessage, body: string) => Promise<void>;
    onRetrySend: (nonce: string) => void;
    onDiscardSend: (nonce: string) => void;
    onNotify: (message: string) => void;
    classes: MessageListClasses;
}

function MessageListBase({
    viewer,
    viewerIdentity = null,
    room,
    emptyLabel = DEFAULT_EMPTY_LABEL,
    entries,
    hasMore,
    loadingMore,
    editingMessageId,
    highlightedMessageId = null,
    viewerTimedOut = false,
    readReceipts = NO_READ_RECEIPTS,
    matchesViewerMention,
    containerRef,
    contentRef,
    endRef,
    onScroll,
    onLightbox,
    onReply,
    onStartEditing,
    onCancelEditing,
    onToggleReaction,
    onTogglePin,
    onDelete,
    onEdit,
    onRetrySend,
    onDiscardSend,
    onNotify,
    classes,
}: MessageListProps) {
    const blockedIDs = useBlockedUserIds();
    const capabilities = roomCapabilities(room, viewer);

    const canPin = capabilities.canPinMessages && onTogglePin !== undefined;
    const canSeeSeenLabels = capabilities.readReceipts === "pairwise";

    const handleReply = useCallback(
        (message: ChatMessage) => {
            onReply({
                id: message.id,
                senderName: message.sender.display_name,
                body: message.body,
            });
        },
        [onReply],
    );

    const handleCopyLink = useMemo(() => {
        if (!room) {
            return undefined;
        }

        return (message: ChatMessage) => {
            const link = siteUrl(messagePermalink(room, message));

            Promise.resolve()
                .then(() => navigator.clipboard.writeText(link))
                .then(() => onNotify(LINK_COPIED))
                .catch((thrown: unknown) => {
                    const reason = errorMessage(thrown, "");

                    onNotify(reason === "" ? LINK_COPY_FAILED : `${LINK_COPY_FAILED}: ${reason}`);
                });
        };
    }, [room, onNotify]);

    return (
        <div className={classes.messages} ref={containerRef} onScroll={onScroll}>
            {entries.length === 0 && !hasMore && <div className={classes.empty}>{emptyLabel}</div>}
            <div ref={contentRef} style={{ display: "flex", flexDirection: "column", gap: "inherit" }}>
                {hasMore && (
                    <div className={classes.loadMoreBar}>
                        {loadingMore ? "Loading older messages..." : "Scroll up for more"}
                    </div>
                )}
                {entries.map((entry, idx) => {
                    if (isPendingSend(entry)) {
                        return (
                            <MessageBubble
                                key={threadEntryKey(entry)}
                                message={pendingPreview(entry, viewer, viewerIdentity)}
                                isOwn
                                delivery={entry}
                                onRetrySend={onRetrySend}
                                onDiscardSend={onDiscardSend}
                                onLightbox={onLightbox}
                            />
                        );
                    }

                    const isOwn = entry.sender.id === viewer.id;
                    const label =
                        room && canSeeSeenLabels && isOwn
                            ? seenLabel(entry, idx, entries, room, viewer.id, readReceipts)
                            : null;

                    return (
                        <MessageBubble
                            key={threadEntryKey(entry)}
                            message={entry}
                            isOwn={isOwn}
                            senderBlocked={blockedIDs.has(entry.sender.id)}
                            highlighted={entry.id === highlightedMessageId}
                            notifiesViewer={
                                entry.reply_to?.sender_id === viewer.id ||
                                (matchesViewerMention ? matchesViewerMention(entry.body) : false)
                            }
                            seenLabel={label}
                            onLightbox={onLightbox}
                            onReply={handleReply}
                            onReactionToggle={onToggleReaction}
                            onPinToggle={canPin ? onTogglePin : undefined}
                            onDelete={onDelete}
                            onEdit={onEdit}
                            onEditStart={onStartEditing}
                            onEditCancel={onCancelEditing}
                            onCopyLink={handleCopyLink}
                            editing={editingMessageId === entry.id}
                            canPin={canPin}
                            canModerate={capabilities.canModerateRoom}
                            canReact={!viewerTimedOut}
                            canEdit={!viewerTimedOut}
                            senderIsStaff={isSiteStaff(entry.sender.role)}
                        />
                    );
                })}
                <div ref={endRef} />
            </div>
        </div>
    );
}

export const MessageList = memo(MessageListBase);
