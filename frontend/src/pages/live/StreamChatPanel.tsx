import { useEffect, useState } from "react";
import { Link } from "react-router";
import { useAuth } from "../../hooks/useAuth";
import { useNotifications } from "../../hooks/useNotifications";
import { joinStreamChat } from "../../api/endpoints";
import { useMessageHistory } from "../../hooks/useMessageHistory";
import { MessageBubble } from "../../components/chat/MessageBubble/MessageBubble";
import { ChatComposer, type ReplyTarget } from "../../components/chat/ChatComposer/ChatComposer";
import { handleIncomingChatMessage, applySharedChatWSBranch } from "../../utils/chatStream";
import type { ChatMessage, WSMessage } from "../../types/api";
import styles from "./live.module.css";

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

    return <StreamChatPanelInner key={streamId} streamId={streamId} userId={user.id} isLive={isLive} />;
}

function StreamChatPanelInner({ streamId, userId, isLive }: { streamId: string; userId: string; isLive: boolean }) {
    const { addWSListener } = useNotifications();
    const [joined, setJoined] = useState(false);
    const [joinError, setJoinError] = useState(false);
    const [replyingTo, setReplyingTo] = useState<ReplyTarget | null>(null);

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
    const { messages, setMessages, containerRef, endRef, scrollToBottomInstant, handleScroll, addMessage } =
        useMessageHistory(roomId);

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

    function handleSent(message: ChatMessage) {
        addMessage(message);
        scrollToBottomInstant({ force: true });
    }

    function handleReply(message: ChatMessage) {
        setReplyingTo({
            id: message.id,
            senderName: message.sender.display_name || message.sender.username,
            bodyPreview: message.body.slice(0, 140),
        });
    }

    return (
        <div className={styles.chatPanel}>
            <div className={styles.chatHeader}>Stream chat</div>
            <div className={styles.chatMessages} ref={containerRef} onScroll={handleScroll}>
                {isLive && joinError && <div className={styles.chatNotice}>Couldn't join the chat.</div>}
                {isLive && !joined && !joinError && <div className={styles.chatNotice}>Joining chat...</div>}
                {messages.map(m => (
                    <MessageBubble key={m.id} message={m} isOwn={m.sender.id === userId} onReply={handleReply} />
                ))}
                <div ref={endRef} />
            </div>
            {isLive && joined && (
                <ChatComposer
                    roomId={streamId}
                    draftRecipientId={null}
                    onSent={handleSent}
                    replyingTo={replyingTo}
                    onCancelReply={() => setReplyingTo(null)}
                    sendOnEnter
                    compact
                />
            )}
            {!isLive && <div className={styles.chatEnded}>Chat is closed while the stream is offline.</div>}
        </div>
    );
}
