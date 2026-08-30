import { describe, expect, it, vi } from "vitest";
import { queryKeys } from "../api/queryKeys";
import { expectInvalidated } from "./query";
import { createTestQueryClient } from "./render";

function spiedClient() {
    const queryClient = createTestQueryClient();
    const invalidate = vi.spyOn(queryClient, "invalidateQueries");

    return { queryClient, invalidate };
}

describe("expectInvalidated", () => {
    it("passes when the key was invalidated", () => {
        // given
        const { queryClient } = spiedClient();

        // when
        queryClient.invalidateQueries({ queryKey: queryKeys.chat.unreadCount() });

        // then
        expectInvalidated(queryClient, queryKeys.chat.unreadCount());
    });

    it("passes when the key was one invalidation among several", () => {
        // given
        const { queryClient } = spiedClient();

        // when
        queryClient.invalidateQueries({ queryKey: queryKeys.chat.rooms() });
        queryClient.invalidateQueries({ queryKey: queryKeys.chat.unreadCount() });
        queryClient.invalidateQueries({ queryKey: queryKeys.notifications.list() });

        // then
        expectInvalidated(queryClient, queryKeys.chat.unreadCount());
    });

    it("fails when a different key was invalidated", () => {
        // given
        const { queryClient } = spiedClient();

        // when
        queryClient.invalidateQueries({ queryKey: queryKeys.chat.rooms() });

        // then
        expect(() => expectInvalidated(queryClient, queryKeys.chat.unreadCount())).toThrow();
    });

    it("fails when nothing was invalidated", () => {
        // given
        const { queryClient } = spiedClient();

        // when
        // then
        expect(() => expectInvalidated(queryClient, queryKeys.chat.unreadCount())).toThrow();
    });

    it("matches on the whole key, not a prefix of it", () => {
        // given
        const { queryClient } = spiedClient();

        // when
        queryClient.invalidateQueries({ queryKey: queryKeys.chat.room("room-1") });

        // then
        expect(() => expectInvalidated(queryClient, queryKeys.chat.roomMembers("room-1"))).toThrow();
    });

    it("ignores an invalidation that carried extra options", () => {
        // given
        const { queryClient } = spiedClient();

        // when
        queryClient.invalidateQueries({ queryKey: queryKeys.chat.unreadCount(), exact: true });

        // then
        expect(() => expectInvalidated(queryClient, queryKeys.chat.unreadCount())).toThrow();
    });

    it("says so when the client was never spied on", () => {
        // given
        const queryClient = createTestQueryClient();

        // when
        queryClient.invalidateQueries({ queryKey: queryKeys.chat.unreadCount() });

        // then
        expect(() => expectInvalidated(queryClient, queryKeys.chat.unreadCount())).toThrow(/vi\.spyOn/);
    });

    it("reads the spy at assertion time, so the act may happen after the helper is imported", () => {
        // given
        const { queryClient, invalidate } = spiedClient();

        // when
        queryClient.invalidateQueries({ queryKey: queryKeys.chat.unreadCount() });

        // then
        expect(invalidate).toHaveBeenCalledWith({ queryKey: queryKeys.chat.unreadCount() });
        expectInvalidated(queryClient, queryKeys.chat.unreadCount());
    });
});
