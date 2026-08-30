export interface MentionCandidate {
    viewer_follows: boolean;
    follows_viewer: boolean;
}

export function mentionAffinity(candidate: MentionCandidate): number {
    return (candidate.viewer_follows ? 2 : 0) + (candidate.follows_viewer ? 1 : 0);
}

export function compareMentionCandidates(a: MentionCandidate, b: MentionCandidate): number {
    return mentionAffinity(b) - mentionAffinity(a);
}

export function rankMentionCandidates<T extends MentionCandidate>(candidates: readonly T[]): T[] {
    return [...candidates].sort(compareMentionCandidates);
}
