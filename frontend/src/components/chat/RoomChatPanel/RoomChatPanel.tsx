import { useCallback, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Link } from "react-router";
import { isPendingSend, pendingPreview, threadEntryKey } from "../../../domain/chat/optimisticSend";
import { useAuth } from "../../../hooks/useAuth";
import { useBlockedUserIds } from "../../../hooks/useBlockedUserIds";
import { useChatSession } from "../../../hooks/chat/useChatSession";
import { MessageBubble } from "../MessageBubble/MessageBubble";
import { Lightbox } from "../../Lightbox/Lightbox";
import { ChatComposer, type ChatComposerHandle, type ReplyTarget } from "../ChatComposer/ChatComposer";
import type { ChatMessage, UserProfile } from "../../../types/api";
import styles from "./RoomChatPanel.module.css";

const ENDED_NOTICE = "This chat has ended.";

function panelClass(flush?: boolean): string {
    if (flush) {
        return `${styles.chatPanel} ${styles.chatPanelFlush}`;
    }

    return styles.chatPanel;
}

interface RoomChatPanelProps {
    roomId: string | undefined;
    title: string;
    canSend: boolean;
    notice?: string | null;
    closedNotice?: string | null;
    loginPrompt?: string;
    maxMessages?: number;
    onPopOut?: () => void;
    flush?: boolean;
    hideHeader?: boolean;
}

export function RoomChatPanel({ loginPrompt = "to join the chat.", ...props }: RoomChatPanelProps) {
    const { user } = useAuth();

    if (!user) {
        return (
            <div className={panelClass(props.flush)}>
                <div className={styles.chatHeader}>
                    <span>{props.title}</span>
                </div>
                <div className={styles.chatLoginPrompt}>
                    <Link to="/login">Log in</Link> {loginPrompt}
                </div>
            </div>
        );
    }

    return <RoomChatPanelInner user={user} {...props} />;
}

function RoomChatPanelInner({
    roomId,
    title,
    canSend,
    notice,
    closedNotice,
    maxMessages,
    onPopOut,
    flush,
    hideHeader,
    user,
}: RoomChatPanelProps & { user: UserProfile }) {
    const blockedIDs = useBlockedUserIds();
    const [replyingTo, setReplyingTo] = useState<ReplyTarget | null>(null);
    const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);
    const composerRef = useRef<ChatComposerHandle>(null);

    const session = useChatSession({ roomId, user, maxMessages, scrollMode: "instant" });
    const { entries, status } = session;
    const { sender, retry: retrySend, discard: discardSend } = session.sends;
    const { hasMore, loadingMore } = session.history;
    const { containerRef, contentRef, endRef, onScroll } = session.scroll;
    const { messageId: editingMessageId, start: startEditing, cancel: cancelEditing, save: saveEdit } = session.editing;

    const handleReply = useCallback((message: ChatMessage) => {
        setReplyingTo({
            id: message.id,
            senderName: message.sender.display_name || message.sender.username,
            body: message.body,
        });
    }, []);

    const handleEditCancel = useCallback(() => {
        cancelEditing();
        composerRef.current?.focus();
    }, [cancelEditing]);

    const handleCancelReply = useCallback(() => {
        setReplyingTo(null);
    }, []);

    const handleLightboxClose = useCallback(() => {
        setLightboxSrc(null);
    }, []);

    const endedNotice = closedNotice ?? (status === "ended" ? ENDED_NOTICE : null);

    return (
        <div className={panelClass(flush)}>
            {!hideHeader && (
                <div className={styles.chatHeader}>
                    <span>{title}</span>
                    {onPopOut && (
                        <button
                            type="button"
                            className={styles.chatPopOutBtn}
                            onClick={onPopOut}
                            title="Open chat in its own window"
                            aria-label="Open chat in its own window"
                        >
                            {"⧉"}
                        </button>
                    )}
                </div>
            )}
            <div className={styles.chatMessages} ref={containerRef} onScroll={onScroll}>
                <div ref={contentRef} className={styles.chatContent}>
                    {notice && <div className={styles.chatNotice}>{notice}</div>}
                    {hasMore && (
                        <div className={styles.chatNotice}>
                            {loadingMore ? "Loading older messages..." : "Scroll up for more"}
                        </div>
                    )}
                    {entries.map(entry =>
                        isPendingSend(entry) ? (
                            <MessageBubble
                                key={threadEntryKey(entry)}
                                message={pendingPreview(entry, user, null)}
                                isOwn
                                delivery={entry}
                                onRetrySend={retrySend}
                                onDiscardSend={discardSend}
                                onLightbox={setLightboxSrc}
                            />
                        ) : (
                            <MessageBubble
                                key={threadEntryKey(entry)}
                                message={entry}
                                isOwn={entry.sender.id === user.id}
                                senderBlocked={blockedIDs.has(entry.sender.id)}
                                onLightbox={setLightboxSrc}
                                onReply={handleReply}
                                onEdit={saveEdit}
                                onEditStart={startEditing}
                                onEditCancel={handleEditCancel}
                                editing={editingMessageId === entry.id}
                            />
                        ),
                    )}
                    <div ref={endRef} />
                </div>
            </div>
            {canSend && sender && status === "live" && (
                <div className={styles.composerSlot}>
                    <ChatComposer
                        ref={composerRef}
                        target={{ kind: "room", sender }}
                        replyingTo={replyingTo}
                        onCancelReply={handleCancelReply}
                        sendOnEnter
                        compact
                    />
                </div>
            )}
            {endedNotice && <div className={styles.chatEnded}>{endedNotice}</div>}
            {lightboxSrc && createPortal(<Lightbox src={lightboxSrc} onClose={handleLightboxClose} />, document.body)}
        </div>
    );
}
