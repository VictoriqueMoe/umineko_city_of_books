import { describe, it, expect, vi } from "vitest";

vi.mock("@capacitor/core", () => ({
    Capacitor: {
        isNativePlatform: () => false,
        getPlatform: () => "web",
    },
}));

import { routeFromPushData } from "./push";

describe("routeFromPushData", () => {
    it("routes a new follower to their profile", () => {
        expect(routeFromPushData({ type: "new_follower", actor_username: "alice" })).toBe("/user/alice");
    });

    it("routes a chat room invite to the room", () => {
        expect(routeFromPushData({ type: "chat_room_invite", reference_id: "room-1" })).toBe("/rooms/room-1");
    });

    it("returns null when the payload has no type", () => {
        expect(routeFromPushData({})).toBeNull();
    });
});
