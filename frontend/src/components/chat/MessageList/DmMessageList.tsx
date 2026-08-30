import { memo, useCallback, type Ref } from "react";
import { seenLabel } from "../../../domain/chat/dm";
import { isSiteStaff } from "../../../domain/permissions";
import type { ChatMessage, ChatRoom, UserProfile } from "../../../types/api";
import { type ReplyTarget } from "../ChatComposer/ChatComposer";
import { MessageBubble } from "../MessageBubble/MessageBubble";

const REPLY_PREVIEW_MAX = 80;

export interface DmMessageListClasses {
    messages: string;
    loadMoreBar: string;
}

export interface DmMessageListProps {
    viewer: UserProfile;
    room: ChatRoom;
    messages: ChatMessage[];
    hasMore: boolean;
    loadingMore: boolean;
    editingMessageId: string | null;
    readReceipts: Record<string, Record<string, string>>;
    matchesViewerMention: ((body: string) => boolean) | null;
    containerRef: Ref<HTMLDivElement>;
    contentRef: Ref<HTMLDivElement>;
    endRef: Ref<HTMLDivElement>;
    onScroll: () => void;
    onLightbox: (src: string) => void;
    onReply: (target: ReplyTarget) => void;
    onStartEditing: (message: ChatMessage) => void;
    onCancelEditing: () => void;
    onDelete: (message: ChatMessage) => void;
    onEdit: (message: ChatMessage, body: string) => Promise<void>;
    classes: DmMessageListClasses;
}

function replyPreview(body: string): string {
    return body.length > REPLY_PREVIEW_MAX ? body.slice(0, REPLY_PREVIEW_MAX) + "..." : body;
}

function DmMessageListBase({
    viewer,
    room,
    messages,
    hasMore,
    loadingMore,
    editingMessageId,
    readReceipts,
    matchesViewerMention,
    containerRef,
    contentRef,
    endRef,
    onScroll,
    onLightbox,
    onReply,
    onStartEditing,
    onCancelEditing,
    onDelete,
    onEdit,
    classes,
}: DmMessageListProps) {
    const isSiteMod = isSiteStaff(viewer.role);

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
            <div ref={contentRef} style={{ display: "flex", flexDirection: "column", gap: "inherit" }}>
                {hasMore && (
                    <div className={classes.loadMoreBar}>
                        {loadingMore ? "Loading older messages..." : "Scroll up for more"}
                    </div>
                )}
                {messages.map((msg, idx) => {
                    const isOwn = msg.sender.id === viewer.id;
                    const label = isOwn ? seenLabel(msg, idx, messages, room, viewer.id, readReceipts) : null;
                    return (
                        <MessageBubble
                            key={msg.id}
                            message={msg}
                            isOwn={isOwn}
                            notifiesViewer={
                                msg.reply_to?.sender_id === viewer.id ||
                                (matchesViewerMention ? matchesViewerMention(msg.body) : false)
                            }
                            seenLabel={label}
                            onLightbox={onLightbox}
                            onReply={handleReply}
                            onDelete={onDelete}
                            onEdit={onEdit}
                            onEditStart={onStartEditing}
                            onEditCancel={onCancelEditing}
                            editing={editingMessageId === msg.id}
                            canModerate={isSiteMod}
                            senderIsStaff={isSiteStaff(msg.sender.role)}
                        />
                    );
                })}
                <div ref={endRef} />
            </div>
        </div>
    );
}

export const DmMessageList = memo(DmMessageListBase);
