import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { CredibilityBadge } from "./CredibilityBadge";

function bandOf(score: number): string {
    const { unmount } = renderWithProviders(<CredibilityBadge score={score} />);
    const band = screen.getByText(String(Math.round(score))).getAttribute("class") ?? "";
    unmount();

    return band;
}

describe("CredibilityBadge", () => {
    it("shows the score rounded to a whole number", () => {
        // given
        const score = 63.4;

        // when
        renderWithProviders(<CredibilityBadge score={score} />);

        // then
        expect(screen.getByText("63")).toBeInTheDocument();
        expect(screen.getByText("Credibility")).toBeInTheDocument();
    });

    it("rounds a half point upwards", () => {
        // given
        const score = 69.5;

        // when
        renderWithProviders(<CredibilityBadge score={score} />);

        // then
        expect(screen.getByText("70")).toBeInTheDocument();
    });

    it("explains how the score is earned", () => {
        // given
        const score = 50;

        // when
        renderWithProviders(<CredibilityBadge score={score} />);

        // then
        expect(screen.getByText(/50 is neutral, higher means stronger community support/)).toBeInTheDocument();
        expect(screen.getByText(/red or gold truth evidence carry more weight/)).toBeInTheDocument();
    });

    it("bands a score of seventy or more as the strongest", () => {
        // given
        const strong = 70;

        // when
        const band = bandOf(strong);

        // then
        expect(band).toBe(bandOf(100));
        expect(band).not.toBe(bandOf(69));
    });

    it("bands a score from forty to sixty nine as middling", () => {
        // given
        const middling = 40;

        // when
        const band = bandOf(middling);

        // then
        expect(band).toBe(bandOf(55));
        expect(band).toBe(bandOf(69));
        expect(band).not.toBe(bandOf(70));
        expect(band).not.toBe(bandOf(39));
    });

    it("bands a score below forty as the weakest", () => {
        // given
        const weak = 39;

        // when
        const band = bandOf(weak);

        // then
        expect(band).toBe(bandOf(0));
        expect(band).not.toBe(bandOf(40));
    });

    it("bands on the number it displays rather than on the exact score", () => {
        // given
        const justUnderTheBoundary = 69.6;

        // when
        const { unmount } = renderWithProviders(<CredibilityBadge score={justUnderTheBoundary} />);
        const band = screen.getByText("70").getAttribute("class") ?? "";
        unmount();

        // then
        expect(band).toBe(bandOf(70));
        expect(band).not.toBe(bandOf(50));
    });

    it("bands a score that rounds up out of the weakest band as middling", () => {
        // given
        const justUnderTheBoundary = 39.5;

        // when
        const { unmount } = renderWithProviders(<CredibilityBadge score={justUnderTheBoundary} />);
        const band = screen.getByText("40").getAttribute("class") ?? "";
        unmount();

        // then
        expect(band).toBe(bandOf(40));
        expect(band).not.toBe(bandOf(39));
    });
});
