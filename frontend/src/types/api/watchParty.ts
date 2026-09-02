import type { User } from "./common";

export interface WatchPartyParticipant {
    user: User;
    has_control: boolean;
    joined_at: string;
}

export interface WatchPartyViewerContext {
    is_participant: boolean;
    has_control: boolean;
    embed_url?: string;
}

export type WatchPartyType = "hyperbeam" | "screenshare";

export interface WatchPartySession {
    id: string;
    room_id: string;
    started_by: string;
    controller_id: string;
    title: string;
    type: WatchPartyType;
    start_url?: string;
    region?: string;
    status: "active" | "ended" | "expired";
    started_at: string;
    ended_at?: string;
    participants: WatchPartyParticipant[];
    viewer?: WatchPartyViewerContext;
}

export interface WatchPartyListResponse {
    sessions: WatchPartySession[];
    enabled: boolean;
    screen_share_enabled: boolean;
}

export interface StartWatchPartyResponse {
    session: WatchPartySession;
    embed_url: string;
}

export interface JoinWatchPartyResponse {
    session: WatchPartySession;
    embed_url: string;
}

export interface WatchPartyStartedEvent {
    session: WatchPartySession;
}

export interface WatchPartyEndedEvent {
    session_id: string;
    room_id: string;
    reason: string;
}

export interface WatchPartyParticipantEvent {
    session_id: string;
    room_id: string;
    participant: WatchPartyParticipant;
}

export interface WatchPartyParticipantLeftEvent {
    session_id: string;
    room_id: string;
    user_id: string;
}

export interface WatchPartyControlChangedEvent {
    session_id: string;
    room_id: string;
    user_id: string;
    has_control: boolean;
}

export interface WatchPartyKickedEvent {
    session_id: string;
    room_id: string;
    actor_id: string;
    reason?: string;
}
