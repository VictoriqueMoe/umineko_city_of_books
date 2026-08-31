import { renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { makeWatchPartySession } from "../../test-utils/fixtures";
import { providerWrapper } from "../../test-utils/render";
import type { WatchPartySession } from "../../types/api";
import { useRoomWatchPartyInvite } from "./useRoomWatchPartyInvite";

function makeSession(overrides: Partial<WatchPartySession> = {}): WatchPartySession {
    return makeWatchPartySession({
        id: "party-1",
        started_by: "u2",
        controller_id: "u2",
        title: "Higurashi rewatch",
        started_at: "2026-08-02T09:00:00Z",
        ...overrides,
    });
}

interface InviteOptions {
    sessions?: WatchPartySession[];
    loaded?: boolean;
    route?: string;
}

function renderInvite(options: InviteOptions = {}) {
    const join = vi.fn().mockResolvedValue(undefined);
    const onError = vi.fn();
    const rendered = renderHook(
        () =>
            useRoomWatchPartyInvite({
                roomReady: true,
                loaded: options.loaded ?? true,
                sessions: options.sessions ?? [],
                join,
                onError,
            }),
        { wrapper: providerWrapper({ route: options.route ?? "/rooms/room-1?party=party-1" }) },
    );

    return { ...rendered, join, onError };
}

describe("useRoomWatchPartyInvite", () => {
    it("opens the watch party the invite link points at", async () => {
        // given
        const options: InviteOptions = { sessions: [makeSession()] };

        // when
        const { join } = renderInvite(options);

        // then
        await waitFor(() => {
            expect(join).toHaveBeenCalledWith("party-1");
        });
    });

    it("reports an invite that points at a watch party which has ended", () => {
        // given
        const options: InviteOptions = { sessions: [] };

        // when
        const { result, join } = renderInvite(options);

        // then
        expect(result.current.invitedPartyMissing).toBe(true);
        expect(join).not.toHaveBeenCalled();
    });

    it("says nothing about invites while the watch parties are still loading", () => {
        // given
        const options: InviteOptions = { sessions: [], loaded: false };

        // when
        const { result } = renderInvite(options);

        // then
        expect(result.current.invitedPartyMissing).toBe(false);
    });
});
