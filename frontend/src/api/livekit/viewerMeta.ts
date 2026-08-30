import type { ViewerMeta } from "../../domain/live/viewers";
import { absolutizeMedia } from "../client";

export function resolveViewerMeta(meta: ViewerMeta): ViewerMeta {
    return absolutizeMedia(meta);
}
