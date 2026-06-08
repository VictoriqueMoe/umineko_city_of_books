import { Button } from "../../components/Button/Button";
import { Input } from "../../components/Input/Input";
import { Modal } from "../../components/Modal/Modal";
import { ChatComposer } from "../../components/chat/ChatComposer/ChatComposer";
import { VoiceBar } from "../../components/chat/Voice/VoiceBar";
import { VoiceButton } from "../../components/chat/Voice/VoiceButton";
import { TypingIndicator } from "../../components/chat/TypingIndicator/TypingIndicator";
import { DmMessageList } from "../../components/chat/MessageList/DmMessageList";
import { Lightbox } from "../../components/Lightbox/Lightbox";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { isSiteStaff } from "../../utils/permissions";
import { forceMuteVoiceParticipant } from "../../api/endpoints";
import { getRoomAvatarUser, getRoomDisplayName, useDmController } from "../../hooks/useDmController";
import { useIsMobile } from "../../hooks/useIsMobile";
import { MobileDmView } from "../../components/chat/mobile/MobileDmView";
import styles from "./ChatPage.module.css";

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
        draftRecipient,
        setDraftRecipient,
        messagesEndRef,
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
        handleRoomSelect,
        handleMobileBack,
        handleSentMessage,
        handleSelectUser,
        handleEditLast,
        handleDeleteChat,
        notifyTyping,
    } = controller;

    if (!user) {
        return null;
    }

    if (loading) {
        return <div className={styles.keysLoading}>Loading chat...</div>;
    }

    if (isMobile) {
        return <MobileDmView controller={controller} />;
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
                                        <span className={styles.roomName}>{getRoomDisplayName(room, user)}</span>
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
                                    Send your first message to {draftRecipient.display_name}.
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
                                        <span>{getRoomDisplayName(activeRoom, user)}</span>
                                    )}
                                </div>
                                <Button variant="danger" size="small" onClick={handleDeleteChat}>
                                    Delete Chat
                                </Button>
                            </div>
                            {voice.status === "connected" && voice.room && (
                                <VoiceBar
                                    room={voice.room}
                                    onLeave={voice.leave}
                                    canModerate={user ? isSiteStaff(user.role) : false}
                                    onForceMute={(id, muted) => {
                                        forceMuteVoiceParticipant(activeRoomId ?? "", id, muted).catch(() => {});
                                    }}
                                />
                            )}
                            <DmMessageList
                                controller={controller}
                                classes={{ messages: styles.messages, loadMoreBar: styles.loadMoreBar }}
                            />
                            <TypingIndicator names={typingNames} />
                            <ChatComposer
                                roomId={activeRoomId}
                                draftRecipientId={null}
                                onSent={handleSentMessage}
                                replyingTo={replyingTo}
                                onCancelReply={() => setReplyingTo(null)}
                                onTyping={notifyTyping}
                                onEditLast={handleEditLast}
                                extraActions={
                                    <VoiceButton
                                        enabled={voiceEnabled}
                                        status={voice.status}
                                        presenceCount={voice.presenceCount}
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
            {lightboxSrc && <Lightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />}
        </div>
    );
}
