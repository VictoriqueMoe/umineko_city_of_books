import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "./chat";
import {
    deleteMock,
    fetchMock,
    patchMock,
    postFormDataMock,
    postMock,
    putMock,
    resetTransports,
    runRequestCases,
    type RequestCase,
} from "./testHarness";

vi.mock("../../platform/capabilities", () => ({
    isNativeApp: () => false,
    clientPlatform: () => "web",
}));

vi.mock("../client", async importOriginal => {
    const actual = await importOriginal<typeof import("../client")>();
    return {
        ...actual,
        apiFetch: vi.fn(),
        apiFetchText: vi.fn(),
        apiPost: vi.fn(),
        apiPut: vi.fn(),
        apiPatch: vi.fn(),
        apiDelete: vi.fn(),
        apiDeleteWithBody: vi.fn(),
        apiPostFormData: vi.fn(),
    };
});

beforeEach(resetTransports);

describe("direct messages and group chat rooms", () => {
    const groupPayload = {
        name: "Rokkenjima",
        description: "the family conference",
        is_public: true,
        is_rp: false,
        tags: ["umineko"],
        member_ids: ["u-1", "u-2"],
    };

    const cases: RequestCase[] = [
        {
            name: "resolveDMRoom reads the conversation with a recipient",
            call: () => api.resolveDMRoom("u-1"),
            transport: fetchMock,
            request: ["/chat/dm/u-1/resolve"],
        },
        {
            name: "createGroupRoom posts the whole payload to the room collection",
            call: () => api.createGroupRoom(groupPayload),
            transport: postMock,
            request: ["/chat/rooms", groupPayload],
        },
        {
            name: "leaveChatRoom posts an empty body",
            call: () => api.leaveChatRoom("r-1"),
            transport: postMock,
            request: ["/chat/rooms/r-1/leave", {}],
        },
        {
            name: "setChatRoomMuted puts the muted flag",
            call: () => api.setChatRoomMuted("r-1", true),
            transport: putMock,
            request: ["/chat/rooms/r-1/mute", { muted: true }],
        },
        {
            name: "setChatRoomMuted can unmute the room again",
            call: () => api.setChatRoomMuted("r-1", false),
            transport: putMock,
            request: ["/chat/rooms/r-1/mute", { muted: false }],
        },
    ];

    runRequestCases(cases);
});

describe("direct message uploads", () => {
    it("sendFirstDMMessage attaches the body and every media file", async () => {
        // given
        const first = new File(["a"], "a.png", { type: "image/png" });
        const second = new File(["b"], "b.png", { type: "image/png" });

        // when
        await api.sendFirstDMMessage("u-1", "good evening", [first, second]);

        // then
        const formData = postFormDataMock.mock.calls[0][1];
        expect(postFormDataMock.mock.calls[0][0]).toBe("/chat/dm/u-1/messages");
        expect(formData.get("body")).toBe("good evening");
        expect(formData.getAll("media")).toEqual([first, second]);
    });

    it("sendFirstDMMessage omits the media when no files are given", async () => {
        // given
        const recipientId = "u-1";

        // when
        await api.sendFirstDMMessage(recipientId, "good evening");

        // then
        const formData = postFormDataMock.mock.calls[0][1];
        expect(formData.get("body")).toBe("good evening");
        expect(formData.has("media")).toBe(false);
    });
});

describe("chat room membership and bans", () => {
    const cases: RequestCase[] = [
        {
            name: "getChatRoomMembers reads the member list of a room",
            call: () => api.getChatRoomMembers("r-1"),
            transport: fetchMock,
            request: ["/chat/rooms/r-1/members"],
        },
        {
            name: "kickChatRoomMember deletes the named member",
            call: () => api.kickChatRoomMember("r-1", "u-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1/members/u-1"],
        },
        {
            name: "banChatRoomMember posts the reason to the ban collection",
            call: () => api.banChatRoomMember("r-1", "u-1", "spoiling the seventh twilight"),
            transport: postMock,
            request: ["/chat/rooms/r-1/bans/u-1", { reason: "spoiling the seventh twilight" }],
        },
        {
            name: "unbanChatRoomMember deletes the ban",
            call: () => api.unbanChatRoomMember("r-1", "u-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1/bans/u-1"],
        },
        {
            name: "listChatRoomBans reads the ban list of a room",
            call: () => api.listChatRoomBans("r-1"),
            transport: fetchMock,
            request: ["/chat/rooms/r-1/bans"],
        },
        {
            name: "inviteChatRoomMembers posts the ids under the snake cased field",
            call: () => api.inviteChatRoomMembers("r-1", ["u-1", "u-2"]),
            transport: postMock,
            request: ["/chat/rooms/r-1/members", { user_ids: ["u-1", "u-2"] }],
        },
        {
            name: "inviteChatRoomMembers still sends an empty list when nobody was chosen",
            call: () => api.inviteChatRoomMembers("r-1", []),
            transport: postMock,
            request: ["/chat/rooms/r-1/members", { user_ids: [] }],
        },
    ];

    runRequestCases(cases);
});

describe("chat room banned word rules", () => {
    const substringRule = {
        pattern: "goat",
        match_mode: "substring" as const,
        case_sensitive: false,
        action: "delete" as const,
    };
    const regexRule = {
        pattern: "^witch$",
        match_mode: "regex" as const,
        case_sensitive: true,
        action: "kick" as const,
    };

    const cases: RequestCase[] = [
        {
            name: "listChatRoomBannedWords reads the rules of a room",
            call: () => api.listChatRoomBannedWords("r-1"),
            transport: fetchMock,
            request: ["/chat/rooms/r-1/banned-words"],
        },
        {
            name: "createChatRoomBannedWord posts the whole rule to the room",
            call: () => api.createChatRoomBannedWord("r-1", substringRule),
            transport: postMock,
            request: ["/chat/rooms/r-1/banned-words", substringRule],
        },
        {
            name: "updateChatRoomBannedWord puts the rule back under its own id",
            call: () => api.updateChatRoomBannedWord("r-1", "rule-1", regexRule),
            transport: putMock,
            request: ["/chat/rooms/r-1/banned-words/rule-1", regexRule],
        },
        {
            name: "deleteChatRoomBannedWord deletes the rule from the room",
            call: () => api.deleteChatRoomBannedWord("r-1", "rule-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1/banned-words/rule-1"],
        },
    ];

    runRequestCases(cases);
});

describe("room voice channels", () => {
    const cases: RequestCase[] = [
        {
            name: "getVoiceToken posts an empty body to the room voice token",
            call: () => api.getVoiceToken("r-1"),
            transport: postMock,
            request: ["/chat/rooms/r-1/voice/token", {}],
        },
        {
            name: "forceMuteVoiceParticipant mutes the named participant",
            call: () => api.forceMuteVoiceParticipant("r-1", "u-1", true),
            transport: postMock,
            request: ["/chat/rooms/r-1/voice/participants/u-1/mute", { muted: true }],
        },
        {
            name: "forceMuteVoiceParticipant can unmute the participant again",
            call: () => api.forceMuteVoiceParticipant("r-1", "u-1", false),
            transport: postMock,
            request: ["/chat/rooms/r-1/voice/participants/u-1/mute", { muted: false }],
        },
    ];

    runRequestCases(cases);
});

describe("chat room reading and housekeeping", () => {
    const cases: RequestCase[] = [
        {
            name: "getUserRooms reads the caller's rooms",
            call: () => api.getUserRooms(),
            transport: fetchMock,
            request: ["/chat/rooms"],
        },
        {
            name: "getRoomMessages defaults to fifty messages and no offset",
            call: () => api.getRoomMessages("r-1"),
            transport: fetchMock,
            request: ["/chat/rooms/r-1/messages?limit=50"],
        },
        {
            name: "getRoomMessages pages through the history",
            call: () => api.getRoomMessages("r-1", 25, 50),
            transport: fetchMock,
            request: ["/chat/rooms/r-1/messages?limit=25&offset=50"],
        },
        {
            name: "deleteChatRoom deletes the room",
            call: () => api.deleteChatRoom("r-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1"],
        },
        {
            name: "getChatUnreadCount reads the unread counter",
            call: () => api.getChatUnreadCount(),
            transport: fetchMock,
            request: ["/chat/unread-count"],
        },
        {
            name: "markChatRoomRead posts an empty body to the read marker",
            call: () => api.markChatRoomRead("r-1"),
            transport: postMock,
            request: ["/chat/rooms/r-1/read", {}],
        },
        {
            name: "updateChatRoomNickname puts the caller's own nickname",
            call: () => api.updateChatRoomNickname("r-1", "Victorique"),
            transport: putMock,
            request: ["/chat/rooms/r-1/me", { nickname: "Victorique" }],
        },
    ];

    runRequestCases(cases);
});

describe("chat room member nicknames, timeouts and avatars", () => {
    const cases: RequestCase[] = [
        {
            name: "setChatRoomMemberNickname puts the nickname on the member",
            call: () => api.setChatRoomMemberNickname("r-1", "u-1", "Beatrice"),
            transport: putMock,
            request: ["/chat/rooms/r-1/members/u-1/nickname", { nickname: "Beatrice" }],
        },
        {
            name: "unlockChatRoomMemberNickname deletes the member's nickname lock",
            call: () => api.unlockChatRoomMemberNickname("r-1", "u-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1/members/u-1/nickname"],
        },
        {
            name: "clearChatRoomMemberTimeout deletes the member's timeout",
            call: () => api.clearChatRoomMemberTimeout("r-1", "u-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1/members/u-1/timeout"],
        },
        {
            name: "clearChatRoomAvatar deletes the caller's own room avatar",
            call: () => api.clearChatRoomAvatar("r-1"),
            transport: deleteMock,
            request: ["/chat/rooms/r-1/me/avatar"],
        },
    ];

    runRequestCases(cases);

    it("uploadChatRoomAvatar posts the file under the avatar field of the caller's membership", async () => {
        // given
        const file = new File(["x"], "room-avatar.png", { type: "image/png" });

        // when
        await api.uploadChatRoomAvatar("r-1", file);

        // then
        expect(postFormDataMock.mock.calls[0][0]).toBe("/chat/rooms/r-1/me/avatar");
        expect(postFormDataMock.mock.calls[0][1].get("avatar")).toBe(file);
    });
});

describe("chat message editing, pinning and reactions", () => {
    const cases: RequestCase[] = [
        {
            name: "deleteChatMessage deletes the message",
            call: () => api.deleteChatMessage("m-1"),
            transport: deleteMock,
            request: ["/chat/messages/m-1"],
        },
        {
            name: "editChatMessage patches only the body",
            call: () => api.editChatMessage("m-1", "without love it cannot be seen"),
            transport: patchMock,
            request: ["/chat/messages/m-1", { body: "without love it cannot be seen" }],
        },
        {
            name: "pinChatMessage posts an empty body to the pin",
            call: () => api.pinChatMessage("m-1"),
            transport: postMock,
            request: ["/chat/messages/m-1/pin", {}],
        },
        {
            name: "unpinChatMessage deletes the pin",
            call: () => api.unpinChatMessage("m-1"),
            transport: deleteMock,
            request: ["/chat/messages/m-1/pin"],
        },
        {
            name: "getChatRoomPinnedMessages reads the pins of a room",
            call: () => api.getChatRoomPinnedMessages("r-1"),
            transport: fetchMock,
            request: ["/chat/rooms/r-1/pins"],
        },
        {
            name: "addChatMessageReaction posts the emoji to the reaction collection",
            call: () => api.addChatMessageReaction("m-1", "🍬"),
            transport: postMock,
            request: ["/chat/messages/m-1/reactions", { emoji: "🍬" }],
        },
    ];

    runRequestCases(cases);
});
