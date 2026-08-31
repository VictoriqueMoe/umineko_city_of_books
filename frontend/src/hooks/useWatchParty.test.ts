import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { makeWatchPartySession } from "../test-utils/fixtures";
import { providerWrapper } from "../test-utils/render";
import { emitRealtimeEvent } from "../test-utils/ws";
import type { RealtimeEvent } from "../api/realtime/events";
import type { User, WatchPartyParticipant, WatchPartySession } from "../types/api";
import { useWatchParty } from "./useWatchParty";

const mocks = vi.hoisted(() => ({
    listWatchParties: vi.fn(),
    startWatchParty: vi.fn(),
    joinWatchParty: vi.fn(),
    leaveWatchParty: vi.fn(),
    endWatchParty: vi.fn(),
    transferWatchPartyControl: vi.fn(),
    kickWatchPartyParticipant: vi.fn(),
    identifyWatchPartyParticipant: vi.fn(),
    getWatchPartyVoiceToken: vi.fn(),
    forceMuteWatchPartyVoiceParticipant: vi.fn(),
    resolveOptimalRegion: vi.fn(),
    sendWatchPartyLeaveBeacon: vi.fn(),
}));

vi.mock("../api/endpoints/watchParty", () => ({
    listWatchParties: mocks.listWatchParties,
    startWatchParty: mocks.startWatchParty,
    joinWatchParty: mocks.joinWatchParty,
    leaveWatchParty: mocks.leaveWatchParty,
    endWatchParty: mocks.endWatchParty,
    transferWatchPartyControl: mocks.transferWatchPartyControl,
    kickWatchPartyParticipant: mocks.kickWatchPartyParticipant,
    identifyWatchPartyParticipant: mocks.identifyWatchPartyParticipant,
    getWatchPartyVoiceToken: mocks.getWatchPartyVoiceToken,
    forceMuteWatchPartyVoiceParticipant: mocks.forceMuteWatchPartyVoiceParticipant,
}));

vi.mock("../api/hyperbeam/region", () => ({ resolveOptimalRegion: mocks.resolveOptimalRegion }));

vi.mock("../api/beacons/watchPartyLeave", () => ({ sendWatchPartyLeaveBeacon: mocks.sendWatchPartyLeaveBeacon }));

const roomId = "room-1";
const viewerId = "user-viewer";

function makeChatUser(overrides: Partial<User> = {}): User {
    return {
        id: viewerId,
        username: "beatrice",
        display_name: "Beatrice",
        ...overrides,
    };
}

function makeParticipant(overrides: Partial<WatchPartyParticipant> = {}): WatchPartyParticipant {
    return {
        user: makeChatUser(),
        has_control: false,
        joined_at: "2026-08-01T10:00:00Z",
        ...overrides,
    };
}

function makeSession(overrides: Partial<WatchPartySession> = {}): WatchPartySession {
    return makeWatchPartySession({
        room_id: roomId,
        started_by: viewerId,
        controller_id: viewerId,
        participants: [makeParticipant()],
        ...overrides,
    });
}

function serverLists(sessions: WatchPartySession[], screenShareEnabled = true) {
    mocks.listWatchParties.mockResolvedValue({
        sessions,
        enabled: true,
        screen_share_enabled: screenShareEnabled,
    });
}

interface SetupOptions {
    roomId?: string | null;
    viewerUserId?: string | null;
}

function setup(options: SetupOptions = {}) {
    return renderHook(
        () =>
            useWatchParty(
                options.roomId === undefined ? roomId : options.roomId,
                options.viewerUserId === undefined ? viewerId : options.viewerUserId,
            ),
        { wrapper: providerWrapper() },
    );
}

function emit(event: RealtimeEvent): void {
    emitRealtimeEvent(event);
}

async function setupLoaded(options: SetupOptions = {}) {
    const view = setup(options);
    await waitFor(() => {
        expect(view.result.current.loaded).toBe(true);
    });
    return view;
}

async function setupActive(session: WatchPartySession = makeSession()) {
    serverLists([session]);
    mocks.joinWatchParty.mockResolvedValue({ session, embed_url: "https://hb.test/embed" });
    const view = await setupLoaded();
    await act(async () => {
        await view.result.current.join(session.id);
    });
    await waitFor(() => {
        expect(view.result.current.activeSession?.session.id).toBe(session.id);
    });
    return view;
}

beforeEach(() => {
    serverLists([]);
    mocks.startWatchParty.mockResolvedValue({ session: makeSession(), embed_url: "https://hb.test/embed" });
    mocks.joinWatchParty.mockResolvedValue({ session: makeSession(), embed_url: "https://hb.test/embed" });
    mocks.leaveWatchParty.mockResolvedValue(undefined);
    mocks.endWatchParty.mockResolvedValue(undefined);
    mocks.transferWatchPartyControl.mockResolvedValue(undefined);
    mocks.kickWatchPartyParticipant.mockResolvedValue(undefined);
    mocks.identifyWatchPartyParticipant.mockResolvedValue(undefined);
    mocks.resolveOptimalRegion.mockResolvedValue("EU");
    mocks.sendWatchPartyLeaveBeacon.mockReset();
    vi.spyOn(console, "error").mockImplementation(() => {});
    vi.spyOn(console, "warn").mockImplementation(() => {});
});

describe("useWatchParty loading", () => {
    it("lists the parties of the room it is pointed at", async () => {
        // given
        serverLists([makeSession()], false);

        // when
        const { result } = await setupLoaded();

        // then
        expect(mocks.listWatchParties).toHaveBeenCalledWith(roomId);
        expect(result.current.sessions).toHaveLength(1);
        expect(result.current.enabled).toBe(true);
        expect(result.current.screenShareEnabled).toBe(false);
    });

    it("asks for nothing when there is no room to look at", () => {
        // given
        const { result } = setup({ roomId: null });

        // when
        const state = result.current;

        // then
        expect(mocks.listWatchParties).not.toHaveBeenCalled();
        expect(state.sessions).toEqual([]);
        expect(state.enabled).toBe(false);
        expect(state.screenShareEnabled).toBe(false);
    });

    it("never claims to have loaded anything while there is no room", () => {
        // given
        const { result } = setup({ roomId: null });

        // when
        const state = result.current;

        // then
        expect(state.loaded).toBe(false);
    });

    it("reports why the parties could not be loaded", async () => {
        // given
        mocks.listWatchParties.mockRejectedValue(new Error("the room has vanished"));
        const { result } = setup();

        // when
        await waitFor(() => {
            expect(result.current.error).toBe("the room has vanished");
        });

        // then
        expect(result.current.loaded).toBe(false);
    });

    it("falls back to a plain message when the failure carries none", async () => {
        // given
        mocks.listWatchParties.mockRejectedValue(new Error(""));
        const { result } = setup();

        // when
        await waitFor(() => {
            expect(result.current.error).toBe("Failed to load watch parties");
        });

        // then
        expect(result.current.sessions).toEqual([]);
    });

    it("forgets an error once it has been acknowledged", async () => {
        // given
        mocks.listWatchParties.mockRejectedValue(new Error("the room has vanished"));
        const { result } = setup();
        await waitFor(() => {
            expect(result.current.error).toBe("the room has vanished");
        });

        // when
        act(() => {
            result.current.clearError();
        });

        // then
        expect(result.current.error).toBeNull();
    });

    it("subscribes to the socket only while a room is being watched", async () => {
        // given
        const withRoom = await setupLoaded();
        const withoutRoom = setup({ roomId: null });

        // when
        emit({ type: "watch_party_started", data: { session: makeSession({ id: "session-2" }) } });

        // then
        expect(withRoom.result.current.sessions.map(s => s.id)).toEqual(["session-2"]);
        expect(withoutRoom.result.current.sessions).toEqual([]);
    });

    it("re-reads the party list when the viewer asks for a refresh", async () => {
        // given
        const { result } = await setupLoaded();

        // when
        await act(async () => {
            await result.current.refresh();
        });

        // then
        expect(mocks.listWatchParties).toHaveBeenCalledTimes(2);
    });
});

describe("useWatchParty start", () => {
    it("resolves the nearest hyperbeam region before starting a virtual browser party", async () => {
        // given
        const { result } = await setupLoaded();

        // when
        await act(async () => {
            await result.current.start({ title: "Chiru rewatch" });
        });

        // then
        expect(mocks.resolveOptimalRegion).toHaveBeenCalledOnce();
        expect(mocks.startWatchParty).toHaveBeenCalledWith(roomId, {
            start_url: undefined,
            region: "EU",
            title: "Chiru rewatch",
            type: "hyperbeam",
        });
    });

    it("never asks for a region when the party is a screen share", async () => {
        // given
        const { result } = await setupLoaded();

        // when
        await act(async () => {
            await result.current.start({ type: "screenshare", title: "my desktop" });
        });

        // then
        expect(mocks.resolveOptimalRegion).not.toHaveBeenCalled();
        expect(mocks.startWatchParty).toHaveBeenCalledWith(roomId, {
            start_url: undefined,
            region: undefined,
            title: "my desktop",
            type: "screenshare",
        });
    });

    it("omits an empty region rather than sending a blank one", async () => {
        // given
        mocks.resolveOptimalRegion.mockResolvedValue("");
        const { result } = await setupLoaded();

        // when
        await act(async () => {
            await result.current.start({});
        });

        // then
        expect(mocks.startWatchParty).toHaveBeenCalledWith(roomId, {
            start_url: undefined,
            region: undefined,
            title: undefined,
            type: "hyperbeam",
        });
    });

    it("opens the new party with its embed url", async () => {
        // given
        const session = makeSession({ id: "session-9" });
        mocks.startWatchParty.mockResolvedValue({ session, embed_url: "https://hb.test/embed-9" });
        const { result } = await setupLoaded();
        serverLists([session]);

        // when
        await act(async () => {
            await result.current.start({});
        });

        // then
        expect(result.current.openSessionId).toBe("session-9");
        expect(result.current.activeSession?.embedURL).toBe("https://hb.test/embed-9");
        expect(result.current.sessions).toHaveLength(1);
    });

    it("keeps the party open once the server list catches up with it", async () => {
        // given
        const session = makeSession({ id: "session-9" });
        mocks.startWatchParty.mockResolvedValue({ session, embed_url: "https://hb.test/embed-9" });
        const { result } = await setupLoaded();
        serverLists([session]);

        // when
        await act(async () => {
            await result.current.start({});
        });
        await waitFor(() => {
            expect(mocks.listWatchParties).toHaveBeenCalledTimes(2);
        });

        // then
        expect(result.current.activeSession?.session.id).toBe("session-9");
        expect(result.current.activeSession?.embedURL).toBe("https://hb.test/embed-9");
    });

    it("reports and re-raises a refusal to start a party", async () => {
        // given
        mocks.startWatchParty.mockRejectedValue(new Error("too many parties already"));
        const { result } = await setupLoaded();

        // when
        await act(async () => {
            await expect(result.current.start({})).rejects.toThrow("too many parties already");
        });

        // then
        expect(result.current.error).toBe("too many parties already");
        expect(result.current.openSessionId).toBeNull();
    });

    it("starts nothing when there is no room", async () => {
        // given
        const { result } = setup({ roomId: null });

        // when
        const session = await result.current.start({});

        // then
        expect(session).toBeNull();
        expect(mocks.startWatchParty).not.toHaveBeenCalled();
    });
});

describe("useWatchParty join and leave", () => {
    it("joins a party and makes it the active one", async () => {
        // given
        const session = makeSession({ id: "session-3" });
        serverLists([session]);
        mocks.joinWatchParty.mockResolvedValue({ session, embed_url: "https://hb.test/embed-3" });
        const { result } = await setupLoaded();

        // when
        await act(async () => {
            await result.current.join("session-3");
        });

        // then
        expect(mocks.joinWatchParty).toHaveBeenCalledWith(roomId, "session-3");
        expect(result.current.activeSession?.session.id).toBe("session-3");
        expect(result.current.activeSession?.embedURL).toBe("https://hb.test/embed-3");
    });

    it("reports and re-raises a refusal to join", async () => {
        // given
        mocks.joinWatchParty.mockRejectedValue(new Error("you are banned from this room"));
        const { result } = await setupLoaded();

        // when
        await act(async () => {
            await expect(result.current.join("session-3")).rejects.toThrow("you are banned from this room");
        });

        // then
        expect(result.current.error).toBe("you are banned from this room");
        expect(result.current.openSessionId).toBeNull();
    });

    it("reports whether the viewer holds control of the active party", async () => {
        // given
        const session = makeSession({
            participants: [makeParticipant({ user: makeChatUser({ id: viewerId }), has_control: true })],
        });

        // when
        const { result } = await setupActive(session);

        // then
        expect(result.current.activeSession?.hasControl).toBe(true);
    });

    it("leaves the party alone in the eyes of a viewer who is not a participant", async () => {
        // given
        const session = makeSession({
            participants: [makeParticipant({ user: makeChatUser({ id: "someone-else" }), has_control: true })],
        });

        // when
        const { result } = await setupActive(session);

        // then
        expect(result.current.activeSession?.hasControl).toBe(false);
    });

    it("leaves the party and clears everything about it", async () => {
        // given
        const { result } = await setupActive();

        // when
        await act(async () => {
            await result.current.leave();
        });

        // then
        expect(mocks.leaveWatchParty).toHaveBeenCalledWith(roomId, "session-1");
        expect(result.current.openSessionId).toBeNull();
        expect(result.current.activeSession).toBeNull();
    });

    it("closes the party locally even when the leave request fails", async () => {
        // given
        mocks.leaveWatchParty.mockRejectedValue(new Error("network is down"));
        const { result } = await setupActive();

        // when
        await act(async () => {
            await expect(result.current.leave()).rejects.toThrow("network is down");
        });

        // then
        expect(result.current.error).toBe("network is down");
        expect(result.current.openSessionId).toBeNull();
    });

    it("does nothing when leaving with no party open", async () => {
        // given
        const { result } = await setupLoaded();

        // when
        await act(async () => {
            await result.current.leave();
        });

        // then
        expect(mocks.leaveWatchParty).not.toHaveBeenCalled();
    });

    it("hides the party window without leaving it", async () => {
        // given
        const { result } = await setupActive();

        // when
        act(() => {
            result.current.close();
        });

        // then
        expect(result.current.openSessionId).toBeNull();
        expect(mocks.leaveWatchParty).not.toHaveBeenCalled();
    });

    it("reopens a party the viewer had only hidden", async () => {
        // given
        const { result } = await setupActive();
        act(() => {
            result.current.close();
        });

        // when
        act(() => {
            result.current.openExisting("session-1");
        });

        // then
        expect(result.current.openSessionId).toBe("session-1");
        expect(mocks.joinWatchParty).toHaveBeenCalledOnce();
    });
});

describe("useWatchParty host controls", () => {
    it("ends the active party for everyone", async () => {
        // given
        const { result } = await setupActive();

        // when
        await act(async () => {
            await result.current.end();
        });

        // then
        expect(mocks.endWatchParty).toHaveBeenCalledWith(roomId, "session-1");
    });

    it("reports and re-raises a refusal to end the party", async () => {
        // given
        mocks.endWatchParty.mockRejectedValue(new Error("only the host may end it"));
        const { result } = await setupActive();

        // when
        await act(async () => {
            await expect(result.current.end()).rejects.toThrow("only the host may end it");
        });

        // then
        expect(result.current.error).toBe("only the host may end it");
    });

    it("hands control of the party to another watcher", async () => {
        // given
        const { result } = await setupActive();

        // when
        await act(async () => {
            await result.current.transferControl("user-battler");
        });

        // then
        expect(mocks.transferWatchPartyControl).toHaveBeenCalledWith(roomId, "session-1", "user-battler");
    });

    it("reports and re-raises a refusal to hand over control", async () => {
        // given
        mocks.transferWatchPartyControl.mockRejectedValue(new Error("you do not outrank the controller"));
        const { result } = await setupActive();

        // when
        await act(async () => {
            await expect(result.current.transferControl("user-battler")).rejects.toThrow(
                "you do not outrank the controller",
            );
        });

        // then
        expect(result.current.error).toBe("you do not outrank the controller");
    });

    it("removes a watcher from the party", async () => {
        // given
        const { result } = await setupActive();

        // when
        await act(async () => {
            await result.current.kick("user-battler");
        });

        // then
        expect(mocks.kickWatchPartyParticipant).toHaveBeenCalledWith(roomId, "session-1", "user-battler");
    });

    it("reports and re-raises a refusal to remove a watcher", async () => {
        // given
        mocks.kickWatchPartyParticipant.mockRejectedValue(new Error("that watcher outranks you"));
        const { result } = await setupActive();

        // when
        await act(async () => {
            await expect(result.current.kick("user-battler")).rejects.toThrow("that watcher outranks you");
        });

        // then
        expect(result.current.error).toBe("that watcher outranks you");
    });

    it("leaves the host controls idle while no party is open", async () => {
        // given
        const { result } = await setupLoaded();

        // when
        await act(async () => {
            await result.current.end();
            await result.current.transferControl("user-battler");
            await result.current.kick("user-battler");
            await result.current.identify("hb-1");
        });

        // then
        expect(mocks.endWatchParty).not.toHaveBeenCalled();
        expect(mocks.transferWatchPartyControl).not.toHaveBeenCalled();
        expect(mocks.kickWatchPartyParticipant).not.toHaveBeenCalled();
        expect(mocks.identifyWatchPartyParticipant).not.toHaveBeenCalled();
    });
});

describe("useWatchParty identify", () => {
    it("tells the server which virtual browser seat the viewer took", async () => {
        // given
        const { result } = await setupActive();

        // when
        await act(async () => {
            await result.current.identify("hb-user-7");
        });

        // then
        expect(mocks.identifyWatchPartyParticipant).toHaveBeenCalledWith(roomId, "session-1", "hb-user-7");
    });

    it("skips identifying with an empty seat", async () => {
        // given
        const { result } = await setupActive();

        // when
        await act(async () => {
            await result.current.identify("");
        });

        // then
        expect(mocks.identifyWatchPartyParticipant).not.toHaveBeenCalled();
    });

    it("swallows a failure to identify rather than disturbing the viewer", async () => {
        // given
        mocks.identifyWatchPartyParticipant.mockRejectedValue(new Error("unknown seat"));
        const { result } = await setupActive();

        // when
        await act(async () => {
            await result.current.identify("hb-user-7");
        });

        // then
        expect(result.current.error).toBeNull();
    });
});

describe("useWatchParty socket events", () => {
    it("adds a party somebody else started in this room", async () => {
        // given
        const { result } = await setupLoaded();

        // when
        emit({ type: "watch_party_started", data: { session: makeSession({ id: "session-2" }) } });

        // then
        expect(result.current.sessions.map(s => s.id)).toEqual(["session-2"]);
    });

    it("ignores a party started in some other room", async () => {
        // given
        const { result } = await setupLoaded();

        // when
        emit({
            type: "watch_party_started",
            data: { session: makeSession({ id: "session-2", room_id: "room-other" }) },
        });

        // then
        expect(result.current.sessions).toEqual([]);
    });

    it("replaces a party it already knows about rather than listing it twice", async () => {
        // given
        serverLists([makeSession({ title: "old title" })]);
        const { result } = await setupLoaded();

        // when
        emit({ type: "watch_party_started", data: { session: makeSession({ title: "new title" }) } });

        // then
        expect(result.current.sessions).toHaveLength(1);
        expect(result.current.sessions[0].title).toBe("new title");
    });

    it("drops a party that has ended and closes it when it was the open one", async () => {
        // given
        const { result } = await setupActive();

        // when
        emit({
            type: "watch_party_ended",
            data: { session_id: "session-1", room_id: roomId, reason: "host left" },
        });

        // then
        expect(result.current.sessions).toEqual([]);
        expect(result.current.openSessionId).toBeNull();
    });

    it("ignores the ending of a party in another room", async () => {
        // given
        const { result } = await setupActive();

        // when
        emit({
            type: "watch_party_ended",
            data: { session_id: "session-1", room_id: "room-other", reason: "host left" },
        });

        // then
        expect(result.current.openSessionId).toBe("session-1");
    });

    it("adds a watcher who joined the party", async () => {
        // given
        const { result } = await setupActive();

        // when
        emit({
            type: "watch_party_participant_joined",
            data: {
                session_id: "session-1",
                room_id: roomId,
                participant: makeParticipant({ user: makeChatUser({ id: "user-battler" }) }),
            },
        });

        // then
        expect(result.current.activeSession?.session.participants).toHaveLength(2);
    });

    it("replaces a watcher who rejoined rather than listing them twice", async () => {
        // given
        const { result } = await setupActive();

        // when
        emit({
            type: "watch_party_participant_joined",
            data: {
                session_id: "session-1",
                room_id: roomId,
                participant: makeParticipant({ has_control: true }),
            },
        });

        // then
        expect(result.current.activeSession?.session.participants).toHaveLength(1);
        expect(result.current.activeSession?.hasControl).toBe(true);
    });

    it("removes a watcher who left the party", async () => {
        // given
        const session = makeSession({
            participants: [makeParticipant(), makeParticipant({ user: makeChatUser({ id: "user-battler" }) })],
        });
        const { result } = await setupActive(session);

        // when
        emit({
            type: "watch_party_participant_left",
            data: { session_id: "session-1", room_id: roomId, user_id: "user-battler" },
        });

        // then
        expect(result.current.activeSession?.session.participants).toHaveLength(1);
        expect(result.current.openSessionId).toBe("session-1");
    });

    it("closes the party for the viewer when it is the viewer who left", async () => {
        // given
        const { result } = await setupActive();

        // when
        emit({
            type: "watch_party_participant_left",
            data: { session_id: "session-1", room_id: roomId, user_id: viewerId },
        });

        // then
        expect(result.current.openSessionId).toBeNull();
        expect(result.current.activeSession).toBeNull();
    });

    it("moves the control badge when control changes hands", async () => {
        // given
        const { result } = await setupActive();

        // when
        emit({
            type: "watch_party_control_changed",
            data: { session_id: "session-1", room_id: roomId, user_id: viewerId, has_control: true },
        });

        // then
        expect(result.current.activeSession?.hasControl).toBe(true);
    });

    it("tells the viewer they were removed and shuts the party window", async () => {
        // given
        const { result } = await setupActive();

        // when
        emit({
            type: "watch_party_kicked",
            data: { session_id: "session-1", room_id: roomId, actor_id: "user-ronove" },
        });

        // then
        expect(result.current.error).toBe("You were removed from the watch party.");
        expect(result.current.openSessionId).toBeNull();
    });

    it("ignores a removal from a party in another room", async () => {
        // given
        const { result } = await setupActive();

        // when
        emit({
            type: "watch_party_kicked",
            data: { session_id: "session-1", room_id: "room-other", actor_id: "user-ronove" },
        });

        // then
        expect(result.current.error).toBeNull();
        expect(result.current.openSessionId).toBe("session-1");
    });

    it("pays no attention to socket traffic about anything else", async () => {
        // given
        const { result } = await setupActive();

        // when
        emit({ type: "chat_message", data: { body: "unrelated" } } as unknown as RealtimeEvent);

        // then
        expect(result.current.openSessionId).toBe("session-1");
        expect(result.current.error).toBeNull();
    });

    it("stops listening to the socket when it is torn down", async () => {
        // given
        const { result, unmount } = await setupActive();

        // when
        unmount();
        emit({
            type: "watch_party_kicked",
            data: { session_id: "session-1", room_id: roomId, actor_id: "user-ronove" },
        });

        // then
        expect(result.current.openSessionId).toBe("session-1");
    });
});

describe("useWatchParty presence beacons", () => {
    it("tells the server the viewer is gone when the tab is closed", async () => {
        // given
        await setupActive();

        // when
        act(() => {
            window.dispatchEvent(new Event("beforeunload"));
        });

        // then
        expect(mocks.sendWatchPartyLeaveBeacon).toHaveBeenCalledWith("room-1", "session-1");
    });

    it("leaves the party after the tab has been hidden for ten minutes", async () => {
        // given
        await setupActive();
        vi.useFakeTimers();
        Object.defineProperty(document, "visibilityState", { configurable: true, get: () => "hidden" });

        // when
        act(() => {
            document.dispatchEvent(new Event("visibilitychange"));
        });
        act(() => {
            vi.advanceTimersByTime(10 * 60 * 1000);
        });

        // then
        expect(mocks.sendWatchPartyLeaveBeacon).toHaveBeenCalledOnce();
    });

    it("keeps the viewer in the party when the tab comes back before the deadline", async () => {
        // given
        await setupActive();
        vi.useFakeTimers();
        let visibility = "hidden";
        Object.defineProperty(document, "visibilityState", { configurable: true, get: () => visibility });

        // when
        act(() => {
            document.dispatchEvent(new Event("visibilitychange"));
        });
        act(() => {
            vi.advanceTimersByTime(5 * 60 * 1000);
        });
        visibility = "visible";
        act(() => {
            document.dispatchEvent(new Event("visibilitychange"));
        });
        act(() => {
            vi.advanceTimersByTime(10 * 60 * 1000);
        });

        // then
        expect(mocks.sendWatchPartyLeaveBeacon).not.toHaveBeenCalled();
    });
});
