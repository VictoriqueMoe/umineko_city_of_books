import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../../test-utils/render";
import { TheoryStatusBadge } from "./TheoryStatusBadge";

describe("TheoryStatusBadge", () => {
    const cases: { status: "open" | "contested" | "refuted"; label: string }[] = [
        { status: "open", label: "Open" },
        { status: "contested", label: "Contested" },
        { status: "refuted", label: "Refuted" },
    ];

    for (const testCase of cases) {
        it(`renders ${testCase.label} for the ${testCase.status} status`, () => {
            // given
            renderWithProviders(<TheoryStatusBadge status={testCase.status} />);

            // then
            expect(screen.getByText(testCase.label)).toBeInTheDocument();
        });
    }

    it("distinguishes refuted from contested", () => {
        // given
        const { unmount } = renderWithProviders(<TheoryStatusBadge status="contested" />);
        const contestedClass = screen.getByText("Contested").className;
        unmount();

        // when
        renderWithProviders(<TheoryStatusBadge status="refuted" />);
        const refutedClass = screen.getByText("Refuted").className;

        // then
        expect(refutedClass).not.toBe(contestedClass);
    });
});
