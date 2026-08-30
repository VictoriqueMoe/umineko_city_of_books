import type {
    BanChangedPayload,
    ChatUnreadBumpedPayload,
    LiveGamesCountPayload,
    LockChangedPayload,
    ProfileChangedPayload,
    RealtimeEvent,
    RealtimeEventName,
    RealtimeEventPayload,
    RoleChangedPayload,
} from "./events";

export type RealtimeEventGuard<K extends RealtimeEventName> = (data: unknown) => data is RealtimeEventPayload<K>;

type RealtimeGuardTable = Partial<{ [K in RealtimeEventName]: RealtimeEventGuard<K> }>;

function asRecord(data: unknown): Record<string, unknown> | null {
    if (data === null || typeof data !== "object" || Array.isArray(data)) {
        return null;
    }

    return data as Record<string, unknown>;
}

function isUserId(value: unknown): boolean {
    return typeof value === "string" && value.length > 0;
}

function isFiniteNumber(value: unknown): boolean {
    return typeof value === "number" && Number.isFinite(value);
}

function isAbsentOrString(value: unknown): boolean {
    return value === undefined || typeof value === "string";
}

export function isRoleChangedPayload(data: unknown): data is RoleChangedPayload {
    const record = asRecord(data);
    if (record === null) {
        return false;
    }

    return isUserId(record.user_id) && typeof record.role === "string";
}

export function isBanChangedPayload(data: unknown): data is BanChangedPayload {
    const record = asRecord(data);
    if (record === null) {
        return false;
    }

    return isUserId(record.user_id) && typeof record.banned === "boolean" && isAbsentOrString(record.ban_reason);
}

export function isLockChangedPayload(data: unknown): data is LockChangedPayload {
    const record = asRecord(data);
    if (record === null) {
        return false;
    }

    return isUserId(record.user_id) && typeof record.locked === "boolean" && isAbsentOrString(record.lock_reason);
}

export function isProfileChangedPayload(data: unknown): data is ProfileChangedPayload {
    const record = asRecord(data);
    if (record === null) {
        return false;
    }

    return isUserId(record.user_id) && isAbsentOrString(record.display_name) && isAbsentOrString(record.avatar_url);
}

export function isChatUnreadBumpedPayload(data: unknown): data is ChatUnreadBumpedPayload {
    const record = asRecord(data);
    if (record === null) {
        return false;
    }

    return isFiniteNumber(record.total);
}

export function isLiveGamesCountPayload(data: unknown): data is LiveGamesCountPayload {
    const record = asRecord(data);
    if (record === null) {
        return false;
    }

    return isFiniteNumber(record.count);
}

export const REALTIME_EVENT_GUARDS = {
    role_changed: isRoleChangedPayload,
    ban_changed: isBanChangedPayload,
    lock_changed: isLockChangedPayload,
    profile_changed: isProfileChangedPayload,
    chat_unread_bumped: isChatUnreadBumpedPayload,
    live_games_count: isLiveGamesCountPayload,
} as const satisfies RealtimeGuardTable;

export type GuardedRealtimeEventName = keyof typeof REALTIME_EVENT_GUARDS;

const GUARDED_EVENT_NAMES: ReadonlySet<string> = new Set<string>(Object.keys(REALTIME_EVENT_GUARDS));

export function hasRealtimeGuard(name: RealtimeEventName): name is GuardedRealtimeEventName {
    return GUARDED_EVENT_NAMES.has(name);
}

export function passesRealtimeGuard(event: RealtimeEvent): boolean {
    if (!hasRealtimeGuard(event.type)) {
        return true;
    }

    const guard: (data: unknown) => boolean = REALTIME_EVENT_GUARDS[event.type];

    return guard(event.data);
}
