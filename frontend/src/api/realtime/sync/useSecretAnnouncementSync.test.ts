import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { FakeWebSocket } from "../../../test-utils/ws";
import type * as SocketModule from "../socket";
import type * as SyncModule from "./useSecretAnnouncementSync";

const { getAuthToken, isNativeApp } = vi.hoisted(() => ({
    getAuthToken: vi.fn(),
    isNativeApp: vi.fn(),
}));

vi.mock("../../authToken", () => ({ getAuthToken }));
vi.mock("../../../platform/capabilities", () => ({ isNativeApp, clientPlatform: () => "web" }));

const beatrice = { id: "user-1", username: "beato", display_name: "Beatrice" };

let sync: typeof SyncModule;
let socket: typeof SocketModule;

function lastSocket(): FakeWebSocket {
    const instance = FakeWebSocket.instances[FakeWebSocket.instances.length - 1];
    if (!instance) {
        throw new Error("no websocket was opened");
    }

    return instance;
}

function connect(): void {
    act(() => {
        socket.openRealtimeSocket("user-1");
        lastSocket().onopen?.();
    });
}

function receive(message: { type: string; data?: unknown }): void {
    act(() => {
        lastSocket().onmessage?.({ data: JSON.stringify(message) });
    });
}

beforeEach(async () => {
    FakeWebSocket.instances = [];
    vi.stubGlobal("WebSocket", FakeWebSocket);
    getAuthToken.mockReturnValue(null);
    isNativeApp.mockReturnValue(false);
    vi.resetModules();
    socket = await import("../socket");
    sync = await import("./useSecretAnnouncementSync");
});

describe("useSecretAnnouncementSync", () => {
    it("holds nothing until a secret closes", () => {
        // given
        const { result } = renderHook(() => sync.useSecretAnnouncementSync());

        // then
        expect(result.current.announcement).toBeNull();
    });

    it("announces a closed secret to the rest of the app", () => {
        // given
        const { result } = renderHook(() => sync.useSecretAnnouncementSync());
        connect();

        // when
        receive({
            type: "secret_closed",
            data: { secret_id: "secret-1", secret_title: "The Golden Truth", solver: beatrice },
        });

        // then
        expect(result.current.announcement).toEqual({
            secret_id: "secret-1",
            secret_title: "The Golden Truth",
            solver: beatrice,
        });
    });

    it("ignores a secret closed event that names no solver", () => {
        // given
        const { result } = renderHook(() => sync.useSecretAnnouncementSync());
        connect();

        // when
        receive({ type: "secret_closed", data: { secret_id: "secret-1" } });

        // then
        expect(result.current.announcement).toBeNull();
    });

    it("ignores a secret closed event that carries no payload at all", () => {
        // given
        const { result } = renderHook(() => sync.useSecretAnnouncementSync());
        connect();

        // when
        receive({ type: "secret_closed" });

        // then
        expect(result.current.announcement).toBeNull();
    });

    it("replaces the standing announcement when a second secret closes", () => {
        // given
        const { result } = renderHook(() => sync.useSecretAnnouncementSync());
        connect();
        receive({
            type: "secret_closed",
            data: { secret_id: "secret-1", secret_title: "The Golden Truth", solver: beatrice },
        });

        // when
        receive({
            type: "secret_closed",
            data: { secret_id: "secret-2", secret_title: "The Red Truth", solver: beatrice },
        });

        // then
        expect(result.current.announcement?.secret_id).toBe("secret-2");
    });

    it("forgets the announcement once it is dismissed", () => {
        // given
        const { result } = renderHook(() => sync.useSecretAnnouncementSync());
        connect();
        receive({
            type: "secret_closed",
            data: { secret_id: "secret-1", secret_title: "The Golden Truth", solver: beatrice },
        });

        // when
        act(() => {
            result.current.dismiss();
        });

        // then
        expect(result.current.announcement).toBeNull();
    });

    it("stops listening once the consumer unmounts", () => {
        // given
        const view = renderHook(() => sync.useSecretAnnouncementSync());
        connect();

        // when
        view.unmount();
        receive({
            type: "secret_closed",
            data: { secret_id: "secret-1", secret_title: "The Golden Truth", solver: beatrice },
        });

        // then
        expect(view.result.current.announcement).toBeNull();
    });
});
