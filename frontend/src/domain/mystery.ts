import type { MysteryAttempt, MysteryClue, User } from "../types/api";
import { parseServerDate } from "../utils/time";
import { contentPermissions, isContentOwner, type ContentViewer } from "./contentPermissions";

export interface AttemptGroup {
    author: User;
    attempts: MysteryAttempt[];
}

export interface AttemptGroupOptions {
    freeForAll: boolean;
    viewerId: string | null | undefined;
}

export interface MysterySubject {
    author: { id: string };
}

export interface MysteryPermissions {
    isAuthor: boolean;
    canEdit: boolean;
    canDelete: boolean;
    canSeeAsGameMaster: boolean;
}

const TIMER_SOLVED = "#66bb6a";
const TIMER_FRESH = "#64b5f6";
const TIMER_WEEK = "#ffd54f";
const TIMER_MONTH = "#ffb74d";
const TIMER_STALE = "#e57373";

const DAY_MS = 86400000;

export function findWinningAttempt(attempts: MysteryAttempt[]): MysteryAttempt | null {
    for (const attempt of attempts) {
        if (attempt.is_winner) {
            return attempt;
        }

        if (attempt.replies && attempt.replies.length > 0) {
            const nested = findWinningAttempt(attempt.replies);
            if (nested) {
                return nested;
            }
        }
    }

    return null;
}

export function groupAttemptsByAuthor(attempts: MysteryAttempt[], options: AttemptGroupOptions): AttemptGroup[] {
    const groups = new Map<string, AttemptGroup>();

    for (const attempt of attempts) {
        const existing = groups.get(attempt.author.id);
        if (existing) {
            existing.attempts.push(attempt);
            continue;
        }
        groups.set(attempt.author.id, { author: attempt.author, attempts: [attempt] });
    }

    const result = Array.from(groups.values());

    if (!options.freeForAll || !options.viewerId) {
        return result;
    }

    result.sort((a, b) => {
        if (a.author.id === options.viewerId) {
            return -1;
        }
        if (b.author.id === options.viewerId) {
            return 1;
        }
        return 0;
    });

    return result;
}

export function winningAuthorIds(attempts: MysteryAttempt[]): Set<string> {
    const winners = new Set<string>();

    for (const attempt of attempts) {
        if (attempt.is_winner) {
            winners.add(attempt.author.id);
        }
    }

    return winners;
}

export function globalClues(clues: MysteryClue[]): MysteryClue[] {
    return clues.filter(clue => !clue.player_id);
}

export function cluesForPlayer(clues: MysteryClue[], playerId: string): MysteryClue[] {
    return clues.filter(clue => clue.player_id === playerId);
}

export function mysteryPermissions(
    viewer: ContentViewer | null | undefined,
    mystery: MysterySubject,
): MysteryPermissions {
    const subject = { family: "mystery", authorId: mystery.author.id } as const;
    const isAuthor = isContentOwner(viewer, subject);
    const { canEdit, canDelete } = contentPermissions(viewer, subject);

    return {
        isAuthor,
        canEdit,
        canDelete,
        canSeeAsGameMaster: isAuthor || viewer?.role === "super_admin",
    };
}

export function timerColour(createdAt: string, solved: boolean, now: number = Date.now()): string {
    if (solved) {
        return TIMER_SOLVED;
    }

    const created = parseServerDate(createdAt);
    if (!created) {
        return TIMER_FRESH;
    }

    const days = (now - created.getTime()) / DAY_MS;

    if (days < 1) {
        return TIMER_FRESH;
    }
    if (days < 7) {
        return TIMER_WEEK;
    }
    if (days < 30) {
        return TIMER_MONTH;
    }

    return TIMER_STALE;
}
