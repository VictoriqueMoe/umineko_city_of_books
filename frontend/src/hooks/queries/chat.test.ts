import type { QueryClient } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { UserProfile } from "../../types/api";
import { makeChatRoom, makeDmRoom, makeUser } from "../../test-utils/fixtures";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import { queryClient as sharedQueryClient } from "../../api/queryClient";
import { queryKeys, type RoomsListParams } from "../../api/queryKeys";
import {
    fetchResolveDMRoom,
    fetchRoomMessages,
    fetchRoomMessagesBefore,
    useChatRoomBannedWords,
    useChatRoomBans,
    useChatRoomMembers,
    useChatRoomPinnedMessages,
    useChatUnreadCount,
    useDmRooms,
    useHostedRooms,
    useJoinedRooms,
    usePublicRooms,
    useUserRooms,
    useUserRoomsCache,
} from "./chat";

const endpoints = vi.hoisted(() => ({
    getChatRoomMembers: vi.fn(),
    getChatRoomPinnedMessages: vi.fn(),
    getChatUnreadCount: vi.fn(),
    getRoomMessages: vi.fn(),
    getRoomMessagesBefore: vi.fn(),
    getUserRooms: vi.fn(),
    listChatRoomBans: vi.fn(),
    listChatRoomBannedWords: vi.fn(),
    listMyChatRooms: vi.fn(),
    listPublicChatRooms: vi.fn(),
    resolveDMRoom: vi.fn(),
}));

vi.mock("../../api/endpoints/chat", () => endpoints);

const dmRoom = makeDmRoom({ id: "room-1" });
const groupRoom = makeChatRoom({ id: "room-2" });

const unfiltered: RoomsListParams = {
    search: "",
    rpOnly: false,
    tagFilter: "",
    includeArchived: false,
    pages: 1,
};

function setup<T>(hook: () => T, user: UserProfile | null = null) {
    const queryClient = createTestQueryClient();
    const rendered = renderHook(hook, { wrapper: providerWrapper({ queryClient, user }) });

    return { ...rendered, queryClient };
}

function firstKey(queryClient: QueryClient): readonly unknown[] {
    return queryClient.getQueryCache().getAll()[0].queryKey;
}

async function settle(): Promise<void> {
    await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
    });
}

beforeEach(() => {
    sharedQueryClient.clear();
    endpoints.getChatRoomMembers.mockResolvedValue({ members: [] });
    endpoints.getChatRoomPinnedMessages.mockResolvedValue({ messages: [] });
    endpoints.getChatUnreadCount.mockResolvedValue({ count: 0 });
    endpoints.getRoomMessages.mockResolvedValue({ messages: [], total: 0 });
    endpoints.getRoomMessagesBefore.mockResolvedValue({ messages: [] });
    endpoints.getUserRooms.mockResolvedValue({ rooms: [] });
    endpoints.listChatRoomBans.mockResolvedValue({ bans: [] });
    endpoints.listChatRoomBannedWords.mockResolvedValue({ rules: [] });
    endpoints.listMyChatRooms.mockResolvedValue({ rooms: [], total: 0 });
    endpoints.listPublicChatRooms.mockResolvedValue({ rooms: [], total: 0 });
    endpoints.resolveDMRoom.mockResolvedValue({ room: { id: "r-1" }, recipient: { id: "u-2" } });
});

describe("fetchRoomMessages", () => {
    it("passes the room and the page window straight through", async () => {
        // given
        endpoints.getRoomMessages.mockResolvedValue({ messages: [{ id: "m-1" }], total: 1 });

        // when
        const result = await fetchRoomMessages("room-1", 30, 60);

        // then
        expect(endpoints.getRoomMessages).toHaveBeenCalledWith("room-1", 30, 60);
        expect(result).toEqual({ messages: [{ id: "m-1" }], total: 1 });
    });

    it("leaves the page window unset when the caller gives none", async () => {
        // given
        await fetchRoomMessages("room-1");

        // when
        const call = endpoints.getRoomMessages.mock.calls[0];

        // then
        expect(call).toEqual(["room-1", undefined, undefined]);
    });
});

describe("fetchRoomMessagesBefore", () => {
    it("passes the room, the cursor and the limit straight through", async () => {
        // given
        endpoints.getRoomMessagesBefore.mockResolvedValue({ messages: [{ id: "m-0" }] });

        // when
        const result = await fetchRoomMessagesBefore("room-1", "2026-01-01T00:00:00Z", 10);

        // then
        expect(endpoints.getRoomMessagesBefore).toHaveBeenCalledWith("room-1", "2026-01-01T00:00:00Z", 10);
        expect(result).toEqual({ messages: [{ id: "m-0" }] });
    });
});

describe("fetchResolveDMRoom", () => {
    it("resolves the direct message room for a recipient through the shared cache", async () => {
        // given
        const resolved = await fetchResolveDMRoom("u-2");

        // when
        const cached = sharedQueryClient.getQueryData(["chat", "dm-resolve", "u-2"]);

        // then
        expect(endpoints.resolveDMRoom).toHaveBeenCalledWith("u-2");
        expect(resolved).toEqual({ room: { id: "r-1" }, recipient: { id: "u-2" } });
        expect(cached).toEqual(resolved);
    });

    it("serves a repeat resolve for the same recipient from the cache", async () => {
        // given
        await fetchResolveDMRoom("u-2");

        // when
        await fetchResolveDMRoom("u-2");

        // then
        expect(endpoints.resolveDMRoom).toHaveBeenCalledTimes(1);
    });

    it("resolves each recipient under its own cache entry", async () => {
        // given
        await fetchResolveDMRoom("u-2");

        // when
        await fetchResolveDMRoom("u-3");

        // then
        expect(endpoints.resolveDMRoom).toHaveBeenCalledTimes(2);
        expect(endpoints.resolveDMRoom).toHaveBeenLastCalledWith("u-3");
    });
});

describe("useUserRooms", () => {
    it("exposes the rooms of the signed in user under the user rooms key", async () => {
        // given
        endpoints.getUserRooms.mockResolvedValue({ rooms: [{ id: "room-1" }] });

        // when
        const { result, queryClient } = setup(() => useUserRooms());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(firstKey(queryClient)).toEqual(["chat", "rooms", "user"]);
        expect(result.current.rooms).toEqual([{ id: "room-1" }]);
    });

    it("reports no rooms while the request is in flight", async () => {
        // given
        const { result } = setup(() => useUserRooms());

        // when
        const initial = result.current;

        // then
        expect(initial.loading).toBe(true);
        expect(initial.rooms).toEqual([]);
        await waitFor(() => expect(result.current.loading).toBe(false));
    });

    it("stays idle while it is switched off", () => {
        // given
        const { result } = setup(() => useUserRooms(false));

        // when
        const current = result.current;

        // then
        expect(endpoints.getUserRooms).not.toHaveBeenCalled();
        expect(current.rooms).toEqual([]);
        expect(current.loading).toBe(false);
    });
});

describe("useDmRooms", () => {
    it("keeps the direct messages apart from the rest of the rooms under the user rooms key", async () => {
        // given
        endpoints.getUserRooms.mockResolvedValue({
            rooms: [dmRoom, groupRoom],
        });

        // when
        const { result, queryClient } = setup(() => useDmRooms());

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(firstKey(queryClient)).toEqual(["chat", "rooms", "user"]);
        expect(result.current.rooms).toEqual([dmRoom]);
        expect(result.current.allRooms).toEqual([dmRoom, groupRoom]);
    });

    it("reports no conversations while the request is in flight", async () => {
        // given
        const { result } = setup(() => useDmRooms());

        // when
        const initial = result.current;

        // then
        expect(initial.loading).toBe(true);
        expect(initial.rooms).toEqual([]);
        expect(initial.allRooms).toEqual([]);
        await waitFor(() => expect(result.current.loading).toBe(false));
    });

    it("stays idle while it is switched off", () => {
        // given
        const { result } = setup(() => useDmRooms(false));

        // when
        const current = result.current;

        // then
        expect(endpoints.getUserRooms).not.toHaveBeenCalled();
        expect(current.rooms).toEqual([]);
        expect(current.loading).toBe(false);
    });

    it("shares one request with the full room list rather than asking twice", async () => {
        // given
        endpoints.getUserRooms.mockResolvedValue({ rooms: [dmRoom, groupRoom] });

        // when
        const { result } = setup(() => ({ dms: useDmRooms(), all: useUserRooms() }));

        // then
        await waitFor(() => expect(result.current.dms.loading).toBe(false));
        expect(endpoints.getUserRooms).toHaveBeenCalledTimes(1);
        expect(result.current.dms.rooms).toEqual([dmRoom]);
        expect(result.current.all.rooms).toEqual([dmRoom, groupRoom]);
    });

    it("hands back the same conversations across a re-render, so readers can memoise them", async () => {
        // given
        endpoints.getUserRooms.mockResolvedValue({ rooms: [dmRoom, groupRoom] });
        const { result, rerender } = setup(() => useDmRooms());
        await waitFor(() => expect(result.current.loading).toBe(false));
        const first = result.current.rooms;

        // when
        rerender();

        // then
        expect(result.current.rooms).toBe(first);
    });
});

describe("useUserRoomsCache", () => {
    function setupRoster() {
        const queryClient = createTestQueryClient();
        const rendered = renderHook(() => ({ roster: useDmRooms(), cache: useUserRoomsCache() }), {
            wrapper: providerWrapper({ queryClient, user: makeUser() }),
        });

        return { ...rendered, queryClient };
    }

    it("rewrites the rooms every reader sees without asking the server again", async () => {
        // given
        endpoints.getUserRooms.mockResolvedValue({ rooms: [dmRoom, groupRoom] });
        const { result } = setupRoster();
        await waitFor(() => expect(result.current.roster.loading).toBe(false));

        // when
        act(() => {
            result.current.cache.patchRooms(rooms => rooms.map(room => ({ ...room, unread: true })));
        });
        await settle();

        // then
        expect(result.current.roster.rooms).toEqual([{ ...dmRoom, unread: true }]);
        expect(result.current.roster.allRooms).toHaveLength(2);
        expect(endpoints.getUserRooms).toHaveBeenCalledTimes(1);
    });

    it("leaves the cache alone while the rooms have not arrived", () => {
        // given
        endpoints.getUserRooms.mockReturnValue(new Promise(() => {}));
        const { result, queryClient } = setupRoster();

        // when
        act(() => {
            result.current.cache.patchRooms(() => [dmRoom]);
        });

        // then
        expect(queryClient.getQueryData(queryKeys.chat.userRooms())).toBeUndefined();
        expect(result.current.roster.rooms).toEqual([]);
    });

    it("asks the server for the rooms again when it is told to refresh", async () => {
        // given
        endpoints.getUserRooms.mockResolvedValue({ rooms: [dmRoom] });
        const { result } = setupRoster();
        await waitFor(() => expect(result.current.roster.loading).toBe(false));
        endpoints.getUserRooms.mockResolvedValue({ rooms: [dmRoom, { ...dmRoom, id: "room-9" }] });

        // when
        act(() => {
            result.current.cache.refreshRooms();
        });

        // then
        await waitFor(() => expect(result.current.roster.rooms).toHaveLength(2));
        expect(endpoints.getUserRooms).toHaveBeenCalledTimes(2);
    });
});

describe("useHostedRooms", () => {
    it("asks for the rooms the member hosts under the hosted rooms list key", async () => {
        // given
        endpoints.listMyChatRooms.mockResolvedValue({ rooms: [{ id: "room-1" }], total: 1 });

        // when
        const { result, queryClient } = setup(() => useHostedRooms(unfiltered));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.listMyChatRooms).toHaveBeenCalledWith({
            role: "host",
            search: "",
            rp: false,
            tag: undefined,
            includeArchived: false,
            limit: 20,
            offset: 0,
        });
        expect(firstKey(queryClient)).toEqual(["chat", "rooms-list", "hosted", "", false, "", false, 1]);
        expect(result.current.rooms).toEqual([{ id: "room-1" }]);
        expect(result.current.total).toBe(1);
    });

    it("widens the page window as further pages are asked for", async () => {
        // given
        const { result } = setup(() => useHostedRooms({ ...unfiltered, pages: 3 }));

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(endpoints.listMyChatRooms).toHaveBeenCalledWith(expect.objectContaining({ limit: 60 }));
    });

    it("carries the filters into the key and the request together", async () => {
        // given
        const filtered: RoomsListParams = {
            search: "tea",
            rpOnly: true,
            tagFilter: "horror",
            includeArchived: true,
            pages: 1,
        };

        // when
        const { result, queryClient } = setup(() => useHostedRooms(filtered));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.listMyChatRooms).toHaveBeenCalledWith(
            expect.objectContaining({ search: "tea", rp: true, tag: "horror", includeArchived: true }),
        );
        expect(firstKey(queryClient)).toEqual(["chat", "rooms-list", "hosted", "tea", true, "horror", true, 1]);
    });

    it("stays idle while it is switched off", () => {
        // given
        const { result } = setup(() => useHostedRooms(unfiltered, false));

        // when
        const current = result.current;

        // then
        expect(endpoints.listMyChatRooms).not.toHaveBeenCalled();
        expect(current.rooms).toEqual([]);
        expect(current.total).toBe(0);
    });
});

describe("useJoinedRooms", () => {
    it("asks for the rooms the member only belongs to under the joined rooms list key", async () => {
        // given
        endpoints.listMyChatRooms.mockResolvedValue({ rooms: [{ id: "room-2" }], total: 1 });

        // when
        const { result, queryClient } = setup(() => useJoinedRooms(unfiltered));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.listMyChatRooms).toHaveBeenCalledWith(expect.objectContaining({ role: "member" }));
        expect(firstKey(queryClient)).toEqual(["chat", "rooms-list", "joined", "", false, "", false, 1]);
        expect(result.current.rooms).toEqual([{ id: "room-2" }]);
    });

    it("stays idle while it is switched off", () => {
        // given
        const { result } = setup(() => useJoinedRooms(unfiltered, false));

        // when
        const current = result.current;

        // then
        expect(endpoints.listMyChatRooms).not.toHaveBeenCalled();
        expect(current.rooms).toEqual([]);
    });
});

describe("usePublicRooms", () => {
    it("asks for the public rooms under the discover rooms list key", async () => {
        // given
        endpoints.listPublicChatRooms.mockResolvedValue({ rooms: [{ id: "room-3" }], total: 7 });

        // when
        const { result, queryClient } = setup(() => usePublicRooms(unfiltered));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.listPublicChatRooms).toHaveBeenCalledWith({
            search: "",
            rp: false,
            tag: undefined,
            includeArchived: false,
            limit: 20,
            offset: 0,
        });
        expect(firstKey(queryClient)).toEqual(["chat", "rooms-list", "discover", "", false, "", false, 1]);
        expect(result.current.total).toBe(7);
    });

    it("asks for public rooms even without a signed in member", async () => {
        // given
        const { result } = setup(() => usePublicRooms(unfiltered), null);

        // when
        await waitFor(() => expect(result.current.loading).toBe(false));

        // then
        expect(endpoints.listPublicChatRooms).toHaveBeenCalledTimes(1);
    });
});

describe("useChatRoomMembers", () => {
    it("loads the members of a room under the room members key", async () => {
        // given
        endpoints.getChatRoomMembers.mockResolvedValue({ members: [{ user_id: "u-1" }] });

        // when
        const { result, queryClient } = setup(() => useChatRoomMembers("room-1"));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.getChatRoomMembers).toHaveBeenCalledWith("room-1");
        expect(firstKey(queryClient)).toEqual(["chat", "room", "room-1", "members"]);
        expect(result.current.members).toEqual([{ user_id: "u-1" }]);
    });

    it("stays idle while it is switched off", () => {
        // given
        const { result } = setup(() => useChatRoomMembers("room-1", false));

        // when
        const current = result.current;

        // then
        expect(endpoints.getChatRoomMembers).not.toHaveBeenCalled();
        expect(current.members).toEqual([]);
        expect(current.loading).toBe(false);
    });

    it("stays idle when no room has been chosen", () => {
        // given
        const { result } = setup(() => useChatRoomMembers(""));

        // when
        const current = result.current;

        // then
        expect(endpoints.getChatRoomMembers).not.toHaveBeenCalled();
        expect(current.members).toEqual([]);
    });
});

describe("useChatUnreadCount", () => {
    it("counts unread messages for a signed in member", async () => {
        // given
        endpoints.getChatUnreadCount.mockResolvedValue({ count: 4 });

        // when
        const { result, queryClient } = setup(() => useChatUnreadCount(), makeUser());

        // then
        await waitFor(() => expect(result.current.count).toBe(4));
        expect(firstKey(queryClient)).toEqual(["chat", "unread-count"]);
    });

    it("asks for nothing and reports zero for a signed out visitor", () => {
        // given
        const { result } = setup(() => useChatUnreadCount(), null);

        // when
        const current = result.current;

        // then
        expect(endpoints.getChatUnreadCount).not.toHaveBeenCalled();
        expect(current.count).toBe(0);
    });

    it("reports zero while the count is still being fetched", async () => {
        // given
        endpoints.getChatUnreadCount.mockResolvedValue({ count: 4 });

        // when
        const { result } = setup(() => useChatUnreadCount(), makeUser());

        // then
        expect(result.current.count).toBe(0);
        await waitFor(() => expect(result.current.count).toBe(4));
    });
});

describe("useChatRoomBans", () => {
    it("loads the bans of a room under the room bans key", async () => {
        // given
        endpoints.listChatRoomBans.mockResolvedValue({ bans: [{ user_id: "u-9" }] });

        // when
        const { result, queryClient } = setup(() => useChatRoomBans("room-1"));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.listChatRoomBans).toHaveBeenCalledWith("room-1");
        expect(firstKey(queryClient)).toEqual(["chat", "rooms", "room-1", "bans"]);
        expect(result.current.bans).toEqual([{ user_id: "u-9" }]);
    });

    it("stays idle while it is switched off", () => {
        // given
        const { result } = setup(() => useChatRoomBans("room-1", false));

        // when
        const current = result.current;

        // then
        expect(endpoints.listChatRoomBans).not.toHaveBeenCalled();
        expect(current.bans).toEqual([]);
    });

    it("stays idle when no room has been chosen", () => {
        // given
        setup(() => useChatRoomBans(""));

        // when
        const calls = endpoints.listChatRoomBans.mock.calls;

        // then
        expect(calls).toHaveLength(0);
    });
});

describe("useChatRoomBannedWords", () => {
    it("loads the banned word rules of a room under the room banned words key", async () => {
        // given
        endpoints.listChatRoomBannedWords.mockResolvedValue({ rules: [{ id: "w-1" }] });

        // when
        const { result, queryClient } = setup(() => useChatRoomBannedWords("room-1"));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.listChatRoomBannedWords).toHaveBeenCalledWith("room-1");
        expect(firstKey(queryClient)).toEqual(["chat", "rooms", "room-1", "banned-words"]);
        expect(result.current.rules).toEqual([{ id: "w-1" }]);
    });

    it("stays idle while it is switched off", () => {
        // given
        const { result } = setup(() => useChatRoomBannedWords("room-1", false));

        // when
        const current = result.current;

        // then
        expect(endpoints.listChatRoomBannedWords).not.toHaveBeenCalled();
        expect(current.rules).toEqual([]);
    });
});

describe("useChatRoomPinnedMessages", () => {
    it("loads the pinned messages of a room under the pinned key", async () => {
        // given
        endpoints.getChatRoomPinnedMessages.mockResolvedValue({ messages: [{ id: "m-1" }] });

        // when
        const { result, queryClient } = setup(() => useChatRoomPinnedMessages("room-1"));

        // then
        await waitFor(() => expect(result.current.loading).toBe(false));
        expect(endpoints.getChatRoomPinnedMessages).toHaveBeenCalledWith("room-1");
        expect(firstKey(queryClient)).toEqual(["chat", "room", "room-1", "pinned"]);
        expect(result.current.messages).toEqual([{ id: "m-1" }]);
    });

    it("stays idle while it is switched off", () => {
        // given
        const { result } = setup(() => useChatRoomPinnedMessages("room-1", false));

        // when
        const current = result.current;

        // then
        expect(endpoints.getChatRoomPinnedMessages).not.toHaveBeenCalled();
        expect(current.messages).toEqual([]);
    });

    it("stays idle when no room has been chosen", () => {
        // given
        const { result } = setup(() => useChatRoomPinnedMessages(""));

        // when
        const current = result.current;

        // then
        expect(endpoints.getChatRoomPinnedMessages).not.toHaveBeenCalled();
        expect(current.messages).toEqual([]);
    });
});
