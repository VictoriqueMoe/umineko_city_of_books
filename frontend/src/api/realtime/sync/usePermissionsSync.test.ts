import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { expectInvalidated } from "../../../test-utils/query";
import { createTestQueryClient, providerWrapper } from "../../../test-utils/render";
import { queryKeys } from "../../queryKeys";
import { dispatch } from "../bus";
import type { RealtimeEvent } from "../events";
import { usePermissionsSync } from "./usePermissionsSync";

function emit(event: RealtimeEvent): void {
    act(() => {
        dispatch(event);
    });
}

describe("usePermissionsSync", () => {
    it("invalidates the signed in viewer when their permissions change", () => {
        // given
        const queryClient = createTestQueryClient();
        vi.spyOn(queryClient, "invalidateQueries");
        renderHook(() => usePermissionsSync(), { wrapper: providerWrapper({ queryClient }) });

        // when
        emit({ type: "permissions_changed", data: {} });

        // then
        expectInvalidated(queryClient, queryKeys.auth.me());
    });

    it("invalidates the signed in viewer when the vanity roles change", () => {
        // given
        const queryClient = createTestQueryClient();
        vi.spyOn(queryClient, "invalidateQueries");
        renderHook(() => usePermissionsSync(), { wrapper: providerWrapper({ queryClient }) });

        // when
        emit({ type: "vanity_roles_changed", data: {} });

        // then
        expectInvalidated(queryClient, queryKeys.auth.me());
    });

    it("ignores an event it did not ask for", () => {
        // given
        const queryClient = createTestQueryClient();
        vi.spyOn(queryClient, "invalidateQueries");
        renderHook(() => usePermissionsSync(), { wrapper: providerWrapper({ queryClient }) });

        // when
        emit({ type: "chatbots_changed", data: {} });

        // then
        expect(queryClient.invalidateQueries).not.toHaveBeenCalled();
    });
});
