import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeSpectatorMessage } from "../test-utils/fixtures";
import { emitRealtimeEvent, makeWSHarness, type WSHarness } from "../test-utils/ws";
import type { SpectatorMessage } from "../types/api";
import type { GameChatHistory } from "./queries/gameRoom";
import { useGameChatMessages, type GameChatEventName } from "./useGameChatMessages";

const holder = vi.hoisted(() => ({ ws: null as unknown as WSHarness }));

vi.mock("../api/realtime/pipeline", () => ({
    ensureRealtimePipeline: () => {},
    getRealtimeEpoch: () => holder.ws.getEpoch(),
    subscribeRealtimeEpoch: (listener: () => void) => holder.ws.subscribeEpoch(listener),
}));

function makeMessage(id: string, body: string): SpectatorMessage {
    return makeSpectatorMessage({ id, body });
}

function makeHistory(messages: SpectatorMessage[], refresh: () => Promise<unknown> = () => Promise.resolve()) {
    const history: GameChatHistory = { messages, loading: false, error: "", refresh };
    return history;
}

function mount(history: GameChatHistory, roomId = "room-1", channel: GameChatEventName = "spectator_chat_message") {
    return renderHook(props => useGameChatMessages(props.roomId, channel, history), {
        initialProps: { roomId },
    });
}

function arrive(
    roomId: string,
    message: SpectatorMessage,
    channel: GameChatEventName = "spectator_chat_message",
): void {
    emitRealtimeEvent({ type: channel, data: { room_id: roomId, message } });
}

describe("useGameChatMessages", () => {
    beforeEach(() => {
        holder.ws = makeWSHarness();
    });

    it("hands back the loaded history while nothing has arrived", () => {
        // given
        const history = makeHistory([makeMessage("m1", "the golden truth")]);

        // when
        const { result } = mount(history);

        // then
        expect(result.current).toBe(history.messages);
    });

    it("adds a message that arrives on the channel it was given", () => {
        // given
        const history = makeHistory([makeMessage("m1", "the golden truth")]);
        const { result } = mount(history);

        // when
        arrive("room-1", makeMessage("m2", "beato is watching"));

        // then
        expect(result.current.map(m => m.body)).toEqual(["the golden truth", "beato is watching"]);
    });

    it.each<GameChatEventName>(["player_chat_message", "spectator_chat_message"])(
        "keeps %s in the order it was sent when your own messages and everyone else's interleave",
        channel => {
            // given
            const history = makeHistory([
                makeSpectatorMessage({ id: "m1", body: "Greetings", created_at: "2026-09-17T14:01:10.000Z" }),
                makeSpectatorMessage({ id: "m3", body: "Wow", created_at: "2026-09-17T14:02:05.000Z" }),
            ]);
            const { result } = mount(history, "room-1", channel);

            // when
            arrive(
                "room-1",
                makeSpectatorMessage({ id: "m2", body: "Yo", created_at: "2026-09-17T14:01:40.000Z" }),
                channel,
            );
            arrive(
                "room-1",
                makeSpectatorMessage({ id: "m4", body: "Beginner's luck", created_at: "2026-09-17T14:02:30.000Z" }),
                channel,
            );

            // then
            expect(result.current.map(m => m.body)).toEqual(["Greetings", "Yo", "Wow", "Beginner's luck"]);
        },
    );

    it("ignores a message that arrives for another room", () => {
        // given
        const history = makeHistory([]);
        const { result } = mount(history);

        // when
        arrive("room-2", makeMessage("m2", "somewhere else"));

        // then
        expect(result.current).toEqual([]);
    });

    it("ignores a message that arrives on the other chat's channel", () => {
        // given
        const history = makeHistory([]);
        const { result } = mount(history);

        // when
        emitRealtimeEvent({
            type: "player_chat_message",
            data: { room_id: "room-1", message: makeMessage("m2", "wrong channel") },
        });

        // then
        expect(result.current).toEqual([]);
    });

    it("never shows an arrival the history already carries", () => {
        // given
        const history = makeHistory([makeMessage("m1", "only once")]);
        const { result } = mount(history);

        // when
        arrive("room-1", makeMessage("m1", "only once"));

        // then
        expect(result.current).toHaveLength(1);
    });

    it("never shows the same arrival twice", () => {
        // given
        const history = makeHistory([]);
        const { result } = mount(history);

        // when
        arrive("room-1", makeMessage("m2", "beato is watching"));
        arrive("room-1", makeMessage("m2", "beato is watching"));

        // then
        expect(result.current).toHaveLength(1);
    });

    it("leaves the arrivals of the room it was showing behind when it moves to another", () => {
        // given
        const history = makeHistory([]);
        const view = mount(history, "room-1");
        arrive("room-1", makeMessage("m2", "beato is watching"));
        expect(view.result.current).toHaveLength(1);

        // when
        view.rerender({ roomId: "room-2" });

        // then
        expect(view.result.current).toEqual([]);
    });

    it("loads the history again once the socket has reconnected", () => {
        // given
        const refresh = vi.fn<() => Promise<unknown>>().mockResolvedValue(undefined);
        mount(makeHistory([], refresh));
        expect(refresh).not.toHaveBeenCalled();

        // when
        holder.ws.reconnect();

        // then
        expect(refresh).toHaveBeenCalledTimes(1);
    });
});
