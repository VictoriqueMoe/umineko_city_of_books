export const MAX_ROOM_TAGS = 10;
export const MAX_ROOM_TAG_LENGTH = 30;

export function normaliseRoomTag(raw: string): string {
    return raw
        .toLowerCase()
        .trim()
        .replace(/\s+/g, "-")
        .replace(/[^a-z0-9-]+/g, "")
        .replace(/^-+|-+$/g, "")
        .slice(0, MAX_ROOM_TAG_LENGTH);
}

export function isRoomTagCommitKey(key: string): boolean {
    return key === "Enter" || key === ",";
}

export function addRoomTags(existing: string[], raw: string): string[] {
    const parts = raw.split(",").map(normaliseRoomTag).filter(Boolean);
    if (parts.length === 0) {
        return existing;
    }

    const next = [...existing];
    for (const tag of parts) {
        if (next.length >= MAX_ROOM_TAGS) {
            break;
        }
        if (!next.includes(tag)) {
            next.push(tag);
        }
    }

    return next;
}

export function removeRoomTag(existing: string[], tag: string): string[] {
    return existing.filter(t => t !== tag);
}

export function finaliseRoomTags(tags: string[], pendingInput: string): string[] {
    return addRoomTags(tags, pendingInput);
}
