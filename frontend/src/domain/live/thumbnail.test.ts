import { describe, expect, it } from "vitest";
import {
    THUMBNAIL_CAPTURE_INTERVAL_MS,
    THUMBNAIL_FALLBACK_HEIGHT,
    THUMBNAIL_FIRST_CAPTURE_MS,
    THUMBNAIL_MIME_TYPE,
    THUMBNAIL_QUALITY,
    THUMBNAIL_WIDTH,
    canCaptureThumbnail,
    thumbnailSize,
} from "./thumbnail";

describe("canCaptureThumbnail", () => {
    it("refuses a video element that has not reported its dimensions yet", () => {
        // given
        const videoWidth = 0;

        // when
        const canCapture = canCaptureThumbnail(videoWidth);

        // then
        expect(canCapture).toBe(false);
    });

    it("allows a video element that has dimensions", () => {
        // given
        const videoWidth = 1920;

        // when
        const canCapture = canCaptureThumbnail(videoWidth);

        // then
        expect(canCapture).toBe(true);
    });
});

describe("thumbnailSize", () => {
    it("fixes the width and scales the height to the source aspect ratio", () => {
        // given
        const sources = [
            [1920, 1080],
            [1280, 720],
            [640, 480],
        ];

        // when
        const sizes = sources.map(([width, height]) => thumbnailSize(width, height));

        // then
        expect(sizes).toEqual([
            { width: THUMBNAIL_WIDTH, height: 270 },
            { width: THUMBNAIL_WIDTH, height: 270 },
            { width: THUMBNAIL_WIDTH, height: 360 },
        ]);
    });

    it("rounds a height that does not divide evenly", () => {
        // given
        const portrait = [1080, 1920];

        // when
        const size = thumbnailSize(portrait[0], portrait[1]);

        // then
        expect(size).toEqual({ width: THUMBNAIL_WIDTH, height: 853 });
    });

    it("falls back to a sixteen by nine height when the source height is zero", () => {
        // given
        const videoWidth = 1920;

        // when
        const size = thumbnailSize(videoWidth, 0);

        // then
        expect(size).toEqual({ width: THUMBNAIL_WIDTH, height: THUMBNAIL_FALLBACK_HEIGHT });
    });

    it("falls back when the source height is not a number", () => {
        // given
        const videoWidth = 1920;

        // when
        const size = thumbnailSize(videoWidth, Number.NaN);

        // then
        expect(size).toEqual({ width: THUMBNAIL_WIDTH, height: THUMBNAIL_FALLBACK_HEIGHT });
    });

    it("has no size at all when the video has not reported its width", () => {
        // given
        const videoWidth = 0;

        // when
        const size = thumbnailSize(videoWidth, 1080);

        // then
        expect(size).toBeNull();
    });
});

describe("capture settings", () => {
    it("encodes lossy webp and waits before the first capture, then repeats slowly", () => {
        // given
        const settings = {
            mime: THUMBNAIL_MIME_TYPE,
            quality: THUMBNAIL_QUALITY,
            first: THUMBNAIL_FIRST_CAPTURE_MS,
            interval: THUMBNAIL_CAPTURE_INTERVAL_MS,
        };

        // when
        const values = { ...settings };

        // then
        expect(values).toEqual({ mime: "image/webp", quality: 0.7, first: 8000, interval: 50000 });
    });
});
