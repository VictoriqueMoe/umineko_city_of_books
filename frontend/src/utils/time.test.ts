import { beforeEach, describe, expect, it, vi } from "vitest";
import {
    formatActiveLabel,
    formatDate,
    formatExactDateTime,
    formatFullDateTime,
    formatMessageTime,
    formatShortDateTime,
    formatTimeOfDay,
    parseServerDate,
    relativeTime,
    shortRelativeTime,
} from "./time";

const NOW = "2026-06-15T12:00:00Z";
const TIME_PART = new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit" });
const SHORT_DATE_PART = new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short" });
const SHORT_DATE_YEAR_PART = new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short", year: "numeric" });

function ago(seconds: number): string {
    return new Date(Date.parse(NOW) - seconds * 1000).toISOString();
}

function pinNow(): void {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(NOW));
}

describe("parseServerDate", () => {
    it("parses an ISO timestamp that already carries a zone", () => {
        // given
        const input = "2026-06-15T12:00:00Z";

        // when
        const parsed = parseServerDate(input);

        // then
        expect(parsed?.toISOString()).toBe("2026-06-15T12:00:00.000Z");
    });

    it("treats a space separated server timestamp without a zone as UTC", () => {
        // given
        const input = "2026-06-15 12:00:00";

        // when
        const parsed = parseServerDate(input);

        // then
        expect(parsed?.toISOString()).toBe("2026-06-15T12:00:00.000Z");
    });

    it("honours an explicit offset with or without a colon", () => {
        // given
        const withColon = "2026-06-15T14:00:00+02:00";
        const withoutColon = "2026-06-15T14:00:00+0200";

        // when
        const a = parseServerDate(withColon);
        const b = parseServerDate(withoutColon);

        // then
        expect(a?.toISOString()).toBe("2026-06-15T12:00:00.000Z");
        expect(b?.toISOString()).toBe("2026-06-15T12:00:00.000Z");
    });

    it("keeps sub second precision from a postgres style timestamp", () => {
        // given
        const input = "2026-06-15 12:00:00.123456+00:00";

        // when
        const parsed = parseServerDate(input);

        // then
        expect(parsed?.toISOString()).toBe("2026-06-15T12:00:00.123Z");
    });

    it("trims surrounding whitespace before parsing", () => {
        // given
        const input = "  2026-06-15T12:00:00Z  ";

        // when
        const parsed = parseServerDate(input);

        // then
        expect(parsed?.toISOString()).toBe("2026-06-15T12:00:00.000Z");
    });

    it("returns null for missing or blank input", () => {
        // given / when / then
        expect(parseServerDate(null)).toBeNull();
        expect(parseServerDate(undefined)).toBeNull();
        expect(parseServerDate("")).toBeNull();
        expect(parseServerDate("   ")).toBeNull();
    });

    it("returns null for a string that is not a date", () => {
        // given / when / then
        expect(parseServerDate("not a date")).toBeNull();
        expect(parseServerDate("2026-13-45T99:00:00Z")).toBeNull();
    });
});

describe("relativeTime", () => {
    beforeEach(() => {
        pinNow();
    });

    it("returns an empty string when there is no date", () => {
        // given / when / then
        expect(relativeTime(null)).toBe("");
        expect(relativeTime(undefined)).toBe("");
        expect(relativeTime("nonsense")).toBe("");
    });

    it("calls anything under a minute just now", () => {
        // given / when / then
        expect(relativeTime(ago(0))).toBe("just now");
        expect(relativeTime(ago(59))).toBe("just now");
    });

    it("switches to minutes at the one minute boundary", () => {
        // given / when / then
        expect(relativeTime(ago(60))).toBe("1m ago");
        expect(relativeTime(ago(59 * 60 + 59))).toBe("59m ago");
    });

    it("switches to hours at the one hour boundary", () => {
        // given / when / then
        expect(relativeTime(ago(60 * 60))).toBe("1h ago");
        expect(relativeTime(ago(24 * 60 * 60 - 1))).toBe("23h ago");
    });

    it("switches to days at the one day boundary", () => {
        // given / when / then
        expect(relativeTime(ago(24 * 60 * 60))).toBe("1d ago");
        expect(relativeTime(ago(30 * 24 * 60 * 60 - 1))).toBe("29d ago");
    });

    it("switches to thirty day months at the thirty day boundary", () => {
        // given / when / then
        expect(relativeTime(ago(30 * 24 * 60 * 60))).toBe("1mo ago");
        expect(relativeTime(ago(59 * 24 * 60 * 60))).toBe("1mo ago");
        expect(relativeTime(ago(60 * 24 * 60 * 60))).toBe("2mo ago");
    });

    it("keeps counting in months rather than rolling over into years", () => {
        // given
        const twoYears = 730 * 24 * 60 * 60;

        // when
        const result = relativeTime(ago(twoYears));

        // then
        expect(result).toBe("24mo ago");
    });

    it("treats a future timestamp as just now", () => {
        // given
        const future = ago(-3600);

        // when
        const result = relativeTime(future);

        // then
        expect(result).toBe("just now");
    });
});

describe("shortRelativeTime", () => {
    beforeEach(() => {
        pinNow();
    });

    it("returns an empty string when there is no date", () => {
        // given / when / then
        expect(shortRelativeTime(null)).toBe("");
        expect(shortRelativeTime("")).toBe("");
    });

    it("drops the ago suffix from every unit but keeps just now", () => {
        // given / when / then
        expect(shortRelativeTime(ago(5))).toBe("just now");
        expect(shortRelativeTime(ago(60))).toBe("1m");
        expect(shortRelativeTime(ago(2 * 60 * 60))).toBe("2h");
        expect(shortRelativeTime(ago(3 * 24 * 60 * 60))).toBe("3d");
        expect(shortRelativeTime(ago(90 * 24 * 60 * 60))).toBe("3mo");
    });

    it("uses the same boundaries as the long form", () => {
        // given / when / then
        expect(shortRelativeTime(ago(59))).toBe("just now");
        expect(shortRelativeTime(ago(59 * 60 + 59))).toBe("59m");
        expect(shortRelativeTime(ago(24 * 60 * 60 - 1))).toBe("23h");
        expect(shortRelativeTime(ago(30 * 24 * 60 * 60 - 1))).toBe("29d");
    });
});

describe("formatMessageTime", () => {
    beforeEach(() => {
        pinNow();
    });

    it("returns an empty string when there is no date", () => {
        // given / when / then
        expect(formatMessageTime(null)).toBe("");
        expect(formatMessageTime("rubbish")).toBe("");
    });

    it("shows only the clock time for a message sent earlier today", () => {
        // given
        const today = new Date();
        today.setHours(9, 5, 0, 0);

        // when
        const result = formatMessageTime(today.toISOString());

        // then
        expect(result).toBe(TIME_PART.format(today));
    });

    it("labels a message from the previous day as yesterday", () => {
        // given
        const yesterday = new Date();
        yesterday.setDate(yesterday.getDate() - 1);
        yesterday.setHours(22, 30, 0, 0);

        // when
        const result = formatMessageTime(yesterday.toISOString());

        // then
        expect(result).toBe(`Yesterday ${TIME_PART.format(yesterday)}`);
    });

    it("shows a day and month without the year for an older message from this year", () => {
        // given
        const earlier = new Date();
        earlier.setMonth(earlier.getMonth() - 3, 10);
        earlier.setHours(14, 45, 0, 0);

        // when
        const result = formatMessageTime(earlier.toISOString());

        // then
        expect(result).toBe(`${SHORT_DATE_PART.format(earlier)} ${TIME_PART.format(earlier)}`);
        expect(result).not.toContain("2026");
    });

    it("includes the year for a message from a previous year", () => {
        // given
        const older = new Date();
        older.setFullYear(older.getFullYear() - 2, 4, 20);
        older.setHours(8, 0, 0, 0);

        // when
        const result = formatMessageTime(older.toISOString());

        // then
        expect(result).toBe(`${SHORT_DATE_YEAR_PART.format(older)} ${TIME_PART.format(older)}`);
        expect(result).toContain("2024");
    });
});

describe("formatActiveLabel", () => {
    beforeEach(() => {
        pinNow();
    });

    it("says there is no activity when the date is missing", () => {
        // given / when / then
        expect(formatActiveLabel(null)).toBe("No activity yet");
        expect(formatActiveLabel(undefined)).toBe("No activity yet");
        expect(formatActiveLabel("not a date")).toBe("No activity yet");
    });

    it("reports anything under a minute as active just now", () => {
        // given / when / then
        expect(formatActiveLabel(ago(0))).toBe("Active just now");
        expect(formatActiveLabel(ago(59))).toBe("Active just now");
    });

    it("counts minutes, hours and days as the gap widens", () => {
        // given / when / then
        expect(formatActiveLabel(ago(60))).toBe("Active 1m ago");
        expect(formatActiveLabel(ago(59 * 60))).toBe("Active 59m ago");
        expect(formatActiveLabel(ago(60 * 60))).toBe("Active 1h ago");
        expect(formatActiveLabel(ago(23 * 60 * 60))).toBe("Active 23h ago");
        expect(formatActiveLabel(ago(24 * 60 * 60))).toBe("Active 1d ago");
        expect(formatActiveLabel(ago(29 * 24 * 60 * 60))).toBe("Active 29d ago");
    });

    it("falls back to an absolute date once thirty days have passed", () => {
        // given
        const stale = ago(30 * 24 * 60 * 60);

        // when
        const result = formatActiveLabel(stale);

        // then
        expect(result).toBe(`Active ${new Date(stale).toLocaleDateString()}`);
    });
});

describe("formatFullDateTime", () => {
    it("returns an empty string when there is no date", () => {
        // given / when / then
        expect(formatFullDateTime(null)).toBe("");
        expect(formatFullDateTime("")).toBe("");
        expect(formatFullDateTime("not a date")).toBe("");
    });

    it("renders the date and time in the requested locale", () => {
        // given
        const local = new Date(2026, 0, 15, 12, 0, 0);

        // when
        const result = formatFullDateTime(local.toISOString(), "en-GB");

        // then
        expect(result).toBe("15/01/2026, 12:00:00");
    });

    it("respects a different locale ordering", () => {
        // given
        const local = new Date(2026, 0, 15, 12, 0, 0);

        // when
        const result = formatFullDateTime(local.toISOString(), "en-US");

        // then
        expect(result).toContain("1/15/2026");
    });
});

describe("formatDate", () => {
    it("returns an empty string when there is no date", () => {
        // given / when / then
        expect(formatDate(null)).toBe("");
        expect(formatDate(undefined)).toBe("");
        expect(formatDate("not a date")).toBe("");
    });

    it("renders day first for a British locale and month first for an American one", () => {
        // given
        const local = new Date(2026, 0, 15, 12, 0, 0);

        // when
        const gb = formatDate(local.toISOString(), "en-GB");
        const us = formatDate(local.toISOString(), "en-US");

        // then
        expect(gb).toBe("15/01/2026");
        expect(us).toBe("1/15/2026");
    });

    it("accepts a list of locales", () => {
        // given
        const local = new Date(2026, 0, 15, 12, 0, 0);

        // when
        const result = formatDate(local.toISOString(), ["en-GB"]);

        // then
        expect(result).toBe("15/01/2026");
    });
});

describe("formatTimeOfDay", () => {
    it("returns an empty string when there is no date", () => {
        // given / when / then
        expect(formatTimeOfDay(null)).toBe("");
        expect(formatTimeOfDay("  ")).toBe("");
    });

    it("pads the hour to two digits on a 24 hour locale", () => {
        // given
        const local = new Date(2026, 0, 15, 9, 5, 0);

        // when
        const result = formatTimeOfDay(local.toISOString(), "en-GB");

        // then
        expect(result).toBe("09:05");
    });

    it("uses a meridiem on a 12 hour locale", () => {
        // given
        const local = new Date(2026, 0, 15, 21, 5, 0);

        // when
        const result = formatTimeOfDay(local.toISOString(), "en-US");

        // then
        expect(result).toMatch(/^09:05\sPM$/);
    });

    it("drops the seconds", () => {
        // given
        const local = new Date(2026, 0, 15, 9, 5, 42);

        // when
        const result = formatTimeOfDay(local.toISOString(), "en-GB");

        // then
        expect(result).toBe("09:05");
    });
});

describe("formatShortDateTime", () => {
    const SHORT_DATE_TIME_PART = new Intl.DateTimeFormat(undefined, {
        month: "short",
        day: "numeric",
        hour: "2-digit",
        minute: "2-digit",
    });

    it("returns an empty string when there is no date", () => {
        // given / when / then
        expect(formatShortDateTime(null)).toBe("");
        expect(formatShortDateTime(undefined)).toBe("");
        expect(formatShortDateTime("")).toBe("");
        expect(formatShortDateTime("not a date")).toBe("");
    });

    it("pairs an abbreviated date with a clock time and leaves the year out", () => {
        // given
        const local = new Date(2026, 0, 15, 14, 30, 0);

        // when
        const result = formatShortDateTime(local.toISOString());

        // then
        expect(result).toBe(SHORT_DATE_TIME_PART.format(local));
        expect(result).not.toContain("2026");
    });
});

describe("formatExactDateTime", () => {
    const britishCases: { name: string; local: Date; expected: string }[] = [
        {
            name: "spells the month out and keeps a 24 hour clock",
            local: new Date(2026, 0, 15, 14, 30, 0),
            expected: "15 January 2026 at 14:30",
        },
        {
            name: "pads a single digit hour to two digits",
            local: new Date(2026, 0, 15, 9, 5, 0),
            expected: "15 January 2026 at 09:05",
        },
        {
            name: "renders midnight as the start of the day",
            local: new Date(2026, 0, 15, 0, 0, 0),
            expected: "15 January 2026 at 00:00",
        },
        {
            name: "renders the last minute of the day without rolling over",
            local: new Date(2026, 0, 15, 23, 59, 0),
            expected: "15 January 2026 at 23:59",
        },
        {
            name: "leaves a single digit day unpadded",
            local: new Date(2026, 2, 5, 8, 0, 0),
            expected: "5 March 2026 at 08:00",
        },
        {
            name: "ignores the seconds",
            local: new Date(2026, 11, 1, 6, 7, 42),
            expected: "1 December 2026 at 06:07",
        },
    ];

    const localeCases: { name: string; locale: string; expected: string }[] = [
        {
            name: "puts the day before the month for a British reader",
            locale: "en-GB",
            expected: "15 January 2026 at 14:30",
        },
        {
            name: "translates the month and the joining word for a French reader",
            locale: "fr-FR",
            expected: "15 janvier 2026 à 14:30",
        },
        {
            name: "translates the month and the joining word for a German reader",
            locale: "de-DE",
            expected: "15. Januar 2026 um 14:30",
        },
    ];

    it("returns an empty string when there is no date", () => {
        // given / when / then
        expect(formatExactDateTime(null)).toBe("");
        expect(formatExactDateTime(undefined)).toBe("");
        expect(formatExactDateTime("")).toBe("");
        expect(formatExactDateTime("   ")).toBe("");
        expect(formatExactDateTime("not a date")).toBe("");
    });

    it.each(britishCases)("$name", ({ local, expected }) => {
        // given the local date and the exact label it should produce, from the table row

        // when
        const result = formatExactDateTime(local.toISOString(), "en-GB");

        // then
        expect(result).toBe(expected);
    });

    it.each(localeCases)("$name", ({ locale, expected }) => {
        // given
        const local = new Date(2026, 0, 15, 14, 30, 0);

        // when
        const result = formatExactDateTime(local.toISOString(), locale);

        // then
        expect(result).toBe(expected);
    });

    it("switches an American reader to a twelve hour clock with the month first", () => {
        // given
        const local = new Date(2026, 0, 15, 14, 30, 0);

        // when
        const result = formatExactDateTime(local.toISOString(), "en-US");

        // then
        expect(result).toMatch(/^January 15, 2026 at 2:30\sPM$/);
    });

    it("follows the reader's own locale when none is given", () => {
        // given
        const local = new Date(2026, 0, 15, 14, 30, 0);

        // when
        const result = formatExactDateTime(local.toISOString());

        // then
        expect(result).toBe(
            new Intl.DateTimeFormat(undefined, { dateStyle: "long", timeStyle: "short" }).format(local),
        );
    });
});
