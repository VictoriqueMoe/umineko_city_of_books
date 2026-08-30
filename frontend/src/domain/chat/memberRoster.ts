import type { ChatRoomMember, User } from "../../types/api";
import { ROLE_GROUPS } from "../permissions";

export type MemberPresence = "active" | "idle";

export type PresenceMap = Record<string, MemberPresence>;

export interface MemberGroup {
    label: string;
    members: ChatRoomMember[];
}

export type TypingRoster = readonly ChatRoomMember[] | readonly User[] | null | undefined;

const HOST_LABEL = "Host";
const DEFAULT_MEMBER_LABEL = "Members";
const IN_VOICE_LABEL = "In Voice";
const UNKNOWN_TYPING_NAME = "Someone";

export function mergePresence(members: readonly ChatRoomMember[], live: Readonly<PresenceMap>): PresenceMap {
    const seed: PresenceMap = {};
    for (const member of members) {
        if (member.presence === "active" || member.presence === "idle") {
            seed[member.user.id] = member.presence;
        }
    }

    return { ...seed, ...live };
}

export function memberOnlineWeight(presence: Readonly<PresenceMap>, id: string): number {
    const state = presence[id];
    if (state === "active" || state === "idle") {
        return 0;
    }

    return 1;
}

export function memberRankWeight(member: ChatRoomMember): number {
    if (member.user.role === "super_admin") {
        return 0;
    }

    if (member.role === "host") {
        return 1;
    }

    const idx = ROLE_GROUPS.findIndex(group => group.role === member.user.role);
    if (idx >= 0) {
        return idx + 1;
    }

    return ROLE_GROUPS.length + 1;
}

export function memberSortName(member: ChatRoomMember): string {
    const nickname = member.nickname?.trim();
    if (nickname) {
        return nickname.toLowerCase();
    }

    const displayName = member.user.display_name?.trim();
    if (displayName) {
        return displayName.toLowerCase();
    }

    return member.user.username.toLowerCase();
}

export function memberRankLabel(member: ChatRoomMember): string {
    const group = ROLE_GROUPS.find(g => g.role === member.user.role);

    if (member.user.role === "super_admin" && group) {
        return group.label;
    }

    if (member.role === "host") {
        return HOST_LABEL;
    }

    if (group) {
        return group.label;
    }

    return DEFAULT_MEMBER_LABEL;
}

export function sortMembers(members: readonly ChatRoomMember[], presence: Readonly<PresenceMap>): ChatRoomMember[] {
    return [...members].sort((a, b) => {
        const rank = memberRankWeight(a) - memberRankWeight(b);
        if (rank !== 0) {
            return rank;
        }

        const online = memberOnlineWeight(presence, a.user.id) - memberOnlineWeight(presence, b.user.id);
        if (online !== 0) {
            return online;
        }

        return memberSortName(a).localeCompare(memberSortName(b));
    });
}

export function groupMembers(sorted: readonly ChatRoomMember[], voiceIds: ReadonlySet<string>): MemberGroup[] {
    const groups: MemberGroup[] = [];
    for (const member of sorted) {
        const label = memberRankLabel(member);
        const last = groups[groups.length - 1];
        if (last && last.label === label) {
            last.members.push(member);
        } else {
            groups.push({ label, members: [member] });
        }
    }

    const inVoice = sorted.filter(member => voiceIds.has(member.user.id));
    if (inVoice.length > 0) {
        groups.unshift({ label: IN_VOICE_LABEL, members: inVoice });
    }

    return groups;
}

export function typingNames(typingUserIds: readonly string[], roster: TypingRoster, selfId?: string | null): string[] {
    const names: string[] = [];
    for (const id of typingUserIds) {
        if (id === selfId) {
            continue;
        }

        const entry = findRosterEntry(roster, id);
        names.push(entry ? rosterEntryName(entry) : UNKNOWN_TYPING_NAME);
    }

    return names;
}

function isRoomMember(entry: ChatRoomMember | User): entry is ChatRoomMember {
    return "user" in entry;
}

function findRosterEntry(roster: TypingRoster, id: string): ChatRoomMember | User | undefined {
    if (!roster) {
        return undefined;
    }

    for (const entry of roster) {
        const entryId = isRoomMember(entry) ? entry.user.id : entry.id;
        if (entryId === id) {
            return entry;
        }
    }

    return undefined;
}

function accountName(user: User): string {
    if (user.display_name && user.display_name.trim() !== "") {
        return user.display_name;
    }

    return user.username;
}

function rosterEntryName(entry: ChatRoomMember | User): string {
    if (!isRoomMember(entry)) {
        return accountName(entry);
    }

    if (entry.nickname && entry.nickname.trim() !== "") {
        return entry.nickname;
    }

    return accountName(entry.user);
}
