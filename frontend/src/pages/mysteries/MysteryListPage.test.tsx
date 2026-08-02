import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { GMLeaderboardEntry, Mystery, MysteryLeaderboardEntry } from "../../types/api";
import { MysteryListPage } from "./MysteryListPage";

const { useMysteryList, useMysteryLeaderboard, useGMLeaderboard } = vi.hoisted(() => ({
    useMysteryList: vi.fn(),
    useMysteryLeaderboard: vi.fn(),
    useGMLeaderboard: vi.fn(),
}));

vi.mock("../../api/queries/mystery", () => ({ useMysteryList, useMysteryLeaderboard, useGMLeaderboard }));
vi.mock("../../components/RulesBox/RulesBox", () => ({
    RulesBox: ({ page }: { page: string }) => <div data-testid="rules-box">{page}</div>,
}));

const viewer = makeUser({ id: "me", username: "me", display_name: "Me" });

function makeMystery(overrides: Partial<Mystery> = {}): Mystery {
    return {
        id: "mystery-1",
        title: "The sealed guest room",
        body: "Six people died behind a chained door.",
        difficulty: "hard",
        author: { id: "gm-1", username: "beatrice", display_name: "Beatrice" },
        solved: false,
        paused: false,
        gm_away: false,
        free_for_all: false,
        keep_open_after_solve: false,
        solver_count: 0,
        paused_duration_seconds: 0,
        attempt_count: 3,
        clue_count: 1,
        created_at: "2026-07-01T10:00:00Z",
        ...overrides,
    };
}

function makeDetective(overrides: Partial<MysteryLeaderboardEntry> = {}): MysteryLeaderboardEntry {
    return {
        user: { id: "det-1", username: "battler", display_name: "Battler" },
        score: 12,
        easy_solved: 1,
        medium_solved: 1,
        hard_solved: 1,
        nightmare_solved: 0,
        score_adjustment: 0,
        ...overrides,
    };
}

function makeGameMaster(overrides: Partial<GMLeaderboardEntry> = {}): GMLeaderboardEntry {
    return {
        user: { id: "gm-1", username: "beatrice", display_name: "Beatrice" },
        score: 20,
        mystery_count: 3,
        player_count: 9,
        ...overrides,
    };
}

interface StubOptions {
    mysteries?: Mystery[];
    total?: number;
    loading?: boolean;
    detectives?: MysteryLeaderboardEntry[];
    gameMasters?: GMLeaderboardEntry[];
}

function stubMysteries(options: StubOptions = {}) {
    useMysteryList.mockReturnValue({
        mysteries: options.mysteries ?? [],
        total: options.total ?? options.mysteries?.length ?? 0,
        loading: options.loading ?? false,
        refresh: vi.fn(),
    });
    useMysteryLeaderboard.mockReturnValue({ entries: options.detectives ?? [], loading: false });
    useGMLeaderboard.mockReturnValue({ entries: options.gameMasters ?? [], loading: false });
}

function lastListParams(): { sort?: string; solved?: string; limit?: number; offset?: number } {
    const calls = useMysteryList.mock.calls;
    return calls[calls.length - 1][0];
}

describe("MysteryListPage", () => {
    it("waits on the board while the mysteries load", () => {
        // given
        stubMysteries({ loading: true, mysteries: [makeMystery()] });

        // when
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // then
        expect(screen.getByText("Loading mysteries...")).toBeInTheDocument();
        expect(screen.queryByText("The sealed guest room")).not.toBeInTheDocument();
    });

    it("invites the first game master when the board is bare", () => {
        // given
        stubMysteries({ mysteries: [] });

        // when
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // then
        expect(
            screen.getByText("No mysteries yet. Be the first game master to challenge the board."),
        ).toBeInTheDocument();
    });

    it("asks for the newest unsolved mysteries by default", () => {
        // given
        stubMysteries();

        // when
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // then
        expect(lastListParams()).toEqual({ sort: "new", solved: "false", limit: 20, offset: 0 });
    });

    it("honours the sort and solved filters carried in the url", () => {
        // given
        stubMysteries();

        // when
        renderWithProviders(<MysteryListPage />, { route: "/mysteries?sort=old&solved=true" });

        // then
        expect(lastListParams()).toMatchObject({ sort: "old", solved: "true" });
    });

    it("reorders the board when a different sort is chosen", async () => {
        // given
        stubMysteries();
        const user = userEvent.setup();
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // when
        await user.selectOptions(screen.getByDisplayValue("Newest"), "old");

        // then
        expect(lastListParams()).toMatchObject({ sort: "old" });
    });

    it("drops the solved filter entirely when all mysteries are wanted", async () => {
        // given
        stubMysteries();
        const user = userEvent.setup();
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // when
        await user.selectOptions(screen.getByDisplayValue("Unsolved"), "");

        // then
        expect(lastListParams().solved).toBeUndefined();
    });

    it("links each mystery card to its own page with its difficulty and counts", () => {
        // given
        stubMysteries({ mysteries: [makeMystery({ id: "mystery-7", clue_count: 1, attempt_count: 3 })] });

        // when
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // then
        const card = screen.getByRole("link", { name: /The sealed guest room/ });
        expect(card).toHaveAttribute("href", "/mystery/mystery-7");
        expect(within(card).getByText("hard")).toBeInTheDocument();
        expect(within(card).getByText("1 clue")).toBeInTheDocument();
        expect(within(card).getByText("3 attempts")).toBeInTheDocument();
        expect(within(card).getByText("Open")).toBeInTheDocument();
    });

    it("pluralises the clue and attempt counts correctly", () => {
        // given
        stubMysteries({ mysteries: [makeMystery({ clue_count: 2, attempt_count: 1 })] });

        // when
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // then
        expect(screen.getByText("2 clues")).toBeInTheDocument();
        expect(screen.getByText("1 attempt")).toBeInTheDocument();
    });

    it("badges a paused mystery and hides the away badge behind it", () => {
        // given
        stubMysteries({ mysteries: [makeMystery({ paused: true, gm_away: true })] });

        // when
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // then
        expect(screen.getByText("Paused")).toBeInTheDocument();
        expect(screen.queryByText("GM Away")).not.toBeInTheDocument();
        expect(screen.getByText(/Paused at/)).toBeInTheDocument();
    });

    it("badges an away game master when the mystery is still running", () => {
        // given
        stubMysteries({ mysteries: [makeMystery({ gm_away: true, free_for_all: true })] });

        // when
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // then
        expect(screen.getByText("GM Away")).toBeInTheDocument();
        expect(screen.getByText("Free-for-all")).toBeInTheDocument();
        expect(screen.getByText(/Unsolved for/)).toBeInTheDocument();
    });

    it("names the winner and how long the solved mystery took", () => {
        // given
        stubMysteries({
            mysteries: [
                makeMystery({
                    solved: true,
                    solved_at: "2026-07-01T12:00:00Z",
                    winner: { id: "det-1", username: "battler", display_name: "Battler" },
                }),
            ],
        });

        // when
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // then
        expect(screen.getByText("Winner: Battler")).toBeInTheDocument();
        expect(screen.getByText(/Solved in/)).toBeInTheDocument();
        expect(screen.getByText("2 hours")).toBeInTheDocument();
    });

    it("trims a long scenario down to a preview", () => {
        // given
        const body = "a".repeat(250);
        stubMysteries({ mysteries: [makeMystery({ body })] });

        // when
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // then
        expect(screen.getByText(`${"a".repeat(200)}...`)).toBeInTheDocument();
    });

    it("hides the new mystery button from a signed out visitor", () => {
        // given
        stubMysteries();

        // when
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // then
        expect(screen.queryByRole("link", { name: "+ New Mystery" })).not.toBeInTheDocument();
    });

    it("offers a signed in member the mystery composer", () => {
        // given
        stubMysteries();

        // when
        renderWithProviders(<MysteryListPage />, { user: viewer, route: "/mysteries" });

        // then
        expect(screen.getByRole("link", { name: "+ New Mystery" })).toHaveAttribute("href", "/mystery/new");
    });

    it("explains the empty detective leaderboard", () => {
        // given
        stubMysteries({ detectives: [] });

        // when
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // then
        expect(
            screen.getByText("No mysteries have been solved yet. Be the first to claim a winner's laurels."),
        ).toBeInTheDocument();
    });

    it("ranks the detectives and crowns the leader with the site's vanity label", () => {
        // given
        stubMysteries({
            detectives: [
                makeDetective({ score: 12 }),
                makeDetective({
                    user: { id: "det-2", username: "ange", display_name: "Ange" },
                    score: 4,
                }),
            ],
        });

        // when
        renderWithProviders(<MysteryListPage />, {
            route: "/mysteries",
            siteInfo: {
                vanity_roles: [
                    {
                        id: "system_top_detective",
                        label: "Golden Sorcerer",
                        color: "#ffd700",
                        is_system: true,
                        sort_order: 0,
                    },
                ],
            },
        });

        // then
        expect(screen.getByText("#1")).toBeInTheDocument();
        expect(screen.getByText("#2")).toBeInTheDocument();
        expect(screen.getByText("12 pts")).toBeInTheDocument();
        expect(screen.getByText("Golden Sorcerer")).toBeInTheDocument();
    });

    it("falls back to the default detective title when the site has no vanity role", () => {
        // given
        stubMysteries({ detectives: [makeDetective()] });

        // when
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // then
        expect(screen.getByText("True Detective")).toBeInTheDocument();
    });

    it("breaks a detective's score down by difficulty when they are clicked", async () => {
        // given
        stubMysteries({
            detectives: [makeDetective({ easy_solved: 2, medium_solved: 0, hard_solved: 1, score_adjustment: -3 })],
        });
        const user = userEvent.setup();
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // when
        await user.click(screen.getByText("Battler"));

        // then
        expect(screen.getByText("3 solved")).toBeInTheDocument();
        expect(screen.getByText("Easy")).toBeInTheDocument();
        expect(screen.getByText("Hard")).toBeInTheDocument();
        expect(screen.queryByText("Medium")).not.toBeInTheDocument();
        expect(screen.getByText("Adjusted score")).toBeInTheDocument();
        expect(screen.getByText("-3")).toBeInTheDocument();
    });

    it("folds a detective's breakdown away when they are clicked again", async () => {
        // given
        stubMysteries({ detectives: [makeDetective()] });
        const user = userEvent.setup();
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // when
        await user.click(screen.getByText("Battler"));
        await user.click(screen.getByText("Battler"));

        // then
        expect(screen.queryByText("3 solved")).not.toBeInTheDocument();
    });

    it("swaps to the game master leaderboard and explains its emptiness", async () => {
        // given
        stubMysteries({ detectives: [makeDetective()], gameMasters: [] });
        const user = userEvent.setup();
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // when
        await user.click(screen.getByRole("button", { name: /Top Game Masters/ }));

        // then
        expect(
            screen.getByText("No mysteries have been solved yet. Create a mystery and have it solved to appear here."),
        ).toBeInTheDocument();
        expect(screen.queryByText("Battler")).not.toBeInTheDocument();
    });

    it("breaks a game master's score down into mysteries and players", async () => {
        // given
        stubMysteries({ gameMasters: [makeGameMaster({ mystery_count: 1, player_count: 4 })] });
        const user = userEvent.setup();
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // when
        await user.click(screen.getByRole("button", { name: "Top Game Masters" }));
        await user.click(screen.getByText("Beatrice"));

        // then
        expect(screen.getByText(/1 mystery solved/)).toBeInTheDocument();
        expect(screen.getByText("Total players")).toBeInTheDocument();
        expect(screen.getByText("4")).toBeInTheDocument();
        expect(screen.getByText("Game Master")).toBeInTheDocument();
    });

    it("pages forward through the mystery list", async () => {
        // given
        stubMysteries({ mysteries: [makeMystery()], total: 45 });
        const user = userEvent.setup();
        renderWithProviders(<MysteryListPage />, { route: "/mysteries" });

        // when
        await user.click(screen.getByRole("button", { name: "Next" }));

        // then
        expect(lastListParams().offset).toBe(20);
    });
});
