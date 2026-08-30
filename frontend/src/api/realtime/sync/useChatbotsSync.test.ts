import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { expectInvalidated } from "../../../test-utils/query";
import { createTestQueryClient, providerWrapper } from "../../../test-utils/render";
import { queryKeys } from "../../queryKeys";
import { dispatch } from "../bus";
import type { RealtimeEvent } from "../events";
import { useChatbotsSync } from "./useChatbotsSync";

function emit(event: RealtimeEvent): void {
    act(() => {
        dispatch(event);
    });
}

describe("useChatbotsSync", () => {
    it("invalidates every chatbot query when the roster changes", () => {
        // given
        const queryClient = createTestQueryClient();
        vi.spyOn(queryClient, "invalidateQueries");
        renderHook(() => useChatbotsSync(), { wrapper: providerWrapper({ queryClient }) });

        // when
        emit({ type: "chatbots_changed", data: {} });

        // then
        expectInvalidated(queryClient, queryKeys.chatbots.all);
    });

    it("ignores an event it did not ask for", () => {
        // given
        const queryClient = createTestQueryClient();
        vi.spyOn(queryClient, "invalidateQueries");
        renderHook(() => useChatbotsSync(), { wrapper: providerWrapper({ queryClient }) });

        // when
        emit({ type: "permissions_changed", data: {} });

        // then
        expect(queryClient.invalidateQueries).not.toHaveBeenCalled();
    });
});
