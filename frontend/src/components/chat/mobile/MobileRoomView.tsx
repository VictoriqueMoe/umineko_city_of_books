import { lazy, Suspense, useMemo } from "react";
import { roomViewerPolicy } from "../../../domain/chat/roomPolicy";
import { FORCE_MUTE_FAILED, useForceMuteVoiceParticipant } from "../../../hooks/mutations/chat";
import { errorMessage } from "../../../utils/errorMessage";
import { useChatViewport } from "../../../hooks/useChatViewport";
import type { RoomController } from "../../../hooks/useRoomController";
import { TypingIndicator } from "../TypingIndicator/TypingIndicator";
import { Button } from "../../Button/Button";
import { ChatComposer } from "../ChatComposer/ChatComposer";
import { MessageList } from "../MessageList/MessageList";
import { RoomMemberDialogs } from "../RoomMemberDialogs/RoomMemberDialogs";
import { RoomMemberList } from "../RoomMemberList/RoomMemberList";
import { RoomOverlays } from "../RoomOverlays/RoomOverlays";
import { WatchPartyButton } from "../WatchParty/WatchPartyButton";
const VoiceBar = lazy(() => import("../Voice/VoiceBar").then(m => ({ default: m.VoiceBar })));
import { VoiceButton } from "../Voice/VoiceButton";
import { Lightbox } from "../../Lightbox/Lightbox";
import styles from "./mobileChat.module.css";

const MESSAGE_LIST_CLASSES = {
    messages: styles.messages,
    loadMoreBar: styles.loadMoreBar,
    empty: styles.empty,
};

export function MobileRoomView({ controller }: { controller: RoomController }) {
    const { room, session, members, moderation, prefs, anchor, voice, watchParty, panels, toast } = controller;
    const user = session.viewer;
    const forceMute = useForceMuteVoiceParticipant(room.id);

    useChatViewport({ scrollToBottom: session.toBottom });

    const mentionPool = useMemo(() => members.list.map(m => m.user), [members.list]);
    const existingMemberIds = useMemo(() => new Set(members.list.map(m => m.user.id)), [members.list]);

    if (!user || !room.data) {
        return null;
    }

    const currentRoom = room.data;
    const { isHost, isSystem, isSiteMod, canModerateRoom } = roomViewerPolicy(currentRoom, user);

    const memberActions = {
        editSelf: () => panels.setEditProfileOpen(true),
        openNicknameDialog: moderation.openNicknameDialog,
        unlockNickname: moderation.handleModUnlockNickname,
        kick: moderation.handleKick,
        ban: moderation.handleBan,
        openTimeoutDialog: moderation.openTimeoutDialog,
        clearTimeout: moderation.handleClearTimeout,
    };

    const overlays = (
        <>
            <RoomOverlays
                room={currentRoom}
                viewer={user}
                viewerIsStaff={isSiteMod}
                canModerateRoom={canModerateRoom}
                voiceEnabled={voice.enabled}
                onJump={anchor.jumpTo}
                onLightbox={panels.setLightboxSrc}
                infoPanel={{
                    tab: panels.panelTab,
                    onTabChange: panels.openPanel,
                    onClose: panels.closePanel,
                }}
                editProfile={{
                    open: panels.editProfileOpen,
                    currentMember: members.current,
                    onClose: () => panels.setEditProfileOpen(false),
                    onSaved: updated => {
                        members.set(prev => prev.map(m => (m.user.id === updated.user.id ? updated : m)));
                    },
                }}
                roomModeration={{
                    open: panels.moderationDialogOpen,
                    onClose: () => panels.setModerationDialogOpen(false),
                    onSaved: updated => room.set(updated),
                }}
                watchParty={{
                    active: watchParty.activeSession,
                    screenShareEnabled: watchParty.screenShareEnabled,
                    onClose: () => watchParty.close(),
                    onLeave: watchParty.leave,
                    onEnd: watchParty.end,
                    onTransferControl: watchParty.transferControl,
                    onKick: watchParty.kick,
                    onIdentify: watchParty.identify,
                }}
                invite={{
                    open: panels.inviteModalOpen,
                    existingMemberIds,
                    onClose: () => panels.setInviteModalOpen(false),
                    onInvited: result => {
                        if (result.invited_count > 0) {
                            toast.show(
                                result.invited_count === 1
                                    ? "1 member invited"
                                    : `${result.invited_count} members invited`,
                            );
                        } else if (result.skipped_count > 0) {
                            toast.show("No one invited (all were already members or blocked)");
                        }
                    },
                }}
            />

            <RoomMemberDialogs
                variant="mobile"
                nickname={{
                    target: moderation.nicknameDialogTarget,
                    value: moderation.nicknameDialogValue,
                    error: moderation.nicknameDialogError,
                    saving: moderation.nicknameDialogSaving,
                    onValueChange: moderation.setNicknameDialogValue,
                    onCancel: () => moderation.setNicknameDialogTarget(null),
                    onSave: moderation.handleModSetNickname,
                }}
                timeout={{
                    target: moderation.timeoutDialogTarget,
                    amount: moderation.timeoutDialogAmount,
                    unit: moderation.timeoutDialogUnit,
                    error: moderation.timeoutDialogError,
                    saving: moderation.timeoutDialogSaving,
                    onAmountChange: moderation.setTimeoutDialogAmount,
                    onUnitChange: moderation.setTimeoutDialogUnit,
                    onCancel: () => moderation.setTimeoutDialogTarget(null),
                    onSave: moderation.handleSetTimeout,
                }}
            />

            {watchParty.invitedPartyMissing && (
                <div className={styles.endedPartyNotice}>That watch party has ended.</div>
            )}
            {toast.message && (
                <div dir="auto" className={styles.toast}>
                    {toast.message}
                </div>
            )}
            {panels.lightboxSrc && <Lightbox src={panels.lightboxSrc} onClose={() => panels.setLightboxSrc(null)} />}
        </>
    );

    if (prefs.mobileView === "members") {
        return (
            <div className={styles.shell}>
                <div className={styles.topBar}>
                    <button
                        type="button"
                        className={styles.iconBtn}
                        onClick={() => prefs.setMobileView("chat")}
                        aria-label="Back to chat"
                    >
                        {"←"}
                    </button>
                    <div className={styles.topInfo}>
                        <span className={styles.topTitle}>Members</span>
                        <span className={styles.topMeta}>{members.list.length} members</span>
                    </div>
                    {canModerateRoom && !isSystem && (
                        <button
                            type="button"
                            className={styles.iconBtn}
                            onClick={() => panels.setInviteModalOpen(true)}
                            aria-label="Invite members"
                        >
                            {"+"}
                        </button>
                    )}
                </div>
                <RoomMemberList
                    variant="mobile"
                    groups={members.groups}
                    presence={members.presence}
                    onlineWeight={members.onlineWeight}
                    viewer={{ selfId: user.id, isSystem, isSiteMod, canModerateRoom }}
                    openMemberMenu={moderation.openMemberMenu}
                    setOpenMemberMenu={moderation.setOpenMemberMenu}
                    busy={moderation.busy}
                    formatTimeoutUntil={moderation.formatTimeoutUntil}
                    actions={memberActions}
                />
                <div className={styles.membersFooter}>
                    <Button
                        variant="secondary"
                        size="small"
                        onClick={room.toggleMute}
                        disabled={moderation.busy === "mute"}
                    >
                        {moderation.busy === "mute" ? "..." : currentRoom.viewer_muted ? "Unmute" : "Mute"}
                    </Button>
                    {!isSystem && canModerateRoom && (
                        <Button variant="secondary" size="small" onClick={() => panels.setModerationDialogOpen(true)}>
                            Moderation
                        </Button>
                    )}
                    {!isSystem && canModerateRoom && (
                        <Button
                            variant="danger"
                            size="small"
                            onClick={room.remove}
                            disabled={moderation.busy === "delete"}
                        >
                            {moderation.busy === "delete" ? "Deleting..." : "Delete"}
                        </Button>
                    )}
                    {!isSystem && !isHost && (
                        <Button
                            variant="danger"
                            size="small"
                            onClick={room.leave}
                            disabled={moderation.busy === "self"}
                        >
                            {moderation.busy === "self" ? "Leaving..." : "Leave"}
                        </Button>
                    )}
                </div>
                {overlays}
            </div>
        );
    }

    return (
        <div className={styles.shell}>
            <div className={styles.topBar}>
                <button type="button" className={styles.iconBtn} onClick={room.backToRooms} aria-label="Back to rooms">
                    {"←"}
                </button>
                <div className={styles.topInfo}>
                    <div className={styles.topTitleRow}>
                        <span dir="auto" className={styles.topTitle}>
                            {currentRoom.name}
                        </span>
                        {currentRoom.is_system && <span className={styles.topBadge}>Staff</span>}
                        {currentRoom.is_rp && <span className={styles.topBadge}>RP</span>}
                    </div>
                    <span className={styles.topMeta}>
                        {currentRoom.member_count ?? currentRoom.members.length} members
                        {currentRoom.is_public ? " · public" : " · private"}
                    </span>
                </div>
                <button
                    type="button"
                    className={styles.iconBtn}
                    onClick={() => panels.openPanel("search")}
                    aria-label="Search messages"
                >
                    {"🔍"}
                </button>
                <button
                    type="button"
                    className={styles.iconBtn}
                    onClick={() => panels.openPanel("pins")}
                    aria-label="Pinned messages"
                >
                    {"📌"}
                </button>
                <button
                    type="button"
                    className={styles.iconBtn}
                    onClick={() => prefs.setMobileView("members")}
                    aria-label="Members"
                >
                    {"☰"}
                </button>
            </div>

            {voice.status === "connected" && voice.room && (
                <Suspense fallback={null}>
                    <VoiceBar
                        room={voice.room}
                        onLeave={voice.leave}
                        canModerate={canModerateRoom}
                        onForceMute={(id, muted) => {
                            forceMute.mutate(
                                { userId: id, muted },
                                { onError: err => toast.show(errorMessage(err, FORCE_MUTE_FAILED)) },
                            );
                        }}
                    />
                </Suspense>
            )}

            <MessageList
                viewer={user}
                room={currentRoom}
                messages={session.messages}
                hasMore={session.hasMore}
                loadingMore={session.loadingMore}
                highlightedMessageId={anchor.highlightedMsgId}
                editingMessageId={session.editingMessageId}
                viewerTimedOut={room.viewerTimedOut}
                matchesViewerMention={session.matchesViewerMention}
                containerRef={session.containerRef}
                contentRef={session.contentRef}
                endRef={session.endRef}
                onScroll={session.onScroll}
                onLightbox={panels.setLightboxSrc}
                onReply={session.setReplyingTo}
                onStartEditing={session.startEditing}
                onCancelEditing={session.cancelEditing}
                onToggleReaction={session.toggleReaction}
                onTogglePin={session.togglePin}
                onDelete={session.deleteMessage}
                onEdit={session.editMessage}
                classes={MESSAGE_LIST_CLASSES}
            />

            <TypingIndicator names={session.typingNames} />

            <div className={styles.composerWrap}>
                <ChatComposer
                    roomId={currentRoom.id}
                    draftRecipientId={null}
                    onSent={session.onSent}
                    mentionPool={mentionPool}
                    replyingTo={session.replyingTo}
                    onCancelReply={() => session.setReplyingTo(null)}
                    onTyping={session.notifyTyping}
                    onEditLast={session.editLast}
                    timeoutUntil={room.viewerTimeoutUntil}
                    sendOnEnter={false}
                    compact
                    extraActions={
                        !isSystem && user ? (
                            <>
                                <VoiceButton
                                    enabled={voice.enabled}
                                    status={voice.status}
                                    presenceCount={voice.presenceCount}
                                    error={voice.error}
                                    onJoin={voice.join}
                                    onLeave={voice.leave}
                                />
                                <WatchPartyButton
                                    enabled={watchParty.enabled}
                                    screenShareEnabled={watchParty.screenShareEnabled}
                                    sessions={watchParty.sessions}
                                    activeSessionId={watchParty.openSessionId}
                                    viewerUserId={user.id}
                                    error={watchParty.error}
                                    onStart={opts => watchParty.start(opts)}
                                    onJoin={sid => watchParty.join(sid)}
                                    onOpenExisting={sid => watchParty.openExisting(sid)}
                                />
                            </>
                        ) : null
                    }
                />
            </div>

            {overlays}
        </div>
    );
}
