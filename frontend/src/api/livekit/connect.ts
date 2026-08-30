import type { Room } from "livekit-client";
import { nonFatal } from "../../utils/nonFatal";

export interface RoomEventHandlers {
    onConnected?: (room: Room) => void;
    onDisconnected?: (room: Room) => void;
    onParticipantConnected?: (room: Room) => void;
    onParticipantDisconnected?: (room: Room) => void;
    onLocalPermissionsChanged?: (room: Room) => void;
}

export interface ConnectRoomOptions {
    url: string;
    token: string;
    autoSubscribe?: boolean;
    signal?: AbortSignal;
    on?: RoomEventHandlers;
}

export async function connectRoom(options: ConnectRoomOptions): Promise<Room | null> {
    const { Room: LiveKitRoom, RoomEvent } = await import("livekit-client");

    if (options.signal?.aborted) {
        return null;
    }

    const room = new LiveKitRoom();
    const handlers = options.on ?? {};

    const onConnected = handlers.onConnected;
    if (onConnected) {
        room.on(RoomEvent.Connected, () => onConnected(room));
    }

    const onDisconnected = handlers.onDisconnected;
    if (onDisconnected) {
        room.on(RoomEvent.Disconnected, () => onDisconnected(room));
    }

    const onParticipantConnected = handlers.onParticipantConnected;
    if (onParticipantConnected) {
        room.on(RoomEvent.ParticipantConnected, () => onParticipantConnected(room));
    }

    const onParticipantDisconnected = handlers.onParticipantDisconnected;
    if (onParticipantDisconnected) {
        room.on(RoomEvent.ParticipantDisconnected, () => onParticipantDisconnected(room));
    }

    const onLocalPermissionsChanged = handlers.onLocalPermissionsChanged;
    if (onLocalPermissionsChanged) {
        room.on(RoomEvent.ParticipantPermissionsChanged, (_previous, participant) => {
            if (participant !== room.localParticipant) {
                return;
            }

            onLocalPermissionsChanged(room);
        });
    }

    const connectOptions = options.autoSubscribe === undefined ? undefined : { autoSubscribe: options.autoSubscribe };

    await room.connect(options.url, options.token, connectOptions);

    if (options.signal?.aborted) {
        disconnectRoom(room);
        return null;
    }

    return room;
}

export function disconnectRoom(room: Room | null | undefined): void {
    if (!room) {
        return;
    }

    room.disconnect().catch(nonFatal);
}
