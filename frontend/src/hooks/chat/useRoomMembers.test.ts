import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { emitRealtimeEvent } from "../../test-utils/ws";
import type { ChatRoomMember, User } from "../../types/api";
import { useRoomMembers, type RoomMembersOptions } from "./useRoomMembers";

const mocks = vi.hoisted(() => ({
    useChatRoomMembers: vi.fn(),
    membersRefresh: vi.fn(),
    memberCountDelta: vi.fn(),
}));

vi.mock("../queries/chat", () => ({ useChatRoomMembers: mocks.useChatRoomMembers }));

function makeRoomMember(overrides: Partial<ChatRoomMember> = {}): ChatRoomMember {
    return {
        user: { id: "u2", username: "battler", display_name: "Battler" },
        role: "member",
        joined_at: "2026-01-01T00:00:00Z",
        nickname: "",
        member_avatar_url: "",
        nickname_locked: false,
        ...overrides,
    };
}

interface RosterOptions {
    members?: ChatRoomMember[];
    viewerId?: string;
    voiceParticipantIds?: string[];
}

function renderRoster(options: RosterOptions = {}) {
    mocks.useChatRoomMembers.mockReturnValue({
        members: options.members ?? [makeRoomMember()],
        loading: false,
        refresh: mocks.membersRefresh,
    });

    const initialProps: RoomMembersOptions = {
        roomId: "room-1",
        enabled: true,
        viewerId: options.viewerId ?? "u1",
        voiceParticipantIds: options.voiceParticipantIds ?? [],
        onMemberCountDelta: mocks.memberCountDelta,
    };

    return renderHook((props: RoomMembersOptions) => useRoomMembers(props), { initialProps });
}

describe("useRoomMembers", () => {
    it("finds the viewer's own membership", () => {
        // given
        const options: RosterOptions = {
            members: [makeRoomMember({ user: { id: "u1", username: "beatrice", display_name: "Beatrice" } })],
        };

        // when
        const { result } = renderRoster(options);

        // then
        expect(result.current.currentMember?.user.id).toBe("u1");
    });

    it("reloads the member list when somebody new arrives", () => {
        // given
        renderRoster();

        // when
        emitRealtimeEvent({
            type: "chat_member_joined",
            data: { room_id: "room-1", user: { id: "u3", username: "ange", display_name: "Ange" } as User },
        });

        // then
        expect(mocks.membersRefresh).toHaveBeenCalled();
    });

    it("ignores somebody arriving in another room", () => {
        // given
        renderRoster();
        mocks.membersRefresh.mockClear();

        // when
        emitRealtimeEvent({
            type: "chat_member_joined",
            data: { room_id: "room-2", user: { id: "u3", username: "ange", display_name: "Ange" } as User },
        });

        // then
        expect(mocks.membersRefresh).not.toHaveBeenCalled();
    });

    it("drops somebody who left the room", () => {
        // given
        const { result } = renderRoster();

        // when
        emitRealtimeEvent({ type: "chat_member_left", data: { room_id: "room-1", user_id: "u2" } });

        // then
        expect(result.current.members).toEqual([]);
    });

    it("ignores somebody leaving another room", () => {
        // given
        const { result } = renderRoster();

        // when
        emitRealtimeEvent({ type: "chat_member_left", data: { room_id: "room-2", user_id: "u2" } });

        // then
        expect(result.current.members).toHaveLength(1);
    });

    it("records a presence change and forgets somebody who goes offline", () => {
        // given
        const { result } = renderRoster({ members: [] });
        emitRealtimeEvent({
            type: "chat_presence_changed",
            data: { room_id: "room-1", user_id: "u2", state: "active" },
        });
        const whilePresent = result.current.presenceMapMerged;

        // when
        emitRealtimeEvent({
            type: "chat_presence_changed",
            data: { room_id: "room-1", user_id: "u2", state: "offline" },
        });

        // then
        expect(whilePresent).toEqual({ u2: "active" });
        expect(result.current.presenceMapMerged).toEqual({});
    });
});
