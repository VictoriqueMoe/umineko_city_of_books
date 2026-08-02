import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { WSMessageHandler } from "../context/notificationContextValue";
import { makeUser } from "../test-utils/fixtures";
import { providerWrapper, type ProviderOptions } from "../test-utils/render";
import type { WSMessage } from "../types/api";
import { useSidebarBadges } from "./useSidebarBadges";

const mocks = vi.hoisted(() => ({
    activity: null as { activity: Record<string, string> } | null,
    visited: null as { visited: Record<string, string> } | null,
    refreshVisited: vi.fn(),
    markVisitedMutate: vi.fn(),
    markVisitedAsync: vi.fn(),
}));

vi.mock("../api/queries/sidebar", () => ({
    useSidebarActivity: () => ({ data: mocks.activity, loading: false }),
    useSidebarLastVisited: () => ({ data: mocks.visited, loading: false, refresh: mocks.refreshVisited }),
}));

vi.mock("../api/mutations/sidebar", () => ({
    useMarkSidebarVisited: () => ({ mutate: mocks.markVisitedMutate, mutateAsync: mocks.markVisitedAsync }),
}));

const user = makeUser();
const storageKey = `sidebarLastVisited:${user.id}`;

function setActivity(activity: Record<string, string>) {
    mocks.activity = { activity };
}

function setVisited(visited: Record<string, string>) {
    mocks.visited = { visited };
}

function setup(options: ProviderOptions = {}) {
    const wrapper = providerWrapper({ user, ...options });

    return renderHook(() => useSidebarBadges(), { wrapper });
}

function captureWS() {
    const handlers: WSMessageHandler[] = [];

    const addWSListener = vi.fn((handler: WSMessageHandler) => {
        handlers.push(handler);
        return () => {};
    });

    function emit(msg: WSMessage) {
        act(() => {
            for (const handler of handlers) {
                handler(msg);
            }
        });
    }

    return { addWSListener, emit };
}

beforeEach(() => {
    mocks.activity = null;
    mocks.visited = null;
    mocks.markVisitedAsync.mockResolvedValue(undefined);
});

describe("useSidebarBadges for a signed out visitor", () => {
    it("reports nothing unread even when the server sent activity", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z" });

        // when
        const { result } = setup({ user: null });

        // then
        expect(result.current.hasUnread("chat")).toBe(false);
        expect(result.current.hasAnyUnread(["chat"])).toBe(false);
        expect(result.current.anyUnread).toBe(false);
    });

    it("does not listen for activity announcements", () => {
        // given
        const ws = captureWS();

        // when
        setup({ user: null, notification: { addWSListener: ws.addWSListener } });

        // then
        expect(ws.addWSListener).not.toHaveBeenCalled();
    });

    it("ignores requests to mark sections as visited", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z" });
        const { result } = setup({ user: null });

        // when
        act(() => {
            result.current.markVisited("chat");
            result.current.markAllVisited();
        });

        // then
        expect(mocks.markVisitedMutate).not.toHaveBeenCalled();
    });

    it("never migrates visits saved in local storage", () => {
        // given
        window.localStorage.setItem(storageKey, JSON.stringify({ chat: "2026-01-05T10:00:00Z" }));

        // when
        setup({ user: null });

        // then
        expect(mocks.markVisitedAsync).not.toHaveBeenCalled();
        expect(window.localStorage.getItem(storageKey)).not.toBeNull();
    });
});

describe("useSidebarBadges unread rules", () => {
    it("flags a section that has activity but has never been visited", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z" });
        setVisited({});

        // when
        const { result } = setup();

        // then
        expect(result.current.hasUnread("chat")).toBe(true);
    });

    it("flags a section whose activity is newer than the last visit", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z" });
        setVisited({ chat: "2026-01-05T09:00:00Z" });

        // when
        const { result } = setup();

        // then
        expect(result.current.hasUnread("chat")).toBe(true);
    });

    it("clears a section that was visited after its last activity", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z" });
        setVisited({ chat: "2026-01-05T11:00:00Z" });

        // when
        const { result } = setup();

        // then
        expect(result.current.hasUnread("chat")).toBe(false);
    });

    it("clears a section that was visited at the very same moment", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z" });
        setVisited({ chat: "2026-01-05T10:00:00Z" });

        // when
        const { result } = setup();

        // then
        expect(result.current.hasUnread("chat")).toBe(false);
    });

    it("reports nothing for a section the server never mentioned", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z" });
        setVisited({});

        // when
        const { result } = setup();

        // then
        expect(result.current.hasUnread("art")).toBe(false);
    });

    it("reads a server timestamp that carries no timezone as utc", () => {
        // given
        setActivity({ chat: "2026-01-05 10:00:00" });
        setVisited({ chat: "2026-01-05 09:00:00" });

        // when
        const { result } = setup();

        // then
        expect(result.current.hasUnread("chat")).toBe(true);
    });

    it("treats an unreadable activity timestamp as read", () => {
        // given
        setActivity({ chat: "nonsense" });
        setVisited({});

        // when
        const { result } = setup();

        // then
        expect(result.current.hasUnread("chat")).toBe(false);
    });

    it("treats an unreadable visit timestamp as never visited", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z" });
        setVisited({ chat: "nonsense" });

        // when
        const { result } = setup();

        // then
        expect(result.current.hasUnread("chat")).toBe(true);
    });

    it("reports nothing unread before the server answers", () => {
        // given
        mocks.activity = null;
        mocks.visited = null;

        // when
        const { result } = setup();

        // then
        expect(result.current.hasUnread("chat")).toBe(false);
        expect(result.current.anyUnread).toBe(false);
    });
});

describe("useSidebarBadges aggregation", () => {
    it("reports a group as unread when any one of its sections is", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z", art: "2026-01-05T10:00:00Z" });
        setVisited({ chat: "2026-01-05T11:00:00Z" });

        // when
        const { result } = setup();

        // then
        expect(result.current.hasAnyUnread(["chat", "art"])).toBe(true);
        expect(result.current.hasAnyUnread(["chat"])).toBe(false);
    });

    it("reports an empty group as read", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z" });
        setVisited({});

        // when
        const { result } = setup();

        // then
        expect(result.current.hasAnyUnread([])).toBe(false);
    });

    it("reports the whole sidebar as read once every section has been visited", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z", art: "2026-01-04T10:00:00Z" });
        setVisited({ chat: "2026-01-05T11:00:00Z", art: "2026-01-04T11:00:00Z" });

        // when
        const { result } = setup();

        // then
        expect(result.current.anyUnread).toBe(false);
    });

    it("reports the whole sidebar as unread when a single section lags behind", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z", art: "2026-01-04T10:00:00Z" });
        setVisited({ chat: "2026-01-05T11:00:00Z" });

        // when
        const { result } = setup();

        // then
        expect(result.current.anyUnread).toBe(true);
    });
});

describe("useSidebarBadges marking visits", () => {
    it("marks an unread section as visited", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z" });
        setVisited({});
        const { result } = setup();

        // when
        act(() => {
            result.current.markVisited("chat");
        });

        // then
        expect(mocks.markVisitedMutate).toHaveBeenCalledWith("chat");
    });

    it("does not tell the server about a section that was already read", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z" });
        setVisited({ chat: "2026-01-05T11:00:00Z" });
        const { result } = setup();

        // when
        act(() => {
            result.current.markVisited("chat");
        });

        // then
        expect(mocks.markVisitedMutate).not.toHaveBeenCalled();
    });

    it("marks only the unread sections when clearing everything", () => {
        // given
        setActivity({
            chat: "2026-01-05T10:00:00Z",
            art: "2026-01-05T10:00:00Z",
            games: "2026-01-05T10:00:00Z",
        });
        setVisited({ art: "2026-01-05T11:00:00Z" });
        const { result } = setup();

        // when
        act(() => {
            result.current.markAllVisited();
        });

        // then
        expect(mocks.markVisitedMutate).toHaveBeenCalledTimes(2);
        expect(mocks.markVisitedMutate).toHaveBeenCalledWith("chat");
        expect(mocks.markVisitedMutate).toHaveBeenCalledWith("games");
    });

    it("stays quiet when clearing everything and nothing is unread", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z" });
        setVisited({ chat: "2026-01-05T11:00:00Z" });
        const { result } = setup();

        // when
        act(() => {
            result.current.markAllVisited();
        });

        // then
        expect(mocks.markVisitedMutate).not.toHaveBeenCalled();
    });
});

describe("useSidebarBadges live announcements", () => {
    it("flags a section when the server announces fresh activity", () => {
        // given
        setActivity({});
        setVisited({ chat: "2026-01-05T09:00:00Z" });
        const ws = captureWS();
        const { result } = setup({ notification: { addWSListener: ws.addWSListener } });
        expect(result.current.hasUnread("chat")).toBe(false);

        // when
        ws.emit({ type: "sidebar_activity", data: { key: "chat", at: "2026-01-05T10:00:00Z" } });

        // then
        expect(result.current.hasUnread("chat")).toBe(true);
    });

    it("keeps the newer timestamp when an older announcement arrives late", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z" });
        setVisited({ chat: "2026-01-05T09:00:00Z" });
        const ws = captureWS();
        const { result } = setup({ notification: { addWSListener: ws.addWSListener } });

        // when
        ws.emit({ type: "sidebar_activity", data: { key: "chat", at: "2026-01-04T10:00:00Z" } });

        // then
        expect(result.current.hasUnread("chat")).toBe(true);
    });

    it("lets fresher server activity overrule an earlier announcement", () => {
        // given
        setActivity({ chat: "2026-01-05T10:00:00Z" });
        setVisited({ chat: "2026-01-05T09:00:00Z" });
        const ws = captureWS();
        const { result, rerender } = setup({ notification: { addWSListener: ws.addWSListener } });
        ws.emit({ type: "sidebar_activity", data: { key: "chat", at: "2026-01-05T10:30:00Z" } });

        // when
        setActivity({ chat: "2026-01-05T11:00:00Z" });
        setVisited({ chat: "2026-01-05T10:45:00Z" });
        rerender();

        // then
        expect(result.current.hasUnread("chat")).toBe(true);
    });

    it("keeps the newer announcement when a later one is written in another format", () => {
        // given
        setActivity({});
        setVisited({ chat: "2026-01-05 10:30:00" });
        const ws = captureWS();
        const { result } = setup({ notification: { addWSListener: ws.addWSListener } });
        ws.emit({ type: "sidebar_activity", data: { key: "chat", at: "2026-01-05 11:00:00" } });

        // when
        ws.emit({ type: "sidebar_activity", data: { key: "chat", at: "2026-01-05T10:00:00Z" } });

        // then
        expect(result.current.hasUnread("chat")).toBe(true);
    });

    it("ignores websocket messages of any other type", () => {
        // given
        setActivity({});
        setVisited({});
        const ws = captureWS();
        const { result } = setup({ notification: { addWSListener: ws.addWSListener } });

        // when
        ws.emit({ type: "notification", data: { key: "chat", at: "2026-01-05T10:00:00Z" } });

        // then
        expect(result.current.hasUnread("chat")).toBe(false);
    });

    it("ignores an announcement that names no section", () => {
        // given
        setActivity({});
        setVisited({});
        const ws = captureWS();
        const { result } = setup({ notification: { addWSListener: ws.addWSListener } });

        // when
        ws.emit({ type: "sidebar_activity", data: { at: "2026-01-05T10:00:00Z" } });

        // then
        expect(result.current.anyUnread).toBe(false);
    });

    it("ignores an announcement that carries no timestamp", () => {
        // given
        setActivity({});
        setVisited({});
        const ws = captureWS();
        const { result } = setup({ notification: { addWSListener: ws.addWSListener } });

        // when
        ws.emit({ type: "sidebar_activity", data: { key: "chat" } });

        // then
        expect(result.current.anyUnread).toBe(false);
    });
});

describe("useSidebarBadges legacy migration", () => {
    it("sends every saved visit to the server and forgets the saved copy", async () => {
        // given
        setActivity({});
        setVisited({});
        window.localStorage.setItem(
            storageKey,
            JSON.stringify({ chat: "2026-01-05T10:00:00Z", art: "2026-01-05T11:00:00Z" }),
        );

        // when
        setup();

        // then
        await waitFor(() => expect(window.localStorage.getItem(storageKey)).toBeNull());
        expect(mocks.markVisitedAsync).toHaveBeenCalledWith("chat");
        expect(mocks.markVisitedAsync).toHaveBeenCalledWith("art");
        expect(mocks.refreshVisited).toHaveBeenCalled();
    });

    it("keeps the saved visits when the server rejects one of them", async () => {
        // given
        setActivity({});
        setVisited({});
        mocks.markVisitedAsync.mockRejectedValue(new Error("nope"));
        window.localStorage.setItem(storageKey, JSON.stringify({ chat: "2026-01-05T10:00:00Z" }));

        // when
        setup();

        // then
        await waitFor(() => expect(mocks.markVisitedAsync).toHaveBeenCalledWith("chat"));
        expect(window.localStorage.getItem(storageKey)).not.toBeNull();
        expect(mocks.refreshVisited).not.toHaveBeenCalled();
    });

    it("throws away an empty saved record without troubling the server", () => {
        // given
        setActivity({});
        setVisited({});
        window.localStorage.setItem(storageKey, JSON.stringify({}));

        // when
        setup();

        // then
        expect(window.localStorage.getItem(storageKey)).toBeNull();
        expect(mocks.markVisitedAsync).not.toHaveBeenCalled();
    });

    it("leaves a saved record that is not valid json alone", () => {
        // given
        setActivity({});
        setVisited({});
        window.localStorage.setItem(storageKey, "{not json");

        // when
        setup();

        // then
        expect(mocks.markVisitedAsync).not.toHaveBeenCalled();
        expect(window.localStorage.getItem(storageKey)).toBe("{not json");
    });

    it("does nothing when there is no saved record at all", () => {
        // given
        setActivity({});
        setVisited({});

        // when
        setup();

        // then
        expect(mocks.markVisitedAsync).not.toHaveBeenCalled();
        expect(mocks.refreshVisited).not.toHaveBeenCalled();
    });
});
