import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { RulesBox } from "./RulesBox";

const { useRules } = vi.hoisted(() => ({ useRules: vi.fn() }));

vi.mock("../../api/queries/misc", () => ({ useRules }));

beforeEach(() => {
    useRules.mockReturnValue({ rules: "", loading: false });
});

describe("RulesBox", () => {
    it("renders nothing while the rules are still loading", () => {
        // given
        useRules.mockReturnValue({ rules: "", loading: true });

        // when
        const { container } = renderWithProviders(<RulesBox page="theories" />);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("renders nothing when the page has no rules", () => {
        // given
        useRules.mockReturnValue({ rules: "", loading: false });

        // when
        const { container } = renderWithProviders(<RulesBox page="theories" />);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("shows the rules under a rules label", () => {
        // given
        useRules.mockReturnValue({ rules: "No spoilers beyond episode four", loading: false });

        // when
        renderWithProviders(<RulesBox page="theories" />);

        // then
        expect(screen.getByText("Rules")).toBeInTheDocument();
        expect(screen.getByText("No spoilers beyond episode four")).toBeInTheDocument();
    });

    it("asks for the rules that belong to the page it was given", () => {
        // given
        const page = "fanfics";
        useRules.mockReturnValue({ rules: "Tag your warnings", loading: false });

        // when
        renderWithProviders(<RulesBox page={page} />);

        // then
        expect(useRules).toHaveBeenCalledWith(page);
    });
});
