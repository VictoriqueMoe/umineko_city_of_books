import { Fragment } from "react";
import { isSiteStaff } from "../../utils/permissions";
import { effectiveMemberUser, memberModPermissions } from "../../utils/chatMembers";
import { useRoomController } from "../../hooks/useRoomController";
import { useIsMobile } from "../../hooks/useIsMobile";
import { MobileRoomView } from "../../components/chat/mobile/MobileRoomView";
import { TypingIndicator } from "../../components/chat/TypingIndicator/TypingIndicator";
import { Button } from "../../components/Button/Button";
import { ChatComposer } from "../../components/chat/ChatComposer/ChatComposer";
import { EditRoomProfileDialog } from "../../components/chat/EditRoomProfileDialog/EditRoomProfileDialog";
import { RoomModerationDialog } from "../../components/chat/RoomModerationDialog/RoomModerationDialog";
import { InviteMembersModal } from "../../components/chat/InviteMembersModal/InviteMembersModal";
import { RoomMessageList } from "../../components/chat/MessageList/RoomMessageList";
import { PinnedMessagesPanel } from "../../components/chat/PinnedMessagesPanel/PinnedMessagesPanel";
import { MessageSearchPanel } from "../../components/chat/MessageSearchPanel/MessageSearchPanel";
import { WatchPartyButton } from "../../components/chat/WatchParty/WatchPartyButton";
import { WatchPartyModal } from "../../components/chat/WatchParty/WatchPartyModal";
import { VoiceBar } from "../../components/chat/Voice/VoiceBar";
import { VoiceButton } from "../../components/chat/Voice/VoiceButton";
import { Lightbox } from "../../components/Lightbox/Lightbox";
import { ProfileLink } from "../../components/ProfileLink/ProfileLink";
import { forceMuteVoiceParticipant } from "../../api/endpoints";
import styles from "./RoomPage.module.css";

export function RoomPage() {
    const controller = useRoomController();
    const isMobile = useIsMobile();
    const {
        user,
        navigate,
        loading,
        room,
        roomId,
        members,
        memberGroups,
        presenceMapMerged,
        memberOnlineWeight,
        currentMember,
        mobileView,
        setMobileView,
        sidebarCollapsed,
        toggleSidebar,
        descExpanded,
        toggleDescExpanded,
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
        joining,
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
        formatTimeoutUntil,
        openNicknameDialog,
        openTimeoutDialog,
        handleSentMessage,
        handleJoin,
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

    if (!user) {
        return null;
    }

    if (loading) {
        return <div className="loading">Loading room...</div>;
    }

    if (!room) {
        return (
            <div className={styles.notMember}>
                <p>You're not a member of this room.</p>
                {roomId && (
                    <Button variant="primary" size="small" onClick={handleJoin} disabled={joining}>
                        {joining ? "Joining..." : "Try to Join"}
                    </Button>
                )}
                <Button variant="ghost" size="small" onClick={() => navigate("/rooms")}>
                    Back to Rooms
                </Button>
                {toast && <div className={styles.toast}>{toast}</div>}
            </div>
        );
    }

    if (isMobile) {
        return <MobileRoomView controller={controller} />;
    }

    const isHost = room.viewer_role === "host";
    const isSystem = room.is_system;
    const isSiteMod = isSiteStaff(user.role);
    const canModerateRoom = isHost || isSiteMod;

    return (
        <div className={styles.roomWrapper}>
            <div
                className={styles.roomLayout}
                data-mobile-view={mobileView}
                data-sidebar-collapsed={sidebarCollapsed ? "true" : "false"}
            >
                <aside className={styles.sidebar}>
                    <div className={styles.sidebarHeader}>
                        <button
                            type="button"
                            className={styles.backButton}
                            onClick={() => {
                                if (mobileView === "members") {
                                    setMobileView("chat");
                                } else {
                                    navigate("/rooms");
                                }
                            }}
                            aria-label={mobileView === "members" ? "Back to chat" : "Back to rooms"}
                        >
                            {"←"}
                        </button>
                        <span className={styles.sidebarTitle}>Members</span>
                        <span className={styles.memberCount}>{members.length}</span>
                        {isHost && !isSystem && (
                            <button
                                type="button"
                                className={styles.inviteButton}
                                onClick={() => setInviteModalOpen(true)}
                            >
                                + Invite
                            </button>
                        )}
                        <button
                            type="button"
                            className={styles.sidebarCollapseBtn}
                            onClick={toggleSidebar}
                            aria-label="Hide members"
                            data-tooltip="Hide members"
                        >
                            {"◀"}
                        </button>
                    </div>
                    <div className={styles.memberList}>
                        {memberGroups.map(group => (
                            <div key={group.label} className={styles.memberGroup}>
                                <div className={styles.memberGroupHeader}>{group.label}</div>
                                {group.members.map((m, memberIndex) => {
                                    const effectiveUser = effectiveMemberUser(m);
                                    const {
                                        isSelf,
                                        timeoutIsActive,
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
                                    const presenceTitle =
                                        presence === "active"
                                            ? "Active in this room"
                                            : presence === "idle"
                                              ? "Idle or tab in background"
                                              : "Not currently viewing";
                                    const isOnline = memberOnlineWeight(m.user.id) === 0;
                                    const prevMember = memberIndex > 0 ? group.members[memberIndex - 1] : null;
                                    const prevOnline = prevMember ? memberOnlineWeight(prevMember.user.id) === 0 : null;
                                    const showStatusHeader =
                                        group.label !== "In Voice" && (memberIndex === 0 || isOnline !== prevOnline);
                                    return (
                                        <Fragment key={m.user.id}>
                                            {showStatusHeader && (
                                                <div className={styles.memberStatusHeader}>
                                                    {isOnline ? "Online" : "Offline"}
                                                </div>
                                            )}
                                            <div className={styles.memberRow}>
                                                <span
                                                    className={`${styles.presenceDot} ${presenceClass}`}
                                                    title={presenceTitle}
                                                    aria-label={presenceTitle}
                                                />
                                                <ProfileLink user={effectiveUser} size="small" />
                                                {m.role === "host" && <span className={styles.hostBadge}>Host</span>}
                                                {m.ghost && (
                                                    <span
                                                        className={styles.ghostBadge}
                                                        title="Ghost member — not visible to non-staff"
                                                    >
                                                        {"👻"}
                                                    </span>
                                                )}
                                                {timeoutIsActive && (
                                                    <span
                                                        className={styles.timeoutIcon}
                                                        title={`Timed out until ${formatTimeoutUntil(m.timeout_until)}`}
                                                        aria-label={`Timed out until ${formatTimeoutUntil(m.timeout_until)}`}
                                                    >
                                                        {"⏱"}
                                                    </span>
                                                )}
                                                {isSelf && (
                                                    <button
                                                        type="button"
                                                        className={styles.editSelfBtn}
                                                        onClick={() => setEditProfileOpen(true)}
                                                        title="Edit profile in this room"
                                                        aria-label="Edit profile in this room"
                                                    >
                                                        {"✎"}
                                                    </button>
                                                )}
                                                {canActOnMember && (
                                                    <div className={styles.memberActions}>
                                                        <button
                                                            type="button"
                                                            className={styles.modActionsBtn}
                                                            onClick={() =>
                                                                setOpenMemberMenu(prev =>
                                                                    prev === m.user.id ? null : m.user.id,
                                                                )
                                                            }
                                                            aria-label="Moderator actions"
                                                            title="Moderator actions"
                                                        >
                                                            {"⋮"}
                                                        </button>
                                                        {menuOpen && (
                                                            <div
                                                                className={styles.modActionsMenu}
                                                                onMouseLeave={() => setOpenMemberMenu(null)}
                                                            >
                                                                {canEditTargetNickname && (
                                                                    <button
                                                                        type="button"
                                                                        onClick={() => openNicknameDialog(m)}
                                                                    >
                                                                        Change nickname
                                                                    </button>
                                                                )}
                                                                {canEditTargetNickname && m.nickname_locked && (
                                                                    <button
                                                                        type="button"
                                                                        onClick={() =>
                                                                            handleModUnlockNickname(m.user.id)
                                                                        }
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
                                                                    <button
                                                                        type="button"
                                                                        onClick={() => openTimeoutDialog(m)}
                                                                    >
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
                                                )}
                                            </div>
                                        </Fragment>
                                    );
                                })}
                            </div>
                        ))}
                    </div>
                    <div className={styles.sidebarFooter}>
                        <Button
                            variant="secondary"
                            size="small"
                            onClick={handleToggleMute}
                            disabled={busy === "mute"}
                            title={room.viewer_muted ? "Unmute notifications" : "Mute notifications"}
                        >
                            {busy === "mute"
                                ? "..."
                                : room.viewer_muted
                                  ? "Unmute notifications"
                                  : "Mute notifications"}
                        </Button>
                        {!isSystem && canModerateRoom && (
                            <Button variant="secondary" size="small" onClick={() => setModerationDialogOpen(true)}>
                                Moderation
                            </Button>
                        )}
                        {!isSystem && canModerateRoom && (
                            <Button variant="danger" size="small" onClick={handleDelete} disabled={busy === "delete"}>
                                {busy === "delete" ? "Deleting..." : "Delete Room"}
                            </Button>
                        )}
                        {!isSystem && !isHost && (
                            <Button variant="danger" size="small" onClick={handleLeave} disabled={busy === "self"}>
                                {busy === "self" ? "Leaving..." : "Leave Room"}
                            </Button>
                        )}
                    </div>
                </aside>

                {sidebarCollapsed && (
                    <button
                        type="button"
                        className={styles.sidebarExpandRail}
                        onClick={toggleSidebar}
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
                            onClick={() => setMobileView("members")}
                            aria-label="Members"
                        >
                            {"☰"}
                        </button>
                        <div className={styles.roomHeaderInfo}>
                            <div className={styles.roomTitleRow}>
                                <span className={styles.roomTitle}>{room.name}</span>
                                {room.is_system && <span className={styles.rpBadge}>Staff</span>}
                                {room.is_rp && <span className={styles.rpBadge}>RP</span>}
                            </div>
                            <span className={styles.roomMeta}>
                                {room.member_count ?? room.members.length} members
                                {room.is_public ? " · public" : " · private"}
                            </span>
                        </div>
                        <button
                            type="button"
                            className={styles.pinHeaderBtn}
                            onClick={() => setSearchOpen(true)}
                            aria-label="Search messages"
                            title="Search messages"
                        >
                            {"🔍"}
                        </button>
                        <button
                            type="button"
                            className={styles.pinHeaderBtn}
                            onClick={() => setPinnedOpen(true)}
                            aria-label="Pinned messages"
                            title="Pinned messages"
                        >
                            {"📌"}
                        </button>
                    </div>
                    {(room.description || (room.tags && room.tags.length > 0)) && (
                        <div className={styles.roomInfoCollapsible} data-expanded={descExpanded}>
                            <button type="button" className={styles.roomInfoToggle} onClick={toggleDescExpanded}>
                                {descExpanded ? "Hide info ▲" : "Show info ▼"}
                            </button>
                            <div className={styles.roomInfoContent}>
                                {room.description && <div className={styles.roomDescription}>{room.description}</div>}
                                {room.tags && room.tags.length > 0 && (
                                    <div className={styles.roomTags}>
                                        {room.tags.map(t => (
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
                        classes={{
                            messages: styles.messages,
                            loadMoreBar: styles.loadMoreBar,
                            empty: styles.messagesEmpty,
                        }}
                    />
                    <TypingIndicator names={typingNames} />
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
            </div>

            {mobileView === "members" && (
                <button
                    type="button"
                    className={styles.mobileBackToChat}
                    onClick={() => setMobileView("chat")}
                    aria-label="Back to chat"
                >
                    {"← Back to chat"}
                </button>
            )}

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
                <div className={styles.nicknameDialogOverlay} onClick={() => setNicknameDialogTarget(null)}>
                    <div className={styles.nicknameDialog} onClick={e => e.stopPropagation()}>
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
                        <div className={styles.nicknameDialogActions}>
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
                <div className={styles.nicknameDialogOverlay} onClick={() => setTimeoutDialogTarget(null)}>
                    <div className={styles.nicknameDialog} onClick={e => e.stopPropagation()}>
                        <h3>Set timeout for {timeoutDialogTarget.user.display_name}</h3>
                        <div className={styles.timeoutDialogRow}>
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
                        <div className={styles.nicknameDialogActions}>
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
        </div>
    );
}
