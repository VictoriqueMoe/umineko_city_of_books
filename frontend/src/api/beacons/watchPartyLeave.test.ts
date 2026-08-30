import { beforeEach, describe, expect, it, vi } from "vitest";
import { setAuthToken } from "../authToken";
import { sendWatchPartyLeaveBeacon } from "./watchPartyLeave";

const capacitor = vi.hoisted(() => ({ native: false, platform: "web" }));

vi.mock("@capacitor/core", () => ({
    Capacitor: {
        isNativePlatform: () => capacitor.native,
        getPlatform: () => capacitor.platform,
    },
}));

vi.mock("@capacitor/preferences", () => ({
    Preferences: {
        get: () => Promise.resolve({ value: null }),
        set: () => Promise.resolve(),
        remove: () => Promise.resolve(),
    },
}));

function stubFetch(): ReturnType<typeof vi.fn<typeof fetch>> {
    const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);
    return fetchMock;
}

beforeEach(() => {
    capacitor.native = false;
    capacitor.platform = "web";
});

describe("sendWatchPartyLeaveBeacon", () => {
    it("deletes the viewer's own participant row on the watch party", () => {
        // given
        const fetchMock = stubFetch();

        // when
        sendWatchPartyLeaveBeacon("room-1", "session-9");

        // then
        expect(fetchMock.mock.calls[0][0]).toBe("/api/v1/chat/rooms/room-1/watch-parties/session-9/participants/me");
        expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: "DELETE" });
    });

    it("keeps the request alive, because it fires while the page is being torn down", () => {
        // given
        const fetchMock = stubFetch();

        // when
        sendWatchPartyLeaveBeacon("room-1", "session-9");

        // then
        expect(fetchMock.mock.calls[0][1]).toMatchObject({ keepalive: true });
    });

    it("sends the session cookie, which is how the web build is authenticated", () => {
        // given
        const fetchMock = stubFetch();

        // when
        sendWatchPartyLeaveBeacon("room-1", "session-9");

        // then
        expect(fetchMock.mock.calls[0][1]).toMatchObject({ credentials: "include" });
        expect(fetchMock.mock.calls[0][1]?.headers).toEqual({});
    });

    it("sends the bearer token instead when the native app is the one leaving", () => {
        // given
        capacitor.native = true;
        capacitor.platform = "android";
        setAuthToken("token-123");
        const fetchMock = stubFetch();

        // when
        sendWatchPartyLeaveBeacon("room-1", "session-9");

        // then
        expect(fetchMock.mock.calls[0][1]?.headers).toEqual({
            "X-Client-Platform": "android",
            Authorization: "Bearer token-123",
        });
    });

    it("catches a rejected beacon, because an unloading page has nowhere to report it", async () => {
        // given
        const fetchMock = vi.fn<typeof fetch>().mockRejectedValue(new Error("network gone"));
        vi.stubGlobal("fetch", fetchMock);

        // when
        sendWatchPartyLeaveBeacon("room-1", "session-9");
        await Promise.resolve();

        // then
        expect(fetchMock).toHaveBeenCalledOnce();
    });

    it("survives a transport that throws before it ever returns a promise", () => {
        // given
        vi.stubGlobal(
            "fetch",
            vi.fn(() => {
                throw new Error("fetch is gone");
            }),
        );

        // then
        expect(() => sendWatchPartyLeaveBeacon("room-1", "session-9")).not.toThrow();
    });
});
