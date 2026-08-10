import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils/render";
import { KnoxContract } from "./KnoxContract";
import { ALL_KNOX_RULES_ON } from "./knoxRules";

describe("KnoxContract", () => {
    it("publishes all ten rules when nothing is waived", () => {
        // given
        renderWithProviders(<KnoxContract contract={ALL_KNOX_RULES_ON} />);

        // when
        const items = screen.getAllByRole("listitem");

        // then
        expect(items).toHaveLength(10);
        expect(screen.getByText(/What is not sworn here is permitted/)).toBeInTheDocument();
        expect(screen.getByText(/GOOD\?/)).toBeInTheDocument();
    });

    it("omits the rules the game master waived", () => {
        // given
        const contract = { ...ALL_KNOX_RULES_ON, no_supernatural: false, no_unannounced_twins: false };
        renderWithProviders(<KnoxContract contract={contract} />);

        // when
        const items = screen.getAllByRole("listitem");

        // then
        expect(items).toHaveLength(8);
        expect(screen.queryByText(/Magic may decorate this tale/)).not.toBeInTheDocument();
        expect(screen.queryByText(/twin, or any double/)).not.toBeInTheDocument();
    });

    it("warns when the game master swears nothing at all", () => {
        // given
        const contract = Object.fromEntries(
            Object.keys(ALL_KNOX_RULES_ON).map(key => [key, false]),
        ) as unknown as typeof ALL_KNOX_RULES_ON;

        // when
        renderWithProviders(<KnoxContract contract={contract} />);

        // then
        expect(screen.getByText(/Nothing is forbidden here/)).toBeInTheDocument();
        expect(screen.queryAllByRole("listitem")).toHaveLength(0);
    });
});
