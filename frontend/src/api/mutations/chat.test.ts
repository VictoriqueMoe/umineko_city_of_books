import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { createTestQueryClient, providerWrapper } from "../../test-utils/render";
import type { CreateBannedWordRequest } from "../../types/api";
import {
    useAddChatMessageReaction,
    useBanChatRoomMember,
    useClearChatRoomAvatar,
    useClearChatRoomMemberTimeout,
    useCreateChatRoomBannedWord,
    useCreateGroupRoom,
    useDeleteChatMessage,
    useDeleteChatRoom,
    useDeleteChatRoomBannedWord,
    useEditChatMessage,
    useInviteChatRoomMembers,
    useJoinChatRoom,
    useKickChatRoomMember,
    useLeaveChatRoom,
    useMarkChatRoomRead,
    usePinChatMessage,
    useRemoveChatMessageReaction,
    useSendChatMessage,
    useSendFirstDMMessage,
    useSetChatRoomMemberNickname,
    useSetChatRoomMemberTimeout,
    useSetChatRoomMuted,
    useUnbanChatRoomMember,
    useUnlockChatRoomMemberNickname,
    useUnpinChatMessage,
    useUpdateChatRoomBannedWord,
    useUpdateChatRoomNickname,
    useUploadChatRoomAvatar,
} from "./chat";

const mocks = vi.hoisted(() => ({
    addChatMessageReaction: vi.fn(),
    banChatRoomMember: vi.fn(),
    clearChatRoomAvatar: vi.fn(),
    clearChatRoomMemberTimeout: vi.fn(),
    createChatRoomBannedWord: vi.fn(),
    createGroupRoom: vi.fn(),
    deleteChatMessage: vi.fn(),
    deleteChatRoom: vi.fn(),
    deleteChatRoomBannedWord: vi.fn(),
    editChatMessage: vi.fn(),
    inviteChatRoomMembers: vi.fn(),
    joinChatRoom: vi.fn(),
    kickChatRoomMember: vi.fn(),
    leaveChatRoom: vi.fn(),
    markChatRoomRead: vi.fn(),
    pinChatMessage: vi.fn(),
    removeChatMessageReaction: vi.fn(),
    sendChatMessage: vi.fn(),
    sendFirstDMMessage: vi.fn(),
    setChatRoomMemberNickname: vi.fn(),
    setChatRoomMemberTimeout: vi.fn(),
    setChatRoomMuted: vi.fn(),
    unbanChatRoomMember: vi.fn(),
    unlockChatRoomMemberNickname: vi.fn(),
    unpinChatMessage: vi.fn(),
    updateChatRoomBannedWord: vi.fn(),
    updateChatRoomNickname: vi.fn(),
    uploadChatRoomAvatar: vi.fn(),
}));

vi.mock("../endpoints", () => mocks);

const roomsKey = ["chat", "rooms"];
const roomId = "11111111-1111-1111-1111-111111111111";
const userId = "22222222-2222-2222-2222-222222222222";
const messageId = "33333333-3333-3333-3333-333333333333";
const ruleId = "44444444-4444-4444-4444-444444444444";
const roomKey = ["chat", "rooms", roomId];
const membersKey = ["chat", "room", roomId, "members"];
const bannedWordsKey = ["chat", "rooms", roomId, "banned-words"];
const pinnedKey = ["chat", "room", roomId, "pinned"];

const bannedWord: CreateBannedWordRequest = {
    pattern: "goat",
    match_mode: "substring",
    case_sensitive: false,
    action: "delete",
};

function harness() {
    const queryClient = createTestQueryClient();
    const invalidateQueries = vi.spyOn(queryClient, "invalidateQueries");

    return { invalidateQueries, queryClient, wrapper: providerWrapper({ queryClient }) };
}

function makeFile(name = "avatar.png") {
    return new File(["gold"], name, { type: "image/png" });
}

beforeEach(() => {
    for (const fn of Object.values(mocks)) {
        fn.mockResolvedValue(undefined);
    }
});

describe("useCreateGroupRoom", () => {
    const payload = {
        name: "the rokkenjima parlour",
        description: "for tea and murder",
        is_public: true,
        is_rp: false,
        tags: ["umineko"],
        member_ids: [userId],
    };

    it("sends the whole room payload to the api", async () => {
        // given
        const { wrapper } = harness();
        mocks.createGroupRoom.mockResolvedValue({ id: roomId });
        const { result } = renderHook(() => useCreateGroupRoom(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(payload);
        });

        // then
        expect(mocks.createGroupRoom).toHaveBeenCalledWith(payload);
    });

    it("refreshes the room lists once the room exists", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.createGroupRoom.mockResolvedValue({ id: roomId });
        const { result } = renderHook(() => useCreateGroupRoom(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(payload);
        });

        // then
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: roomsKey });
    });

    it("leaves the room lists alone when the room could not be created", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.createGroupRoom.mockRejectedValue(new Error("name is taken"));
        const { result } = renderHook(() => useCreateGroupRoom(), { wrapper });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync(payload)).rejects.toThrow("name is taken");
        });

        // then
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});

describe("useJoinChatRoom", () => {
    it("joins the room as a visible member by default", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useJoinChatRoom(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ roomId });
        });

        // then
        expect(mocks.joinChatRoom).toHaveBeenCalledWith(roomId, { ghost: undefined });
    });

    it("passes the ghost flag through when the joiner wants to lurk", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useJoinChatRoom(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ roomId, ghost: true });
        });

        // then
        expect(mocks.joinChatRoom).toHaveBeenCalledWith(roomId, { ghost: true });
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: roomsKey });
    });
});

describe("useLeaveChatRoom", () => {
    it("leaves the room it was handed and refreshes the room lists", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useLeaveChatRoom(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(roomId);
        });

        // then
        expect(mocks.leaveChatRoom).toHaveBeenCalledWith(roomId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: roomsKey });
    });
});

describe("useDeleteChatRoom", () => {
    it("deletes the room it was handed and refreshes the room lists", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useDeleteChatRoom(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(roomId);
        });

        // then
        expect(mocks.deleteChatRoom).toHaveBeenCalledWith(roomId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: roomsKey });
    });
});

describe("useSetChatRoomMuted", () => {
    it("mutes the room and refreshes the room lists", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.setChatRoomMuted.mockResolvedValue({ muted: true });
        const { result } = renderHook(() => useSetChatRoomMuted(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ roomId, muted: true });
        });

        // then
        expect(mocks.setChatRoomMuted).toHaveBeenCalledWith(roomId, true);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: roomsKey });
    });

    it("unmutes the room when it is asked to", async () => {
        // given
        const { wrapper } = harness();
        mocks.setChatRoomMuted.mockResolvedValue({ muted: false });
        const { result } = renderHook(() => useSetChatRoomMuted(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ roomId, muted: false });
        });

        // then
        expect(mocks.setChatRoomMuted).toHaveBeenCalledWith(roomId, false);
    });
});

describe("useKickChatRoomMember", () => {
    it("kicks the member from the room the hook was built for", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useKickChatRoomMember(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(userId);
        });

        // then
        expect(mocks.kickChatRoomMember).toHaveBeenCalledWith(roomId, userId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: roomKey });
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: membersKey });
    });

    it("marks the cached member list stale so the kicked member disappears", async () => {
        // given
        const { wrapper, queryClient } = harness();
        queryClient.setQueryDefaults(["chat", "room"], { gcTime: Infinity });
        queryClient.setQueryData(membersKey, { members: [{ user_id: userId }] });
        const { result } = renderHook(() => useKickChatRoomMember(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(userId);
        });

        // then
        expect(queryClient.getQueryState(membersKey)?.isInvalidated).toBe(true);
    });
});

describe("useBanChatRoomMember", () => {
    it("sends the reason along with the member being banned", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useBanChatRoomMember(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ userId, reason: "spoiling the endgame" });
        });

        // then
        expect(mocks.banChatRoomMember).toHaveBeenCalledWith(roomId, userId, "spoiling the endgame");
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: roomKey });
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: membersKey });
    });

    it("leaves the room caches alone when the ban is refused", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.banChatRoomMember.mockRejectedValue(new Error("cannot ban the host"));
        const { result } = renderHook(() => useBanChatRoomMember(roomId), { wrapper });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync({ userId, reason: "none" })).rejects.toThrow("cannot ban the host");
        });

        // then
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});

describe("useUnbanChatRoomMember", () => {
    it("lifts the ban on the member it was handed", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useUnbanChatRoomMember(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(userId);
        });

        // then
        expect(mocks.unbanChatRoomMember).toHaveBeenCalledWith(roomId, userId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: roomKey });
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: membersKey });
    });
});

describe("useCreateChatRoomBannedWord", () => {
    it("creates the rule against the room and refreshes only the banned words", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.createChatRoomBannedWord.mockResolvedValue({ id: ruleId });
        const { result } = renderHook(() => useCreateChatRoomBannedWord(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(bannedWord);
        });

        // then
        expect(mocks.createChatRoomBannedWord).toHaveBeenCalledWith(roomId, bannedWord);
        expect(invalidateQueries).toHaveBeenCalledExactlyOnceWith({ queryKey: bannedWordsKey });
    });
});

describe("useUpdateChatRoomBannedWord", () => {
    it("sends the rule id and the replacement rule", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const req: CreateBannedWordRequest = { ...bannedWord, action: "kick" };
        const { result } = renderHook(() => useUpdateChatRoomBannedWord(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ ruleId, req });
        });

        // then
        expect(mocks.updateChatRoomBannedWord).toHaveBeenCalledWith(roomId, ruleId, req);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: bannedWordsKey });
    });
});

describe("useDeleteChatRoomBannedWord", () => {
    it("deletes the rule it was handed and refreshes the banned words", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useDeleteChatRoomBannedWord(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(ruleId);
        });

        // then
        expect(mocks.deleteChatRoomBannedWord).toHaveBeenCalledWith(roomId, ruleId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: bannedWordsKey });
    });
});

describe("useInviteChatRoomMembers", () => {
    it("invites the whole batch of users in one call", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        mocks.inviteChatRoomMembers.mockResolvedValue({ invited_count: 2, skipped_count: 1 });
        const { result } = renderHook(() => useInviteChatRoomMembers(roomId), { wrapper });

        // when
        let invited: { invited_count: number; skipped_count: number } | undefined;
        await act(async () => {
            invited = await result.current.mutateAsync([userId, "battler"]);
        });

        // then
        expect(mocks.inviteChatRoomMembers).toHaveBeenCalledWith(roomId, [userId, "battler"]);
        expect(invited).toEqual({ invited_count: 2, skipped_count: 1 });
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: roomKey });
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: membersKey });
    });

    it("marks the cached member list stale so the invited members appear", async () => {
        // given
        const { wrapper, queryClient } = harness();
        queryClient.setQueryDefaults(["chat", "room"], { gcTime: Infinity });
        queryClient.setQueryData(membersKey, { members: [] });
        const { result } = renderHook(() => useInviteChatRoomMembers(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync([userId]);
        });

        // then
        expect(queryClient.getQueryState(membersKey)?.isInvalidated).toBe(true);
    });
});

describe("useSendChatMessage", () => {
    it("sends the message payload to the room the hook was built for", async () => {
        // given
        const { wrapper } = harness();
        mocks.sendChatMessage.mockResolvedValue({ id: messageId });
        const { result } = renderHook(() => useSendChatMessage(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ body: "hello rokkenjima" });
        });

        // then
        expect(mocks.sendChatMessage).toHaveBeenCalledWith(roomId, { body: "hello rokkenjima" });
    });

    it("carries the reply target and the attachments through untouched", async () => {
        // given
        const { wrapper } = harness();
        const file = makeFile("catbox.png");
        const { result } = renderHook(() => useSendChatMessage(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ body: "a reply", reply_to_id: messageId, files: [file] });
        });

        // then
        expect(mocks.sendChatMessage).toHaveBeenCalledWith(roomId, {
            body: "a reply",
            reply_to_id: messageId,
            files: [file],
        });
    });

    it("invalidates nothing because new messages arrive over the socket", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useSendChatMessage(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ body: "hello" });
        });

        // then
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});

describe("useSendFirstDMMessage", () => {
    it("sends the recipient, the body and no files when none were picked", async () => {
        // given
        const { wrapper } = harness();
        mocks.sendFirstDMMessage.mockResolvedValue({ room: { id: roomId }, message: { id: messageId } });
        const { result } = renderHook(() => useSendFirstDMMessage(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ recipientId: userId, body: "are you there" });
        });

        // then
        expect(mocks.sendFirstDMMessage).toHaveBeenCalledWith(userId, "are you there", undefined);
    });

    it("returns the freshly created room together with the message", async () => {
        // given
        const { wrapper } = harness();
        const file = makeFile("gift.png");
        mocks.sendFirstDMMessage.mockResolvedValue({ room: { id: roomId }, message: { id: messageId } });
        const { result } = renderHook(() => useSendFirstDMMessage(), { wrapper });

        // when
        let sent: { room: { id: string }; message: { id: string } } | undefined;
        await act(async () => {
            sent = await result.current.mutateAsync({ recipientId: userId, body: "a gift", files: [file] });
        });

        // then
        expect(mocks.sendFirstDMMessage).toHaveBeenCalledWith(userId, "a gift", [file]);
        expect(sent).toEqual({ room: { id: roomId }, message: { id: messageId } });
    });
});

describe("useMarkChatRoomRead", () => {
    it("marks the room it was handed as read without touching the cache", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useMarkChatRoomRead(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(roomId);
        });

        // then
        expect(mocks.markChatRoomRead).toHaveBeenCalledWith(roomId);
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});

describe("useUpdateChatRoomNickname", () => {
    it("sends the nickname the member chose for themselves", async () => {
        // given
        const { wrapper } = harness();
        mocks.updateChatRoomNickname.mockResolvedValue({ user_id: userId });
        const { result } = renderHook(() => useUpdateChatRoomNickname(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync("the golden witch");
        });

        // then
        expect(mocks.updateChatRoomNickname).toHaveBeenCalledWith(roomId, "the golden witch");
    });
});

describe("useSetChatRoomMemberNickname", () => {
    it("sends the member and the nickname a moderator forced on them", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useSetChatRoomMemberNickname(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ userId, nickname: "goat" });
        });

        // then
        expect(mocks.setChatRoomMemberNickname).toHaveBeenCalledWith(roomId, userId, "goat");
    });
});

describe("useUnlockChatRoomMemberNickname", () => {
    it("unlocks the nickname of the member it was handed", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useUnlockChatRoomMemberNickname(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(userId);
        });

        // then
        expect(mocks.unlockChatRoomMemberNickname).toHaveBeenCalledWith(roomId, userId);
    });
});

describe("useSetChatRoomMemberTimeout", () => {
    it("spreads the amount and the unit into positional arguments", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useSetChatRoomMemberTimeout(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ userId, amount: 10, unit: "minutes" });
        });

        // then
        expect(mocks.setChatRoomMemberTimeout).toHaveBeenCalledWith(roomId, userId, 10, "minutes");
    });
});

describe("useClearChatRoomMemberTimeout", () => {
    it("clears the timeout of the member it was handed", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useClearChatRoomMemberTimeout(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(userId);
        });

        // then
        expect(mocks.clearChatRoomMemberTimeout).toHaveBeenCalledWith(roomId, userId);
    });
});

describe("useUploadChatRoomAvatar", () => {
    it("uploads the chosen file against the room the hook was built for", async () => {
        // given
        const { wrapper } = harness();
        const file = makeFile();
        const { result } = renderHook(() => useUploadChatRoomAvatar(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(file);
        });

        // then
        expect(mocks.uploadChatRoomAvatar).toHaveBeenCalledWith(roomId, file);
    });
});

describe("useClearChatRoomAvatar", () => {
    it("clears the avatar with no arguments beyond the room", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useClearChatRoomAvatar(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync();
        });

        // then
        expect(mocks.clearChatRoomAvatar).toHaveBeenCalledWith(roomId);
    });
});

describe("useDeleteChatMessage", () => {
    it("deletes the message it was handed", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useDeleteChatMessage(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(messageId);
        });

        // then
        expect(mocks.deleteChatMessage).toHaveBeenCalledWith(messageId);
    });

    it("surfaces the failure to the caller", async () => {
        // given
        const { wrapper } = harness();
        mocks.deleteChatMessage.mockRejectedValue(new Error("too old to delete"));
        const { result } = renderHook(() => useDeleteChatMessage(), { wrapper });

        // when
        await act(async () => {
            await expect(result.current.mutateAsync(messageId)).rejects.toThrow("too old to delete");
        });

        // then
        await waitFor(() => {
            expect(result.current.isError).toBe(true);
        });
        expect(result.current.error).toEqual(new Error("too old to delete"));
    });
});

describe("useEditChatMessage", () => {
    it("sends the message id and the new body", async () => {
        // given
        const { wrapper } = harness();
        mocks.editChatMessage.mockResolvedValue({ id: messageId, body: "edited" });
        const { result } = renderHook(() => useEditChatMessage(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ messageId, body: "edited" });
        });

        // then
        expect(mocks.editChatMessage).toHaveBeenCalledWith(messageId, "edited");
    });
});

describe("usePinChatMessage", () => {
    it("pins the message and refreshes the pinned panel of the room it knows about", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => usePinChatMessage(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(messageId);
        });

        // then
        expect(mocks.pinChatMessage).toHaveBeenCalledWith(messageId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: pinnedKey });
    });

    it("pins the message without touching the cache when no room was given", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => usePinChatMessage(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(messageId);
        });

        // then
        expect(mocks.pinChatMessage).toHaveBeenCalledWith(messageId);
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});

describe("useUnpinChatMessage", () => {
    it("unpins the message and refreshes the pinned panel of the room it knows about", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useUnpinChatMessage(roomId), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(messageId);
        });

        // then
        expect(mocks.unpinChatMessage).toHaveBeenCalledWith(messageId);
        expect(invalidateQueries).toHaveBeenCalledWith({ queryKey: pinnedKey });
    });

    it("unpins the message without touching the cache when no room was given", async () => {
        // given
        const { wrapper, invalidateQueries } = harness();
        const { result } = renderHook(() => useUnpinChatMessage(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync(messageId);
        });

        // then
        expect(invalidateQueries).not.toHaveBeenCalled();
    });
});

describe("useAddChatMessageReaction", () => {
    it("sends the message and the emoji as separate arguments", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useAddChatMessageReaction(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ messageId, emoji: "🦋" });
        });

        // then
        expect(mocks.addChatMessageReaction).toHaveBeenCalledWith(messageId, "🦋");
    });
});

describe("useRemoveChatMessageReaction", () => {
    it("sends the message and the emoji being taken back", async () => {
        // given
        const { wrapper } = harness();
        const { result } = renderHook(() => useRemoveChatMessageReaction(), { wrapper });

        // when
        await act(async () => {
            await result.current.mutateAsync({ messageId, emoji: "🦋" });
        });

        // then
        expect(mocks.removeChatMessageReaction).toHaveBeenCalledWith(messageId, "🦋");
    });
});
