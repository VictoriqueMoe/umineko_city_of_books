import { sendRaw } from "./socket";

export const REALTIME_COMMANDS = {
    PING: "ping",
    JOIN_ROOM: "join_room",
    LEAVE_ROOM: "leave_room",
    TYPING: "typing",
    VIEWER_STATE: "viewer_state",
    SECRET_JOIN: "secret_join",
    SECRET_LEAVE: "secret_leave",
    MYSTERY_JOIN: "mystery_join",
    MYSTERY_LEAVE: "mystery_leave",
    GAME_ROOM_JOIN: "game_room_join",
    GAME_ROOM_LEAVE: "game_room_leave",
    GAME_ROOM_INPUT: "game_room_input",
    WATCH_POSTS: "watch_posts",
} as const satisfies Record<string, RealtimeCommandName>;

export interface PingPayload {
    nonce: number;
    rtt?: number;
}

export interface RoomActionPayload {
    room_id: string;
}

export type JoinRoomPayload = RoomActionPayload;

export type LeaveRoomPayload = RoomActionPayload;

export type TypingCommandPayload = RoomActionPayload;

export type ViewerState = "active" | "idle";

export interface ViewerStatePayload {
    room_id: string;
    state: ViewerState;
}

export interface SecretTopicPayload {
    secret_id: string;
}

export type SecretJoinPayload = SecretTopicPayload;

export type SecretLeavePayload = SecretTopicPayload;

export interface MysteryTopicPayload {
    mystery_id: string;
}

export type MysteryJoinPayload = MysteryTopicPayload;

export type MysteryLeavePayload = MysteryTopicPayload;

export type GameRoomJoinPayload = RoomActionPayload;

export type GameRoomLeavePayload = RoomActionPayload;

export interface GameRoomInputPayload<P = unknown> {
    room_id: string;
    payload: P;
}

export interface WatchPostsPayload {
    post_ids: string[];
}

export type RealtimeCommand =
    | { type: "ping"; data: PingPayload }
    | { type: "join_room"; data: JoinRoomPayload }
    | { type: "leave_room"; data: LeaveRoomPayload }
    | { type: "typing"; data: TypingCommandPayload }
    | { type: "viewer_state"; data: ViewerStatePayload }
    | { type: "secret_join"; data: SecretJoinPayload }
    | { type: "secret_leave"; data: SecretLeavePayload }
    | { type: "mystery_join"; data: MysteryJoinPayload }
    | { type: "mystery_leave"; data: MysteryLeavePayload }
    | { type: "game_room_join"; data: GameRoomJoinPayload }
    | { type: "game_room_leave"; data: GameRoomLeavePayload }
    | { type: "game_room_input"; data: GameRoomInputPayload }
    | { type: "watch_posts"; data: WatchPostsPayload };

export type RealtimeCommandName = RealtimeCommand["type"];

export type RealtimeCommandOf<K extends RealtimeCommandName> = Extract<RealtimeCommand, { type: K }>;

export type RealtimeCommandPayload<K extends RealtimeCommandName> = RealtimeCommandOf<K>["data"];

export function sendRealtime(command: RealtimeCommand): void {
    sendRaw(JSON.stringify(command));
}
