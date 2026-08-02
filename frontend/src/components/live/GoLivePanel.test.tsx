import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { LiveStream, StreamCredentials, StreamOwner } from "../../api/endpoints";
import type { WSMessageHandler } from "../../context/notificationContextValue";
import { renderWithProviders } from "../../test-utils/render";
import type { WSMessage } from "../../types/api";
import { GoLivePanel } from "./GoLivePanel";

const streams = vi.hoisted(() => ({
    getMyStream: vi.fn(),
    getStreamCredentials: vi.fn(),
    resetStreamCredentials: vi.fn(),
    startStream: vi.fn(),
    stopStream: vi.fn(),
    updateStreamTitle: vi.fn(),
}));

vi.mock("../../api/endpoints", () => streams);

const WHIP_URL = "https://ingest.example/whip";
const STREAM_KEY = "sk_beatrice_1986";
const STREAM_ID = "stream-1";

function makeStream(overrides: Partial<LiveStream> = {}): LiveStream {
    return {
        id: STREAM_ID,
        userId: "user-1",
        title: "Reading the message bottle",
        status: "pending",
        viewerCount: 0,
        streamerUsername: "beatrice",
        streamerDisplayName: "Beatrice",
        streamerAvatarUrl: "",
        defaultMode: "webrtc",
        ...overrides,
    };
}

function makeOwner(overrides: Partial<LiveStream> = {}): StreamOwner {
    return { stream: makeStream(overrides), whipUrl: WHIP_URL, streamKey: STREAM_KEY };
}

function makeCreds(overrides: Partial<StreamCredentials> = {}): StreamCredentials {
    return { whipUrl: WHIP_URL, streamKey: STREAM_KEY, hlsEnabled: false, ...overrides };
}

function captureWS() {
    const handlers: WSMessageHandler[] = [];

    function addWSListener(handler: WSMessageHandler) {
        handlers.push(handler);
        return () => {};
    }

    function emit(msg: WSMessage) {
        act(() => {
            for (const handler of handlers) {
                handler(msg);
            }
        });
    }

    return { addWSListener, emit };
}

async function renderPanel(onChanged = vi.fn(), ws = captureWS()) {
    const result = renderWithProviders(<GoLivePanel onChanged={onChanged} />, {
        notification: { addWSListener: ws.addWSListener },
    });
    await screen.findByRole("heading");
    await waitFor(() => expect(streams.getStreamCredentials).toHaveBeenCalled());

    return { ...result, onChanged, emit: ws.emit };
}

function titleBox(): HTMLElement {
    return screen.getByPlaceholderText("Stream title");
}

function goLive(): HTMLElement {
    return screen.getByRole("button", { name: "Go live" });
}

async function openSetup(user: ReturnType<typeof userEvent.setup>) {
    await user.click(screen.getByRole("button", { name: /OBS streaming setup/ }));
}

describe("GoLivePanel", () => {
    beforeEach(() => {
        streams.getMyStream.mockResolvedValue(null);
        streams.getStreamCredentials.mockResolvedValue(makeCreds());
        streams.resetStreamCredentials.mockResolvedValue(makeCreds({ streamKey: "sk_new_key" }));
        streams.startStream.mockResolvedValue(makeOwner());
        streams.stopStream.mockResolvedValue(undefined);
        streams.updateStreamTitle.mockResolvedValue(makeStream({ title: "A new title" }));
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    it("asks for a title before it will let anybody go live", async () => {
        // given
        const user = userEvent.setup();
        await renderPanel();
        expect(goLive()).toBeDisabled();

        // when
        await user.type(titleBox(), "Tea with the witch");

        // then
        expect(goLive()).toBeEnabled();
    });

    it("refuses a title that is nothing but whitespace", async () => {
        // given
        const user = userEvent.setup();
        await renderPanel();

        // when
        await user.type(titleBox(), "   ");

        // then
        expect(goLive()).toBeDisabled();
    });

    it("starts the stream on low latency with no bitrate when smooth playback is off", async () => {
        // given
        const user = userEvent.setup();
        const { onChanged } = await renderPanel();

        // when
        await user.type(titleBox(), "  Tea with the witch  ");
        await user.click(goLive());

        // then
        await waitFor(() => expect(streams.startStream).toHaveBeenCalledWith("Tea with the witch", "webrtc", 0));
        expect(onChanged).toHaveBeenCalledOnce();
        expect(localStorage.getItem("stream.bitrateKbps")).toBeNull();
    });

    it("waits for OBS to connect once the stream has been created", async () => {
        // given
        const user = userEvent.setup();
        await renderPanel();

        // when
        await user.type(titleBox(), "Tea with the witch");
        await user.click(goLive());

        // then
        expect(await screen.findByRole("heading", { name: "Going live..." })).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Cancel" })).toBeInTheDocument();
    });

    it("says why the stream would not start", async () => {
        // given
        const user = userEvent.setup();
        streams.startStream.mockRejectedValue(new Error("You are already streaming"));
        await renderPanel();

        // when
        await user.type(titleBox(), "Tea with the witch");
        await user.click(goLive());

        // then
        expect(await screen.findByText("You are already streaming")).toBeInTheDocument();
    });

    it("apologises in general terms when the failure carries no message", async () => {
        // given
        const user = userEvent.setup();
        streams.startStream.mockRejectedValue("something odd");
        await renderPanel();

        // when
        await user.type(titleBox(), "Tea with the witch");
        await user.click(goLive());

        // then
        expect(await screen.findByText("Could not start the stream.")).toBeInTheDocument();
    });

    it("keeps the playback choice and the bitrate hidden when smooth playback is unavailable", async () => {
        // given
        streams.getStreamCredentials.mockResolvedValue(makeCreds({ hlsEnabled: false }));

        // when
        await renderPanel();

        // then
        expect(screen.queryByRole("button", { name: "Smooth" })).not.toBeInTheDocument();
        expect(screen.queryByPlaceholderText("e.g. 6000")).not.toBeInTheDocument();
    });

    it("keeps go live out of reach until the bitrate is sensible", async () => {
        // given
        const user = userEvent.setup();
        streams.getStreamCredentials.mockResolvedValue(makeCreds({ hlsEnabled: true }));
        await renderPanel();
        await user.type(titleBox(), "Tea with the witch");

        // when
        await user.type(screen.getByPlaceholderText("e.g. 6000"), "100");

        // then
        expect(goLive()).toBeDisabled();
    });

    it("refuses a bitrate above the ceiling", async () => {
        // given
        const user = userEvent.setup();
        streams.getStreamCredentials.mockResolvedValue(makeCreds({ hlsEnabled: true }));
        await renderPanel();
        await user.type(titleBox(), "Tea with the witch");

        // when
        await user.type(screen.getByPlaceholderText("e.g. 6000"), "60000");

        // then
        expect(goLive()).toBeDisabled();
    });

    it("sends the chosen playback mode and bitrate and remembers the bitrate", async () => {
        // given
        const user = userEvent.setup();
        streams.getStreamCredentials.mockResolvedValue(makeCreds({ hlsEnabled: true }));
        await renderPanel();

        // when
        await user.type(titleBox(), "Tea with the witch");
        await user.type(screen.getByPlaceholderText("e.g. 6000"), "6000");
        await user.click(screen.getByRole("button", { name: "Smooth" }));
        await user.click(goLive());

        // then
        await waitFor(() => expect(streams.startStream).toHaveBeenCalledWith("Tea with the witch", "hls", 6000));
        expect(localStorage.getItem("stream.bitrateKbps")).toBe("6000");
    });

    it("starts out with the bitrate used last time", async () => {
        // given
        localStorage.setItem("stream.bitrateKbps", "8500");
        streams.getStreamCredentials.mockResolvedValue(makeCreds({ hlsEnabled: true }));

        // when
        await renderPanel();

        // then
        expect(screen.getByPlaceholderText("e.g. 6000")).toHaveValue(8500);
    });

    it("announces the stream once it is live", async () => {
        // given
        streams.getMyStream.mockResolvedValue(makeOwner({ status: "live", title: "Tea with the witch" }));

        // when
        await renderPanel();

        // then
        expect(await screen.findByRole("heading", { name: "You're live" })).toBeInTheDocument();
        expect(screen.getByText("Tea with the witch")).toBeInTheDocument();
        expect(screen.getByRole("button", { name: "Stop streaming" })).toBeInTheDocument();
    });

    it("stops the stream and offers the go live form again", async () => {
        // given
        const user = userEvent.setup();
        streams.getMyStream.mockResolvedValue(makeOwner({ status: "live" }));
        const { onChanged } = await renderPanel();
        await screen.findByRole("button", { name: "Stop streaming" });

        // when
        await user.click(screen.getByRole("button", { name: "Stop streaming" }));

        // then
        await waitFor(() => expect(streams.stopStream).toHaveBeenCalledWith(STREAM_ID));
        expect(await screen.findByRole("heading", { name: "Go live" })).toBeInTheDocument();
        expect(onChanged).toHaveBeenCalledOnce();
    });

    it("says why the stream would not stop", async () => {
        // given
        const user = userEvent.setup();
        streams.getMyStream.mockResolvedValue(makeOwner({ status: "live" }));
        streams.stopStream.mockRejectedValue(new Error("The ingest is wedged"));
        await renderPanel();
        await screen.findByRole("button", { name: "Stop streaming" });

        // when
        await user.click(screen.getByRole("button", { name: "Stop streaming" }));

        // then
        expect(await screen.findByText("The ingest is wedged")).toBeInTheDocument();
    });

    it("opens the title for editing with the current title already in it", async () => {
        // given
        const user = userEvent.setup();
        streams.getMyStream.mockResolvedValue(makeOwner({ status: "live", title: "Tea with the witch" }));
        await renderPanel();
        await screen.findByRole("button", { name: "Edit title" });

        // when
        await user.click(screen.getByRole("button", { name: "Edit title" }));

        // then
        expect(titleBox()).toHaveValue("Tea with the witch");
        expect(screen.getByRole("button", { name: "Save title" })).toBeDisabled();
    });

    it("saves a changed title and shows it back", async () => {
        // given
        const user = userEvent.setup();
        streams.getMyStream.mockResolvedValue(makeOwner({ status: "live", title: "Tea with the witch" }));
        await renderPanel();
        await screen.findByRole("button", { name: "Edit title" });
        await user.click(screen.getByRole("button", { name: "Edit title" }));

        // when
        await user.clear(titleBox());
        await user.type(titleBox(), "A new title");
        await user.click(screen.getByRole("button", { name: "Save title" }));

        // then
        await waitFor(() => expect(streams.updateStreamTitle).toHaveBeenCalledWith(STREAM_ID, "A new title"));
        expect(await screen.findByText("A new title")).toBeInTheDocument();
    });

    it("abandons the edit when it is cancelled", async () => {
        // given
        const user = userEvent.setup();
        streams.getMyStream.mockResolvedValue(makeOwner({ status: "live", title: "Tea with the witch" }));
        await renderPanel();
        await screen.findByRole("button", { name: "Edit title" });
        await user.click(screen.getByRole("button", { name: "Edit title" }));

        // when
        await user.click(screen.getByRole("button", { name: "Cancel" }));

        // then
        expect(streams.updateStreamTitle).not.toHaveBeenCalled();
        expect(screen.getByRole("button", { name: "Edit title" })).toBeInTheDocument();
    });

    it("says why the title would not save", async () => {
        // given
        const user = userEvent.setup();
        streams.getMyStream.mockResolvedValue(makeOwner({ status: "live", title: "Tea with the witch" }));
        streams.updateStreamTitle.mockRejectedValue(new Error("That title is not allowed"));
        await renderPanel();
        await screen.findByRole("button", { name: "Edit title" });
        await user.click(screen.getByRole("button", { name: "Edit title" }));

        // when
        await user.clear(titleBox());
        await user.type(titleBox(), "A new title");
        await user.click(screen.getByRole("button", { name: "Save title" }));

        // then
        expect(await screen.findByText("That title is not allowed")).toBeInTheDocument();
    });

    it("flips to live when the server says the stream went live", async () => {
        // given
        streams.getMyStream.mockResolvedValue(makeOwner({ status: "pending" }));
        const { emit } = await renderPanel();
        await screen.findByRole("heading", { name: "Going live..." });

        // when
        emit({ type: "stream_live", data: makeStream({ status: "live" }) } as WSMessage);

        // then
        expect(await screen.findByRole("heading", { name: "You're live" })).toBeInTheDocument();
    });

    it("takes on a title that was changed somewhere else", async () => {
        // given
        streams.getMyStream.mockResolvedValue(makeOwner({ status: "live", title: "Tea with the witch" }));
        const { emit } = await renderPanel();
        await screen.findByRole("heading", { name: "You're live" });

        // when
        emit({ type: "stream_title", data: { streamId: STREAM_ID, title: "Cake with the witch" } } as WSMessage);

        // then
        expect(await screen.findByText("Cake with the witch")).toBeInTheDocument();
    });

    it("returns to the go live form when the server says the stream went offline", async () => {
        // given
        streams.getMyStream.mockResolvedValue(makeOwner({ status: "live" }));
        const { emit, onChanged } = await renderPanel();
        await screen.findByRole("heading", { name: "You're live" });

        // when
        emit({ type: "stream_offline", data: { streamId: STREAM_ID } } as WSMessage);

        // then
        expect(await screen.findByRole("heading", { name: "Go live" })).toBeInTheDocument();
        expect(onChanged).toHaveBeenCalledOnce();
    });

    it("pays no attention to news about somebody else's stream", async () => {
        // given
        streams.getMyStream.mockResolvedValue(makeOwner({ status: "live", title: "Tea with the witch" }));
        const { emit } = await renderPanel();
        await screen.findByRole("heading", { name: "You're live" });

        // when
        emit({ type: "stream_title", data: { streamId: "stream-other", title: "Not mine" } } as WSMessage);
        emit({ type: "stream_offline", data: { streamId: "stream-other" } } as WSMessage);

        // then
        expect(screen.getByRole("heading", { name: "You're live" })).toBeInTheDocument();
        expect(screen.getByText("Tea with the witch")).toBeInTheDocument();
    });

    it("keeps the OBS setup folded away until it is asked for", async () => {
        // given
        const toggleName = /OBS streaming setup/;

        // when
        await renderPanel();

        // then
        expect(screen.getByRole("button", { name: toggleName })).toHaveAttribute("aria-expanded", "false");
        expect(screen.queryByText(WHIP_URL)).not.toBeInTheDocument();
    });

    it("shows the whip server and the stream key once the setup is opened", async () => {
        // given
        const user = userEvent.setup();
        await renderPanel();

        // when
        await openSetup(user);

        // then
        expect(screen.getByText(WHIP_URL)).toBeInTheDocument();
        expect(screen.getByText(STREAM_KEY)).toBeInTheDocument();
    });

    it("mentions the bitrate calculator only when smooth playback is available", async () => {
        // given
        const user = userEvent.setup();
        streams.getStreamCredentials.mockResolvedValue(makeCreds({ hlsEnabled: true }));
        await renderPanel();

        // when
        await openSetup(user);

        // then
        expect(screen.getByText("Server, key, encoder settings, and a bitrate calculator")).toBeInTheDocument();
    });

    it("copies the whip server to the clipboard", async () => {
        // given
        const user = userEvent.setup();
        const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
        await renderPanel();
        await openSetup(user);

        // when
        await user.click(screen.getAllByRole("button", { name: "Copy" })[0]);

        // then
        expect(writeText).toHaveBeenCalledWith(WHIP_URL);
        expect(await screen.findByRole("button", { name: "Copied" })).toBeInTheDocument();
    });

    it("copies the stream key to the clipboard", async () => {
        // given
        const user = userEvent.setup();
        const writeText = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue(undefined);
        await renderPanel();
        await openSetup(user);

        // when
        await user.click(screen.getAllByRole("button", { name: "Copy" })[1]);

        // then
        expect(writeText).toHaveBeenCalledWith(STREAM_KEY);
    });

    it("will not reset the key while a stream of your own is up", async () => {
        // given
        const user = userEvent.setup();
        streams.getMyStream.mockResolvedValue(makeOwner({ status: "live" }));
        await renderPanel();

        // when
        await openSetup(user);

        // then
        expect(screen.getByRole("button", { name: "Reset stream key" })).toBeDisabled();
    });

    it("leaves the key alone when the reset is not confirmed", async () => {
        // given
        const user = userEvent.setup();
        vi.spyOn(window, "confirm").mockReturnValue(false);
        await renderPanel();
        await openSetup(user);

        // when
        await user.click(screen.getByRole("button", { name: "Reset stream key" }));

        // then
        expect(streams.resetStreamCredentials).not.toHaveBeenCalled();
        expect(screen.getByText(STREAM_KEY)).toBeInTheDocument();
    });

    it("shows the new key once the reset is confirmed", async () => {
        // given
        const user = userEvent.setup();
        vi.spyOn(window, "confirm").mockReturnValue(true);
        await renderPanel();
        await openSetup(user);

        // when
        await user.click(screen.getByRole("button", { name: "Reset stream key" }));

        // then
        expect(await screen.findByText("sk_new_key")).toBeInTheDocument();
    });

    it("says why the key would not reset", async () => {
        // given
        const user = userEvent.setup();
        vi.spyOn(window, "confirm").mockReturnValue(true);
        streams.resetStreamCredentials.mockRejectedValue(new Error("Try again later"));
        await renderPanel();
        await openSetup(user);

        // when
        await user.click(screen.getByRole("button", { name: "Reset stream key" }));

        // then
        expect(await screen.findByText("Try again later")).toBeInTheDocument();
    });

    it("owns up when the stream key could not be loaded at all", async () => {
        // given
        streams.getStreamCredentials.mockRejectedValue(new Error("gone"));

        // when
        renderWithProviders(<GoLivePanel />);

        // then
        expect(
            await screen.findByText("Could not load your stream key. Reload the page to try again."),
        ).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: /OBS streaming setup/ })).not.toBeInTheDocument();
    });

    it("suggests a bitrate for 1080p at sixty frames a second", async () => {
        // given
        const user = userEvent.setup();
        streams.getStreamCredentials.mockResolvedValue(makeCreds({ hlsEnabled: true }));
        await renderPanel();

        // when
        await openSetup(user);

        // then
        expect(screen.getByText((12000).toLocaleString())).toBeInTheDocument();
        expect(
            screen.getByText(`${(8500).toLocaleString()} to ${(15000).toLocaleString()} range, set as CBR`),
        ).toBeInTheDocument();
    });

    it("recalculates the bitrate for a smaller, slower picture", async () => {
        // given
        const user = userEvent.setup();
        streams.getStreamCredentials.mockResolvedValue(makeCreds({ hlsEnabled: true }));
        await renderPanel();
        await openSetup(user);

        // when
        await user.click(screen.getByRole("button", { name: "720p" }));
        await user.click(screen.getByRole("button", { name: "30 fps" }));

        // then
        expect(screen.getByText((2500).toLocaleString())).toBeInTheDocument();
    });

    it("fills the bitrate in from the calculator", async () => {
        // given
        const user = userEvent.setup();
        streams.getStreamCredentials.mockResolvedValue(makeCreds({ hlsEnabled: true }));
        await renderPanel();
        await openSetup(user);

        // when
        await user.click(screen.getByRole("button", { name: `Use ${(12000).toLocaleString()} Kbps` }));

        // then
        expect(screen.getByPlaceholderText("e.g. 6000")).toHaveValue(12000);
    });
});
