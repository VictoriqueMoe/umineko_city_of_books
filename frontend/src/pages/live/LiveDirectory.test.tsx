import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { LiveStream, LiveStreamListResponse } from "../../api/endpoints";
import { makeUser } from "../../test-utils/fixtures";
import { createTestQueryClient, renderWithProviders } from "../../test-utils/render";
import type { WSMessage } from "../../types/api";
import { LiveDirectory } from "./LiveDirectory";

const mocks = vi.hoisted(() => ({ listLiveStreams: vi.fn() }));

vi.mock("../../api/endpoints", () => ({ listLiveStreams: mocks.listLiveStreams }));

vi.mock("../../components/live/GoLivePanel", () => ({
    GoLivePanel: (props: { onChanged?: () => void }) => (
        <button type="button" onClick={props.onChanged}>
            go live panel
        </button>
    ),
}));

function makeStream(overrides: Partial<LiveStream> = {}): LiveStream {
    return {
        id: "stream-1",
        userId: "user-1",
        title: "Reading Episode 4",
        status: "live",
        viewerCount: 7,
        streamerUsername: "beatrice",
        streamerDisplayName: "Beatrice",
        streamerAvatarUrl: "",
        defaultMode: "webrtc",
        ...overrides,
    };
}

function stubStreams(response: Partial<LiveStreamListResponse> = {}) {
    mocks.listLiveStreams.mockResolvedValue({
        streams: response.streams ?? [],
        enabled: response.enabled ?? true,
    });
}

function renderDirectory(options: { user?: ReturnType<typeof makeUser> | null } = {}) {
    const listeners: ((msg: WSMessage) => void)[] = [];
    const queryClient = createTestQueryClient();
    const result = renderWithProviders(<LiveDirectory />, {
        user: options.user ?? null,
        queryClient,
        notification: {
            addWSListener: listener => {
                listeners.push(listener);
                return () => {};
            },
        },
    });

    return { ...result, listeners, queryClient };
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
        expect(screen.getByRole("button", { name: "go live panel" })).toBeInTheDocument();
        await user.click(screen.getByRole("button", { name: "Close" }));
        expect(screen.queryByRole("button", { name: "go live panel" })).not.toBeInTheDocument();
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
        expect(screen.getByRole("link", { name: /Ciconia blind run/ })).toHaveAttribute("href", "/live/stream-9");
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

    it("refetches the directory when a stream goes live", async () => {
        // given
        const { listeners, queryClient } = renderDirectory();
        await screen.findByText("No one is live right now. Be the first!");
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        listeners[0]({ type: "stream_live", data: {} });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["streams", "live"] });
    });

    it("refetches the directory when a stream goes offline", async () => {
        // given
        const { listeners, queryClient } = renderDirectory();
        await screen.findByText("No one is live right now. Be the first!");
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        listeners[0]({ type: "stream_offline", data: {} });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: ["streams", "live"] });
    });

    it("updates a single stream's watcher count in place", async () => {
        // given
        stubStreams({ streams: [makeStream({ id: "stream-9", viewerCount: 2 })] });
        const { listeners } = renderDirectory();
        await screen.findByText("Reading Episode 4");

        // when
        listeners[0]({ type: "stream_viewers", data: { streamId: "stream-9", viewerCount: 44 } });

        // then
        await waitFor(() => {
            expect(screen.getByText(/44/)).toBeInTheDocument();
        });
    });

    it("updates a single stream's title in place", async () => {
        // given
        stubStreams({ streams: [makeStream({ id: "stream-9" })] });
        const { listeners } = renderDirectory();
        await screen.findByText("Reading Episode 4");

        // when
        listeners[0]({ type: "stream_title", data: { streamId: "stream-9", title: "Now solving the epitaph" } });

        // then
        await waitFor(() => {
            expect(screen.getByText("Now solving the epitaph")).toBeInTheDocument();
        });
    });

    it("leaves other streams alone when one of them changes title", async () => {
        // given
        stubStreams({
            streams: [makeStream({ id: "stream-9" }), makeStream({ id: "stream-8", title: "Higurashi marathon" })],
        });
        const { listeners } = renderDirectory();
        await screen.findByText("Higurashi marathon");

        // when
        listeners[0]({ type: "stream_title", data: { streamId: "stream-9", title: "Now solving the epitaph" } });

        // then
        await waitFor(() => {
            expect(screen.getByText("Now solving the epitaph")).toBeInTheDocument();
        });
        expect(screen.getByText("Higurashi marathon")).toBeInTheDocument();
    });
});
