const REPLY_PREVIEW_MAX = 80;

export function replyPreview(body: string): string {
    return body.length > REPLY_PREVIEW_MAX ? body.slice(0, REPLY_PREVIEW_MAX) + "..." : body;
}
