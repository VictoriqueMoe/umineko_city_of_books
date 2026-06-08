import { Button } from "../../Button/Button";
import { Input } from "../../Input/Input";
import { Modal } from "../../Modal/Modal";
import { ChatComposer } from "../ChatComposer/ChatComposer";
import { VoiceBar } from "../Voice/VoiceBar";
import { VoiceButton } from "../Voice/VoiceButton";
import { TypingIndicator } from "../TypingIndicator/TypingIndicator";
import { DmMessageList } from "../MessageList/DmMessageList";
import { Lightbox } from "../../Lightbox/Lightbox";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import { isSiteStaff } from "../../../utils/permissions";
import { forceMuteVoiceParticipant } from "../../../api/endpoints";
import { getRoomAvatarUser, getRoomDisplayName, type DmController } from "../../../hooks/useDmController";
import { useChatViewport } from "../../../hooks/useChatViewport";
import styles from "./mobileChat.module.css";

export function MobileDmView({ controller }: { controller: DmController }) {
    const {
        user,
        mobileView,
        rooms,
        activeRoom,
        activeRoomId,
        draftRecipient,
        setDraftRecipient,
        messagesEndRef,
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
        handleRoomSelect,
        handleMobileBack,
        handleSentMessage,
        handleSelectUser,
        handleEditLast,
        handleDeleteChat,
        notifyTyping,
    } = controller;

    useChatViewport({ scrollToBottom });

    if (!user) {
        return null;
    }

    const isSiteMod = isSiteStaff(user.role);

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
                                    <span className={styles.roomName}>{getRoomDisplayName(room, user)}</span>
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
                        <span className={styles.topTitle}>{headerName}</span>
                    )}
                </div>
                {activeRoom ? (
                    <button
                        type="button"
                        className={styles.iconBtn}
                        onClick={handleDeleteChat}
                        aria-label="Delete chat"
                        title="Delete chat"
                    >
                        {"🗑"}
                    </button>
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
                    canModerate={isSiteMod}
                    onForceMute={(id, muted) => {
                        forceMuteVoiceParticipant(activeRoomId ?? "", id, muted).catch(() => {});
                    }}
                />
            )}

            {activeRoom ? (
                <DmMessageList
                    controller={controller}
                    classes={{ messages: styles.messages, loadMoreBar: styles.loadMoreBar }}
                />
            ) : (
                <div className={styles.draftEmpty}>
                    Send your first message to {draftRecipient?.display_name}.
                    <div ref={messagesEndRef} />
                </div>
            )}

            {activeRoom && <TypingIndicator names={typingNames} />}

            <div className={styles.composerWrap}>
                <ChatComposer
                    roomId={activeRoom ? activeRoomId : null}
                    draftRecipientId={activeRoom ? null : (draftRecipient?.id ?? null)}
                    onSent={handleSentMessage}
                    replyingTo={activeRoom ? replyingTo : null}
                    onCancelReply={() => setReplyingTo(null)}
                    onTyping={activeRoom ? notifyTyping : undefined}
                    onEditLast={activeRoom ? handleEditLast : undefined}
                    sendOnEnter={false}
                    compact
                    extraActions={
                        activeRoom ? (
                            <VoiceButton
                                enabled={voiceEnabled}
                                status={voice.status}
                                presenceCount={voice.presenceCount}
                                onJoin={voice.join}
                                onLeave={voice.leave}
                            />
                        ) : undefined
                    }
                />
            </div>

            {lightboxSrc && <Lightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />}
        </div>
    );
}
