import { useCallback, useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { Link } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { useNotifications } from "../../hooks/useNotifications";
import { joinStreamChat } from "../../api/endpoints";
import { useMessageHistory } from "../../hooks/useMessageHistory";
import { useChatMessageHandlers } from "../../hooks/useChatMessageHandlers";
import { useBlockedUserIds } from "../../hooks/useBlockedUserIds";
import { MessageBubble } from "../../components/chat/MessageBubble/MessageBubble";
import { Lightbox } from "../../components/Lightbox/Lightbox";
import {
    ChatComposer,
    type ChatComposerHandle,
    type ReplyTarget,
} from "../../components/chat/ChatComposer/ChatComposer";
import { handleIncomingChatMessage, applySharedChatWSBranch } from "../../utils/chatStream";
import type { ChatMessage, UserProfile, WSMessage } from "../../types/api";
import styles from "./live.module.css";

const MAX_LIVE_MESSAGES = 50;

export function StreamChatPanel({ streamId, isLive }: { streamId: string; isLive: boolean }) {
    const { user } = useAuth();

    if (!user) {
        return (
            <div className={styles.chatPanel}>
                <div className={styles.chatHeader}>Stream chat</div>
                <div className={styles.chatLoginPrompt}>
                    <Link to="/login">Log in</Link> to join the chat.
                </div>
            </div>
        );
    }

    return <StreamChatPanelInner key={streamId} streamId={streamId} user={user} isLive={isLive} />;
}

function StreamChatPanelInner({ streamId, user, isLive }: { streamId: string; user: UserProfile; isLive: boolean }) {
    const { addWSListener } = useNotifications();
    const blockedIDs = useBlockedUserIds();
    const [joined, setJoined] = useState(false);
    const [joinError, setJoinError] = useState(false);
    const [replyingTo, setReplyingTo] = useState<ReplyTarget | null>(null);
    const [editingMessageId, setEditingMessageId] = useState<string | null>(null);
    const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);
    const composerRef = useRef<ChatComposerHandle>(null);

    useEffect(() => {
        if (!isLive) {
            return;
        }
        let cancelled = false;
        joinStreamChat(streamId)
            .then(() => {
                if (!cancelled) {
                    setJoined(true);
                }
            })
            .catch(() => {
                if (!cancelled) {
                    setJoinError(true);
                }
            });
        return () => {
            cancelled = true;
        };
    }, [streamId, isLive]);

    const roomId = isLive && joined ? streamId : undefined;
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
    } = useMessageHistory(roomId, editingMessageId === null ? MAX_LIVE_MESSAGES : undefined);

    const { handleEditMessage } = useChatMessageHandlers({
        user,
        messages,
        setMessages,
        setEditingMessageId,
    });

    useEffect(() => {
        if (!joined || !isLive) {
            return;
        }
        return addWSListener((msg: WSMessage) => {
            if (msg.type === "chat_message") {
                handleIncomingChatMessage(msg.data as ChatMessage, streamId, setMessages, () =>
                    scrollToBottomInstant(),
                );
                return;
            }
            applySharedChatWSBranch(msg, { activeRoomId: streamId, setMessages, noteTyping: () => {} });
        });
    }, [joined, isLive, streamId, addWSListener, setMessages, scrollToBottomInstant]);

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
        <div className={styles.chatPanel}>
            <div className={styles.chatHeader}>Stream chat</div>
            <div className={styles.chatMessages} ref={containerRef} onScroll={handleScroll}>
                <div ref={contentRef} className={styles.chatContent}>
                    {isLive && joinError && <div className={styles.chatNotice}>Couldn't join the chat.</div>}
                    {isLive && !joined && !joinError && <div className={styles.chatNotice}>Joining chat...</div>}
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
            {isLive && joined && (
                <ChatComposer
                    ref={composerRef}
                    roomId={streamId}
                    draftRecipientId={null}
                    onSent={handleSent}
                    replyingTo={replyingTo}
                    onCancelReply={handleCancelReply}
                    sendOnEnter
                    compact
                />
            )}
            {!isLive && <div className={styles.chatEnded}>Chat is closed while the stream is offline.</div>}
            {lightboxSrc && createPortal(<Lightbox src={lightboxSrc} onClose={handleLightboxClose} />, document.body)}
        </div>
    );
}
