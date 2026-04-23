function escapeRegex(s: string): string {
    return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

export function buildMentionMatcher(username: string | undefined): ((body: string) => boolean) | null {
    if (!username) {
        return null;
    }
    const re = new RegExp(`@${escapeRegex(username)}(?![a-zA-Z0-9_])`, "i");
    return body => re.test(body);
}
