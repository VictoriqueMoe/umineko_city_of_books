import { fireEvent } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import type { ThemeType } from "../../../types/app";
import { Butterflies } from "./Butterflies";

const BUTTERFLY_COUNT = 8;
const PARTICLE_COUNT = 15;

const DEFAULT_BUTTERFLY = "🦋";
const DEFAULT_PARTICLE = "✦";
const LAST_DEFAULT_BUTTERFLY = "✛";
const LAST_DEFAULT_PARTICLE = "◇";
const SWEET_BUTTERFLY = "✨";

function renderField(theme: ThemeType = "featherine") {
    const result = renderWithProviders(<Butterflies />, { theme: { theme } });
    const field = result.container.firstElementChild;
    if (!(field instanceof HTMLElement)) {
        throw new Error("expected the butterfly field to be rendered");
    }

    return { ...result, field };
}

function firstOf(field: HTMLElement, selector: string): HTMLElement {
    const el = field.querySelector<HTMLElement>(selector);
    if (!el) {
        throw new Error(`expected the field to hold a ${selector}`);
    }

    return el;
}

function symbolsOf(field: HTMLElement, selector: string): Set<string> {
    return new Set(Array.from(field.querySelectorAll(selector)).map(el => el.textContent ?? ""));
}

describe("Butterflies", () => {
    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("fills the field with butterflies and particles", () => {
        // given
        const theme: ThemeType = "featherine";

        // when
        const { field } = renderField(theme);

        // then
        expect(field.querySelectorAll(".butterfly")).toHaveLength(BUTTERFLY_COUNT);
        expect(field.querySelectorAll(".particle")).toHaveLength(PARTICLE_COUNT);
    });

    it("uses the ordinary symbols for a theme with nothing special about it", () => {
        // given
        vi.spyOn(Math, "random").mockReturnValue(0);

        // when
        const { field } = renderField("bernkastel");

        // then
        expect(symbolsOf(field, ".butterfly")).toEqual(new Set([DEFAULT_BUTTERFLY]));
        expect(symbolsOf(field, ".particle")).toEqual(new Set([DEFAULT_PARTICLE]));
    });

    it("uses the sweets for the lambdadelta theme", () => {
        // given
        vi.spyOn(Math, "random").mockReturnValue(0);

        // when
        const { field } = renderField("lambdadelta");

        // then
        expect(symbolsOf(field, ".butterfly")).toEqual(new Set([SWEET_BUTTERFLY]));
        expect(symbolsOf(field, ".particle")).toEqual(new Set([SWEET_BUTTERFLY]));
    });

    it("picks the last symbol of the set when the roll lands high", () => {
        // given
        vi.spyOn(Math, "random").mockReturnValue(0.99);

        // when
        const { field } = renderField("beatrice");

        // then
        expect(symbolsOf(field, ".butterfly")).toEqual(new Set([LAST_DEFAULT_BUTTERFLY]));
        expect(symbolsOf(field, ".particle")).toEqual(new Set([LAST_DEFAULT_PARTICLE]));
    });

    it("gives every butterfly a start, a duration and a delay to animate with", () => {
        // given
        vi.spyOn(Math, "random").mockReturnValue(0);

        // when
        const { field } = renderField();

        // then
        const first = firstOf(field, ".butterfly");
        expect(first.style.getPropertyValue("--start-x")).toBe("0vw");
        expect(first.style.getPropertyValue("--duration")).toBe("15s");
        expect(first.style.getPropertyValue("--delay")).toBe("0s");
        expect(first.style.fontSize).toBe("0.8rem");
    });

    it("lets particles drift for longer than butterflies do", () => {
        // given
        vi.spyOn(Math, "random").mockReturnValue(1);

        // when
        const { field } = renderField();

        // then
        expect(firstOf(field, ".butterfly").style.getPropertyValue("--duration")).toBe("30s");
        expect(firstOf(field, ".particle").style.getPropertyValue("--duration")).toBe("35s");
    });

    it("moves a butterfly to a fresh column each time its animation loops", () => {
        // given
        const random = vi.spyOn(Math, "random").mockReturnValue(0);
        const { field } = renderField();
        const first = firstOf(field, ".butterfly");

        // when
        random.mockReturnValue(0.5);
        fireEvent(first, new Event("animationiteration"));

        // then
        expect(first.style.getPropertyValue("--start-x")).toBe("50vw");
    });

    it("moves a particle to a fresh column each time its animation loops", () => {
        // given
        const random = vi.spyOn(Math, "random").mockReturnValue(0);
        const { field } = renderField();
        const first = firstOf(field, ".particle");

        // when
        random.mockReturnValue(0.25);
        fireEvent(first, new Event("animationiteration"));

        // then
        expect(first.style.getPropertyValue("--start-x")).toBe("25vw");
    });

    it("empties the field when it goes away", () => {
        // given
        const { field, unmount } = renderField();
        expect(field.childElementCount).toBe(BUTTERFLY_COUNT + PARTICLE_COUNT);

        // when
        unmount();

        // then
        expect(field.childElementCount).toBe(0);
    });
});
