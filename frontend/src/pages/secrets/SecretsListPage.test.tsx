import { screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { SecretListResponse, SecretSolverEntry, SecretSummary, UserProfile } from "../../types/api";
import { SecretsListPage } from "./SecretsListPage";

const { useSecretList } = vi.hoisted(() => ({ useSecretList: vi.fn() }));

vi.mock("../../api/queries/secret", () => ({ useSecretList }));

const beatrice = { id: "user-1", username: "beatrice", display_name: "Beatrice" };
const ange = { id: "user-2", username: "ange", display_name: "Ange" };
const reader = makeUser({ id: "reader-1", username: "battler", display_name: "Battler" });

function makeSecret(overrides: Partial<SecretSummary> = {}): SecretSummary {
    return {
        id: "secret-1",
        title: "The First Twilight",
        description: "Six chosen by the key.",
        total_pieces: 5,
        solved: false,
        viewer_progress: 0,
        comment_count: 1,
        ...overrides,
    };
}

function makeSolver(overrides: Partial<SecretSolverEntry> = {}): SecretSolverEntry {
    return {
        user: beatrice,
        solved_count: 2,
        last_solved_at: "2026-03-04T10:00:00Z",
        ...overrides,
    };
}

interface StubOptions {
    data?: SecretListResponse | null;
    loading?: boolean;
}

function stubList(options: StubOptions = {}) {
    useSecretList.mockReturnValue({
        data: options.data === undefined ? { secrets: [], solvers_leaderboard: [] } : options.data,
        loading: options.loading ?? false,
        refresh: vi.fn(),
    });
}

function renderPage(user: UserProfile | null = null) {
    return renderWithProviders(<SecretsListPage />, { user, route: "/secrets" });
}

function cardFor(title: string): HTMLElement {
    return screen.getByRole("heading", { name: title }).closest("a") as HTMLElement;
}

describe("SecretsListPage", () => {
    it("consults the game board while the hunts are loading", () => {
        // given
        stubList({ loading: true, data: { secrets: [makeSecret()], solvers_leaderboard: [] } });

        // when
        renderPage();

        // then
        expect(screen.getByText("Consulting the game board...")).toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "The First Twilight" })).not.toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Solvers" })).not.toBeInTheDocument();
    });

    it("says nothing is awake when there are no hunts", () => {
        // given
        stubList({ data: { secrets: [], solvers_leaderboard: [] } });

        // when
        renderPage();

        // then
        expect(screen.getByText("No secrets are awake yet.")).toBeInTheDocument();
    });

    it("survives the server returning nothing at all", () => {
        // given
        stubList({ data: null });

        // when
        renderPage();

        // then
        expect(screen.getByText("No secrets are awake yet.")).toBeInTheDocument();
        expect(screen.getByText("No one has solved a secret yet.")).toBeInTheDocument();
    });

    it("links each hunt to its own page and marks whether it is open", () => {
        // given
        stubList({
            data: {
                secrets: [
                    makeSecret({ id: "secret-1", title: "The First Twilight" }),
                    makeSecret({ id: "secret-2", title: "The Golden Land", solved: true, solver: beatrice }),
                ],
                solvers_leaderboard: [],
            },
        });

        // when
        renderPage();

        // then
        expect(cardFor("The First Twilight")).toHaveAttribute("href", "/secrets/secret-1");
        expect(cardFor("The First Twilight")).toHaveTextContent("Open");
        expect(cardFor("The Golden Land")).toHaveAttribute("href", "/secrets/secret-2");
        expect(cardFor("The Golden Land")).toHaveTextContent("Solved");
    });

    it("names the hunter who closed a solved hunt", () => {
        // given
        stubList({
            data: {
                secrets: [makeSecret({ solved: true, solver: { ...beatrice, avatar_url: "/avatar.png" } })],
                solvers_leaderboard: [],
            },
        });

        // when
        renderPage();

        // then
        expect(cardFor("The First Twilight")).toHaveTextContent("Solved by Beatrice");
    });

    it("counts the comments on a hunt with the right plural", () => {
        // given
        stubList({
            data: {
                secrets: [
                    makeSecret({ id: "secret-1", title: "The First Twilight", comment_count: 1 }),
                    makeSecret({ id: "secret-2", title: "The Golden Land", comment_count: 4 }),
                ],
                solvers_leaderboard: [],
            },
        });

        // when
        renderPage();

        // then
        expect(cardFor("The First Twilight")).toHaveTextContent("1 comment");
        expect(cardFor("The Golden Land")).toHaveTextContent("4 comments");
    });

    it("keeps a visitor's own progress off the card when they are signed out", () => {
        // given
        stubList({
            data: {
                secrets: [makeSecret({ viewer_progress: 3 })],
                solvers_leaderboard: [],
            },
        });

        // when
        renderPage(null);

        // then
        expect(cardFor("The First Twilight")).not.toHaveTextContent("3 / 5 pieces");
    });

    it("shows a signed in hunter how many pieces they hold", () => {
        // given
        stubList({
            data: {
                secrets: [makeSecret({ viewer_progress: 3 })],
                solvers_leaderboard: [],
            },
        });

        // when
        renderPage(reader);

        // then
        expect(cardFor("The First Twilight")).toHaveTextContent("3 / 5 pieces");
    });

    it("hides the progress chip from a hunter who has found nothing yet", () => {
        // given
        stubList({
            data: {
                secrets: [makeSecret({ viewer_progress: 0 })],
                solvers_leaderboard: [],
            },
        });

        // when
        renderPage(reader);

        // then
        expect(cardFor("The First Twilight")).not.toHaveTextContent("pieces");
    });

    it("says the solvers board is empty when nobody has finished a hunt", () => {
        // given
        stubList({ data: { secrets: [makeSecret()], solvers_leaderboard: [] } });

        // when
        renderPage();

        // then
        expect(screen.getByRole("heading", { name: "Solvers" })).toBeInTheDocument();
        expect(screen.getByText("No one has solved a secret yet.")).toBeInTheDocument();
    });

    it("ranks the solvers in the order the server gave them", () => {
        // given
        stubList({
            data: {
                secrets: [],
                solvers_leaderboard: [
                    makeSolver({ user: beatrice, solved_count: 3 }),
                    makeSolver({ user: ange, solved_count: 1 }),
                ],
            },
        });

        // when
        renderPage();

        // then
        expect(screen.getByText("1")).toBeInTheDocument();
        expect(screen.getByText("2")).toBeInTheDocument();
        expect(screen.getByText("3 solved")).toBeInTheDocument();
        expect(screen.getByText("1 solved")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Beatrice/ })).toHaveAttribute("href", "/user/beatrice");
    });

    it("dates the last hunt each solver closed", () => {
        // given
        stubList({
            data: { secrets: [], solvers_leaderboard: [makeSolver({ last_solved_at: "2026-03-04T10:00:00Z" })] },
        });

        // when
        renderPage();

        // then
        expect(screen.getByText(/2026/)).toBeInTheDocument();
    });

    it("leaves the date blank when the server sent an unusable one", () => {
        // given
        stubList({
            data: { secrets: [], solvers_leaderboard: [makeSolver({ last_solved_at: "" })] },
        });

        // when
        renderPage();

        // then
        expect(screen.queryByText(/2026/)).not.toBeInTheDocument();
        expect(screen.getByText("2 solved")).toBeInTheDocument();
    });
});
