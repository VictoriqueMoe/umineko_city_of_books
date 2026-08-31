import { Button } from "../../Button/Button";
import { Input } from "../../Input/Input";
import { Modal } from "../../Modal/Modal";
import { ChatComposer } from "../ChatComposer/ChatComposer";
import { VoiceBar } from "../Voice/VoiceBar";
import { VoiceButton } from "../Voice/VoiceButton";
import { TypingIndicator } from "../TypingIndicator/TypingIndicator";
import { MessageList } from "../MessageList/MessageList";
import { MessageSearchPanel } from "../MessageSearchPanel/MessageSearchPanel";
import { Lightbox } from "../../Lightbox/Lightbox";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { getRoomAvatarUser, getRoomDisplayName } from "../../../domain/chat/dm";
import { FORCE_MUTE_FAILED, useForceMuteVoiceParticipant } from "../../../hooks/mutations/chat";
import { errorMessage } from "../../../utils/errorMessage";
import { type MessageAnchor } from "../../../hooks/chat/useMessageAnchor";
import { type DmController } from "../../../hooks/useDmController";
import { useChatViewport } from "../../../hooks/useChatViewport";
import type { User } from "../../../types/api";
import styles from "./mobileChat.module.css";

const DM_MESSAGE_LIST_CLASSES = {
    messages: styles.messages,
    loadMoreBar: styles.loadMoreBar,
    empty: styles.empty,
};

export interface DmThreadView {
    anchor: MessageAnchor;
    mentionPool: User[];
    viewerTimeoutUntil?: string;
    viewerTimedOut: boolean;
    searchOpen: boolean;
    setSearchOpen: (open: boolean) => void;
}

export function MobileDmView({ controller, thread }: { controller: DmController; thread: DmThreadView }) {
    const {
        user,
        mobileView,
        rooms,
        activeRoom,
        activeRoomId,
        capabilities,
        draftRecipient,
        setDraftRecipient,
        messages,
        hasMore,
        loadingMore,
        messagesContainerRef,
        messagesContentRef,
        messagesEndRef,
        handleDmScroll,
        readReceipts,
        matchesViewerMention,
        editingMessageId,
        startEditing,
        cancelEditing,
        handleDeleteMessage,
        handleEditMessage,
        handleReactionToggle,
        scrollToBottom,
        typingNames,
        voice,
        voiceEnabled,
        replyingTo,
        setReplyingTo,
        lightboxSrc,
        setLightboxSrc,
        showNewDm,
        setShowNewDm,
        dmSearch,
        setDmSearch,
        dmResults,
        dmMutuals,
        dmError,
        dmCreating,
        toast,
        showToast,
        handleRoomSelect,
        handleMobileBack,
        handleSentMessage,
        handleSelectUser,
        handleEditLast,
        handleDeleteChat,
        handleToggleMute,
        mutePending,
        notifyTyping,
    } = controller;
    const forceMute = useForceMuteVoiceParticipant(activeRoomId);

    useChatViewport({ scrollToBottom });

    if (!user) {
        return null;
    }

    const newDmModal = (
        <Modal isOpen={showNewDm} onClose={() => setShowNewDm(false)} title="New Direct Message">
            <div className={styles.dialog}>
                <Input
                    fullWidth
                    type="text"
                    placeholder="Search users..."
                    value={dmSearch}
                    onChange={e => setDmSearch(e.target.value)}
                />
                {dmError && <div className={styles.dialogError}>{dmError}</div>}
                <div className={styles.roomList}>
                    {dmSearch.trim()
                        ? dmResults.map(u => (
                              <button
                                  key={u.id}
                                  className={styles.roomItem}
                                  onClick={() => handleSelectUser(u)}
                                  disabled={dmCreating}
                              >
                                  <ProfileLink user={u} size="small" clickable={false} />
                              </button>
                          ))
                        : dmMutuals.map(u => (
                              <button
                                  key={u.id}
                                  className={styles.roomItem}
                                  onClick={() => handleSelectUser(u)}
                                  disabled={dmCreating}
                              >
                                  <ProfileLink user={u} size="small" clickable={false} />
                              </button>
                          ))}
                </div>
            </div>
        </Modal>
    );

    if (mobileView === "list") {
        return (
            <div className={styles.listScreen}>
                <div className={styles.listHeader}>
                    <span className={styles.listTitle}>Messages</span>
                    <Button variant="ghost" size="small" onClick={() => setShowNewDm(true)}>
                        New DM
                    </Button>
                </div>
                <div className={styles.roomList}>
                    {rooms.length === 0 && <div className={styles.emptyRooms}>No conversations yet</div>}
                    {rooms.map(room => {
                        const avatarUser = getRoomAvatarUser(room, user);
                        return (
                            <button key={room.id} className={styles.roomItem} onClick={() => handleRoomSelect(room.id)}>
                                {avatarUser ? (
                                    <ProfileLink user={avatarUser} size="small" clickable={false} />
                                ) : (
                                    <span dir="auto" className={styles.roomName}>
                                        {getRoomDisplayName(room, user)}
                                    </span>
                                )}
                                {room.unread && <span className={styles.unreadDot} aria-label="unread" />}
                            </button>
                        );
                    })}
                </div>
                {newDmModal}
            </div>
        );
    }

    const headerUser = activeRoom ? getRoomAvatarUser(activeRoom, user) : draftRecipient;
    const headerName = activeRoom ? getRoomDisplayName(activeRoom, user) : (draftRecipient?.display_name ?? "");

    return (
        <div className={styles.shell}>
            <div className={styles.topBar}>
                <button
                    type="button"
                    className={styles.iconBtn}
                    onClick={handleMobileBack}
                    aria-label="Back to conversations"
                >
                    {"←"}
                </button>
                <div className={styles.topInfo}>
                    {headerUser ? (
                        <ProfileLink user={headerUser} size="small" />
                    ) : (
                        <span dir="auto" className={styles.topTitle}>
                            {headerName}
                        </span>
                    )}
                </div>
                {activeRoom ? (
                    <>
                        <button
                            type="button"
                            className={styles.iconBtn}
                            onClick={() => thread.setSearchOpen(true)}
                            aria-label="Search messages"
                            title="Search messages"
                        >
                            {"🔍"}
                        </button>
                        <button
                            type="button"
                            className={styles.iconBtn}
                            onClick={handleToggleMute}
                            disabled={mutePending}
                            aria-label={activeRoom.viewer_muted ? "Unmute notifications" : "Mute notifications"}
                            title={activeRoom.viewer_muted ? "Unmute notifications" : "Mute notifications"}
                        >
                            {activeRoom.viewer_muted ? "🔕" : "🔔"}
                        </button>
                        <button
                            type="button"
                            className={styles.iconBtn}
                            onClick={handleDeleteChat}
                            aria-label="Delete chat"
                            title="Delete chat"
                        >
                            {"🗑"}
                        </button>
                    </>
                ) : (
                    <button
                        type="button"
                        className={styles.iconBtn}
                        onClick={() => setDraftRecipient(null)}
                        aria-label="Cancel"
                    >
                        {"×"}
                    </button>
                )}
            </div>

            {voice.status === "connected" && voice.room && (
                <VoiceBar
                    room={voice.room}
                    onLeave={voice.leave}
                    canModerate={capabilities.canModerateRoom}
                    onForceMute={(id, muted) => {
                        forceMute.mutate(
                            { userId: id, muted },
                            { onError: err => showToast(errorMessage(err, FORCE_MUTE_FAILED)) },
                        );
                    }}
                />
            )}

            {activeRoom ? (
                <MessageList
                    viewer={user}
                    room={activeRoom}
                    messages={messages}
                    hasMore={hasMore}
                    loadingMore={loadingMore}
                    highlightedMessageId={thread.anchor.highlightedMsgId}
                    editingMessageId={editingMessageId}
                    viewerTimedOut={thread.viewerTimedOut}
                    readReceipts={readReceipts}
                    matchesViewerMention={matchesViewerMention}
                    containerRef={messagesContainerRef}
                    contentRef={messagesContentRef}
                    endRef={messagesEndRef}
                    onScroll={handleDmScroll}
                    onLightbox={setLightboxSrc}
                    onReply={setReplyingTo}
                    onStartEditing={startEditing}
                    onCancelEditing={cancelEditing}
                    onToggleReaction={handleReactionToggle}
                    onDelete={handleDeleteMessage}
                    onEdit={handleEditMessage}
                    classes={DM_MESSAGE_LIST_CLASSES}
                />
            ) : (
                <div className={styles.draftEmpty}>
                    <span>
                        Send your first message to <bdi>{draftRecipient?.display_name}</bdi>.
                    </span>
                    <div ref={messagesEndRef} />
                </div>
            )}

            {activeRoom && <TypingIndicator names={typingNames} />}

            <div className={styles.composerWrap}>
                <ChatComposer
                    roomId={activeRoom ? activeRoomId : null}
                    draftRecipientId={activeRoom ? null : (draftRecipient?.id ?? null)}
                    onSent={handleSentMessage}
                    mentionPool={activeRoom ? thread.mentionPool : undefined}
                    replyingTo={activeRoom ? replyingTo : null}
                    onCancelReply={() => setReplyingTo(null)}
                    onTyping={activeRoom ? notifyTyping : undefined}
                    onEditLast={activeRoom ? handleEditLast : undefined}
                    timeoutUntil={activeRoom ? thread.viewerTimeoutUntil : undefined}
                    sendOnEnter={false}
                    compact
                    extraActions={
                        activeRoom ? (
                            <VoiceButton
                                enabled={voiceEnabled}
                                status={voice.status}
                                presenceCount={voice.presenceCount}
                                error={voice.error}
                                onJoin={voice.join}
                                onLeave={voice.leave}
                            />
                        ) : undefined
                    }
                />
            </div>

            {activeRoom && thread.searchOpen && (
                <MessageSearchPanel
                    roomId={activeRoom.id}
                    isOpen={thread.searchOpen}
                    onClose={() => thread.setSearchOpen(false)}
                    onJump={thread.anchor.handleJumpToMessage}
                />
            )}

            {toast && (
                <div dir="auto" className={styles.toast}>
                    {toast}
                </div>
            )}
            {lightboxSrc && <Lightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />}
        </div>
    );
}
