import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { User } from "../../types/api";
import { NewSnakesAndLaddersGamePage } from "./NewSnakesAndLaddersGamePage";

const { useMutualFollowers, useSearchUsers, useInviteToGame, navigate } = vi.hoisted(() => ({
    useMutualFollowers: vi.fn(),
    useSearchUsers: vi.fn(),
    useInviteToGame: vi.fn(),
    navigate: vi.fn(),
}));

vi.mock("../../api/queries/misc", () => ({ useMutualFollowers, useSearchUsers }));
vi.mock("../../api/mutations/gameRoom", () => ({ useInviteToGame }));
vi.mock("react-router", async importOriginal => {
    const actual = await importOriginal<typeof import("react-router")>();
    return { ...actual, useNavigate: () => navigate };
});

const viewer = makeUser({ id: "me", username: "me", display_name: "Me" });

function makeCandidate(overrides: Partial<User> = {}): User {
    return {
        id: "user-1",
        username: "beatrice",
        display_name: "Beatrice",
        avatar_url: "",
        ...overrides,
    };
}

const beatrice = makeCandidate();
const battler = makeCandidate({ id: "user-2", username: "battler", display_name: "Battler" });

interface StubOptions {
    mutuals?: User[];
    results?: User[];
    invite?: () => Promise<{ id: string }>;
    pending?: boolean;
}

function stubInvite(options: StubOptions = {}) {
    useMutualFollowers.mockReturnValue({ mutuals: options.mutuals ?? [], loading: false });
    useSearchUsers.mockReturnValue({ users: options.results ?? [], loading: false });
    const mutateAsync = vi.fn(options.invite ?? (() => Promise.resolve({ id: "room-1" })));
    useInviteToGame.mockReturnValue({ mutateAsync, isPending: options.pending ?? false });

    return { mutateAsync };
}

function renderPage() {
    return renderWithProviders(<NewSnakesAndLaddersGamePage />, {
        user: viewer,
        route: "/games/snakes_and_ladders/new",
    });
}

function searchField(): HTMLElement {
    return screen.getByPlaceholderText("Search for a player by username...");
}

describe("NewSnakesAndLaddersGamePage", () => {
    it("offers the mutual followers as the starting candidates", () => {
        // given
        stubInvite({ mutuals: [beatrice, battler] });

        // when
        renderPage();

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText("@battler")).toBeInTheDocument();
    });

    it("leaves the inviter out of their own candidate list", () => {
        // given
        stubInvite({ mutuals: [beatrice, makeCandidate({ id: "me", username: "me", display_name: "Me" })] });

        // when
        renderPage();

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.queryByText("@me")).not.toBeInTheDocument();
    });

    it("says there is nobody to invite when the candidate list is empty", () => {
        // given
        stubInvite({ mutuals: [] });

        // when
        renderPage();

        // then
        expect(screen.getByText("No matches.")).toBeInTheDocument();
    });

    it("replaces the mutual followers with the search results as soon as a term is typed", async () => {
        // given
        stubInvite({ mutuals: [beatrice], results: [battler] });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.type(searchField(), "batt");

        // then
        expect(screen.getByText("Battler")).toBeInTheDocument();
        expect(screen.queryByText("Beatrice")).not.toBeInTheDocument();
    });

    it("searches for the trimmed term once the typing settles", async () => {
        // given
        stubInvite({ mutuals: [beatrice], results: [battler] });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.type(searchField(), "  batt  ");

        // then
        await waitFor(() => {
            expect(useSearchUsers).toHaveBeenLastCalledWith("batt");
        });
    });

    it("keeps the invite locked until an opponent is picked", () => {
        // given
        stubInvite({ mutuals: [beatrice] });

        // when
        renderPage();

        // then
        expect(screen.getByRole("button", { name: "Pick a player" })).toBeDisabled();
    });

    it("names the picked opponent on the invite button", async () => {
        // given
        stubInvite({ mutuals: [beatrice] });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByText("Beatrice"));

        // then
        expect(screen.getByRole("button", { name: "Invite Beatrice" })).toBeEnabled();
    });

    it("invites the chosen opponent to a snakes and ladders game and opens the new room", async () => {
        // given
        const { mutateAsync } = stubInvite({ mutuals: [beatrice], invite: () => Promise.resolve({ id: "room-9" }) });
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByText("Beatrice"));

        // when
        await user.click(screen.getByRole("button", { name: "Invite Beatrice" }));

        // then
        expect(mutateAsync).toHaveBeenCalledWith({ opponentId: "user-1", gameType: "snakes_and_ladders" });
        expect(navigate).toHaveBeenCalledWith("/games/snakes_and_ladders/room-9");
    });

    it("shows why the invite was refused and stays on the page", async () => {
        // given
        stubInvite({ mutuals: [beatrice], invite: () => Promise.reject(new Error("they are already playing you")) });
        const user = userEvent.setup();
        renderPage();
        await user.click(screen.getByText("Beatrice"));

        // when
        await user.click(screen.getByRole("button", { name: "Invite Beatrice" }));

        // then
        expect(await screen.findByText("they are already playing you")).toBeInTheDocument();
        expect(navigate).not.toHaveBeenCalled();
    });

    it("shows a busy label while the invite is in flight", async () => {
        // given
        stubInvite({ mutuals: [beatrice], pending: true });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByText("Beatrice"));

        // then
        expect(screen.getByRole("button", { name: "Sending..." })).toBeDisabled();
    });

    it("returns to the games list without inviting anyone", async () => {
        // given
        const { mutateAsync } = stubInvite({ mutuals: [beatrice] });
        const user = userEvent.setup();
        renderPage();

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(navigate).toHaveBeenCalledWith("/games");
        expect(mutateAsync).not.toHaveBeenCalled();
    });
});
