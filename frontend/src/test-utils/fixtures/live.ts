import type { HomeActivityResponse, HomeMember, HomePublicRoom, LiveStream, WatchPartySession } from "../../types/api";

export function makeStream(overrides: Partial<LiveStream> = {}): LiveStream {
    return {
        id: "stream-1",
        userId: "user-1",
        title: "Reading the epitaph",
        status: "live",
        viewerCount: 3,
        streamerUsername: "beato",
        streamerDisplayName: "Beatrice",
        streamerAvatarUrl: "",
        defaultMode: "webrtc",
        ...overrides,
    };
}

export function makeHomeMember(overrides: Partial<HomeMember> = {}): HomeMember {
    return {
        id: "member-1",
        username: "battler",
        display_name: "Battler",
        avatar_url: "",
        created_at: "2026-02-01T12:00:00Z",
        ...overrides,
    };
}

export function makeHomePublicRoom(overrides: Partial<HomePublicRoom> = {}): HomePublicRoom {
    return {
        id: "room-1",
        name: "Tea Parlour",
        description: "Somewhere to sit",
        member_count: 3,
        last_message_at: null,
        ...overrides,
    };
}

export function makeWatchPartySession(overrides: Partial<WatchPartySession> = {}): WatchPartySession {
    return {
        id: "session-1",
        room_id: "room-1",
        started_by: "user-viewer",
        controller_id: "user-viewer",
        title: "Chiru rewatch",
        type: "hyperbeam",
        status: "active",
        started_at: "2026-08-01T10:00:00Z",
        participants: [],
        ...overrides,
    };
}

export function makeHomeActivity(overrides: Partial<HomeActivityResponse> = {}): HomeActivityResponse {
    return {
        online_count: 5,
        recent_activity: [],
        echoes: [],
        recent_members: [],
        public_rooms: [],
        corner_activity: [],
        ...overrides,
    };
}
