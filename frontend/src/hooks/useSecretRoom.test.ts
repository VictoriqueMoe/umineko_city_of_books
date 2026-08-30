import { act, renderHook } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { queryKeys } from "../api/queryKeys";
import type * as OutboundModule from "../api/realtime/outbound";
import { providerWrapper } from "../test-utils/render";
import { emitRealtimeEvent, makeWSHarness, type WSHarness } from "../test-utils/ws";
import type { SecretDetailResponse, SecretLeaderboardEntry, User } from "../types/api";
import { useSecretRoom } from "./useSecretRoom";

const holder = vi.hoisted(() => ({ ws: null as unknown as WSHarness }));
const { useSecret } = vi.hoisted(() => ({ useSecret: vi.fn() }));

vi.mock("./queries/secret", () => ({ useSecret }));

vi.mock("../api/realtime/pipeline", () => ({
    ensureRealtimePipeline: () => {},
    getRealtimeEpoch: () => holder.ws.getEpoch(),
    subscribeRealtimeEpoch: (listener: () => void) => holder.ws.subscribeEpoch(listener),
}));

vi.mock("../api/realtime/outbound", async importOriginal => {
    const actual = await importOriginal<typeof OutboundModule>();

    return { ...actual, sendRealtime: (command: OutboundModule.RealtimeCommand) => holder.ws.sendRealtime(command) };
});

const secretId = "secret-1";
const beatrice: User = { id: "user-1", username: "beatrice", display_name: "Beatrice" };
const ange: User = { id: "user-2", username: "ange", display_name: "Ange" };
const battler: User = { id: "user-3", username: "battler", display_name: "Battler" };

function makeEntry(overrides: Partial<SecretLeaderboardEntry> = {}): SecretLeaderboardEntry {
    return { user: beatrice, pieces_collected: 1, solved: false, ...overrides };
}

function makeDetail(overrides: Partial<SecretDetailResponse> = {}): SecretDetailResponse {
    return {
        id: secretId,
        title: "The First Twilight",
        description: "Six chosen by the key.",
        total_pieces: 5,
        solved: false,
        viewer_progress: 0,
        comment_count: 0,
        riddle: "Who is the golden witch?",
        leaderboard: [],
        comments: [],
        ...overrides,
    };
}

interface MountOptions {
    detail?: SecretDetailResponse | null;
    loading?: boolean;
    id?: string;
    cached?: SecretDetailResponse;
}

function mount(options: MountOptions = {}) {
    const refresh = vi.fn(() => Promise.resolve(undefined));
    useSecret.mockReturnValue({
        data: options.detail === undefined ? makeDetail() : options.detail,
        loading: options.loading ?? false,
        refresh,
    });

    const queryClient = new QueryClient({
        defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false } },
    });
    if (options.cached) {
        queryClient.setQueryData<SecretDetailResponse>(queryKeys.secrets.detail(secretId), options.cached);
    }

    const rendered = renderHook(() => useSecretRoom(options.id ?? secretId), {
        wrapper: providerWrapper({ queryClient }),
    });

    return { ...rendered, refresh, queryClient };
}

function cachedDetail(queryClient: QueryClient): SecretDetailResponse | undefined {
    return queryClient.getQueryData<SecretDetailResponse>(queryKeys.secrets.detail(secretId));
}

function joinCommand(secret = secretId) {
    return { type: "secret_join", data: { secret_id: secret } };
}

function leaveCommand(secret = secretId) {
    return { type: "secret_leave", data: { secret_id: secret } };
}

describe("useSecretRoom", () => {
    beforeEach(() => {
        holder.ws = makeWSHarness();
    });

    it("hands the hunt back with its leaderboard already in rank order", () => {
        // given
        const detail = makeDetail({
            leaderboard: [
                makeEntry({ user: ange, pieces_collected: 2 }),
                makeEntry({ user: battler, pieces_collected: 2 }),
                makeEntry({ user: beatrice, pieces_collected: 1, solved: true }),
            ],
        });

        // when
        const { result } = mount({ detail });

        // then
        expect(result.current.detail?.leaderboard.map(e => e.user.display_name)).toEqual([
            "Beatrice",
            "Ange",
            "Battler",
        ]);
    });

    it("passes the loading state and the missing hunt straight through", () => {
        // given
        const options: MountOptions = { detail: null, loading: true };

        // when
        const { result } = mount(options);

        // then
        expect(result.current.detail).toBeNull();
        expect(result.current.loading).toBe(true);
    });

    it("waits for the socket before joining the hunt's live room", () => {
        // given
        const options: MountOptions = {};

        // when
        const { refresh } = mount(options);

        // then
        expect(holder.ws.sendRealtime).not.toHaveBeenCalled();
        expect(refresh).not.toHaveBeenCalled();
    });

    it("joins the hunt's live room and resyncs once the socket comes up", () => {
        // given
        const { refresh } = mount();

        // when
        holder.ws.reconnect();

        // then
        expect(holder.ws.sendRealtime).toHaveBeenCalledWith(joinCommand());
        expect(refresh).toHaveBeenCalledOnce();
    });

    it("rejoins and resyncs after every reconnect, because the old socket carried the subscription", () => {
        // given
        const { refresh } = mount();
        holder.ws.reconnect();

        // when
        holder.ws.reconnect();

        // then
        expect(holder.ws.sendRealtime.mock.calls.map(call => call[0])).toEqual([
            joinCommand(),
            leaveCommand(),
            joinCommand(),
        ]);
        expect(refresh).toHaveBeenCalledTimes(2);
    });

    it("leaves the hunt's live room when the reader walks away", () => {
        // given
        const { unmount } = mount();
        holder.ws.reconnect();

        // when
        unmount();

        // then
        expect(holder.ws.sendRealtime).toHaveBeenLastCalledWith(leaveCommand());
    });

    it("stays out of the live room while there is no hunt to join", () => {
        // given
        mount({ id: "" });

        // when
        holder.ws.reconnect();

        // then
        expect(holder.ws.sendRealtime).not.toHaveBeenCalled();
    });

    it("moves a hunter's piece count when the server announces progress", () => {
        // given
        const { queryClient } = mount({ cached: makeDetail({ leaderboard: [makeEntry()] }) });

        // when
        emitRealtimeEvent({
            type: "secret_progress",
            data: { secret_id: secretId, user: beatrice, pieces_collected: 4, total_pieces: 5 },
        });

        // then
        expect(cachedDetail(queryClient)?.leaderboard).toEqual([
            { user: beatrice, pieces_collected: 4, solved: false },
        ]);
    });

    it("adds a hunter nobody had seen before when they find their first piece", () => {
        // given
        const { queryClient } = mount({ cached: makeDetail({ leaderboard: [makeEntry()] }) });

        // when
        emitRealtimeEvent({
            type: "secret_progress",
            data: { secret_id: secretId, user: ange, pieces_collected: 1, total_pieces: 5 },
        });

        // then
        expect(cachedDetail(queryClient)?.leaderboard).toHaveLength(2);
        expect(cachedDetail(queryClient)?.leaderboard[1]).toEqual({
            user: ange,
            pieces_collected: 1,
            solved: false,
        });
    });

    it("ignores progress announced for a different hunt", () => {
        // given
        const { queryClient } = mount({ cached: makeDetail({ leaderboard: [makeEntry()] }) });

        // when
        emitRealtimeEvent({
            type: "secret_progress",
            data: { secret_id: "secret-2", user: beatrice, pieces_collected: 4, total_pieces: 5 },
        });

        // then
        expect(cachedDetail(queryClient)?.leaderboard[0].pieces_collected).toBe(1);
    });

    it("names the hunter who spoke the witch's name and marks the hunt solved", () => {
        // given
        const { result, queryClient } = mount({ cached: makeDetail({ leaderboard: [makeEntry()] }) });

        // when
        emitRealtimeEvent({
            type: "secret_solved",
            data: { secret_id: secretId, solver: beatrice, solved_at: "2026-07-02T10:00:00Z" },
        });

        // then
        expect(result.current.solvedByName).toBe("Beatrice");
        expect(cachedDetail(queryClient)?.solved).toBe(true);
        expect(cachedDetail(queryClient)?.solver).toEqual(beatrice);
        expect(cachedDetail(queryClient)?.leaderboard[0].solved).toBe(true);
    });

    it("stays quiet when another hunt is solved", () => {
        // given
        const { result } = mount({ cached: makeDetail({ leaderboard: [makeEntry()] }) });

        // when
        emitRealtimeEvent({
            type: "secret_solved",
            data: { secret_id: "secret-2", solver: beatrice, solved_at: "2026-07-02T10:00:00Z" },
        });

        // then
        expect(result.current.solvedByName).toBeNull();
    });

    it("forgets the announcement once the reader dismisses it", () => {
        // given
        const { result } = mount({ cached: makeDetail() });
        emitRealtimeEvent({
            type: "secret_solved",
            data: { secret_id: secretId, solver: beatrice, solved_at: "2026-07-02T10:00:00Z" },
        });

        // when
        act(() => {
            result.current.dismissSolved();
        });

        // then
        expect(result.current.solvedByName).toBeNull();
    });

    it("leaves a hunt it has never cached alone", () => {
        // given
        const { queryClient } = mount();

        // when
        emitRealtimeEvent({
            type: "secret_progress",
            data: { secret_id: secretId, user: beatrice, pieces_collected: 4, total_pieces: 5 },
        });

        // then
        expect(cachedDetail(queryClient)).toBeUndefined();
    });
});
