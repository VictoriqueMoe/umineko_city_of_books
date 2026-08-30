import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
    playMessageSound,
    playNotificationSound,
    playRemoteAudio,
    playVoiceJoinSound,
    playVoiceLeaveSound,
} from "./sound";

function spyOnPlay() {
    return vi.spyOn(HTMLMediaElement.prototype, "play").mockResolvedValue(undefined);
}

let playSpy: ReturnType<typeof spyOnPlay>;

function playedAudio(index: number): HTMLAudioElement {
    return playSpy.mock.contexts[index] as HTMLAudioElement;
}

function playedSources(): string {
    return playSpy.mock.contexts.map(context => (context as HTMLAudioElement).src).join(" ");
}

beforeEach(() => {
    playSpy = spyOnPlay();
});

afterEach(() => {
    vi.restoreAllMocks();
});

describe("playMessageSound", () => {
    it("plays the message chime at the quiet default volume", () => {
        // when
        playMessageSound();

        // then
        expect(playSpy).toHaveBeenCalledOnce();
        expect(playedAudio(0).src).toContain("/sounds/message.wav");
        expect(playedAudio(0).volume).toBe(0.15);
    });

    it("reuses the same audio element for repeated messages", () => {
        // when
        playMessageSound();
        playMessageSound();

        // then
        expect(playSpy).toHaveBeenCalledTimes(2);
        expect(playedAudio(0)).toBe(playedAudio(1));
    });

    it("swallows a playback rejection from a browser that blocks autoplay", async () => {
        // given
        playSpy.mockRejectedValue(new Error("play() was interrupted"));

        // when
        playMessageSound();
        await new Promise<void>(resolve => setTimeout(resolve, 0));

        // then
        expect(playSpy).toHaveBeenCalledOnce();
    });
});

describe("playNotificationSound", () => {
    it("plays the notification chime when nothing else has sounded recently", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2030-01-01T00:00:00.000Z"));

        // when
        playNotificationSound();

        // then
        expect(playSpy).toHaveBeenCalledOnce();
        expect(playedAudio(0).src).toContain("/sounds/notification.wav");
        expect(playedAudio(0).volume).toBe(0.15);
    });

    it("stays quiet while a message chime is still ringing out", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00.000Z"));
        playMessageSound();

        // when
        vi.setSystemTime(new Date("2026-08-02T12:00:01.499Z"));
        playNotificationSound();

        // then
        expect(playSpy).toHaveBeenCalledOnce();
        expect(playedSources()).not.toContain("/sounds/notification.wav");
    });

    it("plays again the moment the suppression window has elapsed", () => {
        // given
        vi.useFakeTimers();
        vi.setSystemTime(new Date("2026-08-02T12:00:00.000Z"));
        playMessageSound();

        // when
        vi.setSystemTime(new Date("2026-08-02T12:00:01.500Z"));
        playNotificationSound();

        // then
        expect(playSpy).toHaveBeenCalledTimes(2);
        expect(playedAudio(1).src).toContain("/sounds/notification.wav");
    });
});

describe("playVoiceJoinSound", () => {
    it("plays the join sound louder than the default chimes", () => {
        // when
        playVoiceJoinSound();

        // then
        expect(playedAudio(0).src).toContain("/sounds/voice-join.wav");
        expect(playedAudio(0).volume).toBe(0.3);
    });
});

describe("playVoiceLeaveSound", () => {
    it("plays the leave sound louder than the default chimes", () => {
        // when
        playVoiceLeaveSound();

        // then
        expect(playedAudio(0).src).toContain("/sounds/voice-leave.wav");
        expect(playedAudio(0).volume).toBe(0.3);
    });
});

describe("playRemoteAudio", () => {
    it("plays an arbitrary url at the default volume", () => {
        // when
        playRemoteAudio("https://cdn.example.test/alert.mp3");

        // then
        expect(playedAudio(0).src).toBe("https://cdn.example.test/alert.mp3");
        expect(playedAudio(0).volume).toBe(0.15);
    });

    it("honours a louder volume when one is asked for", () => {
        // when
        playRemoteAudio("https://cdn.example.test/loud.mp3", 0.75);

        // then
        expect(playedAudio(0).volume).toBe(0.75);
    });

    it("caches one audio element per url", () => {
        // when
        playRemoteAudio("https://cdn.example.test/cached.mp3");
        playRemoteAudio("https://cdn.example.test/cached.mp3");
        playRemoteAudio("https://cdn.example.test/other.mp3");

        // then
        expect(playedAudio(0)).toBe(playedAudio(1));
        expect(playedAudio(2)).not.toBe(playedAudio(0));
    });

    it("lets go of an old url once a flood of new ones has arrived", () => {
        // given
        playRemoteAudio("https://cdn.example.test/oldest.mp3");
        const oldest = playedAudio(0);

        // when
        for (let i = 0; i < 20; i++) {
            playRemoteAudio(`https://cdn.example.test/flood-${i}.mp3`);
        }
        playRemoteAudio("https://cdn.example.test/oldest.mp3");

        // then
        expect(playedAudio(playSpy.mock.contexts.length - 1)).not.toBe(oldest);
    });

    it("never lets go of the built in chimes", () => {
        // given
        playMessageSound();
        const chime = playedAudio(0);

        // when
        for (let i = 0; i < 20; i++) {
            playRemoteAudio(`https://cdn.example.test/deluge-${i}.mp3`);
        }
        playMessageSound();

        // then
        expect(playedAudio(playSpy.mock.contexts.length - 1)).toBe(chime);
    });
});

describe("audio unlocking", () => {
    it("primes every cached sound silently on the first user gesture and ignores later ones", async () => {
        // given
        const before = playSpy.mock.calls.length;

        // when
        document.dispatchEvent(new MouseEvent("click"));

        // then
        expect(playSpy.mock.calls.length).toBeGreaterThan(before);
        const primed = playedSources();
        expect(primed).toContain("/sounds/message.wav");
        expect(primed).toContain("/sounds/notification.wav");
        expect(primed).toContain("/sounds/voice-join.wav");
        expect(primed).toContain("/sounds/voice-leave.wav");
        expect(playedAudio(0).muted).toBe(true);

        await vi.waitFor(() => {
            expect(playedAudio(0).muted).toBe(false);
        });

        const afterUnlock = playSpy.mock.calls.length;
        document.dispatchEvent(new KeyboardEvent("keydown"));
        expect(playSpy).toHaveBeenCalledTimes(afterUnlock);
    });

    it("leaves a chime asked for during the unlock gesture audible", async () => {
        // given
        vi.resetModules();
        const pauseSpy = vi.spyOn(HTMLMediaElement.prototype, "pause");
        const releases: (() => void)[] = [];
        playSpy.mockImplementation(() => new Promise<void>(resolve => releases.push(resolve)));
        const sound = await import("./sound");

        // when
        document.dispatchEvent(new MouseEvent("click"));
        sound.playMessageSound();
        for (const release of releases) {
            release();
        }
        await new Promise<void>(resolve => setTimeout(resolve, 0));

        // then
        const chime = playedAudio(playSpy.mock.contexts.length - 1);
        expect(chime.src).toContain("/sounds/message.wav");
        expect(chime.muted).toBe(false);
        expect(chime.volume).toBe(0.15);
        expect(pauseSpy.mock.contexts).not.toContain(chime);
    });
});
