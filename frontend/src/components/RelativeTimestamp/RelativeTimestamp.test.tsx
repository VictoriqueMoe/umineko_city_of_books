import { render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { RelativeTimestamp } from "./RelativeTimestamp";

type Variant = "relative" | "short" | "message" | "active" | "timeOnly" | "dateTime";

const NOW = "2026-06-15T12:00:00Z";
const TIME_PART = new Intl.DateTimeFormat(undefined, { hour: "2-digit", minute: "2-digit" });
const EXACT_PART = new Intl.DateTimeFormat(undefined, { dateStyle: "long", timeStyle: "short" });
const SHORT_DATE_TIME_PART = new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
});

function ago(seconds: number): string {
    return new Date(Date.parse(NOW) - seconds * 1000).toISOString();
}

function yesterdayAt(hours: number, minutes: number): Date {
    const d = new Date(Date.parse(NOW));
    d.setDate(d.getDate() - 1);
    d.setHours(hours, minutes, 0, 0);

    return d;
}

function pinNow(): void {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(NOW));
}

function timeOf(container: HTMLElement): HTMLTimeElement {
    const element = container.querySelector("time");
    if (!element) {
        throw new Error("expected a time element to be rendered");
    }

    return element;
}

function spanOf(container: HTMLElement): HTMLSpanElement {
    const element = container.querySelector("span");
    if (!element) {
        throw new Error("expected a fallback span to be rendered");
    }

    return element;
}

const THREE_DAYS_AGO = ago(3 * 24 * 60 * 60);
const YESTERDAY_EVENING = yesterdayAt(14, 30);

describe("RelativeTimestamp", () => {
    beforeEach(() => {
        pinNow();
    });

    it("marks the moment up as a time element titled with the exact date in the reader's locale", () => {
        // given
        const local = new Date(2026, 0, 15, 14, 30, 0);

        // when
        const { container } = render(<RelativeTimestamp value={local.toISOString()} />);

        // then
        expect(timeOf(container)).toHaveAttribute("title", EXACT_PART.format(local));
    });

    it("carries a machine readable dateTime attribute for the same instant", () => {
        // given
        const value = new Date(2026, 0, 15, 14, 30, 0).toISOString();

        // when
        const { container } = render(<RelativeTimestamp value={value} />);

        // then
        expect(timeOf(container)).toHaveAttribute("datetime", value);
    });

    it("normalises a server timestamp without a zone into a UTC dateTime attribute", () => {
        // given
        const value = "2026-06-12 12:00:00";

        // when
        const { container } = render(<RelativeTimestamp value={value} />);

        // then
        expect(timeOf(container)).toHaveAttribute("datetime", "2026-06-12T12:00:00.000Z");
    });

    const variantCases: { name: string; variant: Variant | undefined; value: string; expected: string }[] = [
        {
            name: "uses the long relative label when no variant is asked for",
            variant: undefined,
            value: THREE_DAYS_AGO,
            expected: "3d ago",
        },
        {
            name: "uses the long relative label for the relative variant",
            variant: "relative",
            value: THREE_DAYS_AGO,
            expected: "3d ago",
        },
        {
            name: "drops the ago suffix for the short variant",
            variant: "short",
            value: THREE_DAYS_AGO,
            expected: "3d",
        },
        {
            name: "names the previous day for the message variant",
            variant: "message",
            value: YESTERDAY_EVENING.toISOString(),
            expected: `Yesterday ${TIME_PART.format(YESTERDAY_EVENING)}`,
        },
        {
            name: "prefixes the label with Active for the active variant",
            variant: "active",
            value: THREE_DAYS_AGO,
            expected: "Active 3d ago",
        },
        {
            name: "shows the clock time alone for the timeOnly variant",
            variant: "timeOnly",
            value: THREE_DAYS_AGO,
            expected: TIME_PART.format(new Date(THREE_DAYS_AGO)),
        },
        {
            name: "shows a short date beside the clock time for the dateTime variant",
            variant: "dateTime",
            value: THREE_DAYS_AGO,
            expected: SHORT_DATE_TIME_PART.format(new Date(THREE_DAYS_AGO)),
        },
    ];

    it.each(variantCases)("$name", ({ variant, value, expected }) => {
        // given the variant and the label it should produce, from the table row

        // when
        const { container } = render(<RelativeTimestamp value={value} variant={variant} />);

        // then
        expect(timeOf(container).textContent).toBe(expected);
    });

    it("reveals the missing date on hover when the label is only a clock time", () => {
        // given
        const local = new Date(2026, 0, 15, 14, 30, 0);

        // when
        const { container } = render(<RelativeTimestamp value={local.toISOString()} variant="timeOnly" />);

        // then
        const stamp = timeOf(container);
        expect(stamp.textContent).toBe(TIME_PART.format(local));
        expect(stamp.textContent).not.toContain("2026");
        expect(stamp).toHaveAttribute("title", EXACT_PART.format(local));
        expect(stamp.getAttribute("title")).toContain("2026");
    });

    const fallbackCases: { name: string; value: string | null | undefined }[] = [
        { name: "falls back for a null value", value: null },
        { name: "falls back for a missing value", value: undefined },
        { name: "falls back for an empty value", value: "" },
        { name: "falls back for a blank value", value: "   " },
        { name: "falls back for a value that is not a date", value: "not a date" },
    ];

    it.each(fallbackCases)("$name", ({ value }) => {
        // given the unusable value, from the table row

        // when
        const { container } = render(<RelativeTimestamp value={value} />);

        // then
        const span = spanOf(container);
        expect(container.querySelector("time")).toBeNull();
        expect(span).not.toHaveAttribute("title");
        expect(span.textContent).toBe("");
    });

    it("still says there is no activity in the fallback span for the active variant", () => {
        // given
        const value = null;

        // when
        const { container } = render(<RelativeTimestamp value={value} variant="active" />);

        // then
        const span = spanOf(container);
        expect(span.textContent).toBe("No activity yet");
        expect(span).not.toHaveAttribute("title");
    });

    it("carries the class name it was given on the time element", () => {
        // given
        const className = "stamp";

        // when
        const { container } = render(<RelativeTimestamp value={THREE_DAYS_AGO} className={className} />);

        // then
        expect(timeOf(container)).toHaveClass(className);
    });

    it("carries the class name it was given on the fallback span", () => {
        // given
        const className = "stamp";

        // when
        const { container } = render(<RelativeTimestamp value={null} variant="active" className={className} />);

        // then
        expect(spanOf(container)).toHaveClass(className);
    });
});
