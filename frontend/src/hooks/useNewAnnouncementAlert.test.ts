import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type * as BusModule from "../api/realtime/bus";
import { makeUser } from "../test-utils/fixtures";
import { providerWrapper, type ProviderOptions } from "../test-utils/render";
import { makeWSHarness, type RealtimeTestNames, type WSHarness } from "../test-utils/ws";
import { useNewAnnouncementAlert } from "./useNewAnnouncementAlert";

const holder = vi.hoisted(() => ({ ws: null as unknown as WSHarness }));

vi.mock("../api/realtime/bus", async importOriginal => {
    const actual = await importOriginal<typeof BusModule>();

    return {
        ...actual,
        subscribe: (names: RealtimeTestNames, handler: BusModule.RealtimeEventHandler) =>
            holder.ws.subscribe(names, handler),
    };
});

const reader = makeUser({ id: "reader-1", username: "battler", display_name: "Battler" });

function mount(onAnnouncement: () => void, options: ProviderOptions = {}) {
    return renderHook(() => useNewAnnouncementAlert(onAnnouncement), {
        wrapper: providerWrapper({ user: reader, ...options }),
    });
}

beforeEach(() => {
    holder.ws = makeWSHarness();
});

describe("useNewAnnouncementAlert", () => {
    it("listens for announcements alone", () => {
        // when
        mount(vi.fn());

        // then
        expect(holder.ws.subscribe).toHaveBeenCalledTimes(1);
        expect(holder.ws.subscribe.mock.calls[0][0]).toBe("new_announcement");
    });

    it("calls back when somebody else publishes an announcement", () => {
        // given
        const onAnnouncement = vi.fn();
        mount(onAnnouncement);

        // when
        holder.ws.emit({ type: "new_announcement", data: { id: "a1", title: "Tea", author_id: "someone-else" } });

        // then
        expect(onAnnouncement).toHaveBeenCalledOnce();
    });

    it("stays quiet about an announcement the reader wrote themselves", () => {
        // given
        const onAnnouncement = vi.fn();
        mount(onAnnouncement);

        // when
        holder.ws.emit({ type: "new_announcement", data: { id: "a1", title: "Tea", author_id: "reader-1" } });

        // then
        expect(onAnnouncement).not.toHaveBeenCalled();
    });

    it("stays quiet while the announcements page is already open", () => {
        // given
        const onAnnouncement = vi.fn();
        mount(onAnnouncement, { route: "/announcements" });

        // when
        holder.ws.emit({ type: "new_announcement", data: { id: "a1", title: "Tea", author_id: "someone-else" } });

        // then
        expect(onAnnouncement).not.toHaveBeenCalled();
    });

    it("calls back for a signed out visitor", () => {
        // given
        const onAnnouncement = vi.fn();
        mount(onAnnouncement, { user: null });

        // when
        holder.ws.emit({ type: "new_announcement", data: { id: "a1", title: "Tea", author_id: "someone-else" } });

        // then
        expect(onAnnouncement).toHaveBeenCalledOnce();
    });

    it("drops its subscription when it goes away", () => {
        // given
        const view = mount(vi.fn());

        // when
        view.unmount();

        // then
        expect(holder.ws.unsubscribe).toHaveBeenCalledTimes(1);
    });
});
