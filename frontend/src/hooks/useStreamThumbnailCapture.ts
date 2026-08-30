import { useEffect, useRef, type RefObject } from "react";
import type { Room } from "livekit-client";
import {
    THUMBNAIL_CAPTURE_INTERVAL_MS,
    THUMBNAIL_FIRST_CAPTURE_MS,
    THUMBNAIL_MIME_TYPE,
    THUMBNAIL_QUALITY,
    thumbnailSize,
} from "../domain/live/thumbnail";
import { errorMessage } from "../utils/errorMessage";
import { useUploadStreamThumbnail } from "./mutations/stream";

export const THUMBNAIL_UPLOAD_FAILED = "Your stream thumbnail is not updating.";

export interface UseStreamThumbnailCaptureOptions {
    streamId: string | undefined;
    isOwnStream: boolean;
    isLive: boolean;
    room: Room | null;
    stageRef: RefObject<HTMLDivElement | null>;
}

export interface StreamThumbnailCapture {
    lastError: string;
}

export function useStreamThumbnailCapture(options: UseStreamThumbnailCaptureOptions): StreamThumbnailCapture {
    const { streamId, isOwnStream, isLive, room, stageRef } = options;

    const upload = useUploadStreamThumbnail();
    const uploadRef = useRef(upload.mutate);

    useEffect(() => {
        uploadRef.current = upload.mutate;
    });

    useEffect(() => {
        if (!isOwnStream || !isLive || !room || !streamId) {
            return;
        }

        let stopped = false;

        const capture = () => {
            if (stopped || document.visibilityState !== "visible") {
                return;
            }

            const video = stageRef.current?.querySelector<HTMLVideoElement>("video");
            if (!video) {
                return;
            }

            const size = thumbnailSize(video.videoWidth, video.videoHeight);
            if (!size) {
                return;
            }

            const canvas = document.createElement("canvas");
            canvas.width = size.width;
            canvas.height = size.height;

            const context = canvas.getContext("2d");
            if (!context) {
                return;
            }

            context.drawImage(video, 0, 0, size.width, size.height);
            canvas.toBlob(
                blob => {
                    if (!blob || stopped) {
                        return;
                    }

                    uploadRef.current({ streamId, blob });
                },
                THUMBNAIL_MIME_TYPE,
                THUMBNAIL_QUALITY,
            );
        };

        const initial = window.setTimeout(capture, THUMBNAIL_FIRST_CAPTURE_MS);
        const interval = window.setInterval(capture, THUMBNAIL_CAPTURE_INTERVAL_MS);

        return () => {
            stopped = true;
            window.clearTimeout(initial);
            window.clearInterval(interval);
        };
    }, [isOwnStream, isLive, room, streamId, stageRef]);

    if (!upload.error) {
        return { lastError: "" };
    }

    return { lastError: `${THUMBNAIL_UPLOAD_FAILED} ${errorMessage(upload.error, "The upload failed.")}` };
}
