import { useEffect, useRef } from "react";
import type { UploadRef } from "../types/api";
import type { UseStagedMediaResult } from "./useStagedMedia";

export function useAttachWhenReady<T>(
    media: UseStagedMediaResult,
    attach: (ref: UploadRef) => Promise<T>,
    onAttached: (result: T) => void,
): void {
    const attaching = useRef(new Set<string>());
    const attachRef = useRef(attach);
    const onAttachedRef = useRef(onAttached);
    const { items, forget, fail } = media;

    useEffect(() => {
        attachRef.current = attach;
        onAttachedRef.current = onAttached;
    });

    useEffect(() => {
        for (const item of items) {
            if (item.state !== "ready" || item.uploadId === null || attaching.current.has(item.key)) {
                continue;
            }

            attaching.current.add(item.key);
            attachRef
                .current({ upload_id: item.uploadId, is_spoiler: item.isSpoiler })
                .then(result => {
                    onAttachedRef.current(result);
                    forget(item.key);
                })
                .catch((err: unknown) => {
                    fail(item.key, err instanceof Error ? err.message : "Could not attach this file");
                });
        }
    }, [items, forget, fail]);
}
