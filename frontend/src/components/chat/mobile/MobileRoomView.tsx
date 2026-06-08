import { isSiteStaff } from "../../../utils/permissions";
import { effectiveMemberUser, memberModPermissions } from "../../../utils/chatMembers";
import { forceMuteVoiceParticipant } from "../../../api/endpoints";
import { useChatViewport } from "../../../hooks/useChatViewport";
import type { RoomController } from "../../../hooks/useRoomController";
import { TypingIndicator } from "../TypingIndicator/TypingIndicator";
import { Button } from "../../Button/Button";
import { ChatComposer } from "../ChatComposer/ChatComposer";
import { EditRoomProfileDialog } from "../EditRoomProfileDialog/EditRoomProfileDialog";
import { RoomModerationDialog } from "../RoomModerationDialog/RoomModerationDialog";
import { InviteMembersModal } from "../InviteMembersModal/InviteMembersModal";
import { RoomMessageList } from "../MessageList/RoomMessageList";
import { PinnedMessagesPanel } from "../PinnedMessagesPanel/PinnedMessagesPanel";
import { MessageSearchPanel } from "../MessageSearchPanel/MessageSearchPanel";
import { WatchPartyButton } from "../WatchParty/WatchPartyButton";
import { WatchPartyModal } from "../WatchParty/WatchPartyModal";
import { VoiceBar } from "../Voice/VoiceBar";
import { VoiceButton } from "../Voice/VoiceButton";
import { Lightbox } from "../../Lightbox/Lightbox";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import styles from "./mobileChat.module.css";

export function MobileRoomView({ controller }: { controller: RoomController }) {
    const {
        user,
        navigate,
        room,
        roomId,
        members,
        memberGroups,
        presenceMapMerged,
        currentMember,
        mobileView,
        setMobileView,
        scrollToBottom,
        typingNames,
        voice,
        voiceEnabled,
        watchParty,
        replyingTo,
        setReplyingTo,
        viewerTimeoutUntil,
        lightboxSrc,
        setLightboxSrc,
        toast,
        setToast,
        busy,
        sendWSMessage,
        pinnedOpen,
        setPinnedOpen,
        searchOpen,
        setSearchOpen,
        pinnedRefreshKey,
        editProfileOpen,
        setEditProfileOpen,
        inviteModalOpen,
        setInviteModalOpen,
        moderationDialogOpen,
        setModerationDialogOpen,
        openMemberMenu,
        setOpenMemberMenu,
        setMembers,
        nicknameDialogTarget,
        setNicknameDialogTarget,
        nicknameDialogValue,
        setNicknameDialogValue,
        nicknameDialogError,
        nicknameDialogSaving,
        timeoutDialogTarget,
        setTimeoutDialogTarget,
        timeoutDialogAmount,
        setTimeoutDialogAmount,
        timeoutDialogUnit,
        setTimeoutDialogUnit,
        timeoutDialogError,
        timeoutDialogSaving,
        openNicknameDialog,
        openTimeoutDialog,
        handleSentMessage,
        handleModSetNickname,
        handleModUnlockNickname,
        handleSetTimeout,
        handleClearTimeout,
        handleKick,
        handleBan,
        handleToggleMute,
        handleLeave,
        handleDelete,
        handleJumpToMessage,
        handleEditLast,
    } = controller;

    useChatViewport({ scrollToBottom });

    if (!user || !room) {
        return null;
    }

    const isHost = room.viewer_role === "host";
    const isSystem = room.is_system;
    const isSiteMod = isSiteStaff(user.role);
    const canModerateRoom = isHost || isSiteMod;

    const overlays = (
        <>
            {searchOpen && (
                <MessageSearchPanel
                    roomId={room.id}
                    isOpen={searchOpen}
                    onClose={() => setSearchOpen(false)}
                    onJump={handleJumpToMessage}
                />
            )}

            <PinnedMessagesPanel
                roomId={room.id}
                isOpen={pinnedOpen}
                onClose={() => setPinnedOpen(false)}
                onJump={handleJumpToMessage}
                canUnpin={canModerateRoom}
                refreshKey={pinnedRefreshKey}
                onLightbox={setLightboxSrc}
            />

            <EditRoomProfileDialog
                key={`${room.id}:${currentMember?.user.id ?? ""}:${editProfileOpen ? "open" : "closed"}`}
                isOpen={editProfileOpen}
                roomId={room.id}
                currentMember={currentMember}
                onClose={() => setEditProfileOpen(false)}
                onSaved={updated => {
                    setMembers(prev => prev.map(m => (m.user.id === updated.user.id ? updated : m)));
                }}
            />

            <RoomModerationDialog
                isOpen={moderationDialogOpen}
                roomId={room.id}
                onClose={() => setModerationDialogOpen(false)}
            />

            {watchParty.activeSession && user && (
                <WatchPartyModal
                    isOpen={true}
                    onClose={() => watchParty.close()}
                    active={watchParty.activeSession}
                    viewerUserId={user.id}
                    viewerRole={user.role}
                    isStarter={watchParty.activeSession.session.started_by === user.id}
                    viewerIsStaff={isSiteStaff(user.role)}
                    voiceEnabled={watchParty.screenShareEnabled}
                    onLeave={watchParty.leave}
                    onEnd={watchParty.end}
                    onTransferControl={watchParty.transferControl}
                    onKick={watchParty.kick}
                    onIdentify={watchParty.identify}
                    onSendMessage={watchParty.sendMessage}
                />
            )}

            <InviteMembersModal
                isOpen={inviteModalOpen}
                roomId={room.id}
                existingMemberIds={new Set(members.map(m => m.user.id))}
                onClose={() => setInviteModalOpen(false)}
                onInvited={result => {
                    if (result.invited_count > 0) {
                        setToast(
                            result.invited_count === 1 ? "1 member invited" : `${result.invited_count} members invited`,
                        );
                    } else if (result.skipped_count > 0) {
                        setToast("No one invited (all were already members or blocked)");
                    }
                }}
            />

            {nicknameDialogTarget && (
                <div className={styles.dialogOverlay} onClick={() => setNicknameDialogTarget(null)}>
                    <div className={styles.dialog} onClick={e => e.stopPropagation()}>
                        <h3>Change nickname for {nicknameDialogTarget.user.display_name}</h3>
                        <input
                            type="text"
                            value={nicknameDialogValue}
                            maxLength={32}
                            onChange={e => setNicknameDialogValue(e.target.value)}
                            placeholder="Nickname (leave blank to clear)"
                            autoFocus
                        />
                        {nicknameDialogError && <div className={styles.dialogError}>{nicknameDialogError}</div>}
                        <div className={styles.dialogActions}>
                            <Button
                                variant="ghost"
                                size="small"
                                onClick={() => setNicknameDialogTarget(null)}
                                disabled={nicknameDialogSaving}
                            >
                                Cancel
                            </Button>
                            <Button
                                variant="primary"
                                size="small"
                                onClick={handleModSetNickname}
                                disabled={nicknameDialogSaving}
                            >
                                {nicknameDialogSaving ? "Saving..." : "Save"}
                            </Button>
                        </div>
                    </div>
                </div>
            )}

            {timeoutDialogTarget && (
                <div className={styles.dialogOverlay} onClick={() => setTimeoutDialogTarget(null)}>
                    <div className={styles.dialog} onClick={e => e.stopPropagation()}>
                        <h3>Set timeout for {timeoutDialogTarget.user.display_name}</h3>
                        <div className={styles.dialogRow}>
                            <input
                                type="number"
                                min={1}
                                step={1}
                                value={timeoutDialogAmount}
                                onChange={e => setTimeoutDialogAmount(e.target.value)}
                                autoFocus
                            />
                            <select value={timeoutDialogUnit} onChange={e => setTimeoutDialogUnit(e.target.value)}>
                                <option value="seconds">seconds</option>
                                <option value="hours">hours</option>
                                <option value="weeks">weeks</option>
                                <option value="years">years</option>
                                <option value="decades">decades</option>
                                <option value="centuries">centuries</option>
                            </select>
                        </div>
                        {timeoutDialogError && <div className={styles.dialogError}>{timeoutDialogError}</div>}
                        <div className={styles.dialogActions}>
                            <Button
                                variant="ghost"
                                size="small"
                                onClick={() => setTimeoutDialogTarget(null)}
                                disabled={timeoutDialogSaving}
                            >
                                Cancel
                            </Button>
                            <Button
                                variant="danger"
                                size="small"
                                onClick={handleSetTimeout}
                                disabled={timeoutDialogSaving}
                            >
                                {timeoutDialogSaving ? "Saving..." : "Set timeout"}
                            </Button>
                        </div>
                    </div>
                </div>
            )}

            {toast && <div className={styles.toast}>{toast}</div>}
            {lightboxSrc && <Lightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />}
        </>
    );

    if (mobileView === "members") {
        return (
            <div className={styles.shell}>
                <div className={styles.topBar}>
                    <button
                        type="button"
                        className={styles.iconBtn}
                        onClick={() => setMobileView("chat")}
                        aria-label="Back to chat"
                    >
                        {"←"}
                    </button>
                    <div className={styles.topInfo}>
                        <span className={styles.topTitle}>Members</span>
                        <span className={styles.topMeta}>{members.length} members</span>
                    </div>
                    {isHost && !isSystem && (
                        <button
                            type="button"
                            className={styles.iconBtn}
                            onClick={() => setInviteModalOpen(true)}
                            aria-label="Invite members"
                        >
                            {"+"}
                        </button>
                    )}
                </div>
                <div className={styles.memberList}>
                    {memberGroups.map(group => (
                        <div key={group.label}>
                            <div className={styles.memberGroupHeader}>{group.label}</div>
                            {group.members.map(m => {
                                const effectiveUser = effectiveMemberUser(m);
                                const {
                                    isSelf,
                                    canKick: canKickTarget,
                                    canEditNickname: canEditTargetNickname,
                                    canTimeout: canTimeoutTarget,
                                    canClearTimeout: canClearTimeoutTarget,
                                    canActOnMember,
                                } = memberModPermissions(m, {
                                    selfId: user.id,
                                    isSystem,
                                    isSiteMod,
                                    canModerateRoom,
                                });
                                const menuOpen = openMemberMenu === m.user.id;
                                const presence = presenceMapMerged[m.user.id];
                                const presenceClass =
                                    presence === "active"
                                        ? styles.presenceActive
                                        : presence === "idle"
                                          ? styles.presenceIdle
                                          : styles.presenceAway;
                                return (
                                    <div key={m.user.id} className={styles.memberRow}>
                                        <span className={`${styles.presenceDot} ${presenceClass}`} />
                                        <ProfileLink user={effectiveUser} size="small" />
                                        {m.role === "host" && <span className={styles.hostBadge}>Host</span>}
                                        <span className={styles.memberSpacer} />
                                        {isSelf && (
                                            <button
                                                type="button"
                                                className={styles.iconBtn}
                                                onClick={() => setEditProfileOpen(true)}
                                                aria-label="Edit profile in this room"
                                            >
                                                {"✎"}
                                            </button>
                                        )}
                                        {canActOnMember && (
                                            <button
                                                type="button"
                                                className={styles.iconBtn}
                                                onClick={() =>
                                                    setOpenMemberMenu(prev => (prev === m.user.id ? null : m.user.id))
                                                }
                                                aria-label="Moderator actions"
                                            >
                                                {"⋮"}
                                            </button>
                                        )}
                                        {menuOpen && (
                                            <div className={styles.modMenu}>
                                                {canEditTargetNickname && (
                                                    <button type="button" onClick={() => openNicknameDialog(m)}>
                                                        Change nickname
                                                    </button>
                                                )}
                                                {canEditTargetNickname && m.nickname_locked && (
                                                    <button
                                                        type="button"
                                                        onClick={() => handleModUnlockNickname(m.user.id)}
                                                        disabled={busy === m.user.id}
                                                    >
                                                        Reset/unlock nickname
                                                    </button>
                                                )}
                                                {canKickTarget && (
                                                    <button
                                                        type="button"
                                                        className={styles.danger}
                                                        onClick={() => {
                                                            setOpenMemberMenu(null);
                                                            handleKick(m.user.id);
                                                        }}
                                                        disabled={busy === m.user.id}
                                                    >
                                                        Kick member
                                                    </button>
                                                )}
                                                {canKickTarget && (
                                                    <button
                                                        type="button"
                                                        className={styles.danger}
                                                        onClick={() => {
                                                            setOpenMemberMenu(null);
                                                            handleBan(m.user.id);
                                                        }}
                                                        disabled={busy === m.user.id}
                                                    >
                                                        Ban from room
                                                    </button>
                                                )}
                                                {canTimeoutTarget && (
                                                    <button type="button" onClick={() => openTimeoutDialog(m)}>
                                                        Set timeout
                                                    </button>
                                                )}
                                                {canClearTimeoutTarget && (
                                                    <button
                                                        type="button"
                                                        onClick={() => handleClearTimeout(m.user.id)}
                                                        disabled={busy === `timeout:${m.user.id}`}
                                                    >
                                                        Remove timeout
                                                    </button>
                                                )}
                                            </div>
                                        )}
                                    </div>
                                );
                            })}
                        </div>
                    ))}
                </div>
                <div className={styles.membersFooter}>
                    <Button variant="secondary" size="small" onClick={handleToggleMute} disabled={busy === "mute"}>
                        {busy === "mute" ? "..." : room.viewer_muted ? "Unmute" : "Mute"}
                    </Button>
                    {!isSystem && canModerateRoom && (
                        <Button variant="secondary" size="small" onClick={() => setModerationDialogOpen(true)}>
                            Moderation
                        </Button>
                    )}
                    {!isSystem && canModerateRoom && (
                        <Button variant="danger" size="small" onClick={handleDelete} disabled={busy === "delete"}>
                            {busy === "delete" ? "Deleting..." : "Delete"}
                        </Button>
                    )}
                    {!isSystem && !isHost && (
                        <Button variant="danger" size="small" onClick={handleLeave} disabled={busy === "self"}>
                            {busy === "self" ? "Leaving..." : "Leave"}
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
                <button
                    type="button"
                    className={styles.iconBtn}
                    onClick={() => navigate("/rooms")}
                    aria-label="Back to rooms"
                >
                    {"←"}
                </button>
                <div className={styles.topInfo}>
                    <div className={styles.topTitleRow}>
                        <span className={styles.topTitle}>{room.name}</span>
                        {room.is_system && <span className={styles.topBadge}>Staff</span>}
                        {room.is_rp && <span className={styles.topBadge}>RP</span>}
                    </div>
                    <span className={styles.topMeta}>
                        {room.member_count ?? room.members.length} members
                        {room.is_public ? " · public" : " · private"}
                    </span>
                </div>
                <button
                    type="button"
                    className={styles.iconBtn}
                    onClick={() => setSearchOpen(true)}
                    aria-label="Search messages"
                >
                    {"🔍"}
                </button>
                <button
                    type="button"
                    className={styles.iconBtn}
                    onClick={() => setPinnedOpen(true)}
                    aria-label="Pinned messages"
                >
                    {"📌"}
                </button>
                <button
                    type="button"
                    className={styles.iconBtn}
                    onClick={() => setMobileView("members")}
                    aria-label="Members"
                >
                    {"☰"}
                </button>
            </div>

            {voice.status === "connected" && voice.room && (
                <VoiceBar
                    room={voice.room}
                    onLeave={voice.leave}
                    canModerate={canModerateRoom}
                    onForceMute={(id, muted) => {
                        forceMuteVoiceParticipant(roomId ?? "", id, muted).catch(() => {});
                    }}
                />
            )}

            <RoomMessageList
                controller={controller}
                classes={{ messages: styles.messages, loadMoreBar: styles.loadMoreBar, empty: styles.empty }}
            />

            <TypingIndicator names={typingNames} />

            <div className={styles.composerWrap}>
                <ChatComposer
                    roomId={room.id}
                    draftRecipientId={null}
                    onSent={handleSentMessage}
                    mentionPool={members.map(m => m.user)}
                    replyingTo={replyingTo}
                    onCancelReply={() => setReplyingTo(null)}
                    onTyping={() => sendWSMessage({ type: "typing", data: { room_id: room.id } })}
                    onEditLast={handleEditLast}
                    timeoutUntil={viewerTimeoutUntil}
                    sendOnEnter={false}
                    compact
                    extraActions={
                        !isSystem && user ? (
                            <>
                                <VoiceButton
                                    enabled={voiceEnabled}
                                    status={voice.status}
                                    presenceCount={voice.presenceCount}
                                    onJoin={voice.join}
                                    onLeave={voice.leave}
                                />
                                <WatchPartyButton
                                    enabled={watchParty.enabled}
                                    screenShareEnabled={watchParty.screenShareEnabled}
                                    sessions={watchParty.sessions}
                                    activeSessionId={watchParty.openSessionId}
                                    viewerUserId={user.id}
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
