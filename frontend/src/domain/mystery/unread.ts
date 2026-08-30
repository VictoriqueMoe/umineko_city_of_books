import type { MysteryAttempt } from "../../types/api";
import { parseServerDate } from "../../utils/time";

export interface UnreadAuthorsInput {
    attempts: MysteryAttempt[];
    cursor: string | null;
    viewerId: string | null | undefined;
}

export function readCursorKey(mysteryId: string): string {
    return `mystery-read-cursor-${mysteryId}`;
}

function timestamp(value: string | null | undefined): number {
    const parsed = parseServerDate(value);

    return parsed ? parsed.getTime() : 0;
}

export function unreadAuthorIds({ attempts, cursor, viewerId }: UnreadAuthorsInput): Set<string> {
    const unread = new Set<string>();

    if (!cursor) {
        return unread;
    }

    const cursorMs = timestamp(cursor);

    for (const attempt of attempts) {
        if (timestamp(attempt.created_at) > cursorMs && attempt.author.id !== viewerId) {
            unread.add(attempt.author.id);
        }
    }

    return unread;
}
