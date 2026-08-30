import { renderHook, waitFor } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { queryKeys, type RoomsListParams } from "../../api/queryKeys";
import { makeUser } from "../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { useRoomsDirectory } from "./useRoomsDirectory";

const mocks = vi.hoisted(() => ({
    listMyChatRooms: vi.fn(),
    listPublicChatRooms: vi.fn(),
    getUserRooms: vi.fn(),
}));

vi.mock("../../api/endpoints/chat", () => ({
    listMyChatRooms: mocks.listMyChatRooms,
    listPublicChatRooms: mocks.listPublicChatRooms,
    getUserRooms: mocks.getUserRooms,
    getChatRoomMembers: vi.fn(),
    getChatRoomPinnedMessages: vi.fn(),
    getChatUnreadCount: vi.fn(),
    getRoomMessages: vi.fn(),
    getRoomMessagesBefore: vi.fn(),
    listChatRoomBans: vi.fn(),
    listChatRoomBannedWords: vi.fn(),
    resolveDMRoom: vi.fn(),
}));

vi.mock("../mutations/chat", () => ({
    useJoinChatRoom: () => ({ mutateAsync: vi.fn() }),
}));

const viewer = makeUser({ id: "viewer-1", username: "battler", display_name: "Battler" });

const mountedParams: RoomsListParams = {
    search: "",
    rpOnly: false,
    tagFilter: "",
    includeArchived: false,
    pages: 1,
};

interface MountedLists {
    keys: unknown[][];
    queryClient: QueryClient;
}

async function mountRoomsDirectory(): Promise<MountedLists> {
    const queryClient = createTestQueryClient();
    renderHook(() => useRoomsDirectory(), { wrapper: providerWrapper({ queryClient, user: viewer }) });

    await waitFor(() => {
        expect(mocks.listMyChatRooms).toHaveBeenCalledTimes(2);
        expect(mocks.listPublicChatRooms).toHaveBeenCalledTimes(1);
    });

    const keys = queryClient
        .getQueryCache()
        .getAll()
        .map(query => [...query.queryKey]);

    return { keys, queryClient };
}

beforeEach(() => {
    mocks.listMyChatRooms.mockResolvedValue({ rooms: [], total: 0 });
    mocks.listPublicChatRooms.mockResolvedValue({ rooms: [], total: 0 });
    mocks.getUserRooms.mockResolvedValue({ rooms: [] });
});

describe("rooms list query keys", () => {
    it("mounts the hosted list under the key the hosted builder invalidates", async () => {
        // given
        const prefix = queryKeys.chat.roomsListHosted();
        const full = queryKeys.chat.roomsListHosted(mountedParams);

        // when
        const { keys, queryClient } = await mountRoomsDirectory();

        // then
        expect(full.slice(0, prefix.length)).toEqual([...prefix]);
        expect(keys).toContainEqual([...full]);
        expect(queryClient.getQueryCache().findAll({ queryKey: prefix })).toHaveLength(1);
    });

    it("mounts the joined list under the key the joined builder invalidates", async () => {
        // given
        const prefix = queryKeys.chat.roomsListJoined();
        const full = queryKeys.chat.roomsListJoined(mountedParams);

        // when
        const { keys, queryClient } = await mountRoomsDirectory();

        // then
        expect(full.slice(0, prefix.length)).toEqual([...prefix]);
        expect(keys).toContainEqual([...full]);
        expect(queryClient.getQueryCache().findAll({ queryKey: prefix })).toHaveLength(1);
    });

    it("mounts the discover list under the key the discover builder invalidates", async () => {
        // given
        const prefix = queryKeys.chat.roomsListDiscover();
        const full = queryKeys.chat.roomsListDiscover(mountedParams);

        // when
        const { keys, queryClient } = await mountRoomsDirectory();

        // then
        expect(full.slice(0, prefix.length)).toEqual([...prefix]);
        expect(keys).toContainEqual([...full]);
        expect(queryClient.getQueryCache().findAll({ queryKey: prefix })).toHaveLength(1);
    });

    it("reaches all three lists from the shared rooms list builder", async () => {
        // given
        const prefix = queryKeys.chat.roomsList();

        // when
        const { queryClient } = await mountRoomsDirectory();

        // then
        expect(queryClient.getQueryCache().findAll({ queryKey: prefix })).toHaveLength(3);
    });
});
