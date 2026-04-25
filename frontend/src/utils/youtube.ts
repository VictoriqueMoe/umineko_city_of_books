const YOUTUBE_ID_RE =
    /https?:\/\/(?:www\.|m\.)?(?:youtube\.com\/(?:watch\?v=|embed\/|shorts\/)|youtu\.be\/)([a-zA-Z0-9_-]{11})/g;

export function extractYouTubeIDs(text: string, limit = 2): string[] {
    const ids: string[] = [];
    const seen = new Set<string>();
    YOUTUBE_ID_RE.lastIndex = 0;
    let match: RegExpExecArray | null = YOUTUBE_ID_RE.exec(text);
    while (match !== null) {
        const id = match[1];
        if (!seen.has(id)) {
            seen.add(id);
            ids.push(id);
            if (ids.length >= limit) {
                break;
            }
        }
        match = YOUTUBE_ID_RE.exec(text);
    }
    return ids;
}
