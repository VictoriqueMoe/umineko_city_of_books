import { memo, useCallback, type Ref } from "react";
import { seenLabel } from "../../../domain/chat/dm";
import { replyPreview } from "../../../domain/chat/replyPreview";
import { roomCapabilities } from "../../../domain/chat/roomPolicy";
import { isSiteStaff } from "../../../domain/permissions";
import type { ChatMessage, ChatRoom, UserProfile } from "../../../types/api";
import { useBlockedUserIds } from "../../../hooks/useBlockedUserIds";
import { type ReplyTarget } from "../ChatComposer/ChatComposer";
import { MessageBubble } from "../MessageBubble/MessageBubble";

const NO_READ_RECEIPTS: Record<string, Record<string, string>> = {};

export interface MessageListClasses {
    messages: string;
    loadMoreBar: string;
    empty: string;
}

export interface MessageListProps {
    viewer: UserProfile;
    room: ChatRoom;
    messages: ChatMessage[];
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
    classes: MessageListClasses;
}

function MessageListBase({
    viewer,
    room,
    messages,
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
                bodyPreview: replyPreview(message.body),
            });
        },
        [onReply],
    );

    return (
        <div className={classes.messages} ref={containerRef} onScroll={onScroll}>
            {messages.length === 0 && !hasMore && <div className={classes.empty}>No messages yet. Say hello!</div>}
            <div ref={contentRef} style={{ display: "flex", flexDirection: "column", gap: "inherit" }}>
                {hasMore && (
                    <div className={classes.loadMoreBar}>
                        {loadingMore ? "Loading older messages..." : "Scroll up for more"}
                    </div>
                )}
                {messages.map((msg, idx) => {
                    const isOwn = msg.sender.id === viewer.id;
                    const label =
                        canSeeSeenLabels && isOwn ? seenLabel(msg, idx, messages, room, viewer.id, readReceipts) : null;

                    return (
                        <MessageBubble
                            key={msg.id}
                            message={msg}
                            isOwn={isOwn}
                            senderBlocked={blockedIDs.has(msg.sender.id)}
                            highlighted={msg.id === highlightedMessageId}
                            notifiesViewer={
                                msg.reply_to?.sender_id === viewer.id ||
                                (matchesViewerMention ? matchesViewerMention(msg.body) : false)
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
                            editing={editingMessageId === msg.id}
                            canPin={canPin}
                            canModerate={capabilities.canModerateRoom}
                            canReact={!viewerTimedOut}
                            canEdit={!viewerTimedOut}
                            senderIsStaff={isSiteStaff(msg.sender.role)}
                        />
                    );
                })}
                <div ref={endRef} />
            </div>
        </div>
    );
}

export const MessageList = memo(MessageListBase);
