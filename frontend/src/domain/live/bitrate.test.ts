import { describe, expect, it } from "vitest";
import {
    FPS_OPTIONS,
    MAX_BITRATE,
    MIN_BITRATE,
    STREAM_RESOLUTIONS,
    isBitrateValid,
    kbpsForBitsPerPixel,
    parseBitrate,
    recommendedBitrate,
    recommendedBitrateForResolution,
} from "./bitrate";

describe("STREAM_RESOLUTIONS", () => {
    it("carries the four offered resolutions in ascending pixel order", () => {
        // given
        const table = STREAM_RESOLUTIONS;

        // when
        const labels = table.map(resolution => resolution.label);
        const pixels = table.map(resolution => resolution.pixels);

        // then
        expect(labels).toEqual(["720p", "1080p", "1440p", "4K"]);
        expect(pixels).toEqual([921600, 2073600, 3686400, 8294400]);
    });

    it("offers thirty and sixty frames per second", () => {
        // given
        const options = FPS_OPTIONS;

        // when
        const values = [...options];

        // then
        expect(values).toEqual([30, 60]);
    });
});

describe("parseBitrate", () => {
    it("reads a numeric string, treating blank input as zero", () => {
        // given
        const entries = ["6000", " 6000 ", ""];

        // when
        const values = entries.map(parseBitrate);

        // then
        expect(values).toEqual([6000, 6000, 0]);
    });

    it("reports unparseable input as not a number", () => {
        // given
        const entry = "not a bitrate";

        // when
        const value = parseBitrate(entry);

        // then
        expect(Number.isNaN(value)).toBe(true);
    });
});

describe("isBitrateValid", () => {
    it("accepts the inclusive bounds and rejects either side of them", () => {
        // given
        const entries = [String(MIN_BITRATE), String(MAX_BITRATE), String(MIN_BITRATE - 1), String(MAX_BITRATE + 1)];

        // when
        const results = entries.map(isBitrateValid);

        // then
        expect(results).toEqual([true, true, false, false]);
    });

    it("rejects blank and unparseable input", () => {
        // given
        const entries = ["", "   ", "abc"];

        // when
        const results = entries.map(isBitrateValid);

        // then
        expect(results).toEqual([false, false, false]);
    });

    it("accepts a fractional bitrate inside the range", () => {
        // given
        const entry = "6000.5";

        // when
        const valid = isBitrateValid(entry);

        // then
        expect(valid).toBe(true);
    });
});

describe("kbpsForBitsPerPixel", () => {
    it("rounds to the nearest five hundred kilobits", () => {
        // given
        const pixelsPerSecond = 1280 * 720 * 30;

        // when
        const kbps = kbpsForBitsPerPixel(pixelsPerSecond, 0.07);

        // then
        expect(kbps).toBe(2000);
    });

    it("rounds a half step upwards", () => {
        // given
        const pixelsPerSecond = 250000;

        // when
        const kbps = kbpsForBitsPerPixel(pixelsPerSecond, 1);

        // then
        expect(kbps).toBe(500);
    });
});

describe("recommendedBitrate", () => {
    it("brackets 720p30 between two and three and a half megabits", () => {
        // given
        const pixels = 1280 * 720;

        // when
        const recommendation = recommendedBitrate(pixels, 30);

        // then
        expect(recommendation).toEqual({ low: 2000, typical: 2500, high: 3500 });
    });

    it("brackets 1080p60 around twelve megabits", () => {
        // given
        const pixels = 1920 * 1080;

        // when
        const recommendation = recommendedBitrate(pixels, 60);

        // then
        expect(recommendation).toEqual({ low: 8500, typical: 12000, high: 15000 });
    });

    it("recommends a 4K60 upper bound above the bitrate the form will accept", () => {
        // given
        const pixels = 3840 * 2160;

        // when
        const recommendation = recommendedBitrate(pixels, 60);

        // then
        expect(recommendation).toEqual({ low: 35000, typical: 47500, high: 59500 });
        expect(recommendation.high).toBeGreaterThan(MAX_BITRATE);
        expect(recommendation.typical).toBeLessThanOrEqual(MAX_BITRATE);
    });
});

describe("recommendedBitrateForResolution", () => {
    it("reads the resolution out of the offered table by index", () => {
        // given
        const index = 1;

        // when
        const recommendation = recommendedBitrateForResolution(index, 60);

        // then
        expect(recommendation).toEqual(recommendedBitrate(1920 * 1080, 60));
    });
});
