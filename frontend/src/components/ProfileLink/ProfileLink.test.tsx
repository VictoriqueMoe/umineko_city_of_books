import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { User } from "../../types/api";
import { renderWithProviders } from "../../test-utils/render";
import { ProfileLink } from "./ProfileLink";

const USER_ID = "00000000-0000-0000-0000-000000000001";

function makeLinkUser(overrides: Partial<User> = {}): User {
    return {
        id: USER_ID,
        username: "beatrice",
        display_name: "Beatrice",
        ...overrides,
    };
}

describe("ProfileLink", () => {
    it("links to the profile page of the user", () => {
        // given
        const user = makeLinkUser({ username: "battler" });

        // when
        renderWithProviders(<ProfileLink user={user} />);

        // then
        expect(screen.getByRole("link", { name: /Beatrice/ })).toHaveAttribute("href", "/user/battler");
    });

    it("renders a static badge when the caller turns clicking off", () => {
        // given
        const user = makeLinkUser();

        // when
        renderWithProviders(<ProfileLink user={user} clickable={false} />);

        // then
        expect(screen.queryByRole("link")).not.toBeInTheDocument();
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
    });

    it("renders a static badge for a user with no username to link to", () => {
        // given
        const user = makeLinkUser({ username: "" });

        // when
        renderWithProviders(<ProfileLink user={user} />);

        // then
        expect(screen.queryByRole("link")).not.toBeInTheDocument();
    });

    it("shows the avatar the user has uploaded", () => {
        // given
        const user = makeLinkUser({ avatar_url: "https://cdn.test/beato.png" });

        // when
        renderWithProviders(<ProfileLink user={user} />);

        // then
        const avatar = screen.getByRole("presentation");
        expect(avatar).toHaveAttribute("src", "https://cdn.test/beato.png");
        expect(avatar).toHaveAttribute("loading", "lazy");
    });

    it("falls back to the first letter of the display name when there is no avatar", () => {
        // given
        const user = makeLinkUser({ display_name: "Featherine", avatar_url: "" });

        // when
        renderWithProviders(<ProfileLink user={user} />);

        // then
        expect(screen.getByText("F")).toBeInTheDocument();
    });

    it("sizes the avatar according to the requested size", () => {
        // given
        const user = makeLinkUser({ avatar_url: "https://cdn.test/beato.png" });

        // when
        renderWithProviders(<ProfileLink user={user} size="large" />);

        // then
        expect(screen.getByRole("presentation")).toHaveAttribute("width", "40");
    });

    it("defaults to the medium avatar size", () => {
        // given
        const user = makeLinkUser({ avatar_url: "https://cdn.test/beato.png" });

        // when
        renderWithProviders(<ProfileLink user={user} />);

        // then
        expect(screen.getByRole("presentation")).toHaveAttribute("width", "28");
    });

    it("hides the name entirely when only the avatar is wanted", () => {
        // given
        const user = makeLinkUser({ role: "admin" });

        // when
        renderWithProviders(<ProfileLink user={user} showName={false} />);

        // then
        expect(screen.queryByText("Beatrice")).not.toBeInTheDocument();
        expect(screen.queryByText("Voyager Witch")).not.toBeInTheDocument();
    });

    it("puts the given prefix in front of the name", () => {
        // given
        const user = makeLinkUser();

        // when
        renderWithProviders(<ProfileLink user={user} prefix="with" />);

        // then
        expect(screen.getByRole("link", { name: /with Beatrice/ })).toBeInTheDocument();
    });

    it("shows the role pill of a staff member alongside the name", () => {
        // given
        const user = makeLinkUser({ role: "moderator" });

        // when
        renderWithProviders(<ProfileLink user={user} />);

        // then
        expect(screen.getByText("Witch")).toBeInTheDocument();
    });

    it("leaves the role pill out when roles are suppressed", () => {
        // given
        const user = makeLinkUser({ role: "moderator" });

        // when
        renderWithProviders(<ProfileLink user={user} showRoles={false} />);

        // then
        expect(screen.queryByText("Witch")).not.toBeInTheDocument();
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
    });

    it("marks a banned user with a banned pill", () => {
        // given
        const user = makeLinkUser({ banned: true });

        // when
        renderWithProviders(<ProfileLink user={user} />);

        // then
        expect(screen.getByText("banned")).toBeInTheDocument();
    });

    it("leaves the banned pill off an ordinary user", () => {
        // given
        const user = makeLinkUser({ banned: false });

        // when
        renderWithProviders(<ProfileLink user={user} />);

        // then
        expect(screen.queryByText("banned")).not.toBeInTheDocument();
    });

    it("still links a banned user to their profile", () => {
        // given
        const user = makeLinkUser({ banned: true });

        // when
        renderWithProviders(<ProfileLink user={user} />);

        // then
        expect(screen.getByRole("link", { name: /Beatrice/ })).toHaveAttribute("href", "/user/beatrice");
    });

    it("shows the vanity roles configured for this user", () => {
        // given
        const user = makeLinkUser();

        // when
        renderWithProviders(<ProfileLink user={user} />, {
            siteInfo: {
                vanity_roles: [
                    { id: "mine", label: "Golden Butterfly", color: "#102030", is_system: false, sort_order: 0 },
                ],
                vanity_role_assignments: { [USER_ID]: ["mine"] },
            },
        });

        // then
        expect(screen.getByText("Golden Butterfly")).toBeInTheDocument();
    });
});
