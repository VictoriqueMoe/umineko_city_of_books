import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeStream as makeLiveStream, makeUser } from "../../test-utils/fixtures";
import { renderWithProviders } from "../../test-utils/render";
import type { LiveStream, LiveStreamListResponse } from "../../types/api";
import { LiveDirectory } from "./LiveDirectory";

const mocks = vi.hoisted(() => ({
    useLiveDirectory: vi.fn(),
}));

vi.mock("../../hooks/useLiveDirectory", () => ({ useLiveDirectory: mocks.useLiveDirectory }));

vi.mock("../../components/live/GoLivePanel", () => ({
    GoLivePanel: () => <div>go live panel</div>,
}));

function makeStream(overrides: Partial<LiveStream> = {}): LiveStream {
    return makeLiveStream({
        title: "Reading Episode 4",
        viewerCount: 7,
        streamerUsername: "beatrice",
        ...overrides,
    });
}

function stubStreams(response: Partial<LiveStreamListResponse> = {}) {
    mocks.useLiveDirectory.mockReturnValue({
        streams: response.streams ?? [],
        enabled: response.enabled ?? true,
        loading: false,
    });
}

function renderDirectory(options: { user?: ReturnType<typeof makeUser> | null } = {}) {
    return renderWithProviders(<LiveDirectory />, { user: options.user ?? null });
}

beforeEach(() => {
    stubStreams();
});

describe("LiveDirectory", () => {
    it("says live streaming is switched off when the server has disabled it", async () => {
        // given
        stubStreams({ enabled: false });

        // when
        renderDirectory({ user: makeUser() });

        // then
        expect(await screen.findByText("Live streaming is currently disabled.")).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Go live" })).not.toBeInTheDocument();
    });

    it("invites the first broadcaster when nobody is live", async () => {
        // given
        stubStreams({ streams: [] });

        // when
        renderDirectory();

        // then
        expect(await screen.findByText("No one is live right now. Be the first!")).toBeInTheDocument();
    });

    it("explains what live is to anybody who lands here", async () => {
        // given
        stubStreams();

        // when
        renderDirectory();

        // then
        expect(await screen.findByText("What is Live?")).toBeInTheDocument();
    });

    it("hides the go live button from a signed out visitor", async () => {
        // given
        stubStreams({ enabled: true });

        // when
        renderDirectory({ user: null });

        // then
        await screen.findByText("No one is live right now. Be the first!");
        expect(screen.queryByRole("button", { name: "Go live" })).not.toBeInTheDocument();
    });

    it("opens and closes the go live panel for a signed in member", async () => {
        // given
        stubStreams({ enabled: true });
        const user = userEvent.setup();
        renderDirectory({ user: makeUser() });
        const goLive = await screen.findByRole("button", { name: "Go live" });

        // when
        await user.click(goLive);

        // then
        expect(screen.getByText("go live panel")).toBeInTheDocument();
        await user.click(screen.getByRole("button", { name: "Close" }));
        expect(screen.queryByText("go live panel")).not.toBeInTheDocument();
    });

    it("lists each live stream with its title, streamer and watcher count", async () => {
        // given
        stubStreams({
            streams: [makeStream({ id: "stream-9", title: "Ciconia blind run", viewerCount: 12 })],
        });

        // when
        renderDirectory();

        // then
        expect(await screen.findByText("Ciconia blind run")).toBeInTheDocument();
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText(/12/)).toBeInTheDocument();
        expect(screen.getByRole("link", { name: /Ciconia blind run/ })).toHaveAttribute("href", "/beatrice/live");
    });

    it("falls back to the streamer's username when they have no display name", async () => {
        // given
        stubStreams({ streams: [makeStream({ streamerDisplayName: "" })] });

        // when
        renderDirectory();

        // then
        expect(await screen.findByText("beatrice")).toBeInTheDocument();
    });

    it("stands in with the streamer's initial when there is no thumbnail or avatar", async () => {
        // given
        stubStreams({ streams: [makeStream({ thumbnailUrl: "", streamerAvatarUrl: "" })] });

        // when
        renderDirectory();

        // then
        expect(await screen.findByText("B")).toBeInTheDocument();
    });

    it("prefers the thumbnail over the avatar on the card", async () => {
        // given
        stubStreams({
            streams: [makeStream({ thumbnailUrl: "/media/thumb.png", streamerAvatarUrl: "/media/avatar.png" })],
        });

        // when
        const { container } = renderDirectory();

        // then
        await screen.findByText("Reading Episode 4");
        const sources = Array.from(container.querySelectorAll("img")).map(img => img.getAttribute("src"));
        expect(sources).toContain("/media/thumb.png");
        expect(sources.filter(src => src === "/media/avatar.png")).toHaveLength(1);
    });
});
