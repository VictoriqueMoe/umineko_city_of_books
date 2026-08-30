import type { SiteRole, User } from "../../types/api";

export type UserIdentityPatch = Partial<
    Pick<User, "banned" | "ban_reason" | "locked" | "lock_reason" | "role" | "display_name" | "avatar_url">
>;

export interface RoleChangeData {
    user_id?: string;
    role?: SiteRole | "";
}

export interface BanChangeData {
    user_id?: string;
    banned?: boolean;
    ban_reason?: string;
}

export interface LockChangeData {
    user_id?: string;
    locked?: boolean;
    lock_reason?: string;
}

export interface ProfileChangeData {
    user_id?: string;
    display_name?: string;
    avatar_url?: string;
}

export type UserIdentityChange =
    | { type: "role_changed"; data: RoleChangeData }
    | { type: "ban_changed"; data: BanChangeData }
    | { type: "lock_changed"; data: LockChangeData }
    | { type: "profile_changed"; data: ProfileChangeData };

export interface UserIdentityUpdate {
    userId: string;
    patch: UserIdentityPatch;
}

function rolePatch(data: RoleChangeData): UserIdentityPatch {
    return { role: (data.role ?? "") as User["role"] };
}

function banPatch(data: BanChangeData): UserIdentityPatch {
    return { banned: !!data.banned, ban_reason: data.ban_reason ?? "" };
}

function lockPatch(data: LockChangeData): UserIdentityPatch {
    return { locked: !!data.locked, lock_reason: data.lock_reason ?? "" };
}

function profilePatch(data: ProfileChangeData): UserIdentityPatch {
    const patch: UserIdentityPatch = {};

    if (typeof data.display_name === "string") {
        patch.display_name = data.display_name;
    }
    if (typeof data.avatar_url === "string") {
        patch.avatar_url = data.avatar_url;
    }

    return patch;
}

function patchFor(change: UserIdentityChange): UserIdentityPatch {
    switch (change.type) {
        case "role_changed":
            return rolePatch(change.data);
        case "ban_changed":
            return banPatch(change.data);
        case "lock_changed":
            return lockPatch(change.data);
        case "profile_changed":
            return profilePatch(change.data);
    }
}

export function userIdentityPatch(change: UserIdentityChange): UserIdentityUpdate | null {
    const userId = change.data.user_id;
    if (!userId) {
        return null;
    }

    return { userId, patch: patchFor(change) };
}
