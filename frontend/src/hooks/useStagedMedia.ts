import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { REALTIME_EVENTS, type UploadSettledPayload } from "../api/realtime/events";
import { useRealtimeEvent, useRealtimeStatus } from "../api/realtime/useRealtime";
import { uploadPercent } from "../domain/uploadProgress";
import type { PendingUploadStatus, UploadKind, UploadRef } from "../types/api";
import { useBeginUpload, useRemoveUpload, useUploadStatuses } from "./mutations/uploads";

export type AttachmentState = "uploading" | PendingUploadStatus;

export type StagedAttachment = {
    key: string;
    file: File;
    isSpoiler: boolean;
    state: AttachmentState;
    percent: number;
    uploadId: string | null;
    error: string | null;
};

export type AttachmentProgress = Pick<StagedAttachment, "state" | "percent" | "error">;

export type UseStagedMediaResult = {
    items: StagedAttachment[];
    files: File[];
    spoilers: boolean[];
    progress: AttachmentProgress[];
    refs: UploadRef[];
    ready: boolean;
    busy: boolean;
    add: (files: File[]) => void;
    remove: (index: number) => void;
    reorder: (from: number, to: number) => void;
    toggleSpoiler: (index: number) => void;
    forget: (key: string) => void;
    fail: (key: string, error: string) => void;
    reset: () => void;
};

function patch(items: StagedAttachment[], key: string, change: Partial<StagedAttachment>): StagedAttachment[] {
    return items.map(item => (item.key === key ? { ...item, ...change } : item));
}

function errorMessage(err: unknown): string {
    return err instanceof Error ? err.message : "Upload failed";
}

export function useStagedMedia(kind: UploadKind = "media"): UseStagedMediaResult {
    const [items, setItems] = useState<StagedAttachment[]>([]);
    const begin = useBeginUpload(kind).mutateAsync;
    const removeUpload = useRemoveUpload().mutateAsync;
    const fetchStatuses = useUploadStatuses().mutateAsync;
    const itemsRef = useRef(items);
    const aborts = useRef(new Map<string, AbortController>());
    const known = useRef(new Set<string>());
    const settledEarly = useRef(new Map<string, UploadSettledPayload>());

    useEffect(() => {
        itemsRef.current = items;
    }, [items]);

    useEffect(() => {
        const inFlight = aborts.current;

        return () => {
            for (const controller of inFlight.values()) {
                controller.abort();
            }
        };
    }, []);

    const settle = useCallback((upload: UploadSettledPayload) => {
        if (!known.current.has(upload.id)) {
            settledEarly.current.set(upload.id, upload);
            return;
        }

        setItems(prev => prev.map(item => (item.uploadId === upload.id ? { ...item, state: upload.status } : item)));
    }, []);

    useRealtimeEvent([REALTIME_EVENTS.UPLOAD_READY, REALTIME_EVENTS.UPLOAD_FAILED], event => {
        settle(event.data);
    });

    const epoch = useRealtimeStatus();

    useEffect(() => {
        const waiting = itemsRef.current.filter(item => item.state === "processing" && item.uploadId !== null);
        if (waiting.length === 0) {
            return;
        }

        const ids = waiting.map(item => item.uploadId as string);
        fetchStatuses(ids)
            .then(uploads => {
                const found = new Set(uploads.map(upload => upload.id));
                for (const upload of uploads) {
                    settle(upload);
                }
                setItems(prev =>
                    prev.map(item =>
                        item.uploadId !== null && ids.includes(item.uploadId) && !found.has(item.uploadId)
                            ? { ...item, state: "failed", error: "This upload expired, add it again" }
                            : item,
                    ),
                );
            })
            .catch(() => undefined);
    }, [epoch, fetchStatuses, settle]);

    const add = useCallback(
        (incoming: File[]) => {
            for (const file of incoming) {
                const key = crypto.randomUUID();
                const controller = new AbortController();
                aborts.current.set(key, controller);
                setItems(prev => [
                    ...prev,
                    { key, file, isSpoiler: false, state: "uploading", percent: 0, uploadId: null, error: null },
                ]);

                begin({
                    file,
                    signal: controller.signal,
                    onProgress: (sent, total) => {
                        setItems(prev => patch(prev, key, { percent: uploadPercent(sent, total) }));
                    },
                })
                    .then(upload => {
                        aborts.current.delete(key);
                        known.current.add(upload.id);
                        const early = settledEarly.current.get(upload.id);
                        settledEarly.current.delete(upload.id);
                        setItems(prev =>
                            patch(prev, key, {
                                uploadId: upload.id,
                                percent: 100,
                                state: early?.status ?? upload.status,
                            }),
                        );
                    })
                    .catch((err: unknown) => {
                        aborts.current.delete(key);
                        if (controller.signal.aborted) {
                            return;
                        }

                        setItems(prev => patch(prev, key, { state: "failed", error: errorMessage(err) }));
                    });
            }
        },
        [begin],
    );

    const remove = useCallback(
        (index: number) => {
            const item = itemsRef.current[index];
            if (!item) {
                return;
            }

            aborts.current.get(item.key)?.abort();
            aborts.current.delete(item.key);
            if (item.uploadId !== null) {
                removeUpload(item.uploadId).catch(() => undefined);
            }

            setItems(prev => prev.filter(other => other.key !== item.key));
        },
        [removeUpload],
    );

    const reorder = useCallback((from: number, to: number) => {
        setItems(prev => {
            const next = prev.slice();
            const [moved] = next.splice(from, 1);
            next.splice(to, 0, moved);

            return next;
        });
    }, []);

    const toggleSpoiler = useCallback((index: number) => {
        setItems(prev => prev.map((item, i) => (i === index ? { ...item, isSpoiler: !item.isSpoiler } : item)));
    }, []);

    const forget = useCallback((key: string) => {
        setItems(prev => prev.filter(item => item.key !== key));
    }, []);

    const fail = useCallback((key: string, error: string) => {
        setItems(prev => patch(prev, key, { state: "failed", error }));
    }, []);

    const reset = useCallback(() => {
        setItems([]);
    }, []);

    const files = useMemo(() => items.map(item => item.file), [items]);
    const spoilers = useMemo(() => items.map(item => item.isSpoiler), [items]);
    const progress = useMemo(
        () => items.map(item => ({ state: item.state, percent: item.percent, error: item.error })),
        [items],
    );
    const refs = useMemo(
        () =>
            items
                .filter(item => item.uploadId !== null)
                .map(item => ({ upload_id: item.uploadId as string, is_spoiler: item.isSpoiler })),
        [items],
    );
    const ready = items.every(item => item.state === "ready");
    const busy = items.some(item => item.state === "uploading" || item.state === "processing");

    return {
        items,
        files,
        spoilers,
        progress,
        refs,
        ready,
        busy,
        add,
        remove,
        reorder,
        toggleSpoiler,
        forget,
        fail,
        reset,
    };
}
