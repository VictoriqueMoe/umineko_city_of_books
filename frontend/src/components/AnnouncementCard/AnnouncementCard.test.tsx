import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { Announcement } from "../../types/api";
import { renderWithProviders } from "../../test-utils/render";
import { AnnouncementCard } from "./AnnouncementCard";

const { useLatestAnnouncement } = vi.hoisted(() => ({ useLatestAnnouncement: vi.fn() }));

vi.mock("../../api/queries/announcement", () => ({ useLatestAnnouncement }));

const DISMISSED_KEY = "dismissed_announcement";

function makeAnnouncement(overrides: Partial<Announcement> = {}): Announcement {
    return {
        id: "ann-1",
        title: "The Golden Witch returns",
        body: "Read the **rules** before posting.",
        author: {
            id: "00000000-0000-0000-0000-000000000001",
            username: "beatrice",
            display_name: "Beatrice",
        },
        pinned: false,
        created_at: "2026-01-01T00:00:00Z",
        updated_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

function LocationProbe() {
    const location = useLocation();

    return <span>{`at ${location.pathname}`}</span>;
}

function renderCard() {
    return renderWithProviders(
        <>
            <AnnouncementCard />
            <LocationProbe />
        </>,
    );
}

beforeEach(() => {
    useLatestAnnouncement.mockReturnValue({ announcement: makeAnnouncement(), loading: false });
});

describe("AnnouncementCard", () => {
    it("shows nothing while there is no announcement to share", () => {
        // given
        useLatestAnnouncement.mockReturnValue({ announcement: null, loading: false });

        // when
        const { container } = renderWithProviders(<AnnouncementCard />);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("shows the announcement title and its author", () => {
        // given
        const announcement = makeAnnouncement({ title: "Beware the Witch Hunter" });
        useLatestAnnouncement.mockReturnValue({ announcement, loading: false });

        // when
        renderWithProviders(<AnnouncementCard />);

        // then
        expect(screen.getByText("Beware the Witch Hunter")).toBeInTheDocument();
        expect(screen.getByText("Announcement")).toBeInTheDocument();
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
    });

    it("renders the body as markdown rather than raw text", () => {
        // given
        const announcement = makeAnnouncement({ body: "Read the **rules** before posting." });
        useLatestAnnouncement.mockReturnValue({ announcement, loading: false });

        // when
        renderWithProviders(<AnnouncementCard />);

        // then
        expect(screen.getByText("rules").tagName).toBe("STRONG");
    });

    it("strips dangerous markup out of the body", () => {
        // given
        const announcement = makeAnnouncement({ body: "<script>alert(1)</script>\n\nStill safe." });
        useLatestAnnouncement.mockReturnValue({ announcement, loading: false });

        // when
        const { container } = renderWithProviders(<AnnouncementCard />);

        // then
        expect(container.querySelector("script")).toBeNull();
        expect(screen.getByText("Still safe.")).toBeInTheDocument();
    });

    it("shows how long ago the announcement was posted", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-01-01T02:00:00Z"));
        useLatestAnnouncement.mockReturnValue({
            announcement: makeAnnouncement({ created_at: "2026-01-01T00:00:00Z" }),
            loading: false,
        });

        // when
        renderWithProviders(<AnnouncementCard />);

        // then
        expect(screen.getByText("2h ago")).toBeInTheDocument();
    });

    it("stays hidden for an announcement the reader already dismissed", () => {
        // given
        localStorage.setItem(DISMISSED_KEY, "ann-1");

        // when
        const { container } = renderWithProviders(<AnnouncementCard />);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("shows a newer announcement even though an older one was dismissed", () => {
        // given
        localStorage.setItem(DISMISSED_KEY, "ann-0");

        // when
        renderWithProviders(<AnnouncementCard />);

        // then
        expect(screen.getByText("The Golden Witch returns")).toBeInTheDocument();
    });

    it("remembers the announcement the reader dismisses", async () => {
        // given
        const user = userEvent.setup();
        renderWithProviders(<AnnouncementCard />);

        // when
        await user.click(screen.getByTitle("Dismiss"));

        // then
        expect(screen.queryByText("The Golden Witch returns")).not.toBeInTheDocument();
        expect(localStorage.getItem(DISMISSED_KEY)).toBe("ann-1");
    });

    it("opens the announcement when the title is clicked", async () => {
        // given
        const user = userEvent.setup();
        renderCard();

        // when
        await user.click(screen.getByText("The Golden Witch returns"));

        // then
        expect(screen.getByText("at /announcements/ann-1")).toBeInTheDocument();
    });

    it("opens the announcement from the read more link", async () => {
        // given
        const user = userEvent.setup();
        renderCard();

        // when
        await user.click(screen.getByText(/Read more/));

        // then
        expect(screen.getByText("at /announcements/ann-1")).toBeInTheDocument();
    });
});
