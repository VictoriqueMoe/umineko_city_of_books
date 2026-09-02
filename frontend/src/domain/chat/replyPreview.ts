import { ellipsise } from "../../utils/text";

const REPLY_PREVIEW_MAX = 80;

export function replyPreview(body: string): string {
    return ellipsise(body, REPLY_PREVIEW_MAX);
}
