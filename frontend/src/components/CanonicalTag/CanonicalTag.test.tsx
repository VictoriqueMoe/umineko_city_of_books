import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useNavigate } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { CanonicalTag } from "./CanonicalTag";

const SITE_ORIGIN = "https://whentheycry.social";

function canonicalLinks(): HTMLLinkElement[] {
    return Array.from(document.head.querySelectorAll<HTMLLinkElement>('link[rel="canonical"]'));
}

function Harness() {
    const navigate = useNavigate();

    return (
        <>
            <CanonicalTag />
            <button type="button" onClick={() => navigate("/mysteries?page=2")}>
                go elsewhere
            </button>
        </>
    );
}

afterEach(() => {
    for (const link of canonicalLinks()) {
        link.remove();
    }
});

describe("CanonicalTag", () => {
    it("renders nothing of its own into the page", () => {
        // given
        const route = "/";

        // when
        const { container } = renderWithProviders(<CanonicalTag />, { route });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("adds a canonical link for the current route", () => {
        // given
        const route = "/theories";

        // when
        renderWithProviders(<CanonicalTag />, { route });

        // then
        expect(canonicalLinks()).toHaveLength(1);
        expect(canonicalLinks()[0].href).toBe(`${SITE_ORIGIN}/theories`);
    });

    it("keeps the query string in the canonical url", () => {
        // given
        const route = "/theories?sort=new&episode=4";

        // when
        renderWithProviders(<CanonicalTag />, { route });

        // then
        expect(canonicalLinks()[0].href).toBe(`${SITE_ORIGIN}/theories?sort=new&episode=4`);
    });

    it("reuses a canonical link that is already in the head", () => {
        // given
        const existing = document.createElement("link");
        existing.rel = "canonical";
        existing.href = "https://example.test/stale";
        document.head.appendChild(existing);

        // when
        renderWithProviders(<CanonicalTag />, { route: "/mysteries" });

        // then
        expect(canonicalLinks()).toHaveLength(1);
        expect(existing.href).toBe(`${SITE_ORIGIN}/mysteries`);
    });

    it("follows the reader as they navigate", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<Harness />, { route: "/theories" });
        expect(canonicalLinks()[0].href).toBe(`${SITE_ORIGIN}/theories`);

        // when
        await user.click(screen.getByRole("button", { name: "go elsewhere" }));

        // then
        expect(canonicalLinks()).toHaveLength(1);
        expect(canonicalLinks()[0].href).toBe(`${SITE_ORIGIN}/mysteries?page=2`);
    });
});
