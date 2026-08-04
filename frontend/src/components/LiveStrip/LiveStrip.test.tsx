import { screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { HomeActivityResponse, HomeCornerActivity, HomeMember } from "../../types/api";
import { LiveStrip } from "./LiveStrip";

const { homeActivity } = vi.hoisted(() => ({ homeActivity: { data: null as HomeActivityResponse | null } }));

vi.mock("../../api/queries/sidebar", () => ({
    useHomeActivity: () => homeActivity,
}));

const NOW = "2026-02-01T12:00:00Z";

function makeCorner(overrides: Partial<HomeCornerActivity> = {}): HomeCornerActivity {
    return {
        corner: "umineko",
        post_count: 4,
        unique_posters: 2,
        last_post_at: null,
        ...overrides,
    };
}

function makeMember(id: string): HomeMember {
    return {
        id,
        username: `witch-${id}`,
        display_name: `Witch ${id}`,
        avatar_url: "",
        created_at: NOW,
    };
}

function makeActivity(overrides: Partial<HomeActivityResponse> = {}): HomeActivityResponse {
    return {
        online_count: 7,
        recent_activity: [],
        recent_members: [],
        public_rooms: [],
        corner_activity: [],
        ...overrides,
    };
}

function renderStrip(corner?: string) {
    return renderWithProviders(<LiveStrip corner={corner} />);
}

describe("LiveStrip", () => {
    beforeEach(() => {
        homeActivity.data = makeActivity();
    });

    afterEach(() => {
        homeActivity.data = null;
    });

    it("stays out of the way until the activity has arrived", () => {
        // given
        homeActivity.data = null;

        // when
        const { container } = renderStrip();

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("shows how many people are online", () => {
        // given
        homeActivity.data = makeActivity({ online_count: 42 });

        // when
        renderStrip();

        // then
        expect(screen.getByText("42")).toBeInTheDocument();
        expect(screen.getByText("online")).toBeInTheDocument();
    });

    it("always offers a way through to the full activity page", () => {
        // given
        homeActivity.data = makeActivity();

        // when
        renderStrip();

        // then
        expect(screen.getByRole("link", { name: /See full activity/ })).toHaveAttribute("href", "/welcome#live");
    });

    it("breaks the last day down by corner when no corner is in focus", () => {
        // given
        homeActivity.data = makeActivity({
            corner_activity: [makeCorner({ corner: "higurashi", post_count: 3 }), makeCorner({ post_count: 9 })],
        });

        // when
        renderStrip();

        // then
        expect(screen.getByText("24h:")).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Umi9" })).toHaveAttribute("href", "/game-board/umineko");
        expect(screen.getByRole("link", { name: "Higu3" })).toHaveAttribute("href", "/game-board/higurashi");
    });

    it("orders the corner chips the way the board is ordered", () => {
        // given
        homeActivity.data = makeActivity({
            corner_activity: [
                makeCorner({ corner: "general" }),
                makeCorner({ corner: "ciconia" }),
                makeCorner({ corner: "umineko" }),
            ],
        });

        // when
        renderStrip();

        // then
        const chips = screen.getAllByRole("link").slice(0, 3);
        expect(chips.map(chip => chip.textContent)).toEqual(["Umi4", "Cico4", "General4"]);
    });

    it("pushes corners it does not recognise to the end under their own name", () => {
        // given
        homeActivity.data = makeActivity({
            corner_activity: [
                makeCorner({ corner: "zzz-unknown" }),
                makeCorner({ corner: "aaa-unknown" }),
                makeCorner({ corner: "umineko" }),
            ],
        });

        // when
        renderStrip();

        // then
        const chips = screen.getAllByRole("link").slice(0, 3);
        expect(chips.map(chip => chip.textContent)).toEqual(["Umi4", "aaa-unknown4", "zzz-unknown4"]);
        expect(screen.getByRole("link", { name: "aaa-unknown4" })).toHaveAttribute("href", "/game-board");
    });

    it("admits when the board has been quiet for a day", () => {
        // given
        homeActivity.data = makeActivity({ corner_activity: [] });

        // when
        renderStrip();

        // then
        expect(screen.getByText("No new posts in the last 24h across the board.")).toBeInTheDocument();
    });

    it("still breaks things down by corner when the general corner is in focus", () => {
        // given
        homeActivity.data = makeActivity({ corner_activity: [makeCorner({ corner: "higanbana", post_count: 2 })] });

        // when
        renderStrip("general");

        // then
        expect(screen.getByRole("link", { name: "Higan2" })).toBeInTheDocument();
    });

    it("focuses on one corner's numbers when that corner is in view", () => {
        // given
        homeActivity.data = makeActivity({
            corner_activity: [makeCorner({ post_count: 5, unique_posters: 3 }), makeCorner({ corner: "ciconia" })],
        });

        // when
        renderStrip("umineko");

        // then
        expect(screen.getByText("5")).toBeInTheDocument();
        expect(screen.getByText("new posts today")).toBeInTheDocument();
        expect(screen.getByText("3")).toBeInTheDocument();
        expect(screen.getByText("posters")).toBeInTheDocument();
        expect(screen.queryByText("24h:")).not.toBeInTheDocument();
    });

    it("keeps the wording singular for a lone post by a lone poster", () => {
        // given
        homeActivity.data = makeActivity({
            corner_activity: [makeCorner({ post_count: 1, unique_posters: 1 })],
        });

        // when
        renderStrip("umineko");

        // then
        expect(screen.getByText("new post today")).toBeInTheDocument();
        expect(screen.getByText("poster")).toBeInTheDocument();
    });

    it("says how long ago the last post landed", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date(NOW));
        homeActivity.data = makeActivity({
            corner_activity: [makeCorner({ last_post_at: "2026-02-01T10:00:00Z" })],
        });

        // when
        renderStrip("umineko");

        // then
        expect(screen.getByText("2h ago").parentElement).toHaveTextContent("last 2h ago");
    });

    it("says nothing about a last post when there has never been one", () => {
        // given
        homeActivity.data = makeActivity({ corner_activity: [makeCorner({ last_post_at: null })] });

        // when
        renderStrip("umineko");

        // then
        expect(screen.queryByText(/^last /)).not.toBeInTheDocument();
    });

    it("invites the first post when the corner in view has none", () => {
        // given
        homeActivity.data = makeActivity({ corner_activity: [makeCorner({ post_count: 0 })] });

        // when
        renderStrip("umineko");

        // then
        expect(screen.getByText("No posts here in the last 24h. Be the first.")).toBeInTheDocument();
    });

    it("invites the first post when the corner in view is missing from the figures", () => {
        // given
        homeActivity.data = makeActivity({ corner_activity: [makeCorner({ corner: "ciconia" })] });

        // when
        renderStrip("higurashi");

        // then
        expect(screen.getByText("No posts here in the last 24h. Be the first.")).toBeInTheDocument();
    });

    it("greets at most three of the newest members", () => {
        // given
        homeActivity.data = makeActivity({
            recent_members: [makeMember("1"), makeMember("2"), makeMember("3"), makeMember("4")],
        });

        // when
        renderStrip();

        // then
        expect(screen.getByText("New:")).toBeInTheDocument();
        const memberLinks = screen.getAllByRole("link").filter(link => link.getAttribute("href")?.startsWith("/user/"));
        expect(memberLinks).toHaveLength(3);
    });

    it("leaves the new members out entirely when nobody has joined", () => {
        // given
        homeActivity.data = makeActivity({ recent_members: [] });

        // when
        renderStrip();

        // then
        expect(screen.queryByText("New:")).not.toBeInTheDocument();
    });
});
