import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { makeUser } from "../../../test-utils/fixtures";
import { createTestQueryClient } from "../../../test-utils/render";
import { FakeWebSocket } from "../../../test-utils/ws";
import type { User, UserProfile } from "../../../types/api";
import { queryKeys } from "../../queryKeys";
import { closeRealtimeSocket, openRealtimeSocket } from "../socket";
import { useUserIdentitySync } from "./useUserIdentitySync";

const { getAuthToken, isNativeApp } = vi.hoisted(() => ({
    getAuthToken: vi.fn(),
    isNativeApp: vi.fn(),
}));

vi.mock("../../authToken", () => ({ getAuthToken }));
vi.mock("../../../platform/capabilities", () => ({ isNativeApp, clientPlatform: () => "web" }));

const SESSION_KEY = "session-1";

let queryClient: QueryClient;

function wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}

function lastSocket(): FakeWebSocket {
    const instance = FakeWebSocket.instances[FakeWebSocket.instances.length - 1];
    if (!instance) {
        throw new Error("no websocket was opened");
    }

    return instance;
}

function mountAndConnect(): void {
    renderHook(() => useUserIdentitySync(), { wrapper });

    act(() => {
        openRealtimeSocket(SESSION_KEY);
        lastSocket().onopen?.();
    });
}

function receive(message: { type: string; data?: unknown }): void {
    act(() => {
        lastSocket().onmessage?.({ data: JSON.stringify(message) });
    });
}

function signIn(user: UserProfile): void {
    queryClient.setQueryData<UserProfile | null>(queryKeys.auth.me(), user);
}

function me(): UserProfile | null | undefined {
    return queryClient.getQueryData<UserProfile | null>(queryKeys.auth.me());
}

function seedStaff(staff: User[]): void {
    queryClient.setQueryData<User[]>(queryKeys.auth.staff(), staff);
}

function staff(): User[] | undefined {
    return queryClient.getQueryData<User[]>(queryKeys.auth.staff());
}

beforeEach(() => {
    FakeWebSocket.instances = [];
    vi.stubGlobal("WebSocket", FakeWebSocket);
    getAuthToken.mockReturnValue(null);
    isNativeApp.mockReturnValue(false);
    queryClient = createTestQueryClient();
});

afterEach(() => {
    closeRealtimeSocket(SESSION_KEY);
});

describe("useUserIdentitySync", () => {
    it("updates the signed in user when their own role changes", () => {
        // given
        signIn(makeUser({ id: "user-1" }));
        mountAndConnect();

        // when
        receive({ type: "role_changed", data: { user_id: "user-1", role: "moderator" } });

        // then
        expect(me()).toEqual(expect.objectContaining({ id: "user-1", role: "moderator" }));
    });

    it("leaves the signed in user alone when somebody else changes role", () => {
        // given
        const signedIn = makeUser({ id: "user-1" });
        signIn(signedIn);
        mountAndConnect();

        // when
        receive({ type: "role_changed", data: { user_id: "user-2", role: "moderator" } });

        // then
        expect(me()).toEqual(signedIn);
    });

    it("ignores a role change that names nobody", () => {
        // given
        const signedIn = makeUser({ id: "user-1" });
        signIn(signedIn);
        mountAndConnect();

        // when
        receive({ type: "role_changed", data: { role: "moderator" } });

        // then
        expect(me()).toEqual(signedIn);
    });

    it("patches every cached copy of the user the event named", () => {
        // given
        signIn(makeUser({ id: "user-1" }));
        seedStaff([{ id: "user-2", username: "battler", display_name: "Battler" }]);
        mountAndConnect();

        // when
        receive({ type: "role_changed", data: { user_id: "user-2", role: "moderator" } });

        // then
        expect(staff()).toEqual([{ id: "user-2", username: "battler", display_name: "Battler", role: "moderator" }]);
        expect(me()?.role).toBeUndefined();
    });

    it("locks the signed in user when the lock message is about them", () => {
        // given
        signIn(makeUser({ id: "user-1" }));
        mountAndConnect();

        // when
        receive({ type: "lock_changed", data: { user_id: "user-1", locked: true, lock_reason: "too much magic" } });

        // then
        expect(me()).toEqual(expect.objectContaining({ locked: true, lock_reason: "too much magic" }));
    });

    it("bans the signed in user when the ban message is about them", () => {
        // given
        signIn(makeUser({ id: "user-1" }));
        mountAndConnect();

        // when
        receive({ type: "ban_changed", data: { user_id: "user-1", banned: true } });

        // then
        expect(me()).toEqual(expect.objectContaining({ banned: true, ban_reason: "" }));
    });

    it("patches only the profile fields the message carried", () => {
        // given
        signIn(makeUser({ id: "user-1", display_name: "Beatrice", avatar_url: "gold.png" }));
        mountAndConnect();

        // when
        receive({ type: "profile_changed", data: { user_id: "user-1", display_name: "Beato" } });

        // then
        expect(me()).toEqual(expect.objectContaining({ display_name: "Beato", avatar_url: "gold.png" }));
    });

    it("still patches the cache when nobody is signed in", () => {
        // given
        seedStaff([{ id: "user-2", username: "battler", display_name: "Battler" }]);
        mountAndConnect();

        // when
        receive({ type: "ban_changed", data: { user_id: "user-2", banned: true, ban_reason: "endless witch" } });

        // then
        expect(staff()?.[0]).toEqual(
            expect.objectContaining({ id: "user-2", banned: true, ban_reason: "endless witch" }),
        );
        expect(me()).toBeUndefined();
    });

    it("stops reacting once the consumer unmounts", () => {
        // given
        const signedIn = makeUser({ id: "user-1" });
        signIn(signedIn);
        const view = renderHook(() => useUserIdentitySync(), { wrapper });

        act(() => {
            openRealtimeSocket(SESSION_KEY);
            lastSocket().onopen?.();
        });

        // when
        view.unmount();
        receive({ type: "role_changed", data: { user_id: "user-1", role: "moderator" } });

        // then
        expect(me()).toEqual(signedIn);
    });
});
