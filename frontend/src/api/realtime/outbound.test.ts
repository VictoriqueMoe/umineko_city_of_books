import { beforeEach, describe, expect, it, vi } from "vitest";
import { REALTIME_COMMANDS, sendRealtime, type RealtimeCommand, type RealtimeCommandName } from "./outbound";

const { sendRaw } = vi.hoisted(() => ({ sendRaw: vi.fn() }));

vi.mock("./socket", () => ({ sendRaw }));

function lastFrame(): string {
    const call = sendRaw.mock.calls[sendRaw.mock.calls.length - 1];
    if (!call) {
        throw new Error("no frame was sent");
    }

    return call[0] as string;
}

const commands: { name: RealtimeCommandName; command: RealtimeCommand; wire: string }[] = [
    {
        name: "join_room",
        command: { type: "join_room", data: { room_id: "room-1" } },
        wire: '{"type":"join_room","data":{"room_id":"room-1"}}',
    },
    {
        name: "leave_room",
        command: { type: "leave_room", data: { room_id: "room-1" } },
        wire: '{"type":"leave_room","data":{"room_id":"room-1"}}',
    },
    {
        name: "typing",
        command: { type: "typing", data: { room_id: "room-1" } },
        wire: '{"type":"typing","data":{"room_id":"room-1"}}',
    },
    {
        name: "viewer_state",
        command: { type: "viewer_state", data: { room_id: "room-1", state: "active" } },
        wire: '{"type":"viewer_state","data":{"room_id":"room-1","state":"active"}}',
    },
    {
        name: "secret_join",
        command: { type: "secret_join", data: { secret_id: "secret-1" } },
        wire: '{"type":"secret_join","data":{"secret_id":"secret-1"}}',
    },
    {
        name: "secret_leave",
        command: { type: "secret_leave", data: { secret_id: "secret-1" } },
        wire: '{"type":"secret_leave","data":{"secret_id":"secret-1"}}',
    },
    {
        name: "game_room_join",
        command: { type: "game_room_join", data: { room_id: "room-1" } },
        wire: '{"type":"game_room_join","data":{"room_id":"room-1"}}',
    },
    {
        name: "game_room_leave",
        command: { type: "game_room_leave", data: { room_id: "room-1" } },
        wire: '{"type":"game_room_leave","data":{"room_id":"room-1"}}',
    },
    {
        name: "game_room_input",
        command: { type: "game_room_input", data: { room_id: "room-1", payload: { y: 620, seq: 1 } } },
        wire: '{"type":"game_room_input","data":{"room_id":"room-1","payload":{"y":620,"seq":1}}}',
    },
];

beforeEach(() => {
    sendRaw.mockClear();
});

describe("realtime outbound wire format", () => {
    for (const entry of commands) {
        it(`serialises ${entry.name} with the field names the server parses`, () => {
            // given
            // when
            sendRealtime(entry.command);

            // then
            expect(sendRaw).toHaveBeenCalledExactlyOnceWith(entry.wire);
        });
    }

    it("sends one frame per call and nothing on its own", () => {
        // given
        expect(sendRaw).not.toHaveBeenCalled();

        // when
        sendRealtime({ type: "typing", data: { room_id: "room-1" } });
        sendRealtime({ type: "typing", data: { room_id: "room-2" } });

        // then
        expect(sendRaw).toHaveBeenCalledTimes(2);
    });

    it("hands the transport a string, never an object", () => {
        // given
        // when
        sendRealtime({ type: "join_room", data: { room_id: "room-1" } });

        // then
        expect(typeof lastFrame()).toBe("string");
        expect(JSON.parse(lastFrame())).toEqual({ type: "join_room", data: { room_id: "room-1" } });
    });

    it("reports an idle viewer as well as an active one", () => {
        // given
        // when
        sendRealtime({ type: "viewer_state", data: { room_id: "room-1", state: "idle" } });

        // then
        expect(lastFrame()).toBe('{"type":"viewer_state","data":{"room_id":"room-1","state":"idle"}}');
    });

    it("passes a game input payload through untouched", () => {
        // given
        const payload = { y: 300, seq: 7 };

        // when
        sendRealtime({ type: "game_room_input", data: { room_id: "room-1", payload } });

        // then
        expect(JSON.parse(lastFrame())).toEqual({
            type: "game_room_input",
            data: { room_id: "room-1", payload: { y: 300, seq: 7 } },
        });
    });
});

describe("realtime command name table", () => {
    it("names every command the union declares and no others", () => {
        // given
        const declared = commands.map(entry => entry.name);

        // when
        const table = Object.values(REALTIME_COMMANDS);

        // then
        expect([...table].sort()).toEqual([...declared].sort());
    });

    it("has nine commands", () => {
        // given
        // when
        const table = Object.values(REALTIME_COMMANDS);

        // then
        expect(table).toHaveLength(9);
        expect(new Set(table).size).toBe(9);
    });
});
