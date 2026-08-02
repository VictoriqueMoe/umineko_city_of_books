import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { RulesPage } from "./RulesPage";

function setup(rulesPage: string) {
    return renderWithProviders(<RulesPage />, { siteInfo: { rules_page: rulesPage } });
}

describe("RulesPage", () => {
    it("behaves like a missing page when no rules have been written", () => {
        // given
        const rulesPage = "";

        // when
        setup(rulesPage);

        // then
        expect(screen.getByRole("heading", { name: "This fragment was never written" })).toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Rules" })).not.toBeInTheDocument();
    });

    it("treats whitespace only rules as nothing at all", () => {
        // given
        const rulesPage = "   \n\t  ";

        // when
        setup(rulesPage);

        // then
        expect(screen.getByRole("heading", { name: "This fragment was never written" })).toBeInTheDocument();
    });

    it("titles the page once there are rules to show", () => {
        // given
        const rulesPage = "Be kind to the other players.";

        // when
        setup(rulesPage);

        // then
        expect(screen.getByRole("heading", { name: "Rules" })).toBeInTheDocument();
        expect(screen.getByText("Be kind to the other players.")).toBeInTheDocument();
    });

    it("turns the configured markdown into real markup", () => {
        // given
        const rulesPage = "## House rules\n\n- Be **kind**\n- No spoilers\n";

        // when
        const { container } = setup(rulesPage);

        // then
        expect(screen.getByRole("heading", { name: "House rules", level: 2 })).toBeInTheDocument();
        expect(screen.getAllByRole("listitem")).toHaveLength(2);
        expect(container.querySelector("strong")).toHaveTextContent("kind");
    });

    it("renders a link written in markdown", () => {
        // given
        const rulesPage = "See the [game board](/feed) for details.";

        // when
        setup(rulesPage);

        // then
        expect(screen.getByRole("link", { name: "game board" })).toHaveAttribute("href", "/feed");
    });

    it("strips scripts smuggled into the rules", () => {
        // given
        const rulesPage = "<p>Play nicely.</p><script>alert(1)</script>";

        // when
        const { container } = setup(rulesPage);

        // then
        expect(screen.getByText("Play nicely.")).toBeInTheDocument();
        expect(container.querySelector("script")).toBeNull();
    });

    it("strips inline event handlers smuggled into the rules", () => {
        // given
        const rulesPage = '<p onclick="alert(1)">Play nicely.</p>';

        // when
        setup(rulesPage);

        // then
        expect(screen.getByText("Play nicely.")).not.toHaveAttribute("onclick");
    });
});
