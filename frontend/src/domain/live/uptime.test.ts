import { describe, expect, it } from "vitest";
import { formatElapsed, streamUptimeLabel } from "./uptime";

describe("formatElapsed", () => {
    it("shows minutes and seconds, zero padded, below an hour", () => {
        // given
        const spans = [0, 5_000, 65_000, 599_000];

        // when
        const labels = spans.map(formatElapsed);

        // then
        expect(labels).toEqual(["00:00", "00:05", "01:05", "09:59"]);
    });

    it("adds an unpadded hour field once the stream passes an hour", () => {
        // given
        const spans = [3_600_000, 3_661_000, 36_000_000];

        // when
        const labels = spans.map(formatElapsed);

        // then
        expect(labels).toEqual(["1:00:00", "1:01:01", "10:00:00"]);
    });

    it("floors part seconds rather than rounding them", () => {
        // given
        const span = 1_999;

        // when
        const label = formatElapsed(span);

        // then
        expect(label).toBe("00:01");
    });

    it("clamps a negative span to zero, so a clock skew never shows a minus sign", () => {
        // given
        const span = -60_000;

        // when
        const label = formatElapsed(span);

        // then
        expect(label).toBe("00:00");
    });
});

describe("streamUptimeLabel", () => {
    it("measures from the start timestamp to now", () => {
        // given
        const startedAt = "2026-08-28T10:00:00.000Z";
        const now = Date.parse("2026-08-28T10:02:03.000Z");

        // when
        const label = streamUptimeLabel(startedAt, now);

        // then
        expect(label).toBe("02:03");
    });

    it("has no label when the stream never reported a start", () => {
        // given
        const startedAt = undefined;

        // when
        const label = streamUptimeLabel(startedAt, Date.now());

        // then
        expect(label).toBeNull();
    });

    it("has no label when the start timestamp cannot be parsed", () => {
        // given
        const startedAt = "not a timestamp";

        // when
        const label = streamUptimeLabel(startedAt, Date.now());

        // then
        expect(label).toBeNull();
    });
});
