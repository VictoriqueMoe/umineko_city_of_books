import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type * as BusModule from "../api/realtime/bus";
import { makeUser } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import { makeWSHarness, type RealtimeTestNames, type WSHarness } from "../test-utils/ws";
import { useGameForfeitWarning } from "./useGameForfeitWarning";

const holder = vi.hoisted(() => ({ ws: null as unknown as WSHarness }));

vi.mock("../api/realtime/bus", async importOriginal => {
    const actual = await importOriginal<typeof BusModule>();

    return {
        ...actual,
        subscribe: (names: RealtimeTestNames, handler: BusModule.RealtimeEventHandler) =>
            holder.ws.subscribe(names, handler),
    };
});

const player = makeUser({ id: "user-1", username: "battler", display_name: "Battler" });

function mount() {
    return renderHook(() => useGameForfeitWarning(), { wrapper: providerWrapper({ user: player }) });
}

beforeEach(() => {
    holder.ws = makeWSHarness();
});

describe("useGameForfeitWarning", () => {
    it("listens for the warning, the withdrawal and the finish on a single subscription", () => {
        // when
        mount();

        // then
        expect(holder.ws.subscribe).toHaveBeenCalledTimes(1);
        expect(holder.ws.subscribe.mock.calls[0][0]).toEqual([
            "game_forfeit_warning",
            "game_forfeit_cleared",
            "game_room_finished",
        ]);
    });

    it("keeps the one subscription across a re-render rather than resubscribing", () => {
        // given
        const view = mount();

        // when
        view.rerender();

        // then
        expect(holder.ws.subscribe).toHaveBeenCalledTimes(1);
        expect(holder.ws.unsubscribe).not.toHaveBeenCalled();
    });

    it("drops its subscription when it goes away", () => {
        // given
        const view = mount();

        // when
        view.unmount();

        // then
        expect(holder.ws.unsubscribe).toHaveBeenCalledTimes(1);
    });
});
