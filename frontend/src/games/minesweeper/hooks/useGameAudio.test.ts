import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CharacterId } from "../types";
import { useGameAudio } from "./useGameAudio";

const VOICE = "https://quotes.auaurora.moe/api/v1/umineko/audio/voice";
const DLANOR_START = `${VOICE}/combined?segments=47:54600001,47:54600002`;
const DLANOR_WIN = `${VOICE}/combined?segments=47:54600315,47:54600316,47:54600317`;
const DLANOR_LOSE_TO_ERIKA = `${VOICE}/47/64600077`;
const BERN_LOSE = `${VOICE}/28/82100692`;

const created: FakeAudio[] = [];
let playResult: () => Promise<void> = () => Promise.resolve();

class FakeAudio {
    src: string;
    volume = 1;
    playCount = 0;
    pauseCount = 0;
    private readonly listeners = new Map<string, (() => void)[]>();

    constructor(src: string) {
        this.src = src;
        created.push(this);
    }

    addEventListener(type: string, handler: () => void): void {
        const existing = this.listeners.get(type) ?? [];
        existing.push(handler);
        this.listeners.set(type, existing);
    }

    play(): Promise<void> {
        this.playCount += 1;
        return playResult();
    }

    pause(): void {
        this.pauseCount += 1;
    }

    emit(type: string): void {
        const handlers = this.listeners.get(type) ?? [];
        for (const handler of handlers) {
            handler();
        }
    }
}

interface HookProps {
    my: CharacterId | "";
    opponent: CharacterId | "";
}

function setup(props: Partial<HookProps> = {}) {
    const initialProps: HookProps = {
        my: CharacterId.Dlanor,
        opponent: CharacterId.Erika,
        ...props,
    };

    return renderHook(p => useGameAudio(p.my, p.opponent), { initialProps });
}

beforeEach(() => {
    created.length = 0;
    playResult = () => Promise.resolve();
    vi.stubGlobal("Audio", FakeAudio);
});

describe("useGameAudio", () => {
    it("reports nothing playing before anything has been asked for", () => {
        // given
        const props = { my: CharacterId.Dlanor, opponent: CharacterId.Erika };

        // when
        const { result } = setup(props);

        // then
        expect(result.current.playing).toBe(false);
        expect(created).toHaveLength(0);
    });

    it("plays the opening line at a slightly lowered volume", async () => {
        // given
        const { result } = setup();

        // when
        await act(async () => {
            result.current.play("start");
        });

        // then
        expect(created).toHaveLength(1);
        expect(created[0].src).toBe(DLANOR_START);
        expect(created[0].volume).toBe(0.75);
        expect(created[0].playCount).toBe(1);
        expect(result.current.playing).toBe(true);
    });

    it("stays silent for a character with no opening line", async () => {
        // given
        const { result } = setup({ my: CharacterId.Bernkastel });

        // when
        await act(async () => {
            result.current.play("start");
        });

        // then
        expect(created).toHaveLength(0);
        expect(result.current.playing).toBe(false);
    });

    it("stays silent when no character has been chosen", async () => {
        // given
        const { result } = setup({ my: "", opponent: "" });

        // when
        await act(async () => {
            result.current.play("start");
        });

        // then
        expect(created).toHaveLength(0);
        expect(result.current.playing).toBe(false);
    });

    it("plays the matchup specific line when the opponent has one", async () => {
        // given
        const { result } = setup({ my: CharacterId.Dlanor, opponent: CharacterId.Erika });

        // when
        await act(async () => {
            result.current.play("lose");
        });

        // then
        expect(created).toHaveLength(1);
        expect(created[0].src).toBe(DLANOR_LOSE_TO_ERIKA);
    });

    it("falls back to the default line when the matchup has nothing for that outcome", async () => {
        // given
        const { result } = setup({ my: CharacterId.Dlanor, opponent: CharacterId.Erika });

        // when
        await act(async () => {
            result.current.play("win");
        });

        // then
        expect(created[0].src).toBe(DLANOR_WIN);
    });

    it("stays silent on a game over with no opponent character", async () => {
        // given
        const { result } = setup({ my: CharacterId.Dlanor, opponent: "" });

        // when
        await act(async () => {
            result.current.play("lose");
        });

        // then
        expect(created).toHaveLength(0);
        expect(result.current.playing).toBe(false);
    });

    it("reports nothing playing when the browser refuses to start the clip", async () => {
        // given
        playResult = () => Promise.reject(new Error("autoplay is blocked"));
        const { result } = setup();

        // when
        await act(async () => {
            result.current.play("start");
        });

        // then
        expect(created).toHaveLength(1);
        expect(result.current.playing).toBe(false);
    });

    it("reports nothing playing once the clip has finished", async () => {
        // given
        const { result } = setup();
        await act(async () => {
            result.current.play("start");
        });
        expect(result.current.playing).toBe(true);

        // when
        act(() => {
            created[0].emit("ended");
        });

        // then
        expect(result.current.playing).toBe(false);
    });

    it("cuts the current clip short before starting another one", async () => {
        // given
        const { result } = setup();
        await act(async () => {
            result.current.play("start");
        });

        // when
        await act(async () => {
            result.current.play("lose");
        });

        // then
        expect(created).toHaveLength(2);
        expect(created[0].pauseCount).toBe(1);
        expect(created[0].src).toBe("");
        expect(created[1].src).toBe(DLANOR_LOSE_TO_ERIKA);
        expect(result.current.playing).toBe(true);
    });

    it("stops the clip on request", async () => {
        // given
        const { result } = setup();
        await act(async () => {
            result.current.play("start");
        });

        // when
        act(() => {
            result.current.stop();
        });

        // then
        expect(created[0].pauseCount).toBe(1);
        expect(created[0].src).toBe("");
        expect(result.current.playing).toBe(false);
    });

    it("shrugs off a stop when nothing is playing", () => {
        // given
        const { result } = setup();

        // when
        act(() => {
            result.current.stop();
        });

        // then
        expect(created).toHaveLength(0);
        expect(result.current.playing).toBe(false);
    });

    it("uses the characters as they stand at the moment of playing", async () => {
        // given
        const { result, rerender } = setup({ my: "", opponent: "" });

        // when
        rerender({ my: CharacterId.Dlanor, opponent: CharacterId.Erika });
        await act(async () => {
            result.current.play("lose");
        });

        // then
        expect(created).toHaveLength(1);
        expect(created[0].src).toBe(DLANOR_LOSE_TO_ERIKA);
    });

    it("picks the line for whichever character the player has taken", async () => {
        // given
        const { result } = setup({ my: CharacterId.Bernkastel, opponent: CharacterId.Erika });

        // when
        await act(async () => {
            result.current.play("lose");
        });

        // then
        expect(created[0].src).toBe(BERN_LOSE);
    });

    it("silences the clip when the view goes away", async () => {
        // given
        const { result, unmount } = setup();
        await act(async () => {
            result.current.play("start");
        });

        // when
        unmount();

        // then
        expect(created[0].pauseCount).toBe(1);
        expect(created[0].src).toBe("");
    });
});
