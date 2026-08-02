import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { ScreenShareView } from "./ScreenShareView";

const mocks = vi.hoisted(() => {
    class FakeRemoteAudioTrack {
        setVolume = vi.fn();
    }

    return { useTracks: vi.fn(), FakeRemoteAudioTrack };
});

vi.mock("@livekit/components-react", () => ({
    useTracks: mocks.useTracks,
    VideoTrack: ({ trackRef }: { trackRef: { sid: string } }) => <div data-testid="video-track">{trackRef.sid}</div>,
}));

vi.mock("livekit-client", () => ({
    RemoteAudioTrack: mocks.FakeRemoteAudioTrack,
    Track: { Source: { ScreenShare: "screen_share", ScreenShareAudio: "screen_share_audio" } },
}));

interface ViewOptions {
    screen?: { sid: string; isLocal?: boolean } | null;
    audioTrack?: unknown;
    onReload?: (() => void) | undefined;
}

function stubTracks(options: ViewOptions) {
    const videoTracks = options.screen
        ? [{ sid: options.screen.sid, participant: { isLocal: options.screen.isLocal ?? false } }]
        : [];
    const audioTracks = options.audioTrack ? [{ publication: { track: options.audioTrack } }] : [];

    mocks.useTracks.mockImplementation((sources: string[]) => {
        return sources[0] === "screen_share" ? videoTracks : audioTracks;
    });
}

function renderView(options: ViewOptions = {}) {
    stubTracks(options);

    return renderWithProviders(<ScreenShareView placeholder="Waiting for the host." onReload={options.onReload} />);
}

beforeEach(() => {
    mocks.useTracks.mockReturnValue([]);
});

describe("ScreenShareView", () => {
    it("shows the placeholder while nobody is sharing a screen", () => {
        // given
        const options = { screen: null };

        // when
        renderView(options);

        // then
        expect(screen.getByText("Waiting for the host.")).toBeInTheDocument();
        expect(screen.queryByTestId("video-track")).not.toBeInTheDocument();
    });

    it("renders the shared screen once a track arrives", () => {
        // given
        const options = { screen: { sid: "track-1" } };

        // when
        renderView(options);

        // then
        expect(screen.getByTestId("video-track")).toHaveTextContent("track-1");
        expect(screen.queryByText("Waiting for the host.")).not.toBeInTheDocument();
    });

    it("offers a reload for a viewer whose remote stream may have stalled", async () => {
        // given
        const onReload = vi.fn();
        const user = userEvent.setup();
        renderView({ screen: { sid: "track-1", isLocal: false }, onReload });

        // when
        await user.click(screen.getByRole("button", { name: /Reload stream/ }));

        // then
        expect(onReload).toHaveBeenCalledOnce();
    });

    it("never offers the host a reload of their own stream", () => {
        // given
        const onReload = vi.fn();

        // when
        renderView({ screen: { sid: "track-1", isLocal: true }, onReload });

        // then
        expect(screen.queryByRole("button", { name: /Reload stream/ })).not.toBeInTheDocument();
    });

    it("hides the reload when no reload was wired up", () => {
        // given
        const options = { screen: { sid: "track-1", isLocal: false } };

        // when
        renderView(options);

        // then
        expect(screen.queryByRole("button", { name: /Reload stream/ })).not.toBeInTheDocument();
    });

    it("keeps the volume control out of sight while the share carries no sound", () => {
        // given
        const options = { screen: { sid: "track-1" }, audioTrack: null };

        // when
        renderView(options);

        // then
        expect(screen.queryByLabelText("Screen share volume")).not.toBeInTheDocument();
    });

    it("offers a volume control at full volume once the share carries sound", () => {
        // given
        const audioTrack = new mocks.FakeRemoteAudioTrack();

        // when
        renderView({ screen: { sid: "track-1" }, audioTrack });

        // then
        expect(screen.getByLabelText("Screen share volume")).toHaveValue("1");
        expect(audioTrack.setVolume).toHaveBeenCalledWith(1);
    });

    it("applies the chosen volume to the remote audio track", () => {
        // given
        const audioTrack = new mocks.FakeRemoteAudioTrack();
        renderView({ screen: { sid: "track-1" }, audioTrack });
        const slider = screen.getByLabelText("Screen share volume");

        // when
        fireEvent.change(slider, { target: { value: "0.25" } });

        // then
        expect(audioTrack.setVolume).toHaveBeenLastCalledWith(0.25);
        expect(slider).toHaveValue("0.25");
    });

    it("mutes the shared sound entirely when the slider is pulled to nothing", () => {
        // given
        const audioTrack = new mocks.FakeRemoteAudioTrack();
        renderView({ screen: { sid: "track-1" }, audioTrack });

        // when
        fireEvent.change(screen.getByLabelText("Screen share volume"), { target: { value: "0" } });

        // then
        expect(audioTrack.setVolume).toHaveBeenLastCalledWith(0);
    });

    it("leaves a track that is not a remote one alone", () => {
        // given
        const audioTrack = { setVolume: vi.fn() };

        // when
        renderView({ screen: { sid: "track-1" }, audioTrack });

        // then
        expect(screen.getByLabelText("Screen share volume")).toBeInTheDocument();
        expect(audioTrack.setVolume).not.toHaveBeenCalled();
    });
});
