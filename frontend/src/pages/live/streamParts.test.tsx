import { act, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../test-utils/render";
import { StreamStage, StreamUptime, StreamViewers, ViewerCountReporter } from "./streamParts";

const mocks = vi.hoisted(() => ({
    useParticipants: vi.fn(),
    useTracks: vi.fn(),
}));

vi.mock("livekit-client", () => ({
    Track: {
        Source: { Camera: "camera", ScreenShare: "screen_share", Unknown: "unknown" },
        Kind: { Video: "video", Audio: "audio" },
    },
}));

vi.mock("@livekit/components-react", () => ({
    useParticipants: mocks.useParticipants,
    useTracks: mocks.useTracks,
    VideoTrack: ({ trackRef }: { trackRef: { sid: string } }) => <div data-testid="video-track">{trackRef.sid}</div>,
}));

interface FakeParticipant {
    identity: string;
    name?: string;
    metadata?: string;
}

function stubParticipants(participants: FakeParticipant[]): void {
    mocks.useParticipants.mockReturnValue(participants);
}

function stubTracks(tracks: unknown[]): void {
    mocks.useTracks.mockReturnValue(tracks);
}

beforeEach(() => {
    stubParticipants([]);
    stubTracks([]);
});

describe("ViewerCountReporter", () => {
    it("reports only the participants who are watching", () => {
        // given
        const onChange = vi.fn();
        stubParticipants([{ identity: "viewer_1" }, { identity: "viewer_2" }, { identity: "streamer_1" }]);

        // when
        renderWithProviders(<ViewerCountReporter onChange={onChange} />);

        // then
        expect(onChange).toHaveBeenCalledWith(2);
    });

    it("reports nobody watching when the room is empty", () => {
        // given
        const onChange = vi.fn();
        stubParticipants([]);

        // when
        renderWithProviders(<ViewerCountReporter onChange={onChange} />);

        // then
        expect(onChange).toHaveBeenCalledWith(0);
    });

    it("draws nothing of its own", () => {
        // given
        stubParticipants([{ identity: "viewer_1" }]);

        // when
        const { container } = renderWithProviders(<ViewerCountReporter onChange={vi.fn()} />);

        // then
        expect(container).toBeEmptyDOMElement();
    });
});

describe("StreamUptime", () => {
    it("stays hidden until the stream has a start time", () => {
        // given
        const startedAt = undefined;

        // when
        const { container } = renderWithProviders(<StreamUptime startedAt={startedAt} />);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("stays hidden when the start time cannot be read", () => {
        // given
        const startedAt = "not a date at all";

        // when
        const { container } = renderWithProviders(<StreamUptime startedAt={startedAt} />);

        // then
        expect(container).toBeEmptyDOMElement();
    });

    it("counts minutes and seconds for a young stream", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-02-01T12:05:07Z"));

        // when
        renderWithProviders(<StreamUptime startedAt="2026-02-01T12:00:00Z" />);

        // then
        expect(screen.getByTitle("Live for")).toHaveTextContent("05:07");
    });

    it("brings the hours in once the stream has run past one", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-02-01T14:03:09Z"));

        // when
        renderWithProviders(<StreamUptime startedAt="2026-02-01T12:00:00Z" />);

        // then
        expect(screen.getByTitle("Live for")).toHaveTextContent("2:03:09");
    });

    it("ticks the elapsed time forward every second", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-02-01T12:00:10Z"));
        renderWithProviders(<StreamUptime startedAt="2026-02-01T12:00:00Z" />);

        // when
        act(() => {
            vi.advanceTimersByTime(5000);
        });

        // then
        expect(screen.getByTitle("Live for")).toHaveTextContent("00:15");
    });

    it("never counts backwards when the start time is in the future", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-02-01T12:00:00Z"));

        // when
        renderWithProviders(<StreamUptime startedAt="2026-02-01T13:00:00Z" />);

        // then
        expect(screen.getByTitle("Live for")).toHaveTextContent("00:00");
    });
});

describe("StreamStage", () => {
    it("waits politely while no video has been published", () => {
        // given
        stubTracks([]);
        stubParticipants([{ identity: "viewer_1" }]);

        // when
        renderWithProviders(<StreamStage />);

        // then
        expect(screen.getByText("Waiting for the stream to start...")).toBeInTheDocument();
        expect(screen.queryByTestId("video-track")).not.toBeInTheDocument();
    });

    it("shows the video once a video track is published", () => {
        // given
        stubTracks([{ sid: "track-9", publication: { kind: "video" } }]);

        // when
        renderWithProviders(<StreamStage />);

        // then
        expect(screen.getByTestId("video-track")).toHaveTextContent("track-9");
        expect(screen.queryByText("Waiting for the stream to start...")).not.toBeInTheDocument();
    });

    it("ignores an audio only publication", () => {
        // given
        stubTracks([{ sid: "track-audio", publication: { kind: "audio" } }]);

        // when
        renderWithProviders(<StreamStage />);

        // then
        expect(screen.getByText("Waiting for the stream to start...")).toBeInTheDocument();
    });

    it("counts the watchers over the stage", () => {
        // given
        stubParticipants([{ identity: "viewer_a" }, { identity: "viewer_b" }, { identity: "publisher" }]);

        // when
        renderWithProviders(<StreamStage />);

        // then
        expect(screen.getByText(/2/)).toBeInTheDocument();
    });

    it("asks the room for the sources a stream can arrive on", () => {
        // given
        stubTracks([]);

        // when
        renderWithProviders(<StreamStage />);

        // then
        expect(mocks.useTracks).toHaveBeenCalledWith(["camera", "screen_share", "unknown"]);
    });
});

describe("StreamViewers", () => {
    it("names the signed in watchers", () => {
        // given
        stubParticipants([
            {
                identity: "viewer_1",
                name: "Beatrice",
                metadata: JSON.stringify({ userId: "u1", username: "beatrice", avatarUrl: "" }),
            },
        ]);

        // when
        renderWithProviders(<StreamViewers />);

        // then
        expect(screen.getByText("Beatrice")).toBeInTheDocument();
        expect(screen.getByText(/1 watching/)).toBeInTheDocument();
    });

    it("falls back to the metadata username when the participant has no name", () => {
        // given
        stubParticipants([
            { identity: "viewer_1", metadata: JSON.stringify({ userId: "u1", username: "battler" }) },
            { identity: "viewer_2", metadata: JSON.stringify({ userId: "u2" }) },
        ]);

        // when
        renderWithProviders(<StreamViewers />);

        // then
        expect(screen.getByText("battler")).toBeInTheDocument();
        expect(screen.getByText("Member")).toBeInTheDocument();
    });

    it("counts a watcher with no metadata as a guest", () => {
        // given
        stubParticipants([{ identity: "viewer_1" }, { identity: "viewer_2" }]);

        // when
        renderWithProviders(<StreamViewers />);

        // then
        expect(screen.getByText(/2 guests/)).toBeInTheDocument();
    });

    it("keeps the guest wording singular for a lone guest", () => {
        // given
        stubParticipants([{ identity: "viewer_1" }]);

        // when
        renderWithProviders(<StreamViewers />);

        // then
        expect(screen.getByText(/1 guest$/)).toBeInTheDocument();
    });

    it("treats a watcher with unreadable metadata as a guest", () => {
        // given
        stubParticipants([{ identity: "viewer_1", metadata: "{not json" }]);

        // when
        renderWithProviders(<StreamViewers />);

        // then
        expect(screen.getByText(/1 guest/)).toBeInTheDocument();
    });

    it("shows a watcher's avatar when their metadata carries one", () => {
        // given
        stubParticipants([
            {
                identity: "viewer_1",
                name: "Ange",
                metadata: JSON.stringify({ userId: "u1", avatarUrl: "/media/ange.png" }),
            },
        ]);

        // when
        const { container } = renderWithProviders(<StreamViewers />);

        // then
        expect(container.querySelector("img")).toHaveAttribute("src", "/media/ange.png");
    });

    it("lists a watcher who joined twice only once", () => {
        // given
        const metadata = JSON.stringify({ userId: "u1", username: "beatrice" });
        stubParticipants([
            { identity: "viewer_1", name: "Beatrice", metadata },
            { identity: "viewer_2", name: "Beatrice", metadata },
        ]);

        // when
        renderWithProviders(<StreamViewers />);

        // then
        expect(screen.getAllByText("Beatrice")).toHaveLength(1);
        expect(screen.getByText(/2 watching/)).toBeInTheDocument();
    });

    it("ignores participants who are not watching", () => {
        // given
        stubParticipants([{ identity: "publisher_1", name: "Streamer" }]);

        // when
        renderWithProviders(<StreamViewers />);

        // then
        expect(screen.getByText(/0 watching/)).toBeInTheDocument();
        expect(screen.queryByText(/guest/)).not.toBeInTheDocument();
    });
});
