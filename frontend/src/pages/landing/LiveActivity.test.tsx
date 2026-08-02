import { screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import type { HomeActivityEntry, HomeActivityResponse, HomeMember, HomePublicRoom } from "../../types/api";
import { LiveActivity } from "./LiveActivity";

const { homeActivity } = vi.hoisted(() => ({ homeActivity: { data: null as HomeActivityResponse | null } }));

vi.mock("../../api/queries/sidebar", () => ({
    useHomeActivity: () => homeActivity,
}));

const NOW = "2026-02-01T12:00:00Z";

function makeEntry(overrides: Partial<HomeActivityEntry> = {}): HomeActivityEntry {
    return {
        kind: "theory",
        id: "entry-1",
        title: "The Golden Truth",
        excerpt: "",
        corner: "umineko",
        url: "/theories/entry-1",
        created_at: "2026-02-01T10:00:00Z",
        author: {
            id: "author-1",
            username: "beatrice",
            display_name: "Beatrice",
            avatar_url: "",
        },
        ...overrides,
    };
}

function makeMember(overrides: Partial<HomeMember> = {}): HomeMember {
    return {
        id: "member-1",
        username: "battler",
        display_name: "Battler",
        avatar_url: "",
        created_at: NOW,
        ...overrides,
    };
}

function makeRoom(overrides: Partial<HomePublicRoom> = {}): HomePublicRoom {
    return {
        id: "room-1",
        name: "Tea Parlour",
        description: "Somewhere to sit",
        member_count: 3,
        last_message_at: null,
        ...overrides,
    };
}

function makeActivity(overrides: Partial<HomeActivityResponse> = {}): HomeActivityResponse {
    return {
        online_count: 5,
        recent_activity: [],
        recent_members: [],
        public_rooms: [],
        corner_activity: [],
        ...overrides,
    };
}

beforeEach(() => {
    homeActivity.data = makeActivity();
});

describe("LiveActivity", () => {
    it("listens quietly until the activity has arrived", () => {
        // given
        homeActivity.data = null;

        // when
        renderWithProviders(<LiveActivity />);

        // then
        expect(screen.getByText("Listening...")).toBeInTheDocument();
        expect(screen.queryByText("Recent activity")).not.toBeInTheDocument();
    });

    it("announces how many witnesses are online right now", () => {
        // given
        homeActivity.data = makeActivity({ online_count: 42 });

        // when
        renderWithProviders(<LiveActivity />);

        // then
        expect(screen.getByText(/42 online now/)).toBeInTheDocument();
    });

    it("links each recent entry to where it was posted and labels its kind", () => {
        // given
        homeActivity.data = makeActivity({
            recent_activity: [
                makeEntry({ kind: "art", id: "art-1", title: "Witch in Gold", url: "/gallery/art-1" }),
                makeEntry({ kind: "journal", id: "journal-1", title: "First Read", url: "/journals/journal-1" }),
            ],
        });

        // when
        renderWithProviders(<LiveActivity />);

        // then
        expect(screen.getByRole("link", { name: /Witch in Gold/ })).toHaveAttribute("href", "/gallery/art-1");
        expect(screen.getByText("Gallery")).toBeInTheDocument();
        expect(screen.getByText("Journal")).toBeInTheDocument();
    });

    it("falls back to the excerpt when an entry has no title", () => {
        // given
        homeActivity.data = makeActivity({
            recent_activity: [makeEntry({ title: "", excerpt: "  a fragment of blue truth  " })],
        });

        // when
        renderWithProviders(<LiveActivity />);

        // then
        expect(screen.getByText("a fragment of blue truth")).toBeInTheDocument();
    });

    it("trims a very long excerpt down to a readable title", () => {
        // given
        const excerpt = "z".repeat(120);
        homeActivity.data = makeActivity({ recent_activity: [makeEntry({ title: "", excerpt })] });

        // when
        renderWithProviders(<LiveActivity />);

        // then
        expect(screen.getByText(`${"z".repeat(80)}…`)).toBeInTheDocument();
    });

    it("names an entry after its kind when there is neither title nor excerpt", () => {
        // given
        homeActivity.data = makeActivity({
            recent_activity: [makeEntry({ kind: "post", title: "", excerpt: "   " })],
        });

        // when
        renderWithProviders(<LiveActivity />);

        // then
        expect(screen.getByText("Post entry")).toBeInTheDocument();
    });

    it("says the board is quiet when nobody has posted", () => {
        // given
        homeActivity.data = makeActivity({ recent_activity: [] });

        // when
        renderWithProviders(<LiveActivity />);

        // then
        expect(screen.getByText("The board is quiet. Be the first to post.")).toBeInTheDocument();
    });

    it("offers to open a room when there are no public rooms yet", () => {
        // given
        homeActivity.data = makeActivity({ public_rooms: [] });

        // when
        renderWithProviders(<LiveActivity />);

        // then
        expect(screen.getByText(/No public rooms yet/)).toBeInTheDocument();
        expect(screen.getByRole("link", { name: "Open one" })).toHaveAttribute("href", "/rooms");
    });

    it("lists each public room with a link and its member count", () => {
        // given
        homeActivity.data = makeActivity({
            public_rooms: [makeRoom({ id: "room-9", name: "Rokkenjima", member_count: 4 })],
        });

        // when
        renderWithProviders(<LiveActivity />);

        // then
        expect(screen.getByRole("link", { name: /Rokkenjima/ })).toHaveAttribute("href", "/rooms/room-9");
        expect(screen.getByText(/4 witches/)).toBeInTheDocument();
    });

    it("keeps the member wording singular for a room of one", () => {
        // given
        homeActivity.data = makeActivity({ public_rooms: [makeRoom({ member_count: 1 })] });

        // when
        renderWithProviders(<LiveActivity />);

        // then
        expect(screen.getByText(/1 witch/)).toBeInTheDocument();
        expect(screen.queryByText(/witches/)).not.toBeInTheDocument();
    });

    it("gives an unnamed room a placeholder name", () => {
        // given
        homeActivity.data = makeActivity({ public_rooms: [makeRoom({ name: "" })] });

        // when
        renderWithProviders(<LiveActivity />);

        // then
        expect(screen.getByText("Untitled room")).toBeInTheDocument();
    });

    it("says how long ago a room last stirred", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date(NOW));
        homeActivity.data = makeActivity({
            public_rooms: [makeRoom({ last_message_at: "2026-02-01T09:00:00Z" })],
        });

        // when
        renderWithProviders(<LiveActivity />);

        // then
        expect(screen.getByText(/3h ago/)).toBeInTheDocument();
    });

    it("says nothing about new sign-ups when nobody has joined", () => {
        // given
        homeActivity.data = makeActivity({ recent_members: [] });

        // when
        renderWithProviders(<LiveActivity />);

        // then
        expect(screen.getByText("No new sign-ups yet.")).toBeInTheDocument();
    });

    it("welcomes each new witness by name", () => {
        // given
        homeActivity.data = makeActivity({
            recent_members: [
                makeMember({ id: "m1", username: "ange", display_name: "Ange" }),
                makeMember({ id: "m2", username: "maria", display_name: "Maria" }),
            ],
        });

        // when
        renderWithProviders(<LiveActivity />);

        // then
        expect(screen.getByRole("link", { name: /Ange/ })).toHaveAttribute("href", "/user/ange");
        expect(screen.getByRole("link", { name: /Maria/ })).toHaveAttribute("href", "/user/maria");
    });
});
