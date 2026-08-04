import { useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Link } from "react-router";
import { useAuth } from "../../../hooks/useAuth";
import { useNotifications } from "../../../hooks/useNotifications";
import { useMessageHistory } from "../../../hooks/useMessageHistory";
import { useChatMessageHandlers } from "../../../hooks/useChatMessageHandlers";
import { useBlockedUserIds } from "../../../hooks/useBlockedUserIds";
import { MessageBubble } from "../MessageBubble/MessageBubble";
import { Lightbox } from "../../Lightbox/Lightbox";
import { ChatComposer, type ChatComposerHandle, type ReplyTarget } from "../ChatComposer/ChatComposer";
import { handleIncomingChatMessage, applySharedChatWSBranch } from "../../../utils/chatStream";
import type { ChatMessage, UserProfile, WSMessage } from "../../../types/api";
import styles from "./RoomChatPanel.module.css";

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
    const { addWSListener } = useNotifications();
    const blockedIDs = useBlockedUserIds();
    const [replyingTo, setReplyingTo] = useState<ReplyTarget | null>(null);
    const [editingMessageId, setEditingMessageId] = useState<string | null>(null);
    const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);
    const composerRef = useRef<ChatComposerHandle>(null);

    const {
        messages,
        setMessages,
        hasMore,
        loadingMore,
        containerRef,
        contentRef,
        endRef,
        scrollToBottomInstant,
        handleScroll,
        addMessage,
    } = useMessageHistory(roomId, editingMessageId === null ? maxMessages : undefined);

    const { handleEditMessage } = useChatMessageHandlers({
        user,
        messages,
        setMessages,
        setEditingMessageId,
    });

    useEffect(() => {
        if (!roomId) {
            return;
        }
        return addWSListener((msg: WSMessage) => {
            if (msg.type === "chat_message") {
                handleIncomingChatMessage(msg.data as ChatMessage, roomId, setMessages, () => scrollToBottomInstant());
                return;
            }
            applySharedChatWSBranch(msg, { activeRoomId: roomId, setMessages, noteTyping: () => {} });
        });
    }, [roomId, addWSListener, setMessages, scrollToBottomInstant]);

    const handleSent = useCallback(
        (message: ChatMessage) => {
            addMessage(message);
            scrollToBottomInstant({ force: true });
        },
        [addMessage, scrollToBottomInstant],
    );

    const handleReply = useCallback((message: ChatMessage) => {
        setReplyingTo({
            id: message.id,
            senderName: message.sender.display_name || message.sender.username,
            bodyPreview: message.body.slice(0, 140),
        });
    }, []);

    const handleEditStart = useCallback((message: ChatMessage) => {
        setEditingMessageId(message.id);
    }, []);

    const handleEditCancel = useCallback(() => {
        setEditingMessageId(null);
        composerRef.current?.focus();
    }, []);

    const handleCancelReply = useCallback(() => {
        setReplyingTo(null);
    }, []);

    const handleLightboxClose = useCallback(() => {
        setLightboxSrc(null);
    }, []);

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
            <div className={styles.chatMessages} ref={containerRef} onScroll={handleScroll}>
                <div ref={contentRef} className={styles.chatContent}>
                    {notice && <div className={styles.chatNotice}>{notice}</div>}
                    {hasMore && (
                        <div className={styles.chatNotice}>
                            {loadingMore ? "Loading older messages..." : "Scroll up for more"}
                        </div>
                    )}
                    {messages.map(m => (
                        <MessageBubble
                            key={m.id}
                            message={m}
                            isOwn={m.sender.id === user.id}
                            senderBlocked={blockedIDs.has(m.sender.id)}
                            onLightbox={setLightboxSrc}
                            onReply={handleReply}
                            onEdit={handleEditMessage}
                            onEditStart={handleEditStart}
                            onEditCancel={handleEditCancel}
                            editing={editingMessageId === m.id}
                        />
                    ))}
                    <div ref={endRef} />
                </div>
            </div>
            {canSend && roomId && (
                <ChatComposer
                    ref={composerRef}
                    roomId={roomId}
                    draftRecipientId={null}
                    onSent={handleSent}
                    replyingTo={replyingTo}
                    onCancelReply={handleCancelReply}
                    sendOnEnter
                    compact
                />
            )}
            {closedNotice && <div className={styles.chatEnded}>{closedNotice}</div>}
            {lightboxSrc && createPortal(<Lightbox src={lightboxSrc} onClose={handleLightboxClose} />, document.body)}
        </div>
    );
}
