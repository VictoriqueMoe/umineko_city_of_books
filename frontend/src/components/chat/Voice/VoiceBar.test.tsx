import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Room } from "livekit-client";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../test-utils/render";
import { VoiceBar } from "./VoiceBar";

const mocks = vi.hoisted(() => ({
    setMicrophoneEnabled: vi.fn(() => Promise.resolve()),
    useLocalParticipant: vi.fn(),
}));

vi.mock("@livekit/components-react", async () => {
    const { createContext, useContext } = await import("react");
    const RoomContext = createContext<{ name?: string } | undefined>(undefined);

    function RoomAudioRenderer() {
        const room = useContext(RoomContext);

        return <div data-testid="room-audio">{room?.name ?? "no room"}</div>;
    }

    return { RoomContext, RoomAudioRenderer, useLocalParticipant: mocks.useLocalParticipant };
});

vi.mock("./VoiceParticipants", () => ({
    VoiceParticipantList: ({
        canModerate,
        onForceMute,
    }: {
        canModerate?: boolean;
        onForceMute?: (identity: string, muted: boolean) => void;
    }) => (
        <div data-testid="participants" data-can-moderate={String(canModerate)}>
            <button type="button" onClick={() => onForceMute?.("battler", true)}>
                force mute battler
            </button>
        </div>
    ),
}));

const room = { name: "Rokkenjima" } as unknown as Room;

function stubMicrophone(isMicrophoneEnabled: boolean): void {
    mocks.useLocalParticipant.mockReturnValue({
        localParticipant: { setMicrophoneEnabled: mocks.setMicrophoneEnabled },
        isMicrophoneEnabled,
    });
}

beforeEach(() => {
    mocks.setMicrophoneEnabled.mockResolvedValue(undefined);
    stubMicrophone(true);
});

describe("VoiceBar", () => {
    it("hands the connected room down to the livekit audio renderer", () => {
        // given
        stubMicrophone(true);

        // when
        renderWithProviders(<VoiceBar room={room} onLeave={vi.fn()} />);

        // then
        expect(screen.getByTestId("room-audio")).toHaveTextContent("Rokkenjima");
    });

    it("offers to mute while the viewer's microphone is live", () => {
        // given
        stubMicrophone(true);

        // when
        renderWithProviders(<VoiceBar room={room} onLeave={vi.fn()} />);

        // then
        expect(screen.getByRole("button", { name: "Mute" })).toBeInTheDocument();
        expect(screen.queryByRole("button", { name: "Unmute" })).not.toBeInTheDocument();
    });

    it("offers to unmute while the viewer's microphone is off", () => {
        // given
        stubMicrophone(false);

        // when
        renderWithProviders(<VoiceBar room={room} onLeave={vi.fn()} />);

        // then
        expect(screen.getByRole("button", { name: "Unmute" })).toBeInTheDocument();
    });

    it("turns the microphone off when the viewer mutes themselves", async () => {
        // given
        stubMicrophone(true);
        const user = userEvent.setup();
        renderWithProviders(<VoiceBar room={room} onLeave={vi.fn()} />);

        // when
        await user.click(screen.getByRole("button", { name: "Mute" }));

        // then
        expect(mocks.setMicrophoneEnabled).toHaveBeenCalledWith(false);
    });

    it("turns the microphone back on when the viewer unmutes themselves", async () => {
        // given
        stubMicrophone(false);
        const user = userEvent.setup();
        renderWithProviders(<VoiceBar room={room} onLeave={vi.fn()} />);

        // when
        await user.click(screen.getByRole("button", { name: "Unmute" }));

        // then
        expect(mocks.setMicrophoneEnabled).toHaveBeenCalledWith(true);
    });

    it("swallows a microphone toggle the browser refuses", async () => {
        // given
        mocks.setMicrophoneEnabled.mockRejectedValue(new Error("no microphone"));
        const user = userEvent.setup();
        renderWithProviders(<VoiceBar room={room} onLeave={vi.fn()} />);

        // when
        await user.click(screen.getByRole("button", { name: "Mute" }));

        // then
        expect(screen.getByRole("button", { name: "Mute" })).toBeInTheDocument();
    });

    it("leaves the call through the handler it was given", async () => {
        // given
        const onLeave = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<VoiceBar room={room} onLeave={onLeave} />);

        // when
        await user.click(screen.getByRole("button", { name: "Leave" }));

        // then
        expect(onLeave).toHaveBeenCalledTimes(1);
    });

    it("treats the viewer as an ordinary member unless told otherwise", () => {
        // given
        stubMicrophone(true);

        // when
        renderWithProviders(<VoiceBar room={room} onLeave={vi.fn()} />);

        // then
        expect(screen.getByTestId("participants")).toHaveAttribute("data-can-moderate", "false");
    });

    it("tells the participant list when the viewer may moderate the call", () => {
        // given
        const canModerate = true;

        // when
        renderWithProviders(<VoiceBar room={room} onLeave={vi.fn()} canModerate={canModerate} />);

        // then
        expect(screen.getByTestId("participants")).toHaveAttribute("data-can-moderate", "true");
    });

    it("passes a server mute from the participant list back to the parent", async () => {
        // given
        const onForceMute = vi.fn();
        const user = userEvent.setup();
        renderWithProviders(<VoiceBar room={room} onLeave={vi.fn()} canModerate onForceMute={onForceMute} />);

        // when
        await user.click(screen.getByRole("button", { name: "force mute battler" }));

        // then
        expect(onForceMute).toHaveBeenCalledWith("battler", true);
    });
});
