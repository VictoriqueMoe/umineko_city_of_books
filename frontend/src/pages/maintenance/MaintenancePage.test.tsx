import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { MaintenancePage } from "./MaintenancePage";

describe("MaintenancePage", () => {
    it("falls back to the house wording when nothing was configured", () => {
        // given
        const ui = <MaintenancePage />;

        // when
        renderWithProviders(ui);

        // then
        expect(screen.getByRole("heading", { name: "The game board is being prepared" })).toBeInTheDocument();
        expect(screen.getByText("Without love, it cannot be seen. Please check back shortly.")).toBeInTheDocument();
    });

    it("shows the wording the administrators chose", () => {
        // given
        const ui = <MaintenancePage title="The witches are rearranging the pieces" message="Back after the storm." />;

        // when
        renderWithProviders(ui);

        // then
        expect(screen.getByRole("heading", { name: "The witches are rearranging the pieces" })).toBeInTheDocument();
        expect(screen.getByText("Back after the storm.")).toBeInTheDocument();
    });

    it("keeps the house wording when the configured text is blank", () => {
        // given
        const ui = <MaintenancePage title="" message="" />;

        // when
        renderWithProviders(ui);

        // then
        expect(screen.getByRole("heading", { name: "The game board is being prepared" })).toBeInTheDocument();
        expect(screen.getByText("Without love, it cannot be seen. Please check back shortly.")).toBeInTheDocument();
    });

    it("mixes a configured title with the house message", () => {
        // given
        const ui = <MaintenancePage title="The witches are rearranging the pieces" />;

        // when
        renderWithProviders(ui);

        // then
        expect(screen.getByRole("heading", { name: "The witches are rearranging the pieces" })).toBeInTheDocument();
        expect(screen.getByText("Without love, it cannot be seen. Please check back shortly.")).toBeInTheDocument();
    });
});
