import { isSiteStaff } from "../../../utils/permissions";
import type { RoomController } from "../../../hooks/useRoomController";
import { useBlockedUserIds } from "../../../hooks/useBlockedUserIds";
import { MessageBubble } from "../MessageBubble/MessageBubble";

interface RoomMessageListClasses {
    messages: string;
    loadMoreBar: string;
    empty: string;
}

interface RoomMessageListProps {
    controller: RoomController;
    classes: RoomMessageListClasses;
}

export function RoomMessageList({ controller, classes }: RoomMessageListProps) {
    const blockedIDs = useBlockedUserIds();
    const {
        user,
        room,
        messages,
        hasMore,
        loadingMore,
        messagesContainerRef,
        messagesContentRef,
        messagesEndRef,
        handleMessagesScroll,
        highlightedMsgId,
        matchesViewerMention,
        viewerTimedOut,
        setLightboxSrc,
        setReplyingTo,
        editingMessageId,
        setEditingMessageId,
        handleReactionToggle,
        handlePinToggle,
        handleDeleteMessage,
        handleEditMessage,
    } = controller;

    if (!user || !room) {
        return null;
    }

    const isHost = room.viewer_role === "host";
    const isSiteMod = isSiteStaff(user.role);
    const canModerateRoom = isHost || isSiteMod;

    return (
        <div className={classes.messages} ref={messagesContainerRef} onScroll={handleMessagesScroll}>
            {messages.length === 0 && !hasMore && <div className={classes.empty}>No messages yet. Say hello!</div>}
            <div ref={messagesContentRef} style={{ display: "flex", flexDirection: "column", gap: "inherit" }}>
                {hasMore && (
                    <div className={classes.loadMoreBar}>
                        {loadingMore ? "Loading older messages..." : "Scroll up for more"}
                    </div>
                )}
                {messages.map(msg => (
                    <MessageBubble
                        key={msg.id}
                        message={msg}
                        isOwn={msg.sender.id === user.id}
                        senderBlocked={blockedIDs.has(msg.sender.id)}
                        highlighted={msg.id === highlightedMsgId}
                        notifiesViewer={
                            msg.reply_to?.sender_id === user.id ||
                            (matchesViewerMention ? matchesViewerMention(msg.body) : false)
                        }
                        onLightbox={setLightboxSrc}
                        onReply={m =>
                            setReplyingTo({
                                id: m.id,
                                senderName: m.sender.display_name,
                                bodyPreview: m.body.length > 80 ? m.body.slice(0, 80) + "..." : m.body,
                            })
                        }
                        onReactionToggle={handleReactionToggle}
                        onPinToggle={canModerateRoom ? handlePinToggle : undefined}
                        onDelete={handleDeleteMessage}
                        onEdit={handleEditMessage}
                        onEditStart={m => setEditingMessageId(m.id)}
                        onEditCancel={() => setEditingMessageId(null)}
                        editing={editingMessageId === msg.id}
                        canPin={canModerateRoom}
                        canModerate={canModerateRoom}
                        canReact={!viewerTimedOut}
                        canEdit={!viewerTimedOut}
                        senderIsStaff={isSiteStaff(msg.sender.role)}
                    />
                ))}
                <div ref={messagesEndRef} />
            </div>
        </div>
    );
}
