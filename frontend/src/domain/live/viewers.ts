export const VIEWER_IDENTITY_PREFIX = "viewer_";

export const MONITOR_IDENTITY_PREFIX = "monitor_";

export const BROADCASTER_IDENTITY_PREFIX = "broadcaster_";

export const UNNAMED_VIEWER = "Member";

export interface ViewerMeta {
    userId?: string;
    username?: string;
    avatarUrl?: string;
}

export interface ViewerParticipant {
    identity: string;
    name?: string;
    metadata?: string;
}

export interface ViewerChip {
    userId: string;
    name: string;
    avatar?: string;
}

export interface ViewerRoster {
    total: number;
    named: ViewerChip[];
    guests: number;
}

export function isViewerIdentity(identity: string): boolean {
    return identity.startsWith(VIEWER_IDENTITY_PREFIX);
}

export function selectViewers<T extends ViewerParticipant>(participants: readonly T[]): T[] {
    return participants.filter(participant => isViewerIdentity(participant.identity));
}

export function countViewers(participants: readonly ViewerParticipant[]): number {
    return selectViewers(participants).length;
}

export function readViewerMeta(
    participant: ViewerParticipant,
    resolveMeta: (meta: ViewerMeta) => ViewerMeta,
): ViewerMeta | null {
    if (!participant.metadata) {
        return null;
    }

    try {
        return resolveMeta(JSON.parse(participant.metadata) as ViewerMeta);
    } catch {
        return null;
    }
}

export function summariseViewers(
    participants: readonly ViewerParticipant[],
    resolveMeta: (meta: ViewerMeta) => ViewerMeta = meta => meta,
): ViewerRoster {
    const viewers = selectViewers(participants);

    const named = new Map<string, ViewerChip>();
    let guests = 0;

    for (const participant of viewers) {
        const meta = readViewerMeta(participant, resolveMeta);

        if (meta?.userId) {
            named.set(meta.userId, {
                userId: meta.userId,
                name: participant.name || meta.username || UNNAMED_VIEWER,
                avatar: meta.avatarUrl,
            });
        } else {
            guests += 1;
        }
    }

    return {
        total: viewers.length,
        named: Array.from(named.values()),
        guests,
    };
}
