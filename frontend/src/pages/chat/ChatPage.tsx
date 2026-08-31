import { useEffect, useState } from "react";
import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Modal } from "../../components/Modal/Modal";
import { ChatComposer } from "../../components/chat/ChatComposer/ChatComposer";
import { VoiceBar } from "../../components/chat/Voice/VoiceBar";
import { VoiceButton } from "../../components/chat/Voice/VoiceButton";
import { TypingIndicator } from "../../components/chat/TypingIndicator/TypingIndicator";
import { MessageList } from "../../components/chat/MessageList/MessageList";
import { MessageSearchPanel } from "../../components/chat/MessageSearchPanel/MessageSearchPanel";
import { Lightbox } from "../../components/Lightbox/Lightbox";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { getRoomAvatarUser, getRoomDisplayName } from "../../domain/chat/dm";
import { isTimeoutActive } from "../../domain/chat/roomPolicy";
import { FORCE_MUTE_FAILED, useForceMuteVoiceParticipant } from "../../hooks/mutations/chat";
import { errorMessage } from "../../utils/errorMessage";
import { useMessageAnchor } from "../../hooks/chat/useMessageAnchor";
import { useRoomMembers } from "../../hooks/chat/useRoomMembers";
import { useDmController } from "../../hooks/useDmController";
import { useIsMobile } from "../../hooks/useIsMobile";
import { MobileDmView, type DmThreadView } from "../../components/chat/mobile/MobileDmView";
import type { User } from "../../types/api";
import styles from "./ChatPage.module.css";

const MESSAGE_LIST_CLASSES = {
    messages: styles.messages,
    loadMoreBar: styles.loadMoreBar,
    empty: styles.messageAreaEmpty,
};

const NO_MENTION_POOL: User[] = [];
const TIMEOUT_TICK_MS = 30_000;

function ignoreMemberCount(): void {}

export function ChatPage() {
    const controller = useDmController();
    const isMobile = useIsMobile();
    const {
        user,
        loading,
        mobileView,
        rooms,
        activeRoomId,
        activeRoom,
        capabilities,
        draftRecipient,
        setDraftRecipient,
        messages,
        hasMore,
        loadingMore,
        loadUntilMessage,
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

    const [searchOpen, setSearchOpen] = useState(false);
    const anchor = useMessageAnchor({ messages, loadUntilMessage, onError: showToast });

    const { currentMember } = useRoomMembers({
        roomId: activeRoomId ?? undefined,
        enabled: !!activeRoomId,
        viewerId: user?.id,
        voiceParticipantIds: voice.participantIds,
        onMemberCountDelta: ignoreMemberCount,
    });
    const viewerTimeoutUntil = currentMember?.timeout_until;

    const [nowTick, setNowTick] = useState(() => Date.now());
    useEffect(() => {
        if (!viewerTimeoutUntil) {
            return;
        }

        const reseed = setTimeout(() => setNowTick(Date.now()), 0);
        const tick = setInterval(() => setNowTick(Date.now()), TIMEOUT_TICK_MS);

        return () => {
            clearTimeout(reseed);
            clearInterval(tick);
        };
    }, [viewerTimeoutUntil]);

    const mentionPool = activeRoom?.members ?? NO_MENTION_POOL;
    const viewerTimedOut = isTimeoutActive(viewerTimeoutUntil, nowTick);

    const thread: DmThreadView = {
        anchor,
        mentionPool,
        viewerTimeoutUntil,
        viewerTimedOut,
        searchOpen,
        setSearchOpen,
    };

    if (!user) {
        return null;
    }

    if (loading) {
        return <div className={styles.keysLoading}>Loading chat...</div>;
    }

    if (isMobile) {
        return <MobileDmView controller={controller} thread={thread} />;
    }

    return (
        <div className={styles.chatWrapper}>
            <div className={styles.chatLayout} data-mobile-view={mobileView}>
                <div className={styles.roomList}>
                    <div className={styles.roomListHeader}>
                        <span className={styles.roomListTitle}>Messages</span>
                        <Button variant="ghost" size="small" onClick={() => setShowNewDm(true)}>
                            New DM
                        </Button>
                    </div>
                    <div className={styles.rooms}>
                        {rooms.length === 0 && <div className={styles.emptyRooms}>No conversations yet</div>}
                        {rooms.map(room => {
                            const avatarUser = getRoomAvatarUser(room, user);
                            return (
                                <button
                                    key={room.id}
                                    className={`${styles.roomItem}${room.id === activeRoomId ? ` ${styles.roomItemActive}` : ""}`}
                                    onClick={() => handleRoomSelect(room.id)}
                                >
                                    {avatarUser ? (
                                        <ProfileLink user={avatarUser} size="small" />
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
                </div>

                <div className={styles.messageArea}>
                    {!activeRoom && draftRecipient ? (
                        <>
                            <div className={styles.messageHeader}>
                                <div className={styles.messageHeaderLeft}>
                                    <button
                                        type="button"
                                        className={styles.backButton}
                                        onClick={handleMobileBack}
                                        aria-label="Back to conversations"
                                    >
                                        {"←"}
                                    </button>
                                    <ProfileLink user={draftRecipient} size="small" />
                                </div>
                                <Button variant="ghost" size="small" onClick={() => setDraftRecipient(null)}>
                                    Cancel
                                </Button>
                            </div>
                            <div className={styles.messages}>
                                <div className={styles.messageAreaEmpty}>
                                    <span>
                                        Send your first message to <bdi>{draftRecipient.display_name}</bdi>.
                                    </span>
                                </div>
                                <div ref={messagesEndRef} />
                            </div>
                            <ChatComposer
                                roomId={null}
                                draftRecipientId={draftRecipient.id}
                                onSent={handleSentMessage}
                            />
                        </>
                    ) : !activeRoom ? (
                        <div className={styles.messageAreaEmpty}>Select a conversation</div>
                    ) : (
                        <>
                            <div className={styles.messageHeader}>
                                <div className={styles.messageHeaderLeft}>
                                    <button
                                        type="button"
                                        className={styles.backButton}
                                        onClick={handleMobileBack}
                                        aria-label="Back to conversations"
                                    >
                                        {"←"}
                                    </button>
                                    {getRoomAvatarUser(activeRoom, user) ? (
                                        <ProfileLink user={getRoomAvatarUser(activeRoom, user)!} size="small" />
                                    ) : (
                                        <span dir="auto">{getRoomDisplayName(activeRoom, user)}</span>
                                    )}
                                </div>
                                <div className={styles.headerActions}>
                                    <Button
                                        variant="ghost"
                                        size="small"
                                        onClick={() => setSearchOpen(true)}
                                        aria-label="Search messages"
                                        title="Search messages"
                                    >
                                        {"🔍"}
                                    </Button>
                                    <Button
                                        variant="secondary"
                                        size="small"
                                        onClick={handleToggleMute}
                                        disabled={mutePending}
                                        title={activeRoom.viewer_muted ? "Unmute notifications" : "Mute notifications"}
                                    >
                                        {mutePending ? "..." : activeRoom.viewer_muted ? "Unmute" : "Mute"}
                                    </Button>
                                    <Button variant="danger" size="small" onClick={handleDeleteChat}>
                                        Delete Chat
                                    </Button>
                                </div>
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
                            <MessageList
                                viewer={user}
                                room={activeRoom}
                                messages={messages}
                                hasMore={hasMore}
                                loadingMore={loadingMore}
                                highlightedMessageId={anchor.highlightedMsgId}
                                editingMessageId={editingMessageId}
                                viewerTimedOut={viewerTimedOut}
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
                                classes={MESSAGE_LIST_CLASSES}
                            />
                            <TypingIndicator names={typingNames} />
                            <ChatComposer
                                roomId={activeRoomId}
                                draftRecipientId={null}
                                onSent={handleSentMessage}
                                mentionPool={mentionPool}
                                replyingTo={replyingTo}
                                onCancelReply={() => setReplyingTo(null)}
                                onTyping={notifyTyping}
                                onEditLast={handleEditLast}
                                timeoutUntil={viewerTimeoutUntil}
                                extraActions={
                                    <VoiceButton
                                        enabled={voiceEnabled}
                                        status={voice.status}
                                        presenceCount={voice.presenceCount}
                                        error={voice.error}
                                        onJoin={voice.join}
                                        onLeave={voice.leave}
                                    />
                                }
                            />
                        </>
                    )}
                </div>

                <Modal isOpen={showNewDm} onClose={() => setShowNewDm(false)} title="New Direct Message">
                    <div className={styles.modalBody}>
                        <Input
                            fullWidth
                            type="text"
                            placeholder="Search users..."
                            value={dmSearch}
                            onChange={e => setDmSearch(e.target.value)}
                        />
                        {dmError && <div className={styles.modalError}>{dmError}</div>}

                        <div className={styles.userList}>
                            {dmSearch.trim() ? (
                                dmResults.length === 0 ? (
                                    <div className={styles.emptyRooms}>No users found</div>
                                ) : (
                                    dmResults.map(u => (
                                        <button
                                            key={u.id}
                                            className={styles.userOption}
                                            onClick={() => handleSelectUser(u)}
                                            disabled={dmCreating}
                                        >
                                            <ProfileLink user={u} size="small" clickable={false} />
                                        </button>
                                    ))
                                )
                            ) : (
                                <>
                                    {dmMutuals.length > 0 && (
                                        <div className={styles.mutualsLabel}>Mutual followers</div>
                                    )}
                                    {dmMutuals.map(u => (
                                        <button
                                            key={u.id}
                                            className={styles.userOption}
                                            onClick={() => handleSelectUser(u)}
                                            disabled={dmCreating}
                                        >
                                            <ProfileLink user={u} size="small" clickable={false} />
                                        </button>
                                    ))}
                                    {dmMutuals.length === 0 && (
                                        <div className={styles.emptyRooms}>
                                            Search for a user to start a conversation
                                        </div>
                                    )}
                                </>
                            )}
                        </div>
                    </div>
                </Modal>
            </div>
            {activeRoom && searchOpen && (
                <MessageSearchPanel
                    roomId={activeRoom.id}
                    isOpen={searchOpen}
                    onClose={() => setSearchOpen(false)}
                    onJump={anchor.handleJumpToMessage}
                />
            )}
            {toast && <div className={styles.toast}>{toast}</div>}
            {lightboxSrc && <Lightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />}
        </div>
    );
}
