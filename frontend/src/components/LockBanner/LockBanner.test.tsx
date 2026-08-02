import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import { LockBanner } from "./LockBanner";

describe("LockBanner", () => {
    it("renders nothing for a signed out visitor", () => {
        // given
        const user = null;

        // when
        const { container } = renderWithProviders(<LockBanner />, { user });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("renders nothing for an account that is not locked", () => {
        // given
        const user = makeUser({ locked: false });

        // when
        const { container } = renderWithProviders(<LockBanner />, { user });

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("tells a locked account what it can still do", () => {
        // given
        const user = makeUser({ locked: true });

        // when
        renderWithProviders(<LockBanner />, { user });

        // then
        expect(screen.getByText(/Your account is locked/)).toBeInTheDocument();
        expect(screen.getByText(/send direct messages to site staff/)).toBeInTheDocument();
    });

    it("points the locked account at the member list to find a moderator", () => {
        // given
        const user = makeUser({ locked: true });

        // when
        renderWithProviders(<LockBanner />, { user });

        // then
        expect(screen.getByRole("link", { name: "Find a moderator" })).toHaveAttribute("href", "/users");
    });

    it("includes the reason for the lock when one was recorded", () => {
        // given
        const user = makeUser({ locked: true, lock_reason: "repeated spoilers" });

        // when
        renderWithProviders(<LockBanner />, { user });

        // then
        expect(screen.getByText(/Reason: repeated spoilers/)).toBeInTheDocument();
    });

    it("leaves out the reason when none was recorded", () => {
        // given
        const user = makeUser({ locked: true, lock_reason: "" });

        // when
        renderWithProviders(<LockBanner />, { user });

        // then
        expect(screen.queryByText(/Reason:/)).not.toBeInTheDocument();
        expect(screen.getByText(/Your account is locked/)).toBeInTheDocument();
    });

    it("renders nothing when the locked flag is absent altogether", () => {
        // given
        const user = makeUser();

        // when
        const { container } = renderWithProviders(<LockBanner />, { user });

        // then
        expect(container).toBeEmptyDOMElement();
    });
});
