import { screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { AdminStats } from "../../types/api";
import { AdminDashboard } from "./AdminDashboard";

const mocks = vi.hoisted(() => ({
    useAdminStats: vi.fn(),
}));

vi.mock("../../api/queries/admin", () => ({ useAdminStats: mocks.useAdminStats }));

function makeAdminStats(overrides: Partial<AdminStats> = {}): AdminStats {
    return {
        total_users: 1234,
        total_theories: 20,
        total_responses: 30,
        total_votes: 40,
        total_posts: 50,
        total_comments: 60,
        new_users_24h: 1,
        new_users_7d: 2,
        new_users_30d: 3,
        new_theories_24h: 4,
        new_theories_7d: 5,
        new_theories_30d: 6,
        new_responses_24h: 7,
        new_responses_7d: 8,
        new_responses_30d: 9,
        new_posts_24h: 10,
        new_posts_7d: 11,
        new_posts_30d: 12,
        posts_by_corner: {},
        most_active_users: [],
        ...overrides,
    };
}

function stubStats(stats: AdminStats | null, loading = false) {
    mocks.useAdminStats.mockReturnValue({ stats, loading });
}

describe("AdminDashboard", () => {
    it("waits while the statistics are still being gathered", () => {
        // given
        stubStats(null, true);

        // when
        renderWithProviders(<AdminDashboard />);

        // then
        expect(screen.getByText("Loading statistics...")).toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Dashboard" })).not.toBeInTheDocument();
    });

    it("says so when no statistics came back", () => {
        // given
        stubStats(null);

        // when
        renderWithProviders(<AdminDashboard />);

        // then
        expect(screen.getByText("Could not load the statistics.")).toBeInTheDocument();
        expect(screen.queryByRole("heading", { name: "Dashboard" })).not.toBeInTheDocument();
    });

    it("groups the running totals into their own cards", () => {
        // given
        stubStats(makeAdminStats());

        // when
        renderWithProviders(<AdminDashboard />);

        // then
        expect(screen.getByText("Total Users")).toBeInTheDocument();
        expect(screen.getByText((1234).toLocaleString())).toBeInTheDocument();
        expect(screen.getByText("Total Comments")).toBeInTheDocument();
        expect(screen.getByText("60")).toBeInTheDocument();
    });

    it("leaves out the posts by corner section when no corner has any posts", () => {
        // given
        stubStats(makeAdminStats({ posts_by_corner: {} }));

        // when
        renderWithProviders(<AdminDashboard />);

        // then
        expect(screen.queryByRole("heading", { name: "Posts by Corner" })).not.toBeInTheDocument();
    });

    it("breaks the posts down by corner when there are any", () => {
        // given
        stubStats(makeAdminStats({ posts_by_corner: { umineko: 2000, higurashi: 3 } }));

        // when
        renderWithProviders(<AdminDashboard />);

        // then
        expect(screen.getByRole("heading", { name: "Posts by Corner" })).toBeInTheDocument();
        expect(screen.getByText("umineko")).toBeInTheDocument();
        expect(screen.getByText((2000).toLocaleString())).toBeInTheDocument();
        expect(screen.getByText("higurashi")).toBeInTheDocument();
    });

    it("lays the recent activity out as one row per period", () => {
        // given
        stubStats(makeAdminStats());

        // when
        renderWithProviders(<AdminDashboard />);

        // then
        const dayRow = screen.getByText("Last 24 hours").closest("tr");
        const monthRow = screen.getByText("Last 30 days").closest("tr");
        expect(dayRow).not.toBeNull();
        expect(monthRow).not.toBeNull();
        expect(within(dayRow as HTMLElement).getByText("4")).toBeInTheDocument();
        expect(within(monthRow as HTMLElement).getByText("12")).toBeInTheDocument();
    });

    it("says so when nobody has been active yet", () => {
        // given
        stubStats(makeAdminStats({ most_active_users: [] }));

        // when
        renderWithProviders(<AdminDashboard />);

        // then
        expect(screen.getByText("No active users yet")).toBeInTheDocument();
    });

    it("ranks the most active users with their action counts", () => {
        // given
        stubStats(
            makeAdminStats({
                most_active_users: [
                    {
                        id: "u1",
                        username: "beatrice",
                        display_name: "Beatrice",
                        avatar_url: "https://example.com/beato.png",
                        action_count: 42,
                    },
                    { id: "u2", username: "battler", display_name: "Battler", avatar_url: "", action_count: 7 },
                ],
            }),
        );

        // when
        renderWithProviders(<AdminDashboard />);

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText("42 actions")).toBeInTheDocument();
        expect(screen.getByText("7 actions")).toBeInTheDocument();
    });

    it("falls back to the first letter of a display name when a user has no avatar", () => {
        // given
        stubStats(
            makeAdminStats({
                most_active_users: [
                    { id: "u2", username: "battler", display_name: "Battler", avatar_url: "", action_count: 7 },
                ],
            }),
        );

        // when
        const { container } = renderWithProviders(<AdminDashboard />);

        // then
        expect(screen.getByText("B")).toBeInTheDocument();
        expect(container.querySelector("img")).toBeNull();
    });
});
