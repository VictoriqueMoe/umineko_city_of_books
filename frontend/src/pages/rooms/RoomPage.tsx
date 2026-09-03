import { lazy, Suspense, useMemo } from "react";
import { useRoomController } from "../../hooks/useRoomController";
import { useIsMobile } from "../../hooks/useIsMobile";
import { MobileRoomView } from "../../components/chat/mobile/MobileRoomView";
import { TypingIndicator } from "../../components/chat/TypingIndicator/TypingIndicator";
import { Button } from "../../components/Button/Button";
import { ChatComposer } from "../../components/chat/ChatComposer/ChatComposer";
import { MessageList } from "../../components/chat/MessageList/MessageList";
import { RoomMemberDialogs } from "../../components/chat/RoomMemberDialogs/RoomMemberDialogs";
import { RoomMemberList } from "../../components/chat/RoomMemberList/RoomMemberList";
import { RoomOverlays } from "../../components/chat/RoomOverlays/RoomOverlays";
import { WatchPartyButton } from "../../components/chat/WatchParty/WatchPartyButton";
const VoiceBar = lazy(() => import("../../components/chat/Voice/VoiceBar").then(m => ({ default: m.VoiceBar })));
import { VoiceButton } from "../../components/chat/Voice/VoiceButton";
import { Lightbox } from "../../components/Lightbox/Lightbox";
import { FORCE_MUTE_FAILED, useForceMuteVoiceParticipant } from "../../hooks/mutations/chat";
import { errorMessage } from "../../utils/errorMessage";
import styles from "./RoomPage.module.css";

const MESSAGE_LIST_CLASSES = {
    messages: styles.messages,
    loadMoreBar: styles.loadMoreBar,
    empty: styles.messagesEmpty,
};

export function RoomPage() {
    const controller = useRoomController();
    const isMobile = useIsMobile();
    const { room, session, members, moderation, capabilities, prefs, anchor, voice, watchParty, panels, toast } =
        controller;
    const user = session.viewer;
    const forceMute = useForceMuteVoiceParticipant(room.id);

    const mentionPool = useMemo(() => members.list.map(m => m.user), [members.list]);
    const existingMemberIds = useMemo(() => new Set(members.list.map(m => m.user.id)), [members.list]);

    if (!user) {
        return null;
    }

    if (room.loading) {
        return <div className="loading">Loading room...</div>;
    }

    if (!room.data) {
        return (
            <div className={styles.notMember}>
                <p>You're not a member of this room.</p>
                {room.id && (
                    <Button variant="primary" size="small" onClick={room.join} disabled={room.joining}>
                        {room.joining ? "Joining..." : "Try to Join"}
                    </Button>
                )}
                <Button variant="ghost" size="small" onClick={room.backToRooms}>
                    Back to Rooms
                </Button>
                {toast.message && <div className={styles.toast}>{toast.message}</div>}
            </div>
        );
    }

    if (isMobile) {
        return <MobileRoomView controller={controller} />;
    }

    const currentRoom = room.data;
    const { isHost, isSystem, isSiteMod, canModerateRoom, canInvite, canEditSettings, canDestroyRoom } = capabilities;

    const memberActions = {
        editSelf: () => panels.setEditProfileOpen(true),
        openNicknameDialog: moderation.openNicknameDialog,
        unlockNickname: moderation.handleModUnlockNickname,
        kick: moderation.handleKick,
        ban: moderation.handleBan,
        openTimeoutDialog: moderation.openTimeoutDialog,
        clearTimeout: moderation.handleClearTimeout,
    };

    return (
        <div className={styles.roomWrapper}>
            <div
                className={styles.roomLayout}
                data-mobile-view={prefs.mobileView}
                data-sidebar-collapsed={prefs.sidebarCollapsed ? "true" : "false"}
            >
                <aside className={styles.sidebar}>
                    <div className={styles.sidebarHeader}>
                        <button
                            type="button"
                            className={styles.backButton}
                            onClick={() => {
                                if (prefs.mobileView === "members") {
                                    prefs.setMobileView("chat");
                                } else {
                                    room.backToRooms();
                                }
                            }}
                            aria-label={prefs.mobileView === "members" ? "Back to chat" : "Back to rooms"}
                        >
                            {"←"}
                        </button>
                        <span className={styles.sidebarTitle}>Members</span>
                        <span className={styles.memberCount}>{members.list.length}</span>
                        {canInvite && (
                            <button
                                type="button"
                                className={styles.inviteButton}
                                onClick={() => panels.setInviteModalOpen(true)}
                            >
                                + Invite
                            </button>
                        )}
                        <button
                            type="button"
                            className={styles.sidebarCollapseBtn}
                            onClick={prefs.toggleSidebar}
                            aria-label="Hide members"
                            data-tooltip="Hide members"
                        >
                            {"◀"}
                        </button>
                    </div>
                    <RoomMemberList
                        variant="desktop"
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
                    <div className={styles.sidebarFooter}>
                        <Button
                            variant="secondary"
                            size="small"
                            onClick={room.toggleMute}
                            disabled={moderation.busy === "mute"}
                            title={currentRoom.viewer_muted ? "Unmute notifications" : "Mute notifications"}
                        >
                            {moderation.busy === "mute"
                                ? "..."
                                : currentRoom.viewer_muted
                                  ? "Unmute notifications"
                                  : "Mute notifications"}
                        </Button>
                        {canEditSettings && (
                            <Button
                                variant="secondary"
                                size="small"
                                onClick={() => panels.setModerationDialogOpen(true)}
                            >
                                Moderation
                            </Button>
                        )}
                        {canDestroyRoom && (
                            <Button
                                variant="danger"
                                size="small"
                                onClick={room.remove}
                                disabled={moderation.busy === "delete"}
                            >
                                {moderation.busy === "delete" ? "Deleting..." : "Delete Room"}
                            </Button>
                        )}
                        {!isSystem && !isHost && (
                            <Button
                                variant="danger"
                                size="small"
                                onClick={room.leave}
                                disabled={moderation.busy === "self"}
                            >
                                {moderation.busy === "self" ? "Leaving..." : "Leave Room"}
                            </Button>
                        )}
                    </div>
                </aside>

                {prefs.sidebarCollapsed && (
                    <button
                        type="button"
                        className={styles.sidebarExpandRail}
                        onClick={prefs.toggleSidebar}
                        aria-label="Show members"
                        title="Show members"
                    >
                        {"▶"}
                    </button>
                )}
                <div className={styles.messageArea}>
                    <div className={styles.roomHeader}>
                        <button
                            type="button"
                            className={styles.mobileMembersBtn}
                            onClick={() => prefs.setMobileView("members")}
                            aria-label="Members"
                        >
                            {"☰"}
                        </button>
                        <div className={styles.roomHeaderInfo}>
                            <div className={styles.roomTitleRow}>
                                <span dir="auto" className={styles.roomTitle}>
                                    {currentRoom.name}
                                </span>
                                {currentRoom.is_system && <span className={styles.rpBadge}>Staff</span>}
                                {currentRoom.is_rp && <span className={styles.rpBadge}>RP</span>}
                            </div>
                            <span className={styles.roomMeta}>
                                {currentRoom.member_count ?? currentRoom.members.length} members
                                {currentRoom.is_public ? " · public" : " · private"}
                            </span>
                        </div>
                        <button
                            type="button"
                            className={styles.pinHeaderBtn}
                            onClick={() => panels.openPanel("search")}
                            aria-label="Search messages"
                            title="Search messages"
                        >
                            {"🔍"}
                        </button>
                        <button
                            type="button"
                            className={styles.pinHeaderBtn}
                            onClick={() => panels.openPanel("pins")}
                            aria-label="Pinned messages"
                            title="Pinned messages"
                        >
                            {"📌"}
                        </button>
                    </div>
                    {(currentRoom.description || (currentRoom.tags && currentRoom.tags.length > 0)) && (
                        <div className={styles.roomInfoCollapsible} data-expanded={prefs.descExpanded}>
                            <button type="button" className={styles.roomInfoToggle} onClick={prefs.toggleDescExpanded}>
                                {prefs.descExpanded ? "Hide info ▲" : "Show info ▼"}
                            </button>
                            <div className={styles.roomInfoContent}>
                                {currentRoom.description && (
                                    <div dir="auto" className={styles.roomDescription}>
                                        {currentRoom.description}
                                    </div>
                                )}
                                {currentRoom.tags && currentRoom.tags.length > 0 && (
                                    <div className={styles.roomTags}>
                                        {currentRoom.tags.map(t => (
                                            <span key={t} className={styles.roomTag}>
                                                #{t}
                                            </span>
                                        ))}
                                    </div>
                                )}
                            </div>
                        </div>
                    )}

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
            </div>

            {prefs.mobileView === "members" && (
                <button
                    type="button"
                    className={styles.mobileBackToChat}
                    onClick={() => prefs.setMobileView("chat")}
                    aria-label="Back to chat"
                >
                    {"← Back to chat"}
                </button>
            )}

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
                variant="desktop"
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
            {toast.message && <div className={styles.toast}>{toast.message}</div>}
            {panels.lightboxSrc && <Lightbox src={panels.lightboxSrc} onClose={() => panels.setLightboxSrc(null)} />}
        </div>
    );
}
