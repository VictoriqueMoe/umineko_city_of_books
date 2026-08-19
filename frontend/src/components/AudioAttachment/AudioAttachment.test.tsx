import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { AudioAttachment, AudioThumb } from "./AudioAttachment";

function givenDuration(container: HTMLElement, seconds: number): HTMLAudioElement {
    const audio = container.querySelector("audio");
    if (!audio) {
        throw new Error("expected an audio element");
    }

    Object.defineProperty(audio, "duration", { value: seconds, configurable: true });
    fireEvent.loadedMetadata(audio);

    return audio;
}

describe("AudioAttachment", () => {
    it("shows the original filename above the player", () => {
        // given a track whose name carries the only clue which file it is
        render(<AudioAttachment src="/uploads/posts/a.flac" filename="神様の言う通り.flac" />);

        // then
        expect(screen.getByText("神様の言う通り.flac")).toBeInTheDocument();
    });

    it("renders without a filename, because older uploads have none", () => {
        // given
        const { container } = render(<AudioAttachment src="/uploads/posts/a.flac" />);

        // then the player is still there
        expect(screen.getByLabelText("Play")).toBeInTheDocument();
        expect(container.querySelector("audio")).toBeInTheDocument();
    });

    it("keeps seeking disabled until the duration is known", () => {
        // given
        const { container } = render(<AudioAttachment src="/uploads/posts/a.flac" />);

        // then dragging a scrubber with no known length would jump to nowhere
        expect(screen.getByLabelText("Seek")).toBeDisabled();

        // when the metadata arrives
        givenDuration(container, 195);

        // then
        expect(screen.getByLabelText("Seek")).toBeEnabled();
    });

    it("shows elapsed and total time in minutes and seconds", () => {
        // given
        const { container } = render(<AudioAttachment src="/uploads/posts/a.flac" />);
        const audio = givenDuration(container, 195);

        // then
        expect(screen.getByText("0:00 / 3:15")).toBeInTheDocument();

        // when playback moves on
        Object.defineProperty(audio, "currentTime", { value: 61, configurable: true });
        fireEvent.timeUpdate(audio);

        // then seconds stay zero padded
        expect(screen.getByText("1:01 / 3:15")).toBeInTheDocument();
    });

    it("swaps the control label between play and pause", () => {
        // given
        const { container } = render(<AudioAttachment src="/uploads/posts/a.flac" />);
        const audio = container.querySelector("audio")!;

        // when the element reports it started
        fireEvent.play(audio);

        // then
        expect(screen.getByLabelText("Pause")).toBeInTheDocument();

        // when it stops
        fireEvent.pause(audio);

        // then
        expect(screen.getByLabelText("Play")).toBeInTheDocument();
    });

    it("keeps clicks off an enclosing card, which would navigate away mid playback", () => {
        // given a post card that opens the post when its body is clicked
        const onCardClick = vi.fn();
        render(
            <div onClick={onCardClick}>
                <AudioAttachment src="/uploads/posts/a.flac" filename="theme.flac" />
            </div>,
        );

        // when the listener presses play, or drags the scrubber
        fireEvent.click(screen.getByLabelText("Play"));
        fireEvent.click(screen.getByLabelText("Seek"));

        // then
        expect(onCardClick).not.toHaveBeenCalled();
    });

    it("says so rather than pretending to work when the file will not load", () => {
        // given
        const { container } = render(<AudioAttachment src="/uploads/posts/missing.flac" />);
        const audio = container.querySelector("audio")!;

        // when
        fireEvent.error(audio);

        // then
        expect(screen.getByText("unavailable")).toBeInTheDocument();
        expect(screen.getByLabelText("Play")).toBeDisabled();
    });
});

describe("AudioThumb", () => {
    it("is labelled for screen readers", () => {
        // given
        render(<AudioThumb />);

        // then
        expect(screen.getByLabelText("Audio file")).toBeInTheDocument();
    });
});
