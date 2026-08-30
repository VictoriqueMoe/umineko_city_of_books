import { afterEach, describe, expect, it, vi } from "vitest";
import { isRealtimeEventName, isSimulationFrame, parseServerEvent } from "./parse";
import type * as ParseModule from "./parse";

const API_ORIGIN = "https://whentheycry.social";

async function loadWithOrigin(origin: string): Promise<typeof ParseModule> {
    vi.stubEnv("VITE_API_BASE", origin);
    vi.resetModules();
    return import("./parse");
}

describe("parseServerEvent rejects malformed packets", () => {
    it("rejects json that does not parse", () => {
        // given
        const raw = '{"type":"pong",';

        // when
        const result = parseServerEvent(raw);

        // then
        expect(result).toEqual({ ok: false, reason: "invalid-json" });
    });

    it("rejects an empty string", () => {
        // when
        const result = parseServerEvent("");

        // then
        expect(result).toEqual({ ok: false, reason: "invalid-json" });
    });

    it("rejects a json scalar rather than an object", () => {
        // then
        expect(parseServerEvent("42")).toEqual({ ok: false, reason: "not-an-object" });
        expect(parseServerEvent('"pong"')).toEqual({ ok: false, reason: "not-an-object" });
        expect(parseServerEvent("true")).toEqual({ ok: false, reason: "not-an-object" });
        expect(parseServerEvent("null")).toEqual({ ok: false, reason: "not-an-object" });
    });

    it("rejects a json array", () => {
        // when
        const result = parseServerEvent('[{"type":"pong","data":{}}]');

        // then
        expect(result).toEqual({ ok: false, reason: "not-an-object" });
    });

    it("rejects an object with no type", () => {
        // when
        const result = parseServerEvent('{"data":{"count":3}}');

        // then
        expect(result).toEqual({ ok: false, reason: "missing-type" });
    });

    it("rejects an object whose type is not a string", () => {
        // then
        expect(parseServerEvent('{"type":7,"data":{}}')).toEqual({ ok: false, reason: "missing-type" });
        expect(parseServerEvent('{"type":null,"data":{}}')).toEqual({ ok: false, reason: "missing-type" });
    });

    it("rejects an object whose type is not a known event", () => {
        // when
        const result = parseServerEvent('{"type":"totally_made_up","data":{}}');

        // then
        expect(result).toEqual({ ok: false, reason: "unknown-type" });
    });

    it("never throws, whatever arrives on the wire", () => {
        // given
        const hostile = [
            "",
            " ",
            "{",
            "}",
            "[",
            "undefined",
            "NaN",
            "'single quoted'",
            '{"type":',
            '{"type":"pong","data":',
            '{"type":{"nested":true}}',
            "[]{}",
            '{"type":"pong"}{"type":"pong"}',
        ];

        // then
        for (const raw of hostile) {
            expect(() => parseServerEvent(raw)).not.toThrow();
            expect(parseServerEvent(raw).ok).toBe(false);
        }
    });
});

describe("parseServerEvent accepts well formed packets", () => {
    it("returns the event under its declared name", () => {
        // given
        const raw = JSON.stringify({ type: "live_games_count", data: { count: 3 } });

        // when
        const result = parseServerEvent(raw);

        // then
        expect(result).toEqual({ ok: true, event: { type: "live_games_count", data: { count: 3 } } });
    });

    it("delivers a null payload rather than rejecting it, because parse judges the envelope only", () => {
        // given
        const raw = JSON.stringify({ type: "ban_changed", data: null });

        // when
        const result = parseServerEvent(raw);

        // then
        expect(result).toEqual({ ok: true, event: { type: "ban_changed", data: null } });
    });

    it("does not apply the event guards, which are the dispatcher's opt-in job", () => {
        // given
        const raw = JSON.stringify({ type: "role_changed", data: { user_id: 42 } });

        // when
        const result = parseServerEvent(raw);

        // then
        expect(result).toEqual({ ok: true, event: { type: "role_changed", data: { user_id: 42 } } });
    });
});

describe("parseServerEvent normalises media urls on non-frame events", () => {
    afterEach(() => {
        vi.unstubAllEnvs();
        vi.resetModules();
    });

    it("deep walks a non-frame payload so nested media urls are absolutised", async () => {
        // given
        const parse = await loadWithOrigin(API_ORIGIN);
        const raw = JSON.stringify({
            type: "chat_message",
            data: { room_id: "r1", sender: { avatar_url: "/uploads/a.png" }, media: [{ media_url: "/uploads/m.png" }] },
        });

        // when
        const result = parse.parseServerEvent(raw);

        // then
        expect(result).toEqual({
            ok: true,
            event: {
                type: "chat_message",
                data: {
                    room_id: "r1",
                    sender: { avatar_url: `${API_ORIGIN}/uploads/a.png` },
                    media: [{ media_url: `${API_ORIGIN}/uploads/m.png` }],
                },
            },
        });
    });

    it("leaves the payload untouched when no api origin is configured", () => {
        // given
        const raw = JSON.stringify({ type: "chat_message", data: { sender: { avatar_url: "/uploads/a.png" } } });

        // when
        const result = parseServerEvent(raw);

        // then
        expect(result).toEqual({
            ok: true,
            event: { type: "chat_message", data: { sender: { avatar_url: "/uploads/a.png" } } },
        });
    });
});

describe("the *_frame fast path", () => {
    afterEach(() => {
        vi.unstubAllEnvs();
        vi.resetModules();
    });

    it("skips absolutizeMedia entirely, so a media url inside a frame is delivered untouched", async () => {
        // given
        const parse = await loadWithOrigin(API_ORIGIN);
        const raw = JSON.stringify({
            type: "game_pong_frame",
            data: { ball: { x: 1, y: 2 }, avatar_url: "/uploads/a.png" },
        });

        // when
        const result = parse.parseServerEvent(raw);

        // then
        expect(result).toEqual({
            ok: true,
            event: { type: "game_pong_frame", data: { ball: { x: 1, y: 2 }, avatar_url: "/uploads/a.png" } },
        });
    });

    it("normalises the identical payload when the event is not a frame, which is the whole difference", async () => {
        // given
        const parse = await loadWithOrigin(API_ORIGIN);
        const raw = JSON.stringify({
            type: "chat_audio",
            data: { ball: { x: 1, y: 2 }, avatar_url: "/uploads/a.png" },
        });

        // when
        const result = parse.parseServerEvent(raw);

        // then
        expect(result).toEqual({
            ok: true,
            event: { type: "chat_audio", data: { ball: { x: 1, y: 2 }, avatar_url: `${API_ORIGIN}/uploads/a.png` } },
        });
    });

    it("rejects an unrecognised *_frame type like any other unknown event, so the fast path is not a bypass", () => {
        // when
        const result = parseServerEvent('{"type":"game_invented_frame","data":{}}');

        // then
        expect(result).toEqual({ ok: false, reason: "unknown-type" });
    });

    it("treats only a trailing _frame as a simulation frame", () => {
        // then
        expect(isSimulationFrame("game_pong_frame")).toBe(true);
        expect(isSimulationFrame("_frame")).toBe(true);
        expect(isSimulationFrame("frame")).toBe(false);
        expect(isSimulationFrame("_frame_started")).toBe(false);
        expect(isSimulationFrame("chat_message")).toBe(false);
    });
});

describe("isRealtimeEventName", () => {
    it("accepts a declared event name and rejects anything else", () => {
        // then
        expect(isRealtimeEventName("game_pong_frame")).toBe(true);
        expect(isRealtimeEventName("chat_message")).toBe(true);
        expect(isRealtimeEventName("mystery_solved")).toBe(true);
        expect(isRealtimeEventName("not_an_event")).toBe(false);
        expect(isRealtimeEventName("toString")).toBe(false);
        expect(isRealtimeEventName("")).toBe(false);
    });
});
