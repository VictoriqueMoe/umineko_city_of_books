import type { QueryClient } from "@tanstack/react-query";
import type { User } from "../types/api";

export type UserPatch = Partial<
    Pick<User, "banned" | "ban_reason" | "locked" | "lock_reason" | "role" | "display_name" | "avatar_url">
>;

function isUserShape(
    v: unknown,
): v is Record<string, unknown> & { id: string; username: string; display_name: string } {
    if (v === null || typeof v !== "object") {
        return false;
    }
    const obj = v as Record<string, unknown>;
    return typeof obj.id === "string" && typeof obj.username === "string" && typeof obj.display_name === "string";
}

function walk(value: unknown, userId: string, patch: UserPatch): unknown {
    if (value === null || typeof value !== "object") {
        return value;
    }

    if (Array.isArray(value)) {
        let changed = false;
        const next: unknown[] = new Array(value.length);
        for (let i = 0; i < value.length; i++) {
            const child = walk(value[i], userId, patch);
            if (child !== value[i]) {
                changed = true;
            }
            next[i] = child;
        }
        return changed ? next : value;
    }

    const obj = value as Record<string, unknown>;
    let childChanged = false;
    const next: Record<string, unknown> = {};
    for (const key of Object.keys(obj)) {
        const child = walk(obj[key], userId, patch);
        if (child !== obj[key]) {
            childChanged = true;
        }
        next[key] = child;
    }

    let selfChanged = false;
    if (isUserShape(obj) && obj.id === userId) {
        for (const key of Object.keys(patch) as (keyof UserPatch)[]) {
            const patchValue = patch[key];
            if (patchValue !== undefined && next[key] !== patchValue) {
                next[key] = patchValue;
                selfChanged = true;
            }
        }
    }

    if (!childChanged && !selfChanged) {
        return value;
    }
    return next;
}

export function patchUserInCache(qc: QueryClient, userId: string, patch: UserPatch): void {
    qc.setQueriesData({ predicate: () => true }, (data: unknown) => walk(data, userId, patch));
}
