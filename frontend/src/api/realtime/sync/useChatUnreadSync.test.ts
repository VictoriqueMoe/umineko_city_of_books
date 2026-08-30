import { act, renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../../test-utils/render";
import { queryKeys } from "../../queryKeys";
import { dispatch } from "../bus";
import type { RealtimeEvent } from "../events";
import { useChatUnreadSync } from "./useChatUnreadSync";

function emit(event: RealtimeEvent): void {
    act(() => {
        dispatch(event);
    });
}

describe("useChatUnreadSync", () => {
    it("writes the bumped total under the chat unread count key", () => {
        // given
        const queryClient = createTestQueryClient();
        renderHook(() => useChatUnreadSync(), { wrapper: providerWrapper({ queryClient }) });

        // when
        emit({ type: "chat_unread_bumped", data: { room_id: "room-1", total: 7 } });

        // then
        expect(queryClient.getQueryData(queryKeys.chat.unreadCount())).toEqual({ count: 7 });
    });

    it("writes the total from a read receipt for the viewer's own tab", () => {
        // given
        const queryClient = createTestQueryClient();
        queryClient.setQueryData(queryKeys.chat.unreadCount(), { count: 7 });
        renderHook(() => useChatUnreadSync(), { wrapper: providerWrapper({ queryClient }) });

        // when
        emit({ type: "chat_read", data: { room_id: "room-1", total: 0 } });

        // then
        expect(queryClient.getQueryData(queryKeys.chat.unreadCount())).toEqual({ count: 0 });
    });

    it("ignores a chat unread message with no usable total", () => {
        // given
        const queryClient = createTestQueryClient();
        queryClient.setQueryData(queryKeys.chat.unreadCount(), { count: 3 });
        renderHook(() => useChatUnreadSync(), { wrapper: providerWrapper({ queryClient }) });

        // when
        emit({ type: "chat_read", data: { room_id: "room-1" } as never });

        // then
        expect(queryClient.getQueryData(queryKeys.chat.unreadCount())).toEqual({ count: 3 });
    });
});
