import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { queryKeys } from "../../api/queryKeys";
import { makeChatRoom, makeUser } from "../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { emitRealtimeEvent } from "../../test-utils/ws";
import type { ChatRoom } from "../../types/api";
import { useRoomsDirectory } from "./useRoomsDirectory";

const mocks = vi.hoisted(() => ({
    listMyChatRooms: vi.fn(),
    listPublicChatRooms: vi.fn(),
    getUserRooms: vi.fn(),
    joinMutateAsync: vi.fn(),
}));

vi.mock("../../api/endpoints/chat", () => ({
    listMyChatRooms: mocks.listMyChatRooms,
    listPublicChatRooms: mocks.listPublicChatRooms,
    getUserRooms: mocks.getUserRooms,
}));

vi.mock("../mutations/chat", () => ({
    useJoinChatRoom: () => ({ mutateAsync: mocks.joinMutateAsync }),
}));

const viewer = makeUser({ id: "viewer-1", username: "battler", display_name: "Battler" });

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeChatRoom({ member_count: 4, ...overrides });
}

function stubLists(system: ChatRoom[] = []) {
    mocks.listMyChatRooms.mockResolvedValue({ rooms: [], total: 0 });
    mocks.listPublicChatRooms.mockResolvedValue({ rooms: [], total: 0 });
    mocks.getUserRooms.mockResolvedValue({ rooms: system });
}

async function renderDirectory() {
    const queryClient = createTestQueryClient();
    const rendered = renderHook(() => useRoomsDirectory(), {
        wrapper: providerWrapper({ user: viewer, queryClient }),
    });

    await waitFor(() => {
        expect(mocks.getUserRooms).toHaveBeenCalled();
    });
    await act(async () => {});

    return { ...rendered, queryClient };
}

beforeEach(() => {
    stubLists();
    mocks.joinMutateAsync.mockResolvedValue(makeRoom({ id: "room-joined" }));
});

describe("useRoomsDirectory live updates", () => {
    it("refreshes the joined and pinned rooms on an invitation", async () => {
        // given
        const { queryClient } = await renderDirectory();
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        emitRealtimeEvent({ type: "chat_room_invited", data: { room_id: "room-1" } });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.chat.roomsListJoined() });
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.chat.userRooms() });
    });

    it("refreshes every list when the member is kicked", async () => {
        // given
        const { queryClient } = await renderDirectory();
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        emitRealtimeEvent({ type: "chat_kicked", data: { room_id: "room-1" } });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.chat.roomsList() });
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.chat.userRooms() });
    });

    it("refreshes every list when a room is renamed or deleted", async () => {
        // given
        const { queryClient } = await renderDirectory();
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        emitRealtimeEvent({ type: "chat_room_deleted", data: { room_id: "room-1" } });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.chat.roomsList() });
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.chat.userRooms() });
    });

    it("reads the pinned rooms from the shared user rooms entry", async () => {
        // given
        stubLists([makeRoom({ id: "room-sys", name: "Staff Lounge", is_system: true })]);

        // when
        const { result, queryClient } = await renderDirectory();

        // then
        await waitFor(() => {
            expect(result.current.systemRooms.map(room => room.name)).toEqual(["Staff Lounge"]);
        });
        expect(queryClient.getQueryData(queryKeys.chat.userRooms())).toBeDefined();
    });

    it("refreshes the lists when voice presence changes", async () => {
        // given
        const { queryClient } = await renderDirectory();
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        emitRealtimeEvent({ type: "voice_presence", data: { room_id: "room-1", participants: [], count: 0 } });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: queryKeys.chat.roomsList() });
    });

    it("ignores chat events it does not care about", async () => {
        // given
        const { queryClient } = await renderDirectory();
        const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

        // when
        emitRealtimeEvent({ type: "chat_message", data: { id: "m-1", room_id: "room-1" } });

        // then
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});
