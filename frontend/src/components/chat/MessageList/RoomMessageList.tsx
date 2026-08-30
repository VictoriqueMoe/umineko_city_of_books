import { memo, useCallback, type Ref } from "react";
import { isSiteStaff } from "../../../domain/permissions";
import { roomViewerPolicy } from "../../../domain/chat/roomPolicy";
import type { ChatMessage, ChatRoom, UserProfile } from "../../../types/api";
import { useBlockedUserIds } from "../../../hooks/useBlockedUserIds";
import { type ReplyTarget } from "../ChatComposer/ChatComposer";
import { MessageBubble } from "../MessageBubble/MessageBubble";

const REPLY_PREVIEW_MAX = 80;

export interface RoomMessageListClasses {
    messages: string;
    loadMoreBar: string;
    empty: string;
}

export interface RoomMessageListProps {
    viewer: UserProfile;
    room: ChatRoom;
    messages: ChatMessage[];
    hasMore: boolean;
    loadingMore: boolean;
    highlightedMessageId: string | null;
    editingMessageId: string | null;
    viewerTimedOut: boolean;
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
    onTogglePin: (message: ChatMessage) => void;
    onDelete: (message: ChatMessage) => void;
    onEdit: (message: ChatMessage, body: string) => Promise<void>;
    classes: RoomMessageListClasses;
}

function replyPreview(body: string): string {
    return body.length > REPLY_PREVIEW_MAX ? body.slice(0, REPLY_PREVIEW_MAX) + "..." : body;
}

function RoomMessageListBase({
    viewer,
    room,
    messages,
    hasMore,
    loadingMore,
    highlightedMessageId,
    editingMessageId,
    viewerTimedOut,
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
}: RoomMessageListProps) {
    const blockedIDs = useBlockedUserIds();
    const { canModerateRoom } = roomViewerPolicy(room, viewer);

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
                {messages.map(msg => (
                    <MessageBubble
                        key={msg.id}
                        message={msg}
                        isOwn={msg.sender.id === viewer.id}
                        senderBlocked={blockedIDs.has(msg.sender.id)}
                        highlighted={msg.id === highlightedMessageId}
                        notifiesViewer={
                            msg.reply_to?.sender_id === viewer.id ||
                            (matchesViewerMention ? matchesViewerMention(msg.body) : false)
                        }
                        onLightbox={onLightbox}
                        onReply={handleReply}
                        onReactionToggle={onToggleReaction}
                        onPinToggle={canModerateRoom ? onTogglePin : undefined}
                        onDelete={onDelete}
                        onEdit={onEdit}
                        onEditStart={onStartEditing}
                        onEditCancel={onCancelEditing}
                        editing={editingMessageId === msg.id}
                        canPin={canModerateRoom}
                        canModerate={canModerateRoom}
                        canReact={!viewerTimedOut}
                        canEdit={!viewerTimedOut}
                        senderIsStaff={isSiteStaff(msg.sender.role)}
                    />
                ))}
                <div ref={endRef} />
            </div>
        </div>
    );
}

export const RoomMessageList = memo(RoomMessageListBase);
