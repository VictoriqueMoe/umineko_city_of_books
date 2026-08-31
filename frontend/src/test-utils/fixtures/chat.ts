import type { ChatMessage, ChatRoom, ChatRoomMember } from "../../types/api";
import { makePublicUser } from "./user";

export function makeChatRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return {
        id: "room-1",
        name: "Tea Parlour",
        description: "",
        type: "group",
        is_public: true,
        is_rp: false,
        is_system: false,
        tags: [],
        viewer_muted: false,
        viewer_ghost: false,
        is_member: true,
        member_count: 2,
        hot_score: 0,
        members: [],
        created_at: "2026-01-01T00:00:00Z",
        ...overrides,
    };
}

export function makeDmRoom(overrides: Partial<ChatRoom> = {}): ChatRoom {
    return makeChatRoom({ name: "", type: "dm", is_public: false, ...overrides });
}

export function makeRoomMember(overrides: Partial<ChatRoomMember> = {}): ChatRoomMember {
    return {
        user: makePublicUser(),
        role: "member",
        joined_at: "2026-01-01T00:00:00Z",
        nickname: "",
        member_avatar_url: "",
        nickname_locked: false,
        ...overrides,
    };
}

export function makeChatMessage(overrides: Partial<ChatMessage> = {}): ChatMessage {
    return {
        id: "m1",
        room_id: "room-1",
        sender: makePublicUser(),
        body: "the golden truth",
        is_system: false,
        created_at: "2026-01-01T00:00:00Z",
        pinned: false,
        reactions: [],
        ...overrides,
    };
}
