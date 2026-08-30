export const THUMBNAIL_WIDTH = 480;

export const THUMBNAIL_FALLBACK_HEIGHT = 270;

export const THUMBNAIL_MIME_TYPE = "image/webp";

export const THUMBNAIL_QUALITY = 0.7;

export const THUMBNAIL_FIRST_CAPTURE_MS = 8000;

export const THUMBNAIL_CAPTURE_INTERVAL_MS = 50000;

export interface ThumbnailSize {
    width: number;
    height: number;
}

export function canCaptureThumbnail(videoWidth: number): boolean {
    return videoWidth !== 0;
}

export function thumbnailSize(videoWidth: number, videoHeight: number): ThumbnailSize | null {
    if (!canCaptureThumbnail(videoWidth)) {
        return null;
    }

    return {
        width: THUMBNAIL_WIDTH,
        height: Math.round((videoHeight / videoWidth) * THUMBNAIL_WIDTH) || THUMBNAIL_FALLBACK_HEIGHT,
    };
}
