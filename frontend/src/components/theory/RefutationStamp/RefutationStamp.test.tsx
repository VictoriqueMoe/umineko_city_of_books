import { describe, it, expect } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithProviders } from "../../../test-utils/render";
import { makeUser } from "../../../test-utils/fixtures";
import { RefutationStamp } from "./RefutationStamp";

describe("RefutationStamp", () => {
    const refuter = makeUser({ id: "u1", username: "kujo", display_name: "Kujo" });

    it("names the refuter and links to the response", () => {
        // given
        renderWithProviders(<RefutationStamp responseId="r9" refutedBy={refuter} refutedAt="2026-08-10T12:00:00Z" />);

        // then
        expect(screen.getByText("Refuted")).toBeInTheDocument();
        expect(screen.getByText("Kujo")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /read the refutation/i })).toHaveAttribute(
            "href",
            expect.stringContaining("#response-r9"),
        );
    });

    it("says so when the refuting response was deleted", () => {
        // given
        renderWithProviders(<RefutationStamp refutedBy={refuter} refutedAt="2026-08-10T12:00:00Z" />);

        // then
        expect(screen.getByText(/was deleted/i)).toBeInTheDocument();
        expect(screen.queryByRole("link", { name: /read the refutation/i })).not.toBeInTheDocument();
    });

    it("still stamps when the refuter's account is gone", () => {
        // given
        renderWithProviders(<RefutationStamp refutedAt="2026-08-10T12:00:00Z" />);

        // then
        expect(screen.getByText("Refuted")).toBeInTheDocument();
        expect(screen.queryByText("Kujo")).not.toBeInTheDocument();
    });
});
