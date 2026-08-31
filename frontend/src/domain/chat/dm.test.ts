import { describe, expect, it } from "vitest";
import { makeChatMessage, makeDmRoom, makePublicUser, makeUser } from "../../test-utils/fixtures";
import type { ChatMessage, ChatRoom, User } from "../../types/api";
import { getRoomAvatarUser, getRoomDisplayName, seenLabel } from "./dm";

const viewer = makeUser({ id: "u1", username: "beatrice", display_name: "Beatrice" });

function makeMember(overrides: Partial<User> = {}): User {
    return { id: "u2", username: "battler", display_name: "Battler", ...overrides };
}

function makeRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeDmRoom({ members: [makePublicUser(), makeMember()], ...overrides });
}

function makeMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return makeChatMessage({
        sender: makeMember(),
        body: "without love it cannot be seen",
        created_at: "2026-08-02T10:00:00Z",
        ...overrides,
    });
}

describe("getRoomDisplayName", () => {
    it("uses the name a group chat was given", () => {
        // given
        const room = makeRoom({ type: "group", name: "Rokkenjima" });

        // when
        const name = getRoomDisplayName(room, viewer);

        // then
        expect(name).toBe("Rokkenjima");
    });

    it("falls back to a generic label for an unnamed group chat", () => {
        // given
        const room = makeRoom({ type: "group", name: "" });

        // when
        const name = getRoomDisplayName(room, viewer);

        // then
        expect(name).toBe("Group Chat");
    });

    it("names a direct message after the other person", () => {
        // given
        const room = makeRoom();

        // when
        const name = getRoomDisplayName(room, viewer);

        // then
        expect(name).toBe("Battler");
    });

    it("falls back to a generic label when the viewer is the only member left", () => {
        // given
        const room = makeRoom({ members: [{ id: "u1", username: "beatrice", display_name: "Beatrice" }] });

        // when
        const name = getRoomDisplayName(room, viewer);

        // then
        expect(name).toBe("Direct Message");
    });
});

describe("getRoomAvatarUser", () => {
    it("picks the other person in a direct message", () => {
        // given
        const room = makeRoom();

        // when
        const avatarUser = getRoomAvatarUser(room, viewer);

        // then
        expect(avatarUser?.id).toBe("u2");
    });

    it("has nobody to show for a group chat", () => {
        // given
        const room = makeRoom({ type: "group", name: "Rokkenjima" });

        // when
        const avatarUser = getRoomAvatarUser(room, viewer);

        // then
        expect(avatarUser).toBeNull();
    });

    it("has nobody to show when the viewer is alone in the conversation", () => {
        // given
        const room = makeRoom({ members: [{ id: "u1", username: "beatrice", display_name: "Beatrice" }] });

        // when
        const avatarUser = getRoomAvatarUser(room, viewer);

        // then
        expect(avatarUser).toBeNull();
    });
});

describe("seenLabel", () => {
    const readAt = "2026-08-02T10:30:00Z";
    const time = new Date(readAt).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });

    it("says nothing when the room has not loaded", () => {
        // given
        const messages = [makeMessage()];

        // when
        const label = seenLabel(messages[0], 0, messages, undefined, "u1", {});

        // then
        expect(label).toBeNull();
    });

    it("only labels the newest message the viewer sent", () => {
        // given
        const messages = [
            makeMessage({ id: "m1", sender: { id: "u1", username: "beatrice", display_name: "Beatrice" } }),
            makeMessage({ id: "m2", sender: { id: "u1", username: "beatrice", display_name: "Beatrice" } }),
        ];
        const receipts = { "room-1": { u2: readAt } };

        // when
        const label = seenLabel(messages[0], 0, messages, makeRoom(), "u1", receipts);

        // then
        expect(label).toBeNull();
    });

    it("says nothing while nobody has read anything in the room", () => {
        // given
        const messages = [makeMessage()];

        // when
        const label = seenLabel(messages[0], 0, messages, makeRoom(), "u1", { "room-2": { u2: readAt } });

        // then
        expect(label).toBeNull();
    });

    it("ignores a receipt from before the message was sent", () => {
        // given
        const messages = [makeMessage({ created_at: "2026-08-02T11:00:00Z" })];
        const receipts = { "room-1": { u2: readAt } };

        // when
        const label = seenLabel(messages[0], 0, messages, makeRoom(), "u1", receipts);

        // then
        expect(label).toBeNull();
    });

    it("reports the time a direct message was read without naming the reader", () => {
        // given
        const messages = [makeMessage()];
        const receipts = { "room-1": { u2: readAt } };

        // when
        const label = seenLabel(messages[0], 0, messages, makeRoom(), "u1", receipts);

        // then
        expect(label).toBe(`seen ${time}`);
    });

    it("names the most recent reader in a group chat", () => {
        // given
        const room = makeRoom({
            type: "group",
            name: "Rokkenjima",
            members: [
                { id: "u1", username: "beatrice", display_name: "Beatrice" },
                makeMember(),
                makeMember({ id: "u3", username: "ange", display_name: "Ange" }),
            ],
        });
        const messages = [makeMessage()];
        const receipts = { "room-1": { u2: "2026-08-02T10:15:00Z", u3: readAt } };

        // when
        const label = seenLabel(messages[0], 0, messages, room, "u1", receipts);

        // then
        expect(label).toBe(`seen by Ange ${time}`);
    });

    it("never counts the viewer's own read receipt", () => {
        // given
        const messages = [makeMessage()];
        const receipts = { "room-1": { u1: readAt } };

        // when
        const label = seenLabel(messages[0], 0, messages, makeRoom(), "u1", receipts);

        // then
        expect(label).toBeNull();
    });
});
