import { act, renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { queryKeys } from "../api/queryKeys";
import type { RealtimeEvent } from "../api/realtime/events";
import { makeStream as makeLiveStream } from "../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../test-utils/render";
import { emitSettledRealtimeEvent } from "../test-utils/ws";
import type { LiveStream, StreamCredentials, StreamDefaultMode, StreamOwner } from "../types/api";
import { useGoLive } from "./useGoLive";

const mocks = vi.hoisted(() => ({
    getMyStream: vi.fn(),
    getStream: vi.fn(),
    getStreamCredentials: vi.fn(),
    getStreamViewerToken: vi.fn(),
    joinStreamChat: vi.fn(),
    listLiveStreams: vi.fn(),
    resetStreamCredentials: vi.fn(),
    startStream: vi.fn(),
    stopStream: vi.fn(),
    updateStreamTitle: vi.fn(),
    uploadStreamThumbnail: vi.fn(),
}));

vi.mock("../api/endpoints/stream", () => mocks);

const STREAM_ID = "stream-1";
const WHIP_URL = "https://ingest.example/whip";

function makeStream(overrides: Partial<LiveStream> = {}): LiveStream {
    return makeLiveStream({ id: STREAM_ID, status: "pending", viewerCount: 0, ...overrides });
}

function makeOwner(overrides: Partial<LiveStream> = {}): StreamOwner {
    return { stream: makeStream(overrides), whipUrl: WHIP_URL, streamKey: "sk_old" };
}

function makeCredentials(overrides: Partial<StreamCredentials> = {}): StreamCredentials {
    return { whipUrl: WHIP_URL, streamKey: "sk_old", hlsEnabled: false, ...overrides };
}

let queryClient: QueryClient;
let serverOwner: StreamOwner | null;
let serverCredentials: StreamCredentials;

async function emit(event: RealtimeEvent): Promise<void> {
    await emitSettledRealtimeEvent(event);
}

function mount() {
    return renderHook(() => useGoLive(), { wrapper: providerWrapper({ queryClient }) });
}

async function mountLoaded() {
    const view = mount();
    await waitFor(() => expect(view.result.current.owner).toEqual(serverOwner));
    await waitFor(() => expect(view.result.current.credentials).toEqual(serverCredentials));
    await waitFor(() => expect(view.result.current.busy).toBe(false));

    return view;
}

function invalidationsOf(calls: unknown[][], key: readonly unknown[]): number {
    const wanted = JSON.stringify(key);

    return calls.filter(call => JSON.stringify((call[0] as { queryKey?: unknown })?.queryKey) === wanted).length;
}

beforeEach(() => {
    localStorage.clear();
    queryClient = createTestQueryClient();
    serverOwner = null;
    serverCredentials = makeCredentials();

    mocks.getMyStream.mockImplementation(() => Promise.resolve(serverOwner));
    mocks.getStreamCredentials.mockImplementation(() => Promise.resolve(serverCredentials));
    mocks.startStream.mockImplementation((title: string, mode: StreamDefaultMode) => {
        serverOwner = makeOwner({ title, defaultMode: mode });

        return Promise.resolve(serverOwner);
    });
    mocks.stopStream.mockImplementation(() => {
        serverOwner = null;

        return Promise.resolve(undefined);
    });
    mocks.updateStreamTitle.mockImplementation((_streamId: string, title: string) => {
        const current = serverOwner;

        if (!current) {
            return Promise.reject(new Error("not live"));
        }

        serverOwner = { ...current, stream: { ...current.stream, title } };

        return Promise.resolve(serverOwner.stream);
    });
    mocks.resetStreamCredentials.mockImplementation(() => {
        serverCredentials = makeCredentials({ streamKey: "sk_new_key" });

        return Promise.resolve(serverCredentials);
    });
});

afterEach(() => {
    vi.restoreAllMocks();
});

describe("useGoLive loading", () => {
    it("hands over the owner view and the credentials the server holds", async () => {
        // given
        serverOwner = makeOwner({ status: "live" });

        // when
        const view = await mountLoaded();

        // then
        expect(view.result.current.owner?.stream.status).toBe("live");
        expect(view.result.current.credentials?.whipUrl).toBe(WHIP_URL);
        expect(view.result.current.ownerError).toBe("");
    });

    it("says why the owner view would not load instead of looking like nobody is live", async () => {
        // given
        mocks.getMyStream.mockRejectedValue(new Error("The stream service is down"));

        // when
        const view = mount();

        // then
        await waitFor(() => expect(view.result.current.ownerError).toBe("The stream service is down"));
        expect(view.result.current.owner).toBeNull();
    });

    it("apologises in general terms when the owner view fails with no message", async () => {
        // given
        mocks.getMyStream.mockRejectedValue({});

        // when
        const view = mount();

        // then
        await waitFor(() => expect(view.result.current.ownerError).toBe("Could not load your stream."));
    });

    it("says why the stream key would not load", async () => {
        // given
        mocks.getStreamCredentials.mockRejectedValue(new Error("gone"));

        // when
        const view = mount();

        // then
        await waitFor(() => expect(view.result.current.credentialsError).toBe("gone"));
    });
});

describe("useGoLive going live", () => {
    it("starts the stream on low latency with no bitrate when smooth playback is off", async () => {
        // given
        const view = await mountLoaded();
        act(() => view.result.current.setTitle("  Tea with the witch  "));

        // when
        await act(async () => {
            view.result.current.start();
        });

        // then
        expect(mocks.startStream).toHaveBeenCalledWith("Tea with the witch", "webrtc", 0);
        await waitFor(() => expect(view.result.current.owner?.stream.title).toBe("Tea with the witch"));
        expect(localStorage.getItem("stream.bitrateKbps")).toBeNull();
    });

    it("sends the chosen mode and bitrate and remembers the bitrate when smooth playback is on", async () => {
        // given
        serverCredentials = makeCredentials({ hlsEnabled: true });
        const view = await mountLoaded();
        act(() => {
            view.result.current.setTitle("Tea with the witch");
            view.result.current.setBitrate("6000");
            view.result.current.setDefaultMode("hls");
        });

        // when
        await act(async () => {
            view.result.current.start();
        });

        // then
        expect(mocks.startStream).toHaveBeenCalledWith("Tea with the witch", "hls", 6000);
        expect(localStorage.getItem("stream.bitrateKbps")).toBe("6000");
    });

    it("refuses to start on a bitrate the form would reject", async () => {
        // given
        serverCredentials = makeCredentials({ hlsEnabled: true });
        const view = await mountLoaded();
        act(() => {
            view.result.current.setTitle("Tea with the witch");
            view.result.current.setBitrate("100");
        });

        // when
        await act(async () => {
            view.result.current.start();
        });

        // then
        expect(view.result.current.canStart).toBe(false);
        expect(mocks.startStream).not.toHaveBeenCalled();
    });

    it("says why the stream would not start", async () => {
        // given
        mocks.startStream.mockRejectedValue(new Error("You are already streaming"));
        const view = await mountLoaded();
        act(() => view.result.current.setTitle("Tea with the witch"));

        // when
        await act(async () => {
            view.result.current.start();
        });

        // then
        expect(view.result.current.error).toBe("You are already streaming");
    });

    it("reads the stream key back after going live in case the ingress was reprovisioned", async () => {
        // given
        const view = await mountLoaded();
        const before = mocks.getStreamCredentials.mock.calls.length;
        act(() => view.result.current.setTitle("Tea with the witch"));

        // when
        await act(async () => {
            view.result.current.start();
        });

        // then
        await waitFor(() => expect(mocks.getStreamCredentials.mock.calls.length).toBeGreaterThan(before));
    });

    it("refreshes the directory itself instead of asking the page to do it", async () => {
        // given
        const view = await mountLoaded();
        act(() => view.result.current.setTitle("Tea with the witch"));
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        await act(async () => {
            view.result.current.start();
        });

        // then
        expect(invalidationsOf(invalidateQueries.mock.calls, queryKeys.streams.live())).toBe(1);
    });
});

describe("useGoLive stopping", () => {
    it("stops the stream and empties the title box", async () => {
        // given
        serverOwner = makeOwner({ status: "live" });
        const view = await mountLoaded();
        act(() => view.result.current.setTitle("left over"));

        // when
        await act(async () => {
            view.result.current.stop();
        });

        // then
        expect(mocks.stopStream).toHaveBeenCalledWith(STREAM_ID);
        await waitFor(() => expect(view.result.current.owner).toBeNull());
        expect(view.result.current.title).toBe("");
    });

    it("says why the stream would not stop", async () => {
        // given
        serverOwner = makeOwner({ status: "live" });
        mocks.stopStream.mockRejectedValue(new Error("The ingest is wedged"));
        const view = await mountLoaded();

        // when
        await act(async () => {
            view.result.current.stop();
        });

        // then
        expect(view.result.current.error).toBe("The ingest is wedged");
    });
});

describe("useGoLive title editing", () => {
    it("opens the editor on the title that is live", async () => {
        // given
        serverOwner = makeOwner({ status: "live", title: "Tea with the witch" });
        const view = await mountLoaded();

        // when
        act(() => view.result.current.titleEdit.open());

        // then
        expect(view.result.current.titleEdit.editing).toBe(true);
        expect(view.result.current.titleEdit.draft).toBe("Tea with the witch");
        expect(view.result.current.titleEdit.canSave).toBe(false);
    });

    it("saves a changed title and closes the editor", async () => {
        // given
        serverOwner = makeOwner({ status: "live", title: "Tea with the witch" });
        const view = await mountLoaded();
        act(() => view.result.current.titleEdit.open());
        act(() => view.result.current.titleEdit.setDraft("Cake with the witch"));

        // when
        await act(async () => {
            view.result.current.titleEdit.save();
        });

        // then
        expect(mocks.updateStreamTitle).toHaveBeenCalledWith(STREAM_ID, "Cake with the witch");
        await waitFor(() => expect(view.result.current.owner?.stream.title).toBe("Cake with the witch"));
        expect(view.result.current.titleEdit.editing).toBe(false);
    });

    it("just closes the editor when the title has not moved", async () => {
        // given
        serverOwner = makeOwner({ status: "live", title: "Tea with the witch" });
        const view = await mountLoaded();
        act(() => view.result.current.titleEdit.open());

        // when
        await act(async () => {
            view.result.current.titleEdit.save();
        });

        // then
        expect(mocks.updateStreamTitle).not.toHaveBeenCalled();
        expect(view.result.current.titleEdit.editing).toBe(false);
    });

    it("says why the title would not save and keeps the editor open", async () => {
        // given
        serverOwner = makeOwner({ status: "live", title: "Tea with the witch" });
        mocks.updateStreamTitle.mockRejectedValue(new Error("That title is not allowed"));
        const view = await mountLoaded();
        act(() => view.result.current.titleEdit.open());
        act(() => view.result.current.titleEdit.setDraft("Cake with the witch"));

        // when
        await act(async () => {
            view.result.current.titleEdit.save();
        });

        // then
        expect(view.result.current.error).toBe("That title is not allowed");
        expect(view.result.current.titleEdit.editing).toBe(true);
    });
});

describe("useGoLive stream key", () => {
    it("leaves the key alone when the reset is not confirmed", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(false);
        const view = await mountLoaded();

        // when
        await act(async () => {
            view.result.current.resetCredentials();
        });

        // then
        expect(mocks.resetStreamCredentials).not.toHaveBeenCalled();
    });

    it("hands back the new key once the reset is confirmed", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(true);
        const view = await mountLoaded();

        // when
        await act(async () => {
            view.result.current.resetCredentials();
        });

        // then
        await waitFor(() => expect(view.result.current.credentials?.streamKey).toBe("sk_new_key"));
    });

    it("will not reset the key while a stream of your own is up", async () => {
        // given
        serverOwner = makeOwner({ status: "live" });
        const confirm = vi.spyOn(window, "confirm").mockReturnValue(true);
        const view = await mountLoaded();

        // when
        await act(async () => {
            view.result.current.resetCredentials();
        });

        // then
        expect(confirm).not.toHaveBeenCalled();
        expect(mocks.resetStreamCredentials).not.toHaveBeenCalled();
    });

    it("says why the key would not reset", async () => {
        // given
        vi.spyOn(window, "confirm").mockReturnValue(true);
        mocks.resetStreamCredentials.mockRejectedValue(new Error("Try again later"));
        const view = await mountLoaded();

        // when
        await act(async () => {
            view.result.current.resetCredentials();
        });

        // then
        expect(view.result.current.error).toBe("Try again later");
    });
});

describe("useGoLive live updates", () => {
    it("flips its own stream to live when the server says so", async () => {
        // given
        serverOwner = makeOwner({ status: "pending" });
        const view = await mountLoaded();

        // when
        await emit({ type: "stream_live", data: makeStream({ status: "live" }) });

        // then
        expect(view.result.current.owner?.stream.status).toBe("live");
    });

    it("takes on a title that was changed somewhere else", async () => {
        // given
        serverOwner = makeOwner({ status: "live", title: "Tea with the witch" });
        const view = await mountLoaded();

        // when
        await emit({ type: "stream_title", data: { streamId: STREAM_ID, title: "Cake with the witch" } });

        // then
        expect(view.result.current.owner?.stream.title).toBe("Cake with the witch");
    });

    it("drops the owner view and empties the title box when its stream goes offline", async () => {
        // given
        serverOwner = makeOwner({ status: "live" });
        const view = await mountLoaded();
        act(() => view.result.current.setTitle("left over"));

        // when
        await emit({ type: "stream_offline", data: { streamId: STREAM_ID } });

        // then
        expect(view.result.current.owner).toBeNull();
        expect(view.result.current.title).toBe("");
    });

    it("leaves the directory to the sync hook when its own stream goes offline", async () => {
        // given
        serverOwner = makeOwner({ status: "live" });
        await mountLoaded();
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        await emit({ type: "stream_offline", data: { streamId: STREAM_ID } });

        // then
        expect(invalidationsOf(invalidateQueries.mock.calls, queryKeys.streams.live())).toBe(0);
    });

    it("pays no attention to news about somebody else's stream", async () => {
        // given
        serverOwner = makeOwner({ status: "live", title: "Tea with the witch" });
        const view = await mountLoaded();

        // when
        await emit({ type: "stream_title", data: { streamId: "stream-other", title: "Not mine" } });
        await emit({ type: "stream_offline", data: { streamId: "stream-other" } });

        // then
        expect(view.result.current.owner?.stream.title).toBe("Tea with the witch");
    });
});

describe("useGoLive clipboard", () => {
    it("remembers which value was copied", async () => {
        // given
        vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
        const view = await mountLoaded();

        // when
        await act(async () => {
            view.result.current.copy("url", WHIP_URL);
        });

        // then
        expect(view.result.current.copied).toBe("url");
    });

    it("says nothing was copied when the clipboard refuses", async () => {
        // given
        vi.spyOn(navigator.clipboard, "writeText").mockRejectedValue(new Error("denied"));
        const view = await mountLoaded();

        // when
        await act(async () => {
            view.result.current.copy("url", WHIP_URL);
        });

        // then
        expect(view.result.current.copied).toBeNull();
    });
});
