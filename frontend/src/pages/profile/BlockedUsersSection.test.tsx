import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { BlockedUserItem } from "../../api/endpoints";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { UserProfile } from "../../types/api";
import { BlockedUsersSection } from "./BlockedUsersSection";

const mocks = vi.hoisted(() => ({
    useBlockedUsers: vi.fn(),
    useUnblockUser: vi.fn(),
    unblock: vi.fn(),
}));

vi.mock("../../api/queries/user", () => ({ useBlockedUsers: mocks.useBlockedUsers }));
vi.mock("../../api/mutations/misc", () => ({ useUnblockUser: mocks.useUnblockUser }));

function makeBlocked(overrides: Partial<BlockedUserItem> = {}): BlockedUserItem {
    return {
        id: "blocked-1",
        username: "kanon",
        display_name: "Kanon",
        avatar_url: "",
        blocked_at: "2026-02-01T00:00:00Z",
        ...overrides,
    };
}

interface SetupOptions {
    blocked?: BlockedUserItem[];
    loading?: boolean;
    user?: UserProfile | null;
}

function setup(options: SetupOptions = {}) {
    mocks.useBlockedUsers.mockReturnValue({
        blocked: options.blocked ?? [],
        loading: options.loading ?? false,
        refresh: vi.fn(),
    });

    const user = userEvent.setup();
    const viewer = options.user === undefined ? makeUser({ id: "me" }) : options.user;
    const result = renderWithProviders(<BlockedUsersSection />, { user: viewer });

    return { ...result, user };
}

beforeEach(() => {
    mocks.unblock.mockResolvedValue(undefined);
    mocks.useUnblockUser.mockReturnValue({ mutateAsync: mocks.unblock });
});

describe("BlockedUsersSection", () => {
    it("asks for the signed in player's own block list", () => {
        // given
        const options = { user: makeUser({ id: "me" }) };

        // when
        setup(options);

        // then
        expect(mocks.useBlockedUsers).toHaveBeenCalledWith("me");
    });

    it("asks for nothing while nobody is signed in", () => {
        // given
        const options = { user: null };

        // when
        setup(options);

        // then
        expect(mocks.useBlockedUsers).toHaveBeenCalledWith("");
    });

    it("waits quietly while the list is being fetched", () => {
        // given
        const options = { loading: true };

        // when
        setup(options);

        // then
        expect(screen.getByText("Loading...")).toBeInTheDocument();
        expect(screen.queryByText("You haven't blocked anyone.")).not.toBeInTheDocument();
    });

    it("says the player has blocked nobody when the list is empty", () => {
        // given
        const options = { blocked: [] };

        // when
        setup(options);

        // then
        expect(screen.getByText("You haven't blocked anyone.")).toBeInTheDocument();
    });

    it("names everyone the player has blocked", () => {
        // given
        const options = {
            blocked: [makeBlocked(), makeBlocked({ id: "blocked-2", username: "shannon", display_name: "Shannon" })],
        };

        // when
        setup(options);

        // then
        expect(screen.getByText("Kanon")).toBeInTheDocument();
        expect(screen.getByText("Shannon")).toBeInTheDocument();
        expect(screen.getAllByRole("button", { name: "Unblock" })).toHaveLength(2);
    });

    it("unblocks the player whose row was pressed", async () => {
        // given
        const { user } = setup({
            blocked: [makeBlocked(), makeBlocked({ id: "blocked-2", username: "shannon", display_name: "Shannon" })],
        });

        // when
        await user.click(screen.getAllByRole("button", { name: "Unblock" })[1]);

        // then
        expect(mocks.unblock).toHaveBeenCalledWith("blocked-2");
    });

    it("keeps the list on screen when unblocking is refused", async () => {
        // given
        mocks.unblock.mockRejectedValue(new Error("Could not unblock."));
        const { user } = setup({ blocked: [makeBlocked()] });

        // when
        await user.click(screen.getByRole("button", { name: "Unblock" }));

        // then
        expect(screen.getByText("Kanon")).toBeInTheDocument();
    });

    it("links each blocked player to their profile", () => {
        // given
        const options = { blocked: [makeBlocked()] };

        // when
        setup(options);

        // then
        expect(screen.getByRole("link", { name: /Kanon/ })).toHaveAttribute("href", "/user/kanon");
    });
});
