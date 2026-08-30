import { Fragment, type Dispatch, type SetStateAction } from "react";
import { effectiveMemberUser, memberModPermissions, type MemberModContext } from "../../../domain/chat/members";
import type { MemberGroup, MemberPresence, PresenceMap } from "../../../domain/chat/memberRoster";
import type { ChatRoomMember } from "../../../types/api";
import { ProfileLink } from "../../ProfileLink/ProfileLink";
import styles from "./RoomMemberList.module.css";

const VOICE_GROUP_LABEL = "In Voice";
const ACTIVE_TITLE = "Active in this room";
const IDLE_TITLE = "Idle or tab in background";
const AWAY_TITLE = "Not currently viewing";
const GHOST_TITLE = "Ghost member, not visible to non-staff";
const EDIT_SELF_LABEL = "Edit profile in this room";
const MOD_ACTIONS_LABEL = "Moderator actions";

export type RoomMemberListVariant = "desktop" | "mobile";

export interface RoomMemberListActions {
    editSelf: () => void;
    openNicknameDialog: (member: ChatRoomMember) => void;
    unlockNickname: (userId: string) => void;
    kick: (userId: string) => void;
    ban: (userId: string) => void;
    openTimeoutDialog: (member: ChatRoomMember) => void;
    clearTimeout: (userId: string) => void;
}

export interface RoomMemberListProps {
    variant: RoomMemberListVariant;
    groups: MemberGroup[];
    presence: PresenceMap;
    onlineWeight: (userId: string) => number;
    viewer: MemberModContext;
    openMemberMenu: string | null;
    setOpenMemberMenu: Dispatch<SetStateAction<string | null>>;
    busy: string | null;
    formatTimeoutUntil: (value?: string) => string;
    actions: RoomMemberListActions;
}

function presenceClassName(state: MemberPresence | undefined): string {
    if (state === "active") {
        return styles.presenceActive;
    }

    if (state === "idle") {
        return styles.presenceIdle;
    }

    return styles.presenceAway;
}

function presenceTitle(state: MemberPresence | undefined): string {
    if (state === "active") {
        return ACTIVE_TITLE;
    }

    if (state === "idle") {
        return IDLE_TITLE;
    }

    return AWAY_TITLE;
}

export function RoomMemberList({
    variant,
    groups,
    presence,
    onlineWeight,
    viewer,
    openMemberMenu,
    setOpenMemberMenu,
    busy,
    formatTimeoutUntil,
    actions,
}: RoomMemberListProps) {
    return (
        <div className={styles.memberList} data-variant={variant}>
            {groups.map(group => (
                <div key={group.label} className={styles.memberGroup}>
                    <div className={styles.memberGroupHeader}>{group.label}</div>
                    {group.members.map((member, memberIndex) => {
                        const effectiveUser = effectiveMemberUser(member);
                        const {
                            isSelf,
                            timeoutIsActive,
                            canKick,
                            canEditNickname,
                            canTimeout,
                            canClearTimeout,
                            canActOnMember,
                        } = memberModPermissions(member, viewer);
                        const menuOpen = openMemberMenu === member.user.id;
                        const state = presence[member.user.id];
                        const isOnline = onlineWeight(member.user.id) === 0;
                        const previousMember = memberIndex > 0 ? group.members[memberIndex - 1] : null;
                        const previousOnline = previousMember ? onlineWeight(previousMember.user.id) === 0 : null;
                        const showStatusHeader =
                            variant === "desktop" &&
                            group.label !== VOICE_GROUP_LABEL &&
                            (memberIndex === 0 || isOnline !== previousOnline);
                        const timeoutLabel = `Timed out until ${formatTimeoutUntil(member.timeout_until)}`;

                        return (
                            <Fragment key={member.user.id}>
                                {showStatusHeader && (
                                    <div className={styles.memberStatusHeader}>{isOnline ? "Online" : "Offline"}</div>
                                )}
                                <div className={styles.memberRow}>
                                    <span
                                        className={`${styles.presenceDot} ${presenceClassName(state)}`}
                                        title={presenceTitle(state)}
                                        aria-label={presenceTitle(state)}
                                    />
                                    <ProfileLink user={effectiveUser} size="small" />
                                    {member.role === "host" && <span className={styles.hostBadge}>Host</span>}
                                    {member.ghost && (
                                        <span className={styles.ghostBadge} title={GHOST_TITLE}>
                                            {"👻"}
                                        </span>
                                    )}
                                    {timeoutIsActive && (
                                        <span
                                            className={styles.timeoutIcon}
                                            title={timeoutLabel}
                                            aria-label={timeoutLabel}
                                        >
                                            {"⏱"}
                                        </span>
                                    )}
                                    {isSelf && (
                                        <button
                                            type="button"
                                            className={styles.editSelfBtn}
                                            onClick={actions.editSelf}
                                            title={EDIT_SELF_LABEL}
                                            aria-label={EDIT_SELF_LABEL}
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
                                                        prev === member.user.id ? null : member.user.id,
                                                    )
                                                }
                                                title={MOD_ACTIONS_LABEL}
                                                aria-label={MOD_ACTIONS_LABEL}
                                            >
                                                {"⋮"}
                                            </button>
                                            {menuOpen && (
                                                <div
                                                    className={styles.modActionsMenu}
                                                    onMouseLeave={() => setOpenMemberMenu(null)}
                                                >
                                                    {canEditNickname && (
                                                        <button
                                                            type="button"
                                                            onClick={() => actions.openNicknameDialog(member)}
                                                        >
                                                            Change nickname
                                                        </button>
                                                    )}
                                                    {canEditNickname && member.nickname_locked && (
                                                        <button
                                                            type="button"
                                                            onClick={() => actions.unlockNickname(member.user.id)}
                                                            disabled={busy === member.user.id}
                                                        >
                                                            Reset/unlock nickname
                                                        </button>
                                                    )}
                                                    {canKick && (
                                                        <button
                                                            type="button"
                                                            className={styles.danger}
                                                            onClick={() => {
                                                                setOpenMemberMenu(null);
                                                                actions.kick(member.user.id);
                                                            }}
                                                            disabled={busy === member.user.id}
                                                        >
                                                            Kick member
                                                        </button>
                                                    )}
                                                    {canKick && (
                                                        <button
                                                            type="button"
                                                            className={styles.danger}
                                                            onClick={() => {
                                                                setOpenMemberMenu(null);
                                                                actions.ban(member.user.id);
                                                            }}
                                                            disabled={busy === member.user.id}
                                                        >
                                                            Ban from room
                                                        </button>
                                                    )}
                                                    {canTimeout && (
                                                        <button
                                                            type="button"
                                                            onClick={() => actions.openTimeoutDialog(member)}
                                                        >
                                                            Set timeout
                                                        </button>
                                                    )}
                                                    {canClearTimeout && (
                                                        <button
                                                            type="button"
                                                            onClick={() => actions.clearTimeout(member.user.id)}
                                                            disabled={busy === `timeout:${member.user.id}`}
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
    );
}
