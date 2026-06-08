import { isSiteStaff } from "../../../utils/permissions";
import { renderSeenLabel, type DmController } from "../../../hooks/useDmController";
import { MessageBubble } from "../MessageBubble/MessageBubble";

interface DmMessageListClasses {
    messages: string;
    loadMoreBar: string;
}

interface DmMessageListProps {
    controller: DmController;
    classes: DmMessageListClasses;
}

export function DmMessageList({ controller, classes }: DmMessageListProps) {
    const {
        user,
        activeRoom,
        messages,
        hasMore,
        loadingMore,
        messagesContainerRef,
        messagesEndRef,
        handleDmScroll,
        readReceipts,
        matchesViewerMention,
        setLightboxSrc,
        setReplyingTo,
        handleDeleteMessage,
        handleEditMessage,
        editingMessageId,
        setEditingMessageId,
    } = controller;

    if (!user || !activeRoom) {
        return null;
    }

    const isSiteMod = isSiteStaff(user.role);

    return (
        <div className={classes.messages} ref={messagesContainerRef} onScroll={handleDmScroll}>
            {hasMore && (
                <div className={classes.loadMoreBar}>
                    {loadingMore ? "Loading older messages..." : "Scroll up for more"}
                </div>
            )}
            {messages.map((msg, idx) => {
                const isOwn = msg.sender.id === user.id;
                const seenLabel = isOwn ? renderSeenLabel(msg, idx, messages, activeRoom, user.id, readReceipts) : null;
                return (
                    <MessageBubble
                        key={msg.id}
                        message={msg}
                        isOwn={isOwn}
                        notifiesViewer={
                            msg.reply_to?.sender_id === user.id ||
                            (matchesViewerMention ? matchesViewerMention(msg.body) : false)
                        }
                        seenLabel={seenLabel}
                        onLightbox={setLightboxSrc}
                        onReply={m =>
                            setReplyingTo({
                                id: m.id,
                                senderName: m.sender.display_name,
                                bodyPreview: m.body.length > 80 ? m.body.slice(0, 80) + "..." : m.body,
                            })
                        }
                        onDelete={handleDeleteMessage}
                        onEdit={handleEditMessage}
                        onEditStart={m => setEditingMessageId(m.id)}
                        onEditCancel={() => setEditingMessageId(null)}
                        editing={editingMessageId === msg.id}
                        canModerate={isSiteMod}
                        senderIsStaff={isSiteStaff(msg.sender.role)}
                    />
                );
            })}
            <div ref={messagesEndRef} />
        </div>
    );
}
